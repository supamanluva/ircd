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
	
	// Register link in link registry (Phase 7.4)
	if err := s.linkRegistry.AddLink(server.SID, link); err != nil {
		s.logger.Error("Failed to register link", "name", server.Name, "error", err)
		return
	}
	defer s.linkRegistry.RemoveLink(server.SID)
	
	s.logger.Info("Link established, keeping connection alive", "name", server.Name)
	
	// Handle ongoing protocol messages (Phase 7.4)
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
			continue
		}
		
		// Handle incoming messages (Phase 7.4.2)
		if err := s.handleLinkMessage(msg, server); err != nil {
			s.logger.Error("Failed to handle link message", "command", msg.Command, "error", err)
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
	
	// Register link in link registry (Phase 7.4)
	if err := s.linkRegistry.AddLink(server.SID, link); err != nil {
		conn.Close()
		return fmt.Errorf("failed to register link: %v", err)
	}
	
	s.logger.Info("Link established, starting message handler", "name", server.Name)
	
	// Handle ongoing protocol messages in a goroutine (Phase 7.4)
	go func() {
		defer func() {
			s.linkRegistry.RemoveLink(server.SID)
			conn.Close()
		}()
		
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
				continue
			}
			
			// Handle incoming messages (Phase 7.4.2)
			if err := s.handleLinkMessage(msg, server); err != nil {
				s.logger.Error("Failed to handle link message", "command", msg.Command, "error", err)
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

// handleLinkMessage processes incoming messages from linked servers (Phase 7.4)
func (s *Server) handleLinkMessage(msg *linking.Message, fromServer *linking.Server) error {
	switch msg.Command {
	case "PRIVMSG", "NOTICE":
		return s.handleLinkPrivmsg(msg, fromServer)
	
	case "JOIN":
		return s.handleLinkJoin(msg, fromServer)
	
	case "PART":
		return s.handleLinkPart(msg, fromServer)
	
	case "QUIT":
		return s.handleLinkQuit(msg, fromServer)
	
	case "NICK":
		return s.handleLinkNick(msg, fromServer)
	
	case "MODE":
		return s.handleLinkMode(msg, fromServer)
	
	case "TOPIC":
		return s.handleLinkTopic(msg, fromServer)
	
	case "KICK":
		return s.handleLinkKick(msg, fromServer)
	
	case "INVITE":
		return s.handleLinkInvite(msg, fromServer)
	
	default:
		s.logger.Debug("Unhandled link message", "command", msg.Command, "from", fromServer.Name)
	}
	
	return nil
}

// handleLinkPrivmsg handles PRIVMSG/NOTICE from remote servers
func (s *Server) handleLinkPrivmsg(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 2 {
		return fmt.Errorf("invalid %s: need 2 params", msg.Command)
	}
	
	target := msg.Params[0]
	message := msg.Params[1]
	
	// Get source user info
	sourceUID := msg.Source
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		// Source might be a server SID, try to construct something reasonable
		s.logger.Debug("Unknown source user", "uid", sourceUID)
		sourceUser = &linking.RemoteUser{
			UID:  sourceUID,
			Nick: sourceUID,
		}
	}
	
	// Check if target is a channel
	if len(target) > 0 && (target[0] == '#' || target[0] == '&') {
		// Channel message - deliver to local members
		s.mu.RLock()
		ch := s.channels[target]
		s.mu.RUnlock()
		
		if ch == nil {
			// We don't have this channel locally, ignore
			return nil
		}
		
		// Format message and broadcast to local channel members
		msgText := fmt.Sprintf(":%s!%s@%s %s %s :%s",
			sourceUser.Nick, sourceUser.User, sourceUser.Host,
			msg.Command, target, message)
		
		// Broadcast to all local members
		ch.Broadcast(msgText, nil)
		
		s.logger.Debug("Delivered channel message from remote",
			"from", sourceUser.Nick, "channel", target)
		
		return nil
	}
	
	// Private message - find target client locally
	s.mu.RLock()
	targetClient := s.clients[target]
	s.mu.RUnlock()
	
	if targetClient == nil {
		// Try finding by UID
		for _, c := range s.clients {
			if c.GetUID() == target {
				targetClient = c
				break
			}
		}
	}
	
	if targetClient == nil {
		// Target not found locally
		return fmt.Errorf("target %s not found locally", target)
	}
	
	// Format and deliver message
	msgText := fmt.Sprintf(":%s!%s@%s %s %s :%s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host,
		msg.Command, targetClient.GetNickname(), message)
	
	targetClient.Send(msgText)
	
	s.logger.Debug("Delivered private message from remote",
		"from", sourceUser.Nick, "to", targetClient.GetNickname())
	
	return nil
}

// handleLinkJoin handles JOIN from remote servers (Phase 7.4.3)
func (s *Server) handleLinkJoin(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 1 {
		return fmt.Errorf("invalid JOIN: need at least 1 param")
	}
	
	channel := msg.Params[0]
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user JOINing", "uid", sourceUID, "channel", channel)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Check if we have local members in this channel
	s.mu.RLock()
	ch, exists := s.channels[channel]
	s.mu.RUnlock()
	
	if !exists {
		// No local users in this channel, nothing to do
		s.logger.Debug("Remote JOIN to channel with no local users",
			"user", sourceUser.Nick, "channel", channel)
		return nil
	}
	
	// Broadcast JOIN to all local members
	joinMsg := fmt.Sprintf(":%s!%s@%s JOIN %s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host, channel)
	ch.Broadcast(joinMsg, nil)
	
	s.logger.Debug("Delivered remote JOIN",
		"user", sourceUser.Nick, "channel", channel)
	
	return nil
}

// handleLinkPart handles PART from remote servers (Phase 7.4.3)
func (s *Server) handleLinkPart(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 1 {
		return fmt.Errorf("invalid PART: need at least 1 param")
	}
	
	channel := msg.Params[0]
	partMsg := ""
	if len(msg.Params) > 1 {
		partMsg = msg.Params[1]
	}
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user PARTing", "uid", sourceUID, "channel", channel)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Check if we have local members in this channel
	s.mu.RLock()
	ch, exists := s.channels[channel]
	s.mu.RUnlock()
	
	if !exists {
		// No local users in this channel, nothing to do
		s.logger.Debug("Remote PART from channel with no local users",
			"user", sourceUser.Nick, "channel", channel)
		return nil
	}
	
	// Broadcast PART to all local members
	var partNotice string
	if partMsg != "" {
		partNotice = fmt.Sprintf(":%s!%s@%s PART %s :%s",
			sourceUser.Nick, sourceUser.User, sourceUser.Host, channel, partMsg)
	} else {
		partNotice = fmt.Sprintf(":%s!%s@%s PART %s",
			sourceUser.Nick, sourceUser.User, sourceUser.Host, channel)
	}
	ch.BroadcastAll(partNotice)
	
	s.logger.Debug("Delivered remote PART",
		"user", sourceUser.Nick, "channel", channel)
	
	return nil
}

// handleLinkQuit handles QUIT from remote servers (Phase 7.4.3)
func (s *Server) handleLinkQuit(msg *linking.Message, fromServer *linking.Server) error {
	quitMsg := ""
	if len(msg.Params) > 0 {
		quitMsg = msg.Params[0]
	}
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user QUITing", "uid", sourceUID)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Broadcast QUIT to all local channels that have this remote user
	quitNotice := fmt.Sprintf(":%s!%s@%s QUIT :%s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host, quitMsg)
	
	// Find all channels with local members and broadcast
	s.mu.RLock()
	for _, ch := range s.channels {
		if ch.GetMemberCount() > 0 {
			// Broadcast to local members
			ch.BroadcastAll(quitNotice)
		}
	}
	s.mu.RUnlock()
	
	s.logger.Debug("Delivered remote QUIT",
		"user", sourceUser.Nick, "message", quitMsg)
	
	return nil
}

// handleLinkNick handles NICK changes from remote servers (Phase 7.4.3)
func (s *Server) handleLinkNick(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 1 {
		return fmt.Errorf("invalid NICK: need at least 1 param")
	}
	
	newNick := msg.Params[0]
	sourceUID := msg.Source
	
	// Get source user info (with old nickname)
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user changing NICK", "uid", sourceUID)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	oldNick := sourceUser.Nick
	
	// Broadcast NICK change to all local channels that have this remote user
	nickNotice := fmt.Sprintf(":%s!%s@%s NICK :%s",
		oldNick, sourceUser.User, sourceUser.Host, newNick)
	
	// Find all channels with local members and broadcast
	s.mu.RLock()
	for _, ch := range s.channels {
		if ch.GetMemberCount() > 0 {
			// Broadcast to local members
			ch.BroadcastAll(nickNotice)
		}
	}
	s.mu.RUnlock()
	
	s.logger.Debug("Delivered remote NICK",
		"old", oldNick, "new", newNick)
	
	return nil
}

// handleLinkMode handles MODE from remote servers (Phase 7.4.4)
func (s *Server) handleLinkMode(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 2 {
		return fmt.Errorf("invalid MODE: need at least 2 params")
	}
	
	channel := msg.Params[0]
	modeString := msg.Params[1]
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user setting MODE", "uid", sourceUID, "channel", channel)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Check if we have local members in this channel
	s.mu.RLock()
	ch, exists := s.channels[channel]
	s.mu.RUnlock()
	
	if !exists {
		// No local users in this channel, nothing to do
		s.logger.Debug("Remote MODE on channel with no local users",
			"user", sourceUser.Nick, "channel", channel)
		return nil
	}
	
	// Broadcast MODE to all local members
	modeMsg := fmt.Sprintf(":%s!%s@%s MODE %s %s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host, channel, modeString)
	ch.BroadcastAll(modeMsg)
	
	s.logger.Debug("Delivered remote MODE",
		"user", sourceUser.Nick, "channel", channel, "mode", modeString)
	
	return nil
}

// handleLinkTopic handles TOPIC from remote servers (Phase 7.4.4)
func (s *Server) handleLinkTopic(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 2 {
		return fmt.Errorf("invalid TOPIC: need at least 2 params")
	}
	
	channel := msg.Params[0]
	topic := msg.Params[1]
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user setting TOPIC", "uid", sourceUID, "channel", channel)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Check if we have local members in this channel
	s.mu.RLock()
	ch, exists := s.channels[channel]
	s.mu.RUnlock()
	
	if !exists {
		// No local users in this channel, nothing to do
		s.logger.Debug("Remote TOPIC on channel with no local users",
			"user", sourceUser.Nick, "channel", channel)
		return nil
	}
	
	// Update local channel topic
	ch.SetTopic(topic)
	
	// Broadcast TOPIC to all local members
	topicMsg := fmt.Sprintf(":%s!%s@%s TOPIC %s :%s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host, channel, topic)
	ch.BroadcastAll(topicMsg)
	
	s.logger.Debug("Delivered remote TOPIC",
		"user", sourceUser.Nick, "channel", channel)
	
	return nil
}

// handleLinkKick handles KICK from remote servers (Phase 7.4.4)
func (s *Server) handleLinkKick(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 3 {
		return fmt.Errorf("invalid KICK: need at least 3 params")
	}
	
	channel := msg.Params[0]
	targetNick := msg.Params[1]
	reason := msg.Params[2]
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user KICKing", "uid", sourceUID, "channel", channel)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Check if we have local members in this channel
	s.mu.RLock()
	ch, exists := s.channels[channel]
	s.mu.RUnlock()
	
	if !exists {
		// No local users in this channel, nothing to do
		s.logger.Debug("Remote KICK on channel with no local users",
			"user", sourceUser.Nick, "channel", channel)
		return nil
	}
	
	// Broadcast KICK to all local members
	kickMsg := fmt.Sprintf(":%s!%s@%s KICK %s %s :%s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host, channel, targetNick, reason)
	ch.BroadcastAll(kickMsg)
	
	// If the kicked user is local, remove them from the channel
	s.mu.RLock()
	targetClient, isLocal := s.clients[targetNick]
	s.mu.RUnlock()
	
	if isLocal && targetClient != nil {
		ch.RemoveMember(targetClient)
		targetClient.PartChannel(channel)
		s.logger.Debug("Removed local user from channel after remote KICK",
			"user", targetNick, "channel", channel)
	}
	
	s.logger.Debug("Delivered remote KICK",
		"kicker", sourceUser.Nick, "channel", channel, "target", targetNick)
	
	return nil
}

// handleLinkInvite handles INVITE from remote servers (Phase 7.4.4)
func (s *Server) handleLinkInvite(msg *linking.Message, fromServer *linking.Server) error {
	if len(msg.Params) < 2 {
		return fmt.Errorf("invalid INVITE: need at least 2 params")
	}
	
	targetNick := msg.Params[0]
	channel := msg.Params[1]
	sourceUID := msg.Source
	
	// Get source user info
	sourceUser, ok := s.network.GetUserByUID(sourceUID)
	if !ok {
		s.logger.Debug("Unknown user INVITEing", "uid", sourceUID)
		return fmt.Errorf("unknown user %s", sourceUID)
	}
	
	// Check if target is a local user
	s.mu.RLock()
	targetClient, isLocal := s.clients[targetNick]
	s.mu.RUnlock()
	
	if !isLocal || targetClient == nil {
		// Target not local, nothing to do
		s.logger.Debug("Remote INVITE for non-local user",
			"inviter", sourceUser.Nick, "target", targetNick, "channel", channel)
		return nil
	}
	
	// Send INVITE notification to target
	inviteMsg := fmt.Sprintf(":%s!%s@%s INVITE %s %s",
		sourceUser.Nick, sourceUser.User, sourceUser.Host, targetNick, channel)
	targetClient.Send(inviteMsg)
	
	s.logger.Debug("Delivered remote INVITE",
		"inviter", sourceUser.Nick, "target", targetNick, "channel", channel)
	
	return nil
}

