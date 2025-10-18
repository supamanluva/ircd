package server

import (
	"fmt"
	"net"
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
	s.logger.Info("Server link listener started on %s", addr)
	
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
				s.logger.Error("Error accepting link connection: %v", err)
				continue
			}
		}
		
		s.logger.Info("Incoming link connection from %s", conn.RemoteAddr())
		
		// Handle link connection in a goroutine
		go s.handleLinkConnection(conn)
	}
}

// handleLinkConnection handles an incoming server link connection
func (s *Server) handleLinkConnection(conn net.Conn) {
	defer conn.Close()
	
	s.logger.Info("Link connection handler started for %s", conn.RemoteAddr())
	
	// TODO: Implement server handshake protocol
	// 1. Receive PASS command with SID and password
	// 2. Receive CAPAB for capabilities
	// 3. Receive SERVER with server name and description
	// 4. Send our PASS, CAPAB, SERVER
	// 5. Exchange SVINFO
	// 6. Perform burst (send all our users/channels)
	// 7. Begin normal operation
	
	s.logger.Info("Link connection closed for %s (handshake not yet implemented)", conn.RemoteAddr())
}

// ConnectToServer initiates an outbound connection to another server
func (s *Server) ConnectToServer(linkCfg LinkConfig) error {
	if !s.config.LinkingEnabled {
		return fmt.Errorf("server linking is not enabled")
	}
	
	addr := fmt.Sprintf("%s:%d", linkCfg.Host, linkCfg.Port)
	s.logger.Info("Attempting to connect to server %s (%s) at %s", 
		linkCfg.Name, linkCfg.SID, addr)
	
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	
	s.logger.Info("Connected to %s, starting handshake", addr)
	
	// TODO: Implement client side of handshake
	// 1. Send PASS with our SID and password
	// 2. Send CAPAB for our capabilities
	// 3. Send SERVER with our name and description
	// 4. Receive their PASS, CAPAB, SERVER
	// 5. Exchange SVINFO
	// 6. Receive burst (their users/channels)
	// 7. Send our burst
	// 8. Begin normal operation
	
	conn.Close()
	s.logger.Info("Link to %s established (handshake not yet implemented)", linkCfg.Name)
	
	return nil
}

// AutoConnect attempts to connect to all auto-connect servers
func (s *Server) AutoConnect() {
	if !s.config.LinkingEnabled {
		return
	}
	
	for _, link := range s.config.Links {
		if link.AutoConnect {
			s.logger.Info("Auto-connecting to %s", link.Name)
			go func(l LinkConfig) {
				if err := s.ConnectToServer(l); err != nil {
					s.logger.Error("Failed to auto-connect to %s: %v", l.Name, err)
				}
			}(link)
		}
	}
}
