package client

import (
	"net"

	"github.com/supamanluva/ircd/internal/logger"
)

// NewMock creates a mock client for testing (without a real connection)
func NewMock(log *logger.Logger) *Client {
	return &Client{
		conn:         nil, // No real connection
		hostname:     "test.host",
		channels:     make(map[string]bool),
		modes:        make(map[rune]bool),
		connType:     TCP,
		logger:       log,
		sendQueue:    make(chan string, 100),
		disconnected: false,
	}
}

// SetConn sets the connection for testing purposes
func (c *Client) SetConn(conn net.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn = conn
}
