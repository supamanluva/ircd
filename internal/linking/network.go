package linking

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Server represents a linked IRC server in the network
type Server struct {
	SID         string           // Server ID (3 chars: 0AA, 1BB, etc)
	Name        string           // Server name (hub1.example.net)
	Description string           // Server description
	Conn        net.Conn        // Network connection
	IsHub       bool            // Can this server link other servers?
	Uplink      *Server         // Parent server (nil if direct link)
	Downlinks   []*Server       // Child servers
	Distance    int             // Hops from local server
	Users       map[string]*RemoteUser   // UID -> User
	Channels    map[string]*RemoteChannel // Channel name -> Channel
	LastPing    time.Time
	LastPong    time.Time
	Version     string
	Capabilities []string       // Server capabilities (ENCAP, KLN, etc)
	mu          sync.RWMutex
}

// RemoteUser represents a user on a remote server
type RemoteUser struct {
	UID        string    // Unique ID (SID + 6 chars)
	Nick       string    // Current nickname
	User       string    // Username
	Host       string    // Hostname
	IP         string    // IP address
	RealName   string    // Real name
	Server     *Server   // Which server the user is on
	Modes      string    // User modes (+i, +o, etc)
	Away       string    // Away message (empty if not away)
	Channels   map[string]bool // Channels user is in
	Timestamp  int64     // Nick timestamp
	mu         sync.RWMutex
}

// RemoteChannel represents a channel state across the network
type RemoteChannel struct {
	Name       string              // Channel name
	TS         int64              // Channel timestamp
	Modes      string             // Channel modes
	Key        string             // Channel key (+k)
	Limit      int                // User limit (+l)
	Topic      string             // Channel topic
	TopicTime  int64              // When topic was set
	TopicBy    string             // Who set the topic (nick!user@host)
	Members    map[string]string  // UID -> modes (@, +, etc)
	Bans       []string           // Ban masks
	mu         sync.RWMutex
}

// Network represents the entire IRC network
type Network struct {
	LocalSID   string                    // Our server's SID
	LocalName  string                    // Our server's name
	Servers    map[string]*Server        // SID -> Server
	Users      map[string]*RemoteUser    // UID -> User
	Channels   map[string]*RemoteChannel // Name -> Channel
	NickToUID  map[string]string         // Nick -> UID (for lookups)
	UIDCounter uint32                    // Counter for generating UIDs
	mu         sync.RWMutex
}

// NewNetwork creates a new network manager
func NewNetwork(sid, name string) *Network {
	return &Network{
		LocalSID:   sid,
		LocalName:  name,
		Servers:    make(map[string]*Server),
		Users:      make(map[string]*RemoteUser),
		Channels:   make(map[string]*RemoteChannel),
		NickToUID:  make(map[string]string),
		UIDCounter: 0,
	}
}

// GenerateUID generates a unique user ID for the local server
func (n *Network) GenerateUID() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	n.UIDCounter++
	
	// Format: SID + 6 alphanumeric chars (AAAAAA - ZZZZZZ)
	uid := fmt.Sprintf("%s%06s", n.LocalSID, encodeBase36(n.UIDCounter))
	return uid
}

// encodeBase36 converts a number to base36 (0-9A-Z)
func encodeBase36(n uint32) string {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if n == 0 {
		return "000000"
	}
	
	result := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		result[i] = chars[n%36]
		n /= 36
	}
	return string(result)
}

// AddServer adds a server to the network
func (n *Network) AddServer(srv *Server) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if _, exists := n.Servers[srv.SID]; exists {
		return fmt.Errorf("server with SID %s already exists", srv.SID)
	}
	
	n.Servers[srv.SID] = srv
	return nil
}

// RemoveServer removes a server and all its users from the network
func (n *Network) RemoveServer(sid string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	srv, exists := n.Servers[sid]
	if !exists {
		return
	}
	
	// Remove all users from this server
	for uid, user := range n.Users {
		if user.Server.SID == sid {
			// Remove from nick lookup
			delete(n.NickToUID, user.Nick)
			
			// Remove from channels
			for chanName := range user.Channels {
				if ch, ok := n.Channels[chanName]; ok {
					ch.mu.Lock()
					delete(ch.Members, uid)
					ch.mu.Unlock()
				}
			}
			
			delete(n.Users, uid)
		}
	}
	
	// Remove all downlink servers recursively
	for _, downlink := range srv.Downlinks {
		n.RemoveServer(downlink.SID)
	}
	
	delete(n.Servers, sid)
}

// AddUser adds a remote user to the network
func (n *Network) AddUser(user *RemoteUser) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if _, exists := n.Users[user.UID]; exists {
		return fmt.Errorf("user with UID %s already exists", user.UID)
	}
	
	// Check for nick collision
	if existingUID, exists := n.NickToUID[user.Nick]; exists {
		// Nick collision - need to resolve
		existingUser := n.Users[existingUID]
		
		// Lower timestamp wins
		if user.Timestamp < existingUser.Timestamp {
			// New user wins, rename existing
			delete(n.NickToUID, existingUser.Nick)
			// Existing user will be renamed by their server
		} else {
			// Existing user wins, reject new user
			return fmt.Errorf("nick collision: %s already exists with older timestamp", user.Nick)
		}
	}
	
	n.Users[user.UID] = user
	n.NickToUID[user.Nick] = user.UID
	return nil
}

// RemoveUser removes a user from the network
func (n *Network) RemoveUser(uid string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	user, exists := n.Users[uid]
	if !exists {
		return
	}
	
	// Remove from nick lookup
	delete(n.NickToUID, user.Nick)
	
	// Remove from all channels
	for chanName := range user.Channels {
		if ch, ok := n.Channels[chanName]; ok {
			ch.mu.Lock()
			delete(ch.Members, uid)
			
			// Remove channel if empty
			if len(ch.Members) == 0 {
				delete(n.Channels, chanName)
			}
			ch.mu.Unlock()
		}
	}
	
	delete(n.Users, uid)
}

// UpdateNick updates a user's nickname
func (n *Network) UpdateNick(uid, newNick string, ts int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	user, exists := n.Users[uid]
	if !exists {
		return fmt.Errorf("user %s not found", uid)
	}
	
	// Check for nick collision
	if existingUID, exists := n.NickToUID[newNick]; exists && existingUID != uid {
		existingUser := n.Users[existingUID]
		
		// Timestamp-based resolution
		if ts < existingUser.Timestamp {
			// New nick wins
			delete(n.NickToUID, existingUser.Nick)
			// Existing user will be renamed
		} else {
			// Existing user wins
			return fmt.Errorf("nick collision: %s already taken", newNick)
		}
	}
	
	// Update nick
	delete(n.NickToUID, user.Nick)
	user.Nick = newNick
	user.Timestamp = ts
	n.NickToUID[newNick] = uid
	
	return nil
}

// GetUserByNick finds a user by nickname
func (n *Network) GetUserByNick(nick string) (*RemoteUser, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	if uid, ok := n.NickToUID[nick]; ok {
		user, exists := n.Users[uid]
		return user, exists
	}
	return nil, false
}

// GetUserByUID finds a user by UID
func (n *Network) GetUserByUID(uid string) (*RemoteUser, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	user, exists := n.Users[uid]
	return user, exists
}

// GetServer finds a server by SID
func (n *Network) GetServer(sid string) (*Server, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	srv, exists := n.Servers[sid]
	return srv, exists
}

// GetServerByName finds a server by name (Phase 7.4.5)
func (n *Network) GetServerByName(name string) *Server {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	for _, srv := range n.Servers {
		if srv.Name == name {
			return srv
		}
	}
	return nil
}

// GetUsersBySID returns all users from a specific server (Phase 7.4.5)
func (n *Network) GetUsersBySID(sid string) []*RemoteUser {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	users := make([]*RemoteUser, 0)
	for _, user := range n.Users {
		// UID format is SID + 6 chars, so check if UID starts with the SID
		if len(user.UID) >= len(sid) && user.UID[:len(sid)] == sid {
			users = append(users, user)
		}
	}
	return users
}

// AddChannel adds or updates a channel in the network
func (n *Network) AddChannel(ch *RemoteChannel) {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	existing, exists := n.Channels[ch.Name]
	if exists {
		// Channel already exists - need to merge based on TS
		existing.mu.Lock()
		defer existing.mu.Unlock()
		
		if ch.TS < existing.TS {
			// New channel state is older, it wins
			existing.TS = ch.TS
			existing.Modes = ch.Modes
			existing.Key = ch.Key
			existing.Limit = ch.Limit
			// Clear ops/voices - new state is authoritative
			for uid := range existing.Members {
				existing.Members[uid] = ""
			}
		} else if ch.TS == existing.TS {
			// Same timestamp, merge members
			for uid, modes := range ch.Members {
				existing.Members[uid] = modes
			}
		}
		// If ch.TS > existing.TS, ignore new state (ours is older)
	} else {
		// New channel
		n.Channels[ch.Name] = ch
	}
}

// GetChannel finds a channel by name
func (n *Network) GetChannel(name string) (*RemoteChannel, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	ch, exists := n.Channels[name]
	return ch, exists
}

// GetServerCount returns the number of linked servers
func (n *Network) GetServerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.Servers)
}

// GetUserCount returns the total number of users across the network
func (n *Network) GetUserCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.Users)
}

// GetChannelCount returns the number of channels
func (n *Network) GetChannelCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.Channels)
}

// ValidateSID checks if a SID is valid format
func ValidateSID(sid string) bool {
	if len(sid) != 3 {
		return false
	}
	
	// First char must be digit
	if sid[0] < '0' || sid[0] > '9' {
		return false
	}
	
	// Last two chars must be alphanumeric
	for i := 1; i < 3; i++ {
		c := sid[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	
	return true
}

// ValidateUID checks if a UID is valid format
func ValidateUID(uid string) bool {
	if len(uid) != 9 {
		return false
	}
	
	// First 3 chars are SID
	if !ValidateSID(uid[:3]) {
		return false
	}
	
	// Last 6 chars are alphanumeric
	for i := 3; i < 9; i++ {
		c := uid[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	
	return true
}

// GetServerSIDs returns a slice of all connected server SIDs
func (n *Network) GetServerSIDs() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	sids := make([]string, 0, len(n.Servers))
	for sid := range n.Servers {
		sids = append(sids, sid)
	}
	return sids
}
