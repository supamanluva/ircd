package commands

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/supamanluva/ircd/internal/channel"
	"github.com/supamanluva/ircd/internal/client"
	"github.com/supamanluva/ircd/internal/linking"
	"github.com/supamanluva/ircd/internal/logger"
	"github.com/supamanluva/ircd/internal/parser"
)

// Handler processes IRC commands
type Handler struct {
	serverName string
	logger     *logger.Logger
	clients    ClientRegistry
	channels   ChannelRegistry
	operators  map[string]string // name -> bcrypt password hash
	router     MessageRouter     // Message router for server linking (Phase 7.4)
}

// ClientRegistry interface for managing clients
type ClientRegistry interface {
	GetClient(nickname string) *client.Client
	AddClient(c *client.Client) error
	RemoveClient(c *client.Client)
	IsNicknameInUse(nickname string) bool
}

// ChannelRegistry interface for managing channels
type ChannelRegistry interface {
	GetChannel(name string) *channel.Channel
	CreateChannel(name string) *channel.Channel
	RemoveChannel(name string)
}

// MessageRouter interface for routing messages to remote servers (Phase 7.4)
type MessageRouter interface {
	// RoutePrivmsg routes a PRIVMSG to a remote user
	RoutePrivmsg(sourceNick, sourceUser, sourceHost, targetNick, message string) error
	// RouteNotice routes a NOTICE to a remote user
	RouteNotice(sourceNick, sourceUser, sourceHost, targetNick, message string) error
	// RouteChannelMessage routes a message to all servers with channel members
	RouteChannelMessage(sourceNick, sourceUser, sourceHost, channel, message, msgType string) error
	// IsUserLocal checks if a user is on the local server
	IsUserLocal(nickname string) bool
	
	// PropagateJoin propagates a JOIN to remote servers (Phase 7.4.3)
	PropagateJoin(nick, user, host, uid, channel string, ts int64) error
	// PropagatePart propagates a PART to remote servers (Phase 7.4.3)
	PropagatePart(nick, user, host, uid, channel, message string) error
	// PropagateQuit propagates a QUIT to remote servers (Phase 7.4.3)
	PropagateQuit(nick, user, host, uid, message string) error
	// PropagateNick propagates a NICK change to remote servers (Phase 7.4.3)
	PropagateNick(oldNick, newNick, user, host, uid string, ts int64) error
	
	// PropagateMode propagates a MODE change to remote servers (Phase 7.4.4)
	PropagateMode(nick, user, host, uid, channel, modeString string, ts int64) error
	// PropagateTopic propagates a TOPIC change to remote servers (Phase 7.4.4)
	PropagateTopic(nick, user, host, uid, channel, topic string, ts int64) error
	// PropagateKick propagates a KICK to remote servers (Phase 7.4.4)
	PropagateKick(nick, user, host, uid, channel, target, reason string) error
	// PropagateInvite propagates an INVITE to remote servers (Phase 7.4.4)
	PropagateInvite(nick, user, host, uid, target, channel string) error
	
	// PropagateUser propagates a new user registration to remote servers
	PropagateUser(nick, user, host, uid, realname string, ts int64) error
	
	// GetRemoteChannel gets a remote channel by name (for NAMES list)
	GetRemoteChannel(name string) (*linking.RemoteChannel, bool)
	// GetRemoteUserByUID gets a remote user by UID (for NAMES list)
	GetRemoteUserByUID(uid string) (*linking.RemoteUser, bool)
	
	// DisconnectServer disconnects a linked server (Phase 7.4.5)
	DisconnectServer(serverName, reason string) error
}

// CommandFunc is the signature for command handler functions
type CommandFunc func(c *client.Client, msg *parser.Message) error

// Operator represents a server operator configuration
type Operator struct {
	Name     string
	Password string // bcrypt hashed
}

// New creates a new command handler
func New(serverName string, log *logger.Logger, clients ClientRegistry, channels ChannelRegistry, operators []Operator) *Handler {
	// Build operator map for quick lookup
	operMap := make(map[string]string)
	for _, op := range operators {
		operMap[op.Name] = op.Password
	}
	
	return &Handler{
		serverName: serverName,
		logger:     log,
		clients:    clients,
		channels:   channels,
		operators:  operMap,
		router:     nil, // Will be set by SetRouter if linking is enabled
	}
}

// SetRouter sets the message router for server linking (Phase 7.4)
func (h *Handler) SetRouter(router MessageRouter) {
	h.router = router
}

// Handle processes a parsed IRC message
func (h *Handler) Handle(c *client.Client, msg *parser.Message) error {
	if !msg.IsValid() {
		return fmt.Errorf("invalid message")
	}

	h.logger.Debug("Handling command", "command", msg.Command, "client", c.GetNickname())

	// Route to appropriate handler
	switch msg.Command {
	case "NICK":
		return h.handleNick(c, msg)
	case "USER":
		return h.handleUser(c, msg)
	case "PING":
		return h.handlePing(c, msg)
	case "PONG":
		return h.handlePong(c, msg)
	case "JOIN":
		return h.handleJoin(c, msg)
	case "PART":
		return h.handlePart(c, msg)
	case "PRIVMSG":
		return h.handlePrivmsg(c, msg)
	case "NOTICE":
		return h.handleNotice(c, msg)
	case "NAMES":
		return h.handleNames(c, msg)
	case "TOPIC":
		return h.handleTopic(c, msg)
	case "MODE":
		return h.handleMode(c, msg)
	case "KICK":
		return h.handleKick(c, msg)
	case "QUIT":
		return h.handleQuit(c, msg)
	case "WHO":
		return h.handleWho(c, msg)
	case "WHOIS":
		return h.handleWhois(c, msg)
	case "LIST":
		return h.handleList(c, msg)
	case "INVITE":
		return h.handleInvite(c, msg)
	case "OPER":
		return h.handleOper(c, msg)
	case "AWAY":
		return h.handleAway(c, msg)
	case "USERHOST":
		return h.handleUserhost(c, msg)
	case "ISON":
		return h.handleIson(c, msg)
	case "SQUIT":
		return h.handleSquit(c, msg)
	default:
		// Unknown command
		h.sendNumeric(c, ERR_UNKNOWNCOMMAND, msg.Command+" :Unknown command")
		return nil
	}
}

// sendNumeric sends a numeric reply to the client
func (h *Handler) sendNumeric(c *client.Client, code, message string) {
	nick := c.GetNickname()
	reply := NumericReply(h.serverName, code, nick, message)
	c.Send(reply)
}

// sendWelcome sends the welcome message sequence after registration
func (h *Handler) sendWelcome(c *client.Client) {
	nick := c.GetNickname()
	
	// 001 RPL_WELCOME
	h.sendNumeric(c, RPL_WELCOME, fmt.Sprintf(":Welcome to the Internet Relay Network %s", c.GetHostmask()))
	
	// 002 RPL_YOURHOST
	h.sendNumeric(c, RPL_YOURHOST, fmt.Sprintf(":Your host is %s, running version ircd-0.1.0", h.serverName))
	
	// 003 RPL_CREATED
	h.sendNumeric(c, RPL_CREATED, ":This server was created just now")
	
	// 004 RPL_MYINFO
	h.sendNumeric(c, RPL_MYINFO, fmt.Sprintf("%s ircd-0.1.0 o o", h.serverName))
	
	h.logger.Info("Client registered", "nickname", nick, "hostmask", c.GetHostmask())
}

// handleNick handles the NICK command
func (h *Handler) handleNick(c *client.Client, msg *parser.Message) error {
	// Check if nickname parameter is provided
	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NONICKNAMEGIVEN, ":No nickname given")
		return nil
	}

	newNick := msg.GetParam(0)

	// Validate nickname
	if !isValidNickname(newNick) {
		h.sendNumeric(c, ERR_ERRONEUSNICKNAME, newNick+" :Erroneous nickname")
		return nil
	}

	// Check if nickname is already in use
	if h.clients.IsNicknameInUse(newNick) && c.GetNickname() != newNick {
		h.sendNumeric(c, ERR_NICKNAMEINUSE, newNick+" :Nickname is already in use")
		return nil
	}

	oldNick := c.GetNickname()
	c.SetNickname(newNick)

	// If client is already registered, broadcast the nick change
	if c.IsRegistered() && oldNick != "" {
		// Notify the client and all channels they're in
		notification := fmt.Sprintf(":%s NICK :%s", oldNick, newNick)
		c.Send(notification)
		
		// Broadcast to all channels
		for _, channelName := range c.GetChannels() {
			if ch := h.channels.GetChannel(channelName); ch != nil {
				ch.Broadcast(notification, c)
			}
		}
		
		// Propagate NICK to remote servers (Phase 7.4.3)
		if h.router != nil {
			parts := strings.SplitN(c.GetHostmask(), "!", 2)
			user := ""
			host := ""
			if len(parts) == 2 {
				userhost := strings.SplitN(parts[1], "@", 2)
				if len(userhost) == 2 {
					user = userhost[0]
					host = userhost[1]
				}
			}
			
			uid := c.GetUID()
			if uid == "" {
				uid = newNick
			}
			
			if err := h.router.PropagateNick(oldNick, newNick, user, host, uid, time.Now().Unix()); err != nil {
				h.logger.Debug("Failed to propagate NICK", "error", err)
			}
		}
	}

	// Check if client should be registered now
	h.tryRegister(c)
	
	// If client just became registered, add them to the registry (for UID assignment and propagation)
	if c.IsRegistered() && oldNick == "" {
		h.logger.Info("Attempting to add newly registered client", "nick", newNick, "isRegistered", c.IsRegistered(), "oldNick", oldNick)
		if err := h.clients.AddClient(c); err != nil {
			h.logger.Warn("Failed to add client to registry", "error", err, "nick", newNick)
		} else {
			h.logger.Info("Successfully added client to registry", "nick", newNick)
		}
	}

	return nil
}

// handleUser handles the USER command
func (h *Handler) handleUser(c *client.Client, msg *parser.Message) error {
	// Check if already registered
	if c.IsRegistered() {
		h.sendNumeric(c, ERR_ALREADYREGISTERED, ":You may not reregister")
		return nil
	}

	// USER command requires 4 parameters: username, mode, unused, realname
	if len(msg.Params) < 4 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "USER :Not enough parameters")
		return nil
	}

	username := msg.GetParam(0)
	realname := msg.GetParam(3)

	c.SetUsername(username, realname)

	// Check if client should be registered now
	wasRegistered := c.IsRegistered()
	h.tryRegister(c)
	
	// If client just became registered, add them to the registry (for UID assignment and propagation)
	if !wasRegistered && c.IsRegistered() {
		h.logger.Info("Attempting to add newly registered client from USER", "nick", c.GetNickname(), "user", username)
		if err := h.clients.AddClient(c); err != nil {
			h.logger.Warn("Failed to add client to registry", "error", err, "nick", c.GetNickname())
		} else {
			h.logger.Info("Successfully added client to registry from USER", "nick", c.GetNickname())
		}
	}

	return nil
}

// handlePing handles the PING command
func (h *Handler) handlePing(c *client.Client, msg *parser.Message) error {
	// PING requires at least one parameter
	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "PING :Not enough parameters")
		return nil
	}

	// Respond with PONG
	token := msg.GetParam(0)
	c.Send(fmt.Sprintf(":%s PONG %s :%s", h.serverName, h.serverName, token))

	return nil
}

// handlePong handles the PONG command
func (h *Handler) handlePong(c *client.Client, msg *parser.Message) error {
	// PONG is a response to PING, just update last activity
	// The client's lastActivity is already updated in Receive()
	h.logger.Debug("Received PONG", "client", c.GetNickname())
	return nil
}

// handleQuit handles the QUIT command
func (h *Handler) handleQuit(c *client.Client, msg *parser.Message) error {
	// Get quit message
	quitMsg := "Client quit"
	if msg.HasParam(0) {
		quitMsg = msg.GetParam(0)
	}

	h.logger.Info("Client quit", "nickname", c.GetNickname(), "message", quitMsg)

	// Broadcast quit to all channels
	quitNotice := fmt.Sprintf(":%s QUIT :%s", c.GetHostmask(), quitMsg)
	for _, channelName := range c.GetChannels() {
		if ch := h.channels.GetChannel(channelName); ch != nil {
			ch.Broadcast(quitNotice, c)
			ch.RemoveMember(c)
			// Remove empty channels
			if ch.IsEmpty() {
				h.channels.RemoveChannel(channelName)
			}
		}
	}
	
	// Propagate QUIT to remote servers (Phase 7.4.3)
	if h.router != nil && c.IsRegistered() {
		parts := strings.SplitN(c.GetHostmask(), "!", 2)
		user := ""
		host := ""
		if len(parts) == 2 {
			userhost := strings.SplitN(parts[1], "@", 2)
			if len(userhost) == 2 {
				user = userhost[0]
				host = userhost[1]
			}
		}
		
		uid := c.GetUID()
		if uid == "" {
			uid = c.GetNickname()
		}
		
		if err := h.router.PropagateQuit(c.GetNickname(), user, host, uid, quitMsg); err != nil {
			h.logger.Debug("Failed to propagate QUIT", "error", err)
		}
	}

	// Send ERROR to client
	c.Send(fmt.Sprintf("ERROR :Closing Link: %s (%s)", c.GetHostmask(), quitMsg))

	return fmt.Errorf("client quit: %s", quitMsg)
}

// handleJoin handles the JOIN command
func (h *Handler) handleJoin(c *client.Client, msg *parser.Message) error {
	// Check if registered
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// JOIN requires at least one parameter
	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "JOIN :Not enough parameters")
		return nil
	}

	channelNames := strings.Split(msg.GetParam(0), ",")
	
	// Parse keys if provided (JOIN #chan1,#chan2 key1,key2)
	var keys []string
	if msg.HasParam(1) {
		keys = strings.Split(msg.GetParam(1), ",")
	}

	for i, channelName := range channelNames {
		channelName = strings.TrimSpace(channelName)

		// Validate channel name (must start with # or &)
		if !isValidChannelName(channelName) {
			h.sendNumeric(c, ERR_NOSUCHCHANNEL, channelName+" :No such channel")
			continue
		}

		// Get or create channel
		ch := h.channels.CreateChannel(channelName)

		// Check if already a member
		if ch.HasMember(c) {
			continue
		}

		// Check channel key if +k mode is set
		if ch.HasMode('k') {
			providedKey := ""
			if i < len(keys) {
				providedKey = strings.TrimSpace(keys[i])
			}
			
			if !ch.CheckKey(providedKey) {
				h.sendNumeric(c, ERR_BADCHANNELKEY, channelName+" :Cannot join channel (+k)")
				continue
			}
		}

		// Add client to channel
		ch.AddMember(c)
		c.JoinChannel(channelName)

		h.logger.Info("Client joined channel", "nickname", c.GetNickname(), "channel", channelName)

		// Send JOIN confirmation to the client
		joinMsg := fmt.Sprintf(":%s JOIN %s", c.GetHostmask(), channelName)
		c.Send(joinMsg)

		// Broadcast JOIN to other members
		ch.Broadcast(joinMsg, c)
		
		// Propagate JOIN to remote servers (Phase 7.4.3)
		if h.router != nil {
			parts := strings.SplitN(c.GetHostmask(), "!", 2)
			user := ""
			host := ""
			if len(parts) == 2 {
				userhost := strings.SplitN(parts[1], "@", 2)
				if len(userhost) == 2 {
					user = userhost[0]
					host = userhost[1]
				}
			}
			
			uid := c.GetUID()
			if uid == "" {
				uid = c.GetNickname() // Fallback if no UID
			}
			
			if err := h.router.PropagateJoin(c.GetNickname(), user, host, uid, channelName, time.Now().Unix()); err != nil {
				h.logger.Debug("Failed to propagate JOIN", "error", err, "channel", channelName)
			}
		}

		// Send topic if it exists
		topic := ch.GetTopic()
		if topic != "" {
			h.sendNumeric(c, RPL_TOPIC, fmt.Sprintf("%s :%s", channelName, topic))
		} else {
			h.sendNumeric(c, RPL_NOTOPIC, fmt.Sprintf("%s :No topic is set", channelName))
		}

		// Send NAMES list
		h.sendNamesList(c, ch)
	}

	return nil
}

// handlePart handles the PART command
func (h *Handler) handlePart(c *client.Client, msg *parser.Message) error {
	// Check if registered
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// PART requires at least one parameter
	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "PART :Not enough parameters")
		return nil
	}

	channelNames := strings.Split(msg.GetParam(0), ",")
	partMsg := "Leaving"
	if msg.HasParam(1) {
		partMsg = msg.GetParam(1)
	}

	for _, channelName := range channelNames {
		channelName = strings.TrimSpace(channelName)

		// Get channel
		ch := h.channels.GetChannel(channelName)
		if ch == nil {
			h.sendNumeric(c, ERR_NOSUCHCHANNEL, channelName+" :No such channel")
			continue
		}

		// Check if member
		if !ch.HasMember(c) {
			h.sendNumeric(c, ERR_NOTONCHANNEL, channelName+" :You're not on that channel")
			continue
		}

		h.logger.Info("Client left channel", "nickname", c.GetNickname(), "channel", channelName)

		// Send PART to everyone including the client
		partNotice := fmt.Sprintf(":%s PART %s :%s", c.GetHostmask(), channelName, partMsg)
		ch.BroadcastAll(partNotice)

		// Remove client from channel
		ch.RemoveMember(c)
		c.PartChannel(channelName)
		
		// Propagate PART to remote servers (Phase 7.4.3)
		if h.router != nil {
			parts := strings.SplitN(c.GetHostmask(), "!", 2)
			user := ""
			host := ""
			if len(parts) == 2 {
				userhost := strings.SplitN(parts[1], "@", 2)
				if len(userhost) == 2 {
					user = userhost[0]
					host = userhost[1]
				}
			}
			
			uid := c.GetUID()
			if uid == "" {
				uid = c.GetNickname()
			}
			
			if err := h.router.PropagatePart(c.GetNickname(), user, host, uid, channelName, partMsg); err != nil {
				h.logger.Debug("Failed to propagate PART", "error", err, "channel", channelName)
			}
		}

		// Remove empty channels
		if ch.IsEmpty() {
			h.channels.RemoveChannel(channelName)
		}
	}

	return nil
}

// handlePrivmsg handles the PRIVMSG command
func (h *Handler) handlePrivmsg(c *client.Client, msg *parser.Message) error {
	return h.handleMessage(c, msg, "PRIVMSG")
}

// handleNotice handles the NOTICE command
func (h *Handler) handleNotice(c *client.Client, msg *parser.Message) error {
	return h.handleMessage(c, msg, "NOTICE")
}

// handleMessage is a common handler for PRIVMSG and NOTICE
func (h *Handler) handleMessage(c *client.Client, msg *parser.Message, cmdType string) error {
	// Check if registered
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// Need target and message
	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NORECIPIENT, fmt.Sprintf(":No recipient given (%s)", cmdType))
		return nil
	}

	if !msg.HasParam(1) {
		h.sendNumeric(c, ERR_NOTEXTTOSEND, ":No text to send")
		return nil
	}

	target := msg.GetParam(0)
	message := msg.GetParam(1)

	// Check if target is a channel
	if isValidChannelName(target) {
		ch := h.channels.GetChannel(target)
		if ch == nil {
			h.sendNumeric(c, ERR_NOSUCHCHANNEL, target+" :No such channel")
			return nil
		}

		// Check if sender is a member
		if !ch.HasMember(c) {
			h.sendNumeric(c, ERR_CANNOTSENDTOCHAN, target+" :Cannot send to channel")
			return nil
		}

		// Check if channel is moderated and user can speak
		if ch.HasMode('m') && !ch.CanSpeak(c) {
			h.sendNumeric(c, ERR_CANNOTSENDTOCHAN, target+" :Cannot send to channel (+m)")
			return nil
		}

		// Broadcast message to channel (excluding sender)
		msgText := fmt.Sprintf(":%s %s %s :%s", c.GetHostmask(), cmdType, target, message)
		ch.Broadcast(msgText, c)

		// Route to remote servers with channel members (Phase 7.4)
		if h.router != nil {
			nick := c.GetNickname()
			parts := strings.SplitN(c.GetHostmask(), "!", 2)
			user := ""
			host := ""
			if len(parts) == 2 {
				userhost := strings.SplitN(parts[1], "@", 2)
				if len(userhost) == 2 {
					user = userhost[0]
					host = userhost[1]
				}
			}
			
			if err := h.router.RouteChannelMessage(nick, user, host, target, message, cmdType); err != nil {
				h.logger.Debug("Failed to route channel message", "error", err, "channel", target)
			}
		}

		h.logger.Debug("Channel message", "from", c.GetNickname(), "channel", target)
	} else {
		// Private message to user
		targetClient := h.clients.GetClient(target)
		if targetClient == nil {
			// Check if user exists on remote server (Phase 7.4)
			if h.router != nil && !h.router.IsUserLocal(target) {
				// User might be on a remote server, try routing
				nick := c.GetNickname()
				parts := strings.SplitN(c.GetHostmask(), "!", 2)
				user := ""
				host := ""
				if len(parts) == 2 {
					userhost := strings.SplitN(parts[1], "@", 2)
					if len(userhost) == 2 {
						user = userhost[0]
						host = userhost[1]
					}
				}
				
				var err error
				if cmdType == "PRIVMSG" {
					err = h.router.RoutePrivmsg(nick, user, host, target, message)
				} else {
					err = h.router.RouteNotice(nick, user, host, target, message)
				}
				
				if err != nil {
					// User not found on remote servers either
					h.sendNumeric(c, ERR_NOSUCHNICK, target+" :No such nick/channel")
				} else {
					h.logger.Debug("Routed to remote server", "from", nick, "to", target, "type", cmdType)
				}
				return nil
			}
			
			h.sendNumeric(c, ERR_NOSUCHNICK, target+" :No such nick/channel")
			return nil
		}

		msgText := fmt.Sprintf(":%s %s %s :%s", c.GetHostmask(), cmdType, target, message)
		targetClient.Send(msgText)

		// If target is away, notify sender (only for PRIVMSG, not NOTICE)
		if cmdType == "PRIVMSG" && targetClient.IsAway() {
			h.sendNumeric(c, RPL_AWAY, fmt.Sprintf("%s :%s", target, targetClient.GetAwayMessage()))
		}

		h.logger.Debug("Private message", "from", c.GetNickname(), "to", target)
	}

	return nil
}

// handleNames handles the NAMES command
func (h *Handler) handleNames(c *client.Client, msg *parser.Message) error {
	// Check if registered
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// If no parameter, list all channels the user is in
	if !msg.HasParam(0) {
		for _, channelName := range c.GetChannels() {
			if ch := h.channels.GetChannel(channelName); ch != nil {
				h.sendNamesList(c, ch)
			}
		}
		h.sendNumeric(c, RPL_ENDOFNAMES, "* :End of NAMES list")
		return nil
	}

	channelNames := strings.Split(msg.GetParam(0), ",")

	for _, channelName := range channelNames {
		channelName = strings.TrimSpace(channelName)

		ch := h.channels.GetChannel(channelName)
		if ch == nil {
			continue
		}

		h.sendNamesList(c, ch)
	}

	return nil
}

// handleTopic handles the TOPIC command
func (h *Handler) handleTopic(c *client.Client, msg *parser.Message) error {
	// Check if registered
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// TOPIC requires at least one parameter (channel name)
	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "TOPIC :Not enough parameters")
		return nil
	}

	channelName := msg.GetParam(0)
	ch := h.channels.GetChannel(channelName)

	if ch == nil {
		h.sendNumeric(c, ERR_NOSUCHCHANNEL, channelName+" :No such channel")
		return nil
	}

	// Check if member of channel
	if !ch.HasMember(c) {
		h.sendNumeric(c, ERR_NOTONCHANNEL, channelName+" :You're not on that channel")
		return nil
	}

	// If no topic parameter, return current topic
	if !msg.HasParam(1) {
		topic := ch.GetTopic()
		if topic != "" {
			h.sendNumeric(c, RPL_TOPIC, fmt.Sprintf("%s :%s", channelName, topic))
		} else {
			h.sendNumeric(c, RPL_NOTOPIC, fmt.Sprintf("%s :No topic is set", channelName))
		}
		return nil
	}

	// Set new topic
	newTopic := msg.GetParam(1)
	ch.SetTopic(newTopic)

	// Broadcast topic change to all members
	topicMsg := fmt.Sprintf(":%s TOPIC %s :%s", c.GetHostmask(), channelName, newTopic)
	ch.BroadcastAll(topicMsg)

	// Propagate TOPIC to remote servers (Phase 7.4.4)
	if h.router != nil {
		parts := strings.SplitN(c.GetHostmask(), "!", 2)
		user := ""
		host := ""
		if len(parts) == 2 {
			userhost := strings.SplitN(parts[1], "@", 2)
			if len(userhost) == 2 {
				user = userhost[0]
				host = userhost[1]
			}
		}
		
		uid := c.GetUID()
		if uid == "" {
			uid = c.GetNickname()
		}
		
		if err := h.router.PropagateTopic(c.GetNickname(), user, host, uid, channelName, newTopic, time.Now().Unix()); err != nil {
			h.logger.Debug("Failed to propagate TOPIC", "error", err)
		}
	}

	h.logger.Info("Topic changed", "channel", channelName, "by", c.GetNickname())

	return nil
}

// sendNamesList sends the NAMES list for a channel
func (h *Handler) sendNamesList(c *client.Client, ch *channel.Channel) {
	nicks := ch.GetMemberNicks()
	
	// Add remote users from network state if we have a router (server linking enabled)
	if h.router != nil {
		channelName := ch.GetName()
		if remoteChan, exists := h.router.GetRemoteChannel(channelName); exists {
			// Add remote users who are in this channel
			for uid := range remoteChan.Members {
				if remoteUser, ok := h.router.GetRemoteUserByUID(uid); ok {
					// Don't duplicate local users
					isLocal := false
					for _, localNick := range nicks {
						cleanNick := strings.TrimPrefix(strings.TrimPrefix(localNick, "@"), "+")
						if cleanNick == remoteUser.Nick {
							isLocal = true
							break
						}
					}
					if !isLocal {
						// Add remote user (no prefix since we don't track their op status locally)
						nicks = append(nicks, remoteUser.Nick)
					}
				}
			}
		}
	}
	
	nickList := strings.Join(nicks, " ")

	// Split into chunks if necessary (RFC says max 512 bytes per message)
	channelName := ch.GetName()
	h.sendNumeric(c, RPL_NAMREPLY, fmt.Sprintf("= %s :%s", channelName, nickList))
	h.sendNumeric(c, RPL_ENDOFNAMES, fmt.Sprintf("%s :End of NAMES list", channelName))
}

// tryRegister attempts to register the client if all requirements are met
func (h *Handler) tryRegister(c *client.Client) {
	// Already registered?
	if c.IsRegistered() {
		return
	}

	// Check if we have both nickname and username
	if c.GetNickname() == "" || !c.HasUsername() {
		return
	}

	// Mark as registered
	c.SetRegistered(true)

	// Send welcome messages
	h.sendWelcome(c)
	
	// Note: User propagation is handled in AddClient() where UID is assigned
}

// isValidNickname checks if a nickname is valid according to RFC 2812
// Valid nicknames: letter or special, followed by any combination of letters, digits, or specials
// Specials: [ ] \ ` _ ^ { | }
// Maximum length: 9 characters per RFC 1459, but we allow up to 16 per modern IRC
func isValidNickname(nick string) bool {
	if len(nick) == 0 || len(nick) > 16 {
		return false
	}

	// First character must be letter or special
	first := nick[0]
	if !isLetter(first) && !isSpecial(first) {
		return false
	}

	// Remaining characters can be letter, digit, or special
	for i := 1; i < len(nick); i++ {
		ch := nick[i]
		if !isLetter(ch) && !isDigit(ch) && !isSpecial(ch) && ch != '-' {
			return false
		}
	}

	return true
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isSpecial(ch byte) bool {
	return strings.ContainsRune("[]\\`_^{|}", rune(ch))
}

// isValidChannelName checks if a channel name is valid
// Valid channels start with # or & and contain no spaces, commas, or control characters
func isValidChannelName(name string) bool {
	if len(name) < 2 || len(name) > 50 {
		return false
	}

	// Must start with # or &
	if name[0] != '#' && name[0] != '&' {
		return false
	}

	// Check for invalid characters
	for i := 1; i < len(name); i++ {
		ch := name[i]
		// No spaces, commas, or control characters
		if ch == ' ' || ch == ',' || ch == 7 || ch < 32 {
			return false
		}
	}

	return true
}

// handleMode handles the MODE command for users and channels
// MODE <target> [<modestring> [<mode arguments>...]]
func (h *Handler) handleMode(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if len(msg.Params) < 1 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "MODE :Not enough parameters")
		return nil
	}

	target := msg.Params[0]

	// Check if target is a channel or user
	if target[0] == '#' || target[0] == '&' {
		return h.handleChannelMode(c, msg)
	}
	return h.handleUserMode(c, msg)
}

// handleUserMode handles MODE for users
func (h *Handler) handleUserMode(c *client.Client, msg *parser.Message) error {
	target := msg.Params[0]

	// Users can only set modes on themselves
	if target != c.GetNickname() {
		h.sendNumeric(c, ERR_USERSDONTMATCH, ":Cannot change mode for other users")
		return nil
	}

	// If no mode string, return current modes
	if len(msg.Params) < 2 {
		modes := c.GetModes()
		if modes == "" {
			modes = "+"
		}
		h.sendNumeric(c, RPL_UMODEIS, modes)
		return nil
	}

	modeString := msg.Params[1]
	adding := true

	for _, ch := range modeString {
		switch ch {
		case '+':
			adding = true
		case '-':
			adding = false
		case 'i': // invisible
			c.SetMode('i', adding)
		case 'o': // operator (can only be removed, not added by user)
			if !adding {
				c.SetMode('o', false)
			}
		case 'w': // wallops
			c.SetMode('w', adding)
		default:
			h.sendNumeric(c, ERR_UMODEUNKNOWNFLAG, ":Unknown MODE flag")
		}
	}

	// Confirm mode change
	modes := c.GetModes()
	if modes == "" {
		modes = "+"
	}
	c.Send(fmt.Sprintf(":%s MODE %s %s", c.GetHostmask(), c.GetNickname(), modes))

	return nil
}

// handleChannelMode handles MODE for channels
func (h *Handler) handleChannelMode(c *client.Client, msg *parser.Message) error {
	channelName := msg.Params[0]

	ch := h.channels.GetChannel(channelName)
	if ch == nil {
		h.sendNumeric(c, ERR_NOSUCHCHANNEL, channelName+" :No such channel")
		return nil
	}

	// Check if user is in channel
	if !ch.HasMember(c) {
		h.sendNumeric(c, ERR_NOTONCHANNEL, channelName+" :You're not on that channel")
		return nil
	}

	// If no mode string, return current modes
	if len(msg.Params) < 2 {
		modes := ch.GetModes()
		h.sendNumeric(c, RPL_CHANNELMODEIS, channelName+" "+modes)
		return nil
	}

	// Check if user is channel operator
	if !ch.IsOperator(c) {
		h.sendNumeric(c, ERR_CHANOPRIVSNEEDED, channelName+" :You're not channel operator")
		return nil
	}

	modeString := msg.Params[1]
	modeArgs := msg.Params[2:]
	argIndex := 0
	adding := true
	changes := ""

	for _, modeChar := range modeString {
		switch modeChar {
		case '+':
			adding = true
			changes += "+"
		case '-':
			adding = false
			changes += "-"
		case 'o': // operator
			if argIndex < len(modeArgs) {
				targetNick := modeArgs[argIndex]
				argIndex++
				targetClient := ch.GetMemberByNick(targetNick)
				if targetClient != nil {
					ch.SetOperator(targetClient, adding)
					changes += "o"
				}
			}
		case 'v': // voice
			if argIndex < len(modeArgs) {
				targetNick := modeArgs[argIndex]
				argIndex++
				targetClient := ch.GetMemberByNick(targetNick)
				if targetClient != nil {
					ch.SetVoice(targetClient, adding)
					changes += "v"
				}
			}
		case 'i': // invite-only
			ch.SetMode('i', adding)
			changes += "i"
		case 'm': // moderated
			ch.SetMode('m', adding)
			changes += "m"
		case 'n': // no external messages
			ch.SetMode('n', adding)
			changes += "n"
		case 't': // topic protection
			ch.SetMode('t', adding)
			changes += "t"
		case 'b': // ban
			if adding {
				if argIndex < len(modeArgs) {
					mask := modeArgs[argIndex]
					argIndex++
					ch.AddBan(mask)
					changes += "b"
				}
			} else {
				if argIndex < len(modeArgs) {
					mask := modeArgs[argIndex]
					argIndex++
					ch.RemoveBan(mask)
					changes += "b"
				}
			}
		case 'k': // channel key (password)
			if adding {
				if argIndex < len(modeArgs) {
					key := modeArgs[argIndex]
					argIndex++
					ch.SetKey(key)
					ch.SetMode('k', true)
					changes += "k"
				}
			} else {
				// Removing key
				ch.SetKey("")
				ch.SetMode('k', false)
				changes += "k"
			}
		default:
			h.sendNumeric(c, ERR_UNKNOWNMODE, string(modeChar)+" :is unknown mode char to me")
		}
	}

	// Broadcast mode change
	if changes != "" {
		ch.BroadcastAll(fmt.Sprintf(":%s MODE %s %s", c.GetHostmask(), channelName, changes))
		
		// Propagate MODE to remote servers (Phase 7.4.4)
		if h.router != nil {
			parts := strings.SplitN(c.GetHostmask(), "!", 2)
			user := ""
			host := ""
			if len(parts) == 2 {
				userhost := strings.SplitN(parts[1], "@", 2)
				if len(userhost) == 2 {
					user = userhost[0]
					host = userhost[1]
				}
			}
			
			uid := c.GetUID()
			if uid == "" {
				uid = c.GetNickname()
			}
			
			// Build full mode string with args for propagation
			fullModeStr := changes
			if len(modeArgs) > 0 {
				fullModeStr += " " + strings.Join(modeArgs[:argIndex], " ")
			}
			
			if err := h.router.PropagateMode(c.GetNickname(), user, host, uid, channelName, fullModeStr, time.Now().Unix()); err != nil {
				h.logger.Debug("Failed to propagate MODE", "error", err)
			}
		}
	}

	return nil
}

// handleKick handles the KICK command
// KICK <channel> <user> [<comment>]
func (h *Handler) handleKick(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if len(msg.Params) < 2 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "KICK :Not enough parameters")
		return nil
	}

	channelName := msg.Params[0]
	targetNick := msg.Params[1]
	reason := "Kicked"
	if len(msg.Params) > 2 {
		reason = msg.Params[2]
	}

	ch := h.channels.GetChannel(channelName)
	if ch == nil {
		h.sendNumeric(c, ERR_NOSUCHCHANNEL, channelName+" :No such channel")
		return nil
	}

	// Check if kicker is in channel
	if !ch.HasMember(c) {
		h.sendNumeric(c, ERR_NOTONCHANNEL, channelName+" :You're not on that channel")
		return nil
	}

	// Check if kicker is channel operator
	if !ch.IsOperator(c) {
		h.sendNumeric(c, ERR_CHANOPRIVSNEEDED, channelName+" :You're not channel operator")
		return nil
	}

	// Get target client
	targetClient := ch.GetMemberByNick(targetNick)
	if targetClient == nil {
		h.sendNumeric(c, ERR_USERNOTINCHANNEL, targetNick+" "+channelName+" :They aren't on that channel")
		return nil
	}

	// Broadcast KICK message
	kickMsg := fmt.Sprintf(":%s KICK %s %s :%s", c.GetHostmask(), channelName, targetNick, reason)
	ch.BroadcastAll(kickMsg)

	// Propagate KICK to remote servers (Phase 7.4.4)
	if h.router != nil {
		parts := strings.SplitN(c.GetHostmask(), "!", 2)
		user := ""
		host := ""
		if len(parts) == 2 {
			userhost := strings.SplitN(parts[1], "@", 2)
			if len(userhost) == 2 {
				user = userhost[0]
				host = userhost[1]
			}
		}
		
		uid := c.GetUID()
		if uid == "" {
			uid = c.GetNickname()
		}
		
		if err := h.router.PropagateKick(c.GetNickname(), user, host, uid, channelName, targetNick, reason); err != nil {
			h.logger.Debug("Failed to propagate KICK", "error", err)
		}
	}

	// Remove target from channel
	ch.RemoveMember(targetClient)
	targetClient.PartChannel(channelName)

	h.logger.Info("User kicked from channel", "channel", channelName, "target", targetNick, "by", c.GetNickname())

	return nil
}

// handleWho handles the WHO command
// Syntax: WHO <mask>
// Mask can be a channel name (#channel) or a pattern (* for all)
func (h *Handler) handleWho(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if len(msg.Params) == 0 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "WHO :Not enough parameters")
		return nil
	}

	mask := msg.Params[0]

	// Check if mask is a channel
	if strings.HasPrefix(mask, "#") {
		ch := h.channels.GetChannel(mask)
		if ch == nil {
			// No such channel, just send end of WHO
			h.sendNumeric(c, RPL_ENDOFWHO, mask+" :End of WHO list")
			return nil
		}

		// Send WHO reply for each member
		members := ch.GetMembers()
		for _, member := range members {
			h.sendWhoReply(c, member, mask, ch)
		}
	} else {
		// For now, just return all registered users (could implement pattern matching)
		// This is a simplified implementation
		h.sendNumeric(c, RPL_ENDOFWHO, mask+" :End of WHO list")
		return nil
	}

	h.sendNumeric(c, RPL_ENDOFWHO, mask+" :End of WHO list")
	return nil
}

// sendWhoReply sends a single WHO reply for a user
// Format: <channel> <user> <host> <server> <nick> <flags> :<hopcount> <realname>
func (h *Handler) sendWhoReply(c *client.Client, target *client.Client, channel string, ch *channel.Channel) {
	nick := target.GetNickname()
	hostmask := target.GetHostmask()
	
	// Parse hostmask to get user and host
	parts := strings.Split(hostmask, "!")
	user := nick
	host := "unknown"
	if len(parts) == 2 {
		user = parts[1]
		hostParts := strings.Split(user, "@")
		if len(hostParts) == 2 {
			user = hostParts[0]
			host = hostParts[1]
		}
	}

	// Build flags: H=here, G=away, *=ircop, @=chanop, +=voice
	flags := "H" // H = here (not away), G = gone (away)
	if target.IsAway() {
		flags = "G" // User is away
	}
	if target.HasMode('o') {
		flags += "*" // IRC operator
	}
	if ch != nil && ch.IsOperator(target) {
		flags += "@" // Channel operator
	} else if ch != nil && ch.IsVoiced(target) {
		flags += "+" // Voiced
	}

	// Format: <channel> <user> <host> <server> <nick> <flags> :<hopcount> <realname>
	reply := fmt.Sprintf("%s %s %s %s %s %s :0 User", channel, user, host, h.serverName, nick, flags)
	h.sendNumeric(c, RPL_WHOREPLY, reply)
}

// handleWhois handles the WHOIS command
// Syntax: WHOIS <nickname>
func (h *Handler) handleWhois(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if len(msg.Params) == 0 {
		h.sendNumeric(c, ERR_NONICKNAMEGIVEN, ":No nickname given")
		return nil
	}

	targetNick := msg.Params[0]
	target := h.clients.GetClient(targetNick)

	if target == nil {
		h.sendNumeric(c, ERR_NOSUCHNICK, targetNick+" :No such nick/channel")
		h.sendNumeric(c, RPL_ENDOFWHOIS, targetNick+" :End of WHOIS list")
		return nil
	}

	// Parse hostmask
	hostmask := target.GetHostmask()
	parts := strings.Split(hostmask, "!")
	user := targetNick
	host := "unknown"
	if len(parts) == 2 {
		userHost := parts[1]
		hostParts := strings.Split(userHost, "@")
		if len(hostParts) == 2 {
			user = hostParts[0]
			host = hostParts[1]
		}
	}

	// RPL_WHOISUSER: <nick> <user> <host> * :<realname>
	h.sendNumeric(c, RPL_WHOISUSER, fmt.Sprintf("%s %s %s * :User", targetNick, user, host))

	// RPL_AWAY: Show if user is away
	if target.IsAway() {
		h.sendNumeric(c, RPL_AWAY, fmt.Sprintf("%s :%s", targetNick, target.GetAwayMessage()))
	}

	// RPL_WHOISSERVER: <nick> <server> :<server info>
	h.sendNumeric(c, RPL_WHOISSERVER, fmt.Sprintf("%s %s :IRC Server", targetNick, h.serverName))

	// RPL_WHOISCHANNELS: <nick> :<channels>
	channels := target.GetChannels()
	if len(channels) > 0 {
		channelList := ""
		for i, chName := range channels {
			if i > 0 {
				channelList += " "
			}
			ch := h.channels.GetChannel(chName)
			if ch != nil && ch.IsOperator(target) {
				channelList += "@" + chName
			} else {
				channelList += chName
			}
		}
		h.sendNumeric(c, RPL_WHOISCHANNELS, fmt.Sprintf("%s :%s", targetNick, channelList))
	}

	// RPL_WHOISIDLE: <nick> <seconds> :seconds idle
	idleTime := int(time.Since(target.GetLastActivity()).Seconds())
	h.sendNumeric(c, RPL_WHOISIDLE, fmt.Sprintf("%s %d :seconds idle", targetNick, idleTime))

	// Check if user is an operator
	if target.HasMode('o') {
		h.sendNumeric(c, RPL_WHOISOPERATOR, targetNick+" :is an IRC operator")
	}

	// RPL_ENDOFWHOIS
	h.sendNumeric(c, RPL_ENDOFWHOIS, targetNick+" :End of WHOIS list")

	return nil
}

// handleList handles the LIST command
// Syntax: LIST [<channel>]
func (h *Handler) handleList(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// RPL_LISTSTART
	h.sendNumeric(c, RPL_LISTSTART, "Channel :Users  Name")

	// If specific channel requested
	if len(msg.Params) > 0 {
		channelName := msg.Params[0]
		ch := h.channels.GetChannel(channelName)
		if ch != nil {
			memberCount := len(ch.GetMembers())
			topic := ch.GetTopic()
			if topic == "" {
				topic = "No topic"
			}
			h.sendNumeric(c, RPL_LIST, fmt.Sprintf("%s %d :%s", channelName, memberCount, topic))
		}
	} else {
		// List all channels (simplified - in production you'd want to filter secret channels)
		// For now, we'll just send empty list as we don't have a global channel list
		// In a full implementation, you'd iterate through all channels
	}

	// RPL_LISTEND
	h.sendNumeric(c, RPL_LISTEND, ":End of LIST")

	return nil
}

// handleInvite handles the INVITE command
// Syntax: INVITE <nickname> <channel>
func (h *Handler) handleInvite(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if len(msg.Params) < 2 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "INVITE :Not enough parameters")
		return nil
	}

	targetNick := msg.Params[0]
	channelName := msg.Params[1]

	// Check if channel exists
	ch := h.channels.GetChannel(channelName)
	if ch == nil {
		h.sendNumeric(c, ERR_NOSUCHCHANNEL, channelName+" :No such channel")
		return nil
	}

	// Check if inviter is on the channel
	if !ch.HasMember(c) {
		h.sendNumeric(c, ERR_NOTONCHANNEL, channelName+" :You're not on that channel")
		return nil
	}

	// Check if inviter is an operator (required for invite-only channels)
	if ch.HasMode('i') && !ch.IsOperator(c) {
		h.sendNumeric(c, ERR_CHANOPRIVSNEEDED, channelName+" :You're not channel operator")
		return nil
	}

	// Check if target user exists
	target := h.clients.GetClient(targetNick)
	if target == nil {
		h.sendNumeric(c, ERR_NOSUCHNICK, targetNick+" :No such nick/channel")
		return nil
	}

	// Check if target is already on channel
	if ch.HasMember(target) {
		h.sendNumeric(c, ERR_USERONCHANNEL, targetNick+" "+channelName+" :is already on channel")
		return nil
	}

	// Send confirmation to inviter
	h.sendNumeric(c, RPL_INVITING, channelName+" "+targetNick)

	// Send INVITE notification to target
	inviteMsg := fmt.Sprintf(":%s INVITE %s %s", c.GetHostmask(), targetNick, channelName)
	target.Send(inviteMsg)

	// Propagate INVITE to remote servers (Phase 7.4.4)
	if h.router != nil {
		parts := strings.SplitN(c.GetHostmask(), "!", 2)
		user := ""
		host := ""
		if len(parts) == 2 {
			userhost := strings.SplitN(parts[1], "@", 2)
			if len(userhost) == 2 {
				user = userhost[0]
				host = userhost[1]
			}
		}
		
		uid := c.GetUID()
		if uid == "" {
			uid = c.GetNickname()
		}
		
		if err := h.router.PropagateInvite(c.GetNickname(), user, host, uid, targetNick, channelName); err != nil {
			h.logger.Debug("Failed to propagate INVITE", "error", err)
		}
	}

	h.logger.Info("User invited to channel", "channel", channelName, "target", targetNick, "by", c.GetNickname())

	return nil
}

// handleOper handles the OPER command
// OPER <name> <password>
func (h *Handler) handleOper(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if len(msg.Params) < 2 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "OPER :Not enough parameters")
		return nil
	}

	name := msg.Params[0]
	password := msg.Params[1]

	// Check if operator exists
	hashedPassword, exists := h.operators[name]
	if !exists {
		h.sendNumeric(c, ERR_PASSWDMISMATCH, ":Password incorrect")
		h.logger.Warn("OPER attempt with unknown name", "name", name, "client", c.GetNickname())
		return nil
	}

	// Verify password using bcrypt
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		h.sendNumeric(c, ERR_PASSWDMISMATCH, ":Password incorrect")
		h.logger.Warn("OPER attempt with wrong password", "name", name, "client", c.GetNickname())
		return nil
	}

	// Grant operator status
	c.SetMode('o', true)
	h.sendNumeric(c, RPL_YOUREOPER, ":You are now an IRC operator")

	h.logger.Info("User gained operator status", "nickname", c.GetNickname(), "oper_name", name)

	return nil
}

// handleSquit handles the SQUIT command (Phase 7.4.5)
// SQUIT <server> <comment>
func (h *Handler) handleSquit(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// Only operators can use SQUIT
	if !c.HasMode('o') {
		h.sendNumeric(c, ERR_NOPRIVILEGES, ":Permission Denied- You're not an IRC operator")
		return nil
	}

	if len(msg.Params) < 1 {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "SQUIT :Not enough parameters")
		return nil
	}

	serverName := msg.Params[0]
	reason := "No reason"
	if len(msg.Params) > 1 {
		reason = msg.Params[1]
	}

	// Check if router is available
	if h.router == nil {
		h.sendNumeric(c, ERR_UNKNOWNCOMMAND, "SQUIT :Server linking not enabled")
		return nil
	}

	// Disconnect the server
	if err := h.router.DisconnectServer(serverName, reason); err != nil {
		h.sendNumeric(c, ERR_NOSUCHSERVER, serverName+" :"+err.Error())
		h.logger.Warn("SQUIT failed", "server", serverName, "error", err, "operator", c.GetNickname())
		return nil
	}

	h.logger.Info("Server disconnected via SQUIT", "server", serverName, "reason", reason, "operator", c.GetNickname())

	return nil
}

// handleAway handles the AWAY command
// AWAY [<message>]
func (h *Handler) handleAway(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	// If no message provided, mark as no longer away
	if !msg.HasParam(0) || msg.GetParam(0) == "" {
		c.SetAway("")
		h.sendNumeric(c, RPL_UNAWAY, ":You are no longer marked as being away")
		h.logger.Debug("User no longer away", "nickname", c.GetNickname())
		return nil
	}

	// Set away message
	awayMsg := msg.GetParam(0)
	c.SetAway(awayMsg)
	h.sendNumeric(c, RPL_NOWAWAY, ":You have been marked as being away")

	h.logger.Debug("User marked as away", "nickname", c.GetNickname(), "message", awayMsg)

	return nil
}

// handleUserhost handles the USERHOST command
// USERHOST <nickname> [<nickname> ...]
func (h *Handler) handleUserhost(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "USERHOST :Not enough parameters")
		return nil
	}

	// Build response for up to 5 nicknames (RFC limit)
	var responses []string
	maxNicks := 5

	for i := 0; i < len(msg.Params) && len(responses) < maxNicks; i++ {
		nick := msg.Params[i]
		target := h.clients.GetClient(nick)
		
		if target != nil {
			// Parse hostmask to get user and host
			hostmask := target.GetHostmask()
			parts := strings.Split(hostmask, "!")
			user := nick
			host := "unknown"
			if len(parts) == 2 {
				userHost := parts[1]
				hostParts := strings.Split(userHost, "@")
				if len(hostParts) == 2 {
					user = hostParts[0]
					host = hostParts[1]
				}
			}
			
			// Format: <nick>[*]=<+|-><user>@<host>
			// * = operator, + = not away, - = away
			operFlag := ""
			if target.IsServerOperator() {
				operFlag = "*"
			}
			
			awayFlag := "+"
			if target.IsAway() {
				awayFlag = "-"
			}
			
			response := fmt.Sprintf("%s%s=%s%s@%s",
				target.GetNickname(),
				operFlag,
				awayFlag,
				user,
				host)
			
			responses = append(responses, response)
		}
	}

	if len(responses) > 0 {
		h.sendNumeric(c, RPL_USERHOST, ":"+strings.Join(responses, " "))
	}

	return nil
}

// handleIson handles the ISON command
// ISON <nickname> [<nickname> ...]
func (h *Handler) handleIson(c *client.Client, msg *parser.Message) error {
	if !c.IsRegistered() {
		h.sendNumeric(c, ERR_NOTREGISTERED, ":You have not registered")
		return nil
	}

	if !msg.HasParam(0) {
		h.sendNumeric(c, ERR_NEEDMOREPARAMS, "ISON :Not enough parameters")
		return nil
	}

	// Check which nicknames are online
	var onlineNicks []string

	for _, nick := range msg.Params {
		if h.clients.GetClient(nick) != nil {
			onlineNicks = append(onlineNicks, nick)
		}
	}

	// Always send RPL_ISON, even if empty
	if len(onlineNicks) > 0 {
		h.sendNumeric(c, RPL_ISON, ":"+strings.Join(onlineNicks, " "))
	} else {
		h.sendNumeric(c, RPL_ISON, ":")
	}

	return nil
}
