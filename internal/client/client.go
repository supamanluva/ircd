package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/supamanluva/ircd/internal/logger"
	"github.com/supamanluva/ircd/internal/security"
)

// ConnectionType represents the type of client connection
type ConnectionType int

const (
	TCP ConnectionType = iota
	WebSocket
)

// Client represents a connected IRC client
type Client struct {
	conn           net.Conn
	nickname       string
	username       string
	realname       string
	hostname       string
	uid            string          // Unique ID for server linking (TS6 format: SIDAAAAAA)
	registered     bool
	channels       map[string]bool // channel names the client has joined
	modes          map[rune]bool   // user modes (o=operator, i=invisible, etc.)
	awayMessage    string          // away message (empty if not away)
	connType       ConnectionType
	lastActivity   time.Time
	lastPing       time.Time
	connectTime    time.Time       // When client connected
	mu             sync.RWMutex
	logger         *logger.Logger
	sendQueue      chan string
	disconnected   bool
	rateLimiter    *security.RateLimiter
}

// New creates a new client instance
func New(conn net.Conn, log *logger.Logger) *Client {
	c := &Client{
		conn:         conn,
		hostname:     conn.RemoteAddr().String(),
		channels:     make(map[string]bool),
		modes:        make(map[rune]bool),
		connType:     TCP,
		lastActivity: time.Now(),
		lastPing:     time.Now(),
		connectTime:  time.Now(),
		logger:       log,
		sendQueue:    make(chan string, 100),
		disconnected: false,
		rateLimiter:  security.NewRateLimiter(5.0, 10.0), // 5 msg/sec, burst of 10
	}

	// Start send worker
	go c.sendWorker()

	return c
}

// Send queues a message to be sent to the client
func (c *Client) Send(message string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.disconnected {
		return
	}

	select {
	case c.sendQueue <- message:
	default:
		c.logger.Warn("Send queue full, dropping message", "client", c.nickname)
	}
}

// sendWorker handles sending messages to the client
func (c *Client) sendWorker() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Panic in send worker", "error", r)
		}
	}()

	for msg := range c.sendQueue {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err := fmt.Fprintf(c.conn, "%s\r\n", msg)
		if err != nil {
			c.logger.Error("Failed to send message", "error", err, "client", c.nickname)
			return
		}
	}
}

// Receive reads messages from the client
func (c *Client) Receive() (string, error) {
	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()

	reader := bufio.NewReader(c.conn)
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove trailing \r\n
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return line, nil
}

// SetNickname sets the client's nickname
func (c *Client) SetNickname(nick string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nickname = nick
}

// GetNickname returns the client's nickname
func (c *Client) GetNickname() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nickname
}

// SetUsername sets the client's username and realname
func (c *Client) SetUsername(username, realname string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.username = username
	c.realname = realname
}

// HasUsername returns whether the client has set a username
func (c *Client) HasUsername() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.username != ""
}

// IsRegistered returns whether the client has completed registration
func (c *Client) IsRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

// SetRegistered marks the client as registered
func (c *Client) SetRegistered(registered bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.registered = registered
}

// JoinChannel adds a channel to the client's channel list
func (c *Client) JoinChannel(channelName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channelName] = true
}

// PartChannel removes a channel from the client's channel list
func (c *Client) PartChannel(channelName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channelName)
}

// GetChannels returns the list of channels the client is in
func (c *Client) GetChannels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	channels := make([]string, 0, len(c.channels))
	for ch := range c.channels {
		channels = append(channels, ch)
	}
	return channels
}

// Disconnect closes the client connection
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.disconnected {
		return
	}

	c.disconnected = true
	close(c.sendQueue)
	c.conn.Close()
}

// GetHostmask returns the client's hostmask (nick!user@host)
func (c *Client) GetHostmask() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("%s!%s@%s", c.nickname, c.username, c.hostname)
}

// CheckRateLimit checks if the client is within rate limits
func (c *Client) CheckRateLimit() bool {
	return c.rateLimiter.Allow()
}

// GetLastActivity returns the time of last activity
func (c *Client) GetLastActivity() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActivity
}

// GetLastPing returns the time of last PING
func (c *Client) GetLastPing() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPing
}

// UpdatePingTime updates the last ping time
func (c *Client) UpdatePingTime() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastPing = time.Now()
}

// IsIdle checks if the client has been idle for too long
func (c *Client) IsIdle(timeout time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// For WebSocket clients, check both lastActivity and lastPing
	if c.connType == WebSocket {
		idle := time.Since(c.lastActivity) > timeout && time.Since(c.lastPing) > timeout
		return idle
	}
	return time.Since(c.lastActivity) > timeout
}

// NeedsPing checks if the client needs a PING
func (c *Client) NeedsPing(interval time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastPing) > interval
}

// SetMode sets a user mode
func (c *Client) SetMode(mode rune, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if enabled {
		c.modes[mode] = true
	} else {
		delete(c.modes, mode)
	}
}

// HasMode checks if a user has a specific mode
func (c *Client) HasMode(mode rune) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.modes[mode]
}

// GetModes returns a string representation of user modes
func (c *Client) GetModes() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	modes := ""
	for mode := range c.modes {
		modes += string(mode)
	}
	if modes == "" {
		return ""
	}
	return "+" + modes
}

// IsServerOperator checks if the client is a server operator
func (c *Client) IsServerOperator() bool {
	return c.HasMode('o')
}

// SetAway sets the away message (empty string = not away)
func (c *Client) SetAway(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.awayMessage = message
}

// GetAwayMessage returns the away message
func (c *Client) GetAwayMessage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.awayMessage
}

// IsAway checks if the client is marked as away
func (c *Client) IsAway() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.awayMessage != ""
}

// SetUID sets the client's unique ID (for server linking)
func (c *Client) SetUID(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.uid = uid
}

// GetUID returns the client's unique ID
func (c *Client) GetUID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.uid
}

// GetUsername returns the client's username
func (c *Client) GetUsername() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.username
}

// GetRealname returns the client's real name
func (c *Client) GetRealname() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.realname
}

// GetHostname returns the client's hostname
func (c *Client) GetHostname() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hostname
}

// GetIP returns the client's IP address
func (c *Client) GetIP() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Extract IP from hostname (which is set to RemoteAddr)
	if c.conn != nil {
		if addr, ok := c.conn.RemoteAddr().(*net.TCPAddr); ok {
			return addr.IP.String()
		}
	}
	return c.hostname
}

// GetConnectTime returns when the client connected
func (c *Client) GetConnectTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connectTime
}

