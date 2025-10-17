package websocket

import (
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// Conn wraps a WebSocket connection to implement net.Conn interface
// This allows WebSocket connections to be treated like regular TCP connections
// by the IRC server
type Conn struct {
	ws         *websocket.Conn
	reader     io.Reader
	remoteAddr net.Addr
	localAddr  net.Addr
}

// NewConn creates a new WebSocket connection wrapper
func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{
		ws:         ws,
		remoteAddr: ws.RemoteAddr(),
		localAddr:  ws.LocalAddr(),
	}
}

// Read implements net.Conn interface
// Reads IRC protocol text from WebSocket text messages
func (c *Conn) Read(b []byte) (int, error) {
	// If we have a reader from a previous message, use it
	if c.reader != nil {
		n, err := c.reader.Read(b)
		if err == io.EOF {
			// Message fully read, clear reader for next message
			c.reader = nil
			return n, nil
		}
		return n, err
	}

	// Read next message from WebSocket
	msgType, reader, err := c.ws.NextReader()
	if err != nil {
		return 0, err
	}

	// Only accept text messages for IRC protocol
	if msgType != websocket.TextMessage {
		return 0, io.EOF
	}

	c.reader = reader
	return c.reader.Read(b)
}

// Write implements net.Conn interface
// Writes IRC protocol text as WebSocket text messages
func (c *Conn) Write(b []byte) (int, error) {
	err := c.ws.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close implements net.Conn interface
func (c *Conn) Close() error {
	// Send close message
	err := c.ws.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	if err != nil {
		return c.ws.Close()
	}
	return c.ws.Close()
}

// LocalAddr implements net.Conn interface
func (c *Conn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr implements net.Conn interface
func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline implements net.Conn interface
func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.ws.SetReadDeadline(t); err != nil {
		return err
	}
	return c.ws.SetWriteDeadline(t)
}

// SetReadDeadline implements net.Conn interface
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn interface
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}
