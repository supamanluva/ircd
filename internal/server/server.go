package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/supamanluva/ircd/internal/channel"
	"github.com/supamanluva/ircd/internal/client"
	"github.com/supamanluva/ircd/internal/commands"
	"github.com/supamanluva/ircd/internal/logger"
	"github.com/supamanluva/ircd/internal/parser"
)

// Config holds server configuration
type Config struct {
	ServerName   string
	Host         string
	Port         int
	MaxClients   int
	TLSEnabled   bool
	TLSPort      int
	TLSCertFile  string
	TLSKeyFile   string
	PingInterval time.Duration
	Timeout      time.Duration
	Operators    []Operator // Server operators for OPER command
}

// Operator represents a server operator
type Operator struct {
	Name     string
	Password string // bcrypt hashed password
}

// Server represents the IRC server
type Server struct {
	config      *Config
	logger      *logger.Logger
	listener    net.Listener
	tlsListener net.Listener
	clients     map[string]*client.Client  // nickname -> client
	clientsAddr map[string]*client.Client  // address -> client
	channels    map[string]*channel.Channel
	mu          sync.RWMutex
	shutdown    chan struct{}
	handler     *commands.Handler
}

// GetClient returns a client by nickname
func (s *Server) GetClient(nickname string) *client.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[nickname]
}

// AddClient adds a client to the registry
func (s *Server) AddClient(c *client.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	nick := c.GetNickname()
	if nick == "" {
		return fmt.Errorf("client has no nickname")
	}
	
	if _, exists := s.clients[nick]; exists {
		return fmt.Errorf("nickname already in use")
	}
	
	s.clients[nick] = c
	return nil
}

// RemoveClient removes a client from the registry
func (s *Server) RemoveClient(c *client.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	nick := c.GetNickname()
	if nick != "" {
		delete(s.clients, nick)
	}
}

// IsNicknameInUse checks if a nickname is already taken
func (s *Server) IsNicknameInUse(nickname string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.clients[nickname]
	return exists
}

// GetChannel returns a channel by name
func (s *Server) GetChannel(name string) *channel.Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.channels[name]
}

// CreateChannel creates a new channel or returns existing one
func (s *Server) CreateChannel(name string) *channel.Channel {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if channel already exists
	if ch, exists := s.channels[name]; exists {
		return ch
	}
	
	// Create new channel
	ch := channel.New(name)
	s.channels[name] = ch
	s.logger.Info("Channel created", "channel", name)
	return ch
}

// RemoveChannel removes a channel if it's empty
func (s *Server) RemoveChannel(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if ch, exists := s.channels[name]; exists {
		if ch.IsEmpty() {
			delete(s.channels, name)
			s.logger.Info("Channel removed", "channel", name)
		}
	}
}

// New creates a new IRC server
func New(cfg *Config, log *logger.Logger) (*Server, error) {
	if cfg.PingInterval == 0 {
		cfg.PingInterval = 60 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 300 * time.Second
	}

	srv := &Server{
		config:      cfg,
		logger:      log,
		clients:     make(map[string]*client.Client),
		clientsAddr: make(map[string]*client.Client),
		channels:    make(map[string]*channel.Channel),
		shutdown:    make(chan struct{}),
	}
	
	// Convert config operators to commands.Operator
	cmdOperators := make([]commands.Operator, len(cfg.Operators))
	for i, op := range cfg.Operators {
		cmdOperators[i] = commands.Operator{
			Name:     op.Name,
			Password: op.Password,
		}
	}
	
	// Initialize command handler with server as registry
	srv.handler = commands.New(cfg.ServerName, log, srv, srv, cmdOperators)
	
	return srv, nil
}

// Start begins listening for connections
func (s *Server) Start(ctx context.Context) error {
	// Start regular TCP listener
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("Starting IRC server", "address", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	s.logger.Info("Server listening", "address", addr)

	// Start TLS listener if enabled
	if s.config.TLSEnabled && s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		if err := s.startTLSListener(ctx); err != nil {
			s.logger.Error("Failed to start TLS listener", "error", err)
		}
	}

	// Start connection acceptor
	go s.acceptConnections(ctx, listener, false)

	// Start maintenance routines
	go s.pingClients(ctx)
	go s.checkTimeouts(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// startTLSListener starts the TLS listener
func (s *Server) startTLSListener(ctx context.Context) error {
	cert, err := tls.LoadX509KeyPair(s.config.TLSCertFile, s.config.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	tlsAddr := fmt.Sprintf("%s:%d", s.config.Host, s.config.TLSPort)
	tlsListener, err := tls.Listen("tcp", tlsAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to start TLS listener: %w", err)
	}

	s.tlsListener = tlsListener
	s.logger.Info("TLS server listening", "address", tlsAddr)

	// Start TLS connection acceptor
	go s.acceptConnections(ctx, tlsListener, true)

	return nil
}

// acceptConnections handles incoming client connections
func (s *Server) acceptConnections(ctx context.Context, listener net.Listener, isTLS bool) {
	connType := "TCP"
	if isTLS {
		connType = "TLS"
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				s.logger.Error("Failed to accept connection", "error", err, "type", connType)
				continue
			}

			// Check client limit
			s.mu.RLock()
			clientCount := len(s.clients)
			s.mu.RUnlock()

			if clientCount >= s.config.MaxClients {
				s.logger.Warn("Max clients reached, rejecting connection", "from", conn.RemoteAddr(), "type", connType)
				conn.Close()
				continue
			}

			// Handle client in a new goroutine
			go s.handleClient(conn)
		}
	}
}

// pingClients sends periodic PINGs to all connected clients
func (s *Server) pingClients(ctx context.Context) {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			clients := make([]*client.Client, 0, len(s.clientsAddr))
			for _, c := range s.clientsAddr {
				clients = append(clients, c)
			}
			s.mu.RUnlock()

			// Send PING to clients that need it
			for _, c := range clients {
				if c.IsRegistered() && c.NeedsPing(s.config.PingInterval) {
					c.Send(fmt.Sprintf("PING :%s", s.config.ServerName))
					c.UpdatePingTime()
				}
			}
		}
	}
}

// checkTimeouts disconnects clients that have timed out
func (s *Server) checkTimeouts(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			clients := make([]*client.Client, 0, len(s.clientsAddr))
			for _, c := range s.clientsAddr {
				clients = append(clients, c)
			}
			s.mu.RUnlock()

			// Check for idle clients
			for _, c := range clients {
				if c.IsIdle(s.config.Timeout) {
					s.logger.Info("Client timed out", "nickname", c.GetNickname())
					c.Send("ERROR :Closing Link: (Ping timeout)")
					c.Disconnect()
				}
			}
		}
	}
}

// handleClient manages a single client connection
func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic in client handler", "error", r)
		}
		conn.Close()
	}()

	clientAddr := conn.RemoteAddr().String()
	s.logger.Info("New connection", "from", clientAddr)

	// Create client instance
	c := client.New(conn, s.logger)

	// Register client by address temporarily
	s.mu.Lock()
	s.clientsAddr[clientAddr] = c
	s.mu.Unlock()

	// Send initial message
	c.Send(fmt.Sprintf("NOTICE AUTH :*** Looking up your hostname..."))

	// Message processing loop
	for {
		// Read message from client
		line, err := c.Receive()
		if err != nil {
			s.logger.Debug("Client read error", "from", clientAddr, "error", err)
			break
		}

		// Check rate limit
		if !c.CheckRateLimit() {
			s.logger.Warn("Client exceeded rate limit", "from", clientAddr, "nickname", c.GetNickname())
			c.Send("ERROR :Excess Flood")
			break
		}

		// Parse IRC message
		msg, err := parser.Parse(line)
		if err != nil {
			s.logger.Warn("Failed to parse message", "from", clientAddr, "line", line, "error", err)
			continue
		}

		// Handle the command
		if err := s.handler.Handle(c, msg); err != nil {
			s.logger.Debug("Command handler error", "from", clientAddr, "command", msg.Command, "error", err)
			// QUIT command returns an error to signal disconnect
			if msg.Command == "QUIT" {
				break
			}
		}
	}

	// Cleanup
	s.mu.Lock()
	delete(s.clientsAddr, clientAddr)
	if c.IsRegistered() {
		delete(s.clients, c.GetNickname())
	}
	s.mu.Unlock()

	// TODO: Remove from channels in Phase 2

	c.Disconnect()
	s.logger.Info("Client disconnected", "from", clientAddr, "nickname", c.GetNickname())
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() {
	s.logger.Info("Shutting down server")

	// Close listeners
	if s.listener != nil {
		s.listener.Close()
	}
	if s.tlsListener != nil {
		s.tlsListener.Close()
	}

	// Disconnect all clients
	s.mu.Lock()
	for _, c := range s.clients {
		c.Disconnect()
	}
	s.mu.Unlock()

	close(s.shutdown)
	s.logger.Info("Server shutdown complete")
}
