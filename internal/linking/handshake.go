package linking

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// LinkState represents the state of a server link connection
type LinkState int

const (
	LinkStateConnected   LinkState = iota // Initial connection
	LinkStatePassRecv                     // Received PASS
	LinkStateCapabRecv                    // Received CAPAB
	LinkStateServerRecv                   // Received SERVER
	LinkStateRegistered                   // Fully registered and ready
)

// Link represents an active server-to-server connection
type Link struct {
	conn           net.Conn
	server         *Server
	state          LinkState
	reader         *bufio.Reader
	writer         *bufio.Writer
	mu             sync.RWMutex
	receivedPass   bool
	receivedCapab  bool
	receivedServer bool
	receivedSVINFO bool
	remoteSID      string
	remoteName     string
	remotePass     string
	capabilities   []string
	closeOnce      sync.Once
	closed         chan struct{}
}

// NewLink creates a new Link for a connection
func NewLink(conn net.Conn) *Link {
	return &Link{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		state:  LinkStateConnected,
		closed: make(chan struct{}),
	}
}

// ReadMessage reads a protocol message from the link
func (l *Link) ReadMessage() (*Message, error) {
	line, err := l.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	
	// Trim newline
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	
	return ParseMessage(line)
}

// WriteMessage sends a protocol message to the link
func (l *Link) WriteMessage(msg *Message) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	line := msg.String() + "\r\n"
	_, err := l.writer.WriteString(line)
	if err != nil {
		return err
	}
	
	return l.writer.Flush()
}

// Close closes the link connection
func (l *Link) Close() error {
	var err error
	l.closeOnce.Do(func() {
		close(l.closed)
		err = l.conn.Close()
	})
	return err
}

// IsClosed returns true if the link is closed
func (l *Link) IsClosed() bool {
	select {
	case <-l.closed:
		return true
	default:
		return false
	}
}

// RemoteAddr returns the remote address
func (l *Link) RemoteAddr() string {
	return l.conn.RemoteAddr().String()
}

// HandshakeServer performs the server side of the handshake
// (receiving incoming connection)
func (l *Link) HandshakeServer(network *Network, password string) error {
	// Expect: PASS, CAPAB, SERVER, SVINFO from remote
	
	for {
		msg, err := l.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error during handshake: %v", err)
		}
		
		switch msg.Command {
		case "PASS":
			if l.receivedPass {
				return fmt.Errorf("duplicate PASS command")
			}
			
			remotePass, remoteSID, version, err := ParsePASS(msg)
			if err != nil {
				return fmt.Errorf("invalid PASS: %v", err)
			}
			
			// Validate password
			if remotePass != password {
				l.WriteMessage(BuildERROR("Invalid password"))
				return fmt.Errorf("authentication failed")
			}
			
			// Check TS version compatibility
			if version < MinTSVersion {
				return fmt.Errorf("incompatible TS version: %d (need >= %d)", version, MinTSVersion)
			}
			
			// Check SID doesn't conflict
			if remoteSID == network.LocalSID {
				return fmt.Errorf("SID conflict: %s already in use", remoteSID)
			}
			
			if _, exists := network.GetServer(remoteSID); exists {
				return fmt.Errorf("SID %s already linked", remoteSID)
			}
			
			l.remoteSID = remoteSID
			l.receivedPass = true
			l.state = LinkStatePassRecv
			
		case "CAPAB":
			if !l.receivedPass {
				return fmt.Errorf("CAPAB before PASS")
			}
			if l.receivedCapab {
				return fmt.Errorf("duplicate CAPAB command")
			}
			
			caps, err := ParseCAPAB(msg)
			if err != nil {
				return fmt.Errorf("invalid CAPAB: %v", err)
			}
			
			l.capabilities = caps
			l.receivedCapab = true
			l.state = LinkStateCapabRecv
			
		case "SERVER":
			if !l.receivedPass || !l.receivedCapab {
				return fmt.Errorf("SERVER before PASS/CAPAB")
			}
			if l.receivedServer {
				return fmt.Errorf("duplicate SERVER command")
			}
			
			name, _, description, err := ParseSERVER(msg)
			if err != nil {
				return fmt.Errorf("invalid SERVER: %v", err)
			}
			
			l.remoteName = name
			l.receivedServer = true
			l.state = LinkStateServerRecv
			
			// Create Server object
			l.server = &Server{
				SID:          l.remoteSID,
				Name:         name,
				Description:  description,
				Conn:         l.conn,
				IsHub:        false, // Determined by config
				Distance:     1,
				Users:        make(map[string]*RemoteUser),
				Channels:     make(map[string]*RemoteChannel),
				Capabilities: l.capabilities,
			}
			
			// Send our handshake
			if err := l.SendHandshake(network, password); err != nil {
				return fmt.Errorf("failed to send handshake: %v", err)
			}
			
		case "SVINFO":
			if !l.receivedServer {
				return fmt.Errorf("SVINFO before SERVER")
			}
			if l.receivedSVINFO {
				return fmt.Errorf("duplicate SVINFO command")
			}
			
			tsVersion, minVersion, serverTime, err := ParseSVINFO(msg)
			if err != nil {
				return fmt.Errorf("invalid SVINFO: %v", err)
			}
			
			// Verify version compatibility
			if tsVersion < MinTSVersion || minVersion > TS6Version {
				return fmt.Errorf("incompatible TS versions: %d/%d", tsVersion, minVersion)
			}
			
			// Check time delta (warn if > 60 seconds)
			timeDelta := time.Now().Unix() - serverTime
			if timeDelta < -60 || timeDelta > 60 {
				// Just a warning, not fatal
				fmt.Printf("WARNING: Server %s time delta: %d seconds\n", l.remoteName, timeDelta)
			}
			
			l.receivedSVINFO = true
			l.state = LinkStateRegistered
			
			// Handshake complete!
			return nil
			
		case "ERROR":
			reason := ""
			if len(msg.Params) > 0 {
				reason = msg.Params[0]
			}
			return fmt.Errorf("remote error: %s", reason)
			
		default:
			return fmt.Errorf("unexpected command during handshake: %s", msg.Command)
		}
	}
}

// HandshakeClient performs the client side of the handshake
// (initiating outbound connection)
func (l *Link) HandshakeClient(network *Network, password, remoteSID, remoteName string) error {
	// Send: PASS, CAPAB, SERVER, SVINFO
	
	// Send PASS
	passMsg := BuildPASS(password, network.LocalSID)
	if err := l.WriteMessage(passMsg); err != nil {
		return fmt.Errorf("failed to send PASS: %v", err)
	}
	
	// Send CAPAB
	capabMsg := BuildCAPAB(DefaultCapabilities)
	if err := l.WriteMessage(capabMsg); err != nil {
		return fmt.Errorf("failed to send CAPAB: %v", err)
	}
	
	// Send SERVER
	serverMsg := BuildSERVER(network.LocalName, 1, "IRC Server")
	if err := l.WriteMessage(serverMsg); err != nil {
		return fmt.Errorf("failed to send SERVER: %v", err)
	}
	
	// Send SVINFO
	svinfoMsg := BuildSVINFO()
	if err := l.WriteMessage(svinfoMsg); err != nil {
		return fmt.Errorf("failed to send SVINFO: %v", err)
	}
	
	// Now receive their handshake
	for {
		msg, err := l.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error during handshake: %v", err)
		}
		
		switch msg.Command {
		case "PASS":
			if l.receivedPass {
				return fmt.Errorf("duplicate PASS command")
			}
			
			_, gotSID, version, err := ParsePASS(msg)
			if err != nil {
				return fmt.Errorf("invalid PASS: %v", err)
			}
			
			if gotSID != remoteSID {
				return fmt.Errorf("SID mismatch: got %s, expected %s", gotSID, remoteSID)
			}
			
			if version < MinTSVersion {
				return fmt.Errorf("incompatible TS version: %d", version)
			}
			
			l.remoteSID = gotSID
			l.receivedPass = true
			
		case "CAPAB":
			if !l.receivedPass {
				return fmt.Errorf("CAPAB before PASS")
			}
			
			caps, err := ParseCAPAB(msg)
			if err != nil {
				return fmt.Errorf("invalid CAPAB: %v", err)
			}
			
			l.capabilities = caps
			l.receivedCapab = true
			
		case "SERVER":
			if !l.receivedPass || !l.receivedCapab {
				return fmt.Errorf("SERVER before PASS/CAPAB")
			}
			
			name, _, description, err := ParseSERVER(msg)
			if err != nil {
				return fmt.Errorf("invalid SERVER: %v", err)
			}
			
			if name != remoteName {
				return fmt.Errorf("server name mismatch: got %s, expected %s", name, remoteName)
			}
			
			l.remoteName = name
			l.receivedServer = true
			
			// Create Server object
			l.server = &Server{
				SID:          l.remoteSID,
				Name:         name,
				Description:  description,
				Conn:         l.conn,
				IsHub:        false,
				Distance:     1,
				Users:        make(map[string]*RemoteUser),
				Channels:     make(map[string]*RemoteChannel),
				Capabilities: l.capabilities,
			}
			
		case "SVINFO":
			if !l.receivedServer {
				return fmt.Errorf("SVINFO before SERVER")
			}
			
			tsVersion, minVersion, _, err := ParseSVINFO(msg)
			if err != nil {
				return fmt.Errorf("invalid SVINFO: %v", err)
			}
			
			if tsVersion < MinTSVersion || minVersion > TS6Version {
				return fmt.Errorf("incompatible TS versions: %d/%d", tsVersion, minVersion)
			}
			
			l.receivedSVINFO = true
			l.state = LinkStateRegistered
			
			// Handshake complete!
			return nil
			
		case "ERROR":
			reason := ""
			if len(msg.Params) > 0 {
				reason = msg.Params[0]
			}
			return fmt.Errorf("remote error: %s", reason)
			
		default:
			return fmt.Errorf("unexpected command during handshake: %s", msg.Command)
		}
	}
}

// SendHandshake sends our handshake to the remote server
func (l *Link) SendHandshake(network *Network, password string) error {
	// Send PASS
	passMsg := BuildPASS(password, network.LocalSID)
	if err := l.WriteMessage(passMsg); err != nil {
		return err
	}
	
	// Send CAPAB
	capabMsg := BuildCAPAB(DefaultCapabilities)
	if err := l.WriteMessage(capabMsg); err != nil {
		return err
	}
	
	// Send SERVER
	serverMsg := BuildSERVER(network.LocalName, 1, "IRC Server")
	if err := l.WriteMessage(serverMsg); err != nil {
		return err
	}
	
	// Send SVINFO
	svinfoMsg := BuildSVINFO()
	if err := l.WriteMessage(svinfoMsg); err != nil {
		return err
	}
	
	return nil
}

// IsRegistered returns true if the link is fully registered
func (l *Link) IsRegistered() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state == LinkStateRegistered
}

// GetServer returns the Server object for this link
func (l *Link) GetServer() *Server {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.server
}
