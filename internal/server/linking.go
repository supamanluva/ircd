package server

import (
	"fmt"
	"net"
	
	"github.com/supamanluva/ircd/internal/linking"
)

// StartLinkListener starts listening for incoming server links
func (s *Server) StartLinkListener() error {
	if !s.config.LinkingEnabled {
		return nil // Linking not enabled
	}

	addr := fmt.Sprintf("%s:%d", s.config.LinkingHost, s.config.LinkingPort)
	
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start link listener: %v", err)
	}
	
	s.linkListener = listener
	s.logger.Info("Server link listener started on", "address", addr)
	
	// Accept connections in a goroutine
	go s.acceptLinks()
	
	return nil
}

// acceptLinks accepts incoming server link connections
func (s *Server) acceptLinks() {
	for {
		conn, err := s.linkListener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return
			default:
				s.logger.Error("Error accepting link connection", "error", err)
				continue
			}
		}
		
		s.logger.Info("Incoming link connection from", "address", conn.RemoteAddr().String())
		
		// Handle link connection in a goroutine
		go s.handleLinkConnection(conn)
	}
}

// handleLinkConnection handles an incoming server link connection
func (s *Server) handleLinkConnection(conn net.Conn) {
	defer conn.Close()
	
	s.logger.Info("Link connection handler started for", "address", conn.RemoteAddr().String())
	
	// Create link
	link := linking.NewLink(conn)
	
	// Perform handshake (server side - receiving connection)
	err := link.HandshakeServer(s.network, s.config.LinkPassword)
	if err != nil {
		s.logger.Error("Handshake failed from", "address", conn.RemoteAddr().String(), "error", err)
		return
	}
	
	// Get the registered server
	server := link.GetServer()
	if server == nil {
		s.logger.Error("No server object after handshake from", "address", conn.RemoteAddr().String())
		return
	}
	
	s.logger.Info("Server link established", "name", server.Name, "sid", server.SID, "address", conn.RemoteAddr().String())
	
	// Add server to network
	if err := s.network.AddServer(server); err != nil {
		s.logger.Error("Failed to add server to network", "name", server.Name, "error", err)
		return
	}
	
	s.logger.Info("Server registered in network", "name", server.Name, "total_servers", s.network.GetServerCount())
	
	// Perform burst (Phase 7.3)
	s.logger.Info("Receiving burst from", "name", server.Name)
	
	burstState, err := link.ReceiveBurst(s.network)
	if err != nil {
		s.logger.Error("Failed to receive burst", "name", server.Name, "error", err)
		return
	}
	
	s.logger.Info("Burst received", "name", server.Name, "users", burstState.UsersRecv, "channels", burstState.ChansRecv)
	
	// Send our burst
	s.logger.Info("Sending burst to", "name", server.Name)
	
	err = link.SendBurstFromClients(
		s.network,
		func() []linking.BurstClient { return s.GetBurstClients() },
		func() []linking.BurstChannel { return s.GetBurstChannels() },
	)
	if err != nil {
		s.logger.Error("Failed to send burst", "name", server.Name, "error", err)
		return
	}
	
	localUsers := len(s.GetBurstClients())
	localChans := len(s.GetBurstChannels())
	s.logger.Info("Burst sent", "name", server.Name, "users", localUsers, "channels", localChans)
	
	// Log network statistics
	s.logger.Info("Network state", "total_servers", s.network.GetServerCount(), 
		"total_users", s.network.GetUserCount(), "total_channels", s.network.GetChannelCount())
	
	// TODO (Phase 7.4): Handle ongoing protocol messages
	// For now, just keep the connection alive and log incoming messages
	s.logger.Info("Link established, keeping connection alive", "name", server.Name)
	
	// Keep reading messages to prevent connection close
	// In Phase 7.4 we'll process these messages for routing
	for {
		msg, err := link.ReadMessage()
		if err != nil {
			s.logger.Info("Link connection closed", "name", server.Name, "error", err)
			break
		}
		
		// Log incoming messages for now
		s.logger.Debug("Received from linked server", "name", server.Name, "command", msg.Command, "params", msg.Params)
		
		// Handle PING to keep connection alive
		if msg.Command == "PING" {
			pong := linking.BuildPONG(s.network.LocalSID, msg.Source)
			if err := link.WriteMessage(pong); err != nil {
				s.logger.Error("Failed to send PONG", "name", server.Name, "error", err)
				break
			}
		}
	}
	s.logger.Info("Link connection closed for", "address", conn.RemoteAddr().String())
}

// ConnectToServer initiates an outbound connection to another server
func (s *Server) ConnectToServer(linkCfg LinkConfig) error {
	if !s.config.LinkingEnabled {
		return fmt.Errorf("server linking is not enabled")
	}
	
	addr := fmt.Sprintf("%s:%d", linkCfg.Host, linkCfg.Port)
	s.logger.Info("Attempting to connect to server", "name", linkCfg.Name, "sid", linkCfg.SID, "address", addr)
	
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	
	s.logger.Info("Connected, starting handshake", "address", addr)
	
	// Create link
	link := linking.NewLink(conn)
	
	// Perform handshake (client side - initiating connection)
	err = link.HandshakeClient(s.network, linkCfg.Password, linkCfg.SID, linkCfg.Name)
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake failed with %s: %v", linkCfg.Name, err)
	}
	
	// Get the registered server
	server := link.GetServer()
	if server == nil {
		conn.Close()
		return fmt.Errorf("no server object after handshake with %s", linkCfg.Name)
	}
	
	// Mark as hub if configured
	server.IsHub = linkCfg.IsHub
	
	s.logger.Info("Server link established", "name", server.Name, "sid", server.SID)
	
	// Add server to network
	if err := s.network.AddServer(server); err != nil {
		conn.Close()
		return fmt.Errorf("failed to add server %s to network: %v", server.Name, err)
	}
	
	s.logger.Info("Server registered in network", "name", server.Name, "total_servers", s.network.GetServerCount())
	
	// Perform burst (Phase 7.3)
	s.logger.Info("Sending burst to", "name", server.Name)
	
	err = link.SendBurstFromClients(
		s.network,
		func() []linking.BurstClient { return s.GetBurstClients() },
		func() []linking.BurstChannel { return s.GetBurstChannels() },
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to send burst: %v", err)
	}
	
	localUsers := len(s.GetBurstClients())
	localChans := len(s.GetBurstChannels())
	s.logger.Info("Burst sent", "name", server.Name, "users", localUsers, "channels", localChans)
	
	// Receive their burst
	s.logger.Info("Receiving burst from", "name", server.Name)
	
	burstState, err := link.ReceiveBurst(s.network)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive burst: %v", err)
	}
	
	s.logger.Info("Burst received", "name", server.Name, "users", burstState.UsersRecv, "channels", burstState.ChansRecv)
	
	// Log network statistics
	s.logger.Info("Network state", "total_servers", s.network.GetServerCount(), 
		"total_users", s.network.GetUserCount(), "total_channels", s.network.GetChannelCount())
	
	// TODO (Phase 7.4): Handle ongoing protocol messages in a goroutine
	// For now, just keep the connection alive and log incoming messages
	s.logger.Info("Link established, starting message handler", "name", server.Name)
	
	// Keep reading messages in a goroutine to prevent blocking
	go func() {
		defer conn.Close()
		
		for {
			msg, err := link.ReadMessage()
			if err != nil {
				s.logger.Info("Link connection closed", "name", server.Name, "error", err)
				return
			}
			
			// Log incoming messages for now
			s.logger.Debug("Received from linked server", "name", server.Name, "command", msg.Command, "params", msg.Params)
			
			// Handle PING to keep connection alive
			if msg.Command == "PING" {
				pong := linking.BuildPONG(s.network.LocalSID, msg.Source)
				if err := link.WriteMessage(pong); err != nil {
					s.logger.Error("Failed to send PONG", "name", server.Name, "error", err)
					return
				}
			}
		}
	}()
	
	return nil
}

// AutoConnect attempts to connect to all auto-connect servers
func (s *Server) AutoConnect() {
	if !s.config.LinkingEnabled {
		return
	}
	
	for _, link := range s.config.Links {
		if link.AutoConnect {
			s.logger.Info("Auto-connecting to", "name", link.Name)
			go func(l LinkConfig) {
				if err := s.ConnectToServer(l); err != nil {
					s.logger.Error("Failed to auto-connect", "name", l.Name, "error", err)
				}
			}(link)
		}
	}
}
