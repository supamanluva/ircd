package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/supamanluva/ircd/internal/channel"
	"github.com/supamanluva/ircd/internal/client"
	"github.com/supamanluva/ircd/internal/logger"
	"github.com/supamanluva/ircd/internal/parser"
)

// Handler processes IRC commands
type Handler struct {
	serverName string
	logger     *logger.Logger
	clients    ClientRegistry
	channels   ChannelRegistry
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

// CommandFunc is the signature for command handler functions
type CommandFunc func(c *client.Client, msg *parser.Message) error

// New creates a new command handler
func New(serverName string, log *logger.Logger, clients ClientRegistry, channels ChannelRegistry) *Handler {
	return &Handler{
		serverName: serverName,
		logger:     log,
		clients:    clients,
		channels:   channels,
	}
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
		// TODO: Broadcast to channels in Phase 2
	}

	// Check if client should be registered now
	h.tryRegister(c)

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
	h.tryRegister(c)

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

	for _, channelName := range channelNames {
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

		// Add client to channel
		ch.AddMember(c)
		c.JoinChannel(channelName)

		h.logger.Info("Client joined channel", "nickname", c.GetNickname(), "channel", channelName)

		// Send JOIN confirmation to the client
		joinMsg := fmt.Sprintf(":%s JOIN %s", c.GetHostmask(), channelName)
		c.Send(joinMsg)

		// Broadcast JOIN to other members
		ch.Broadcast(joinMsg, c)

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

		// Broadcast message to channel (excluding sender)
		msgText := fmt.Sprintf(":%s %s %s :%s", c.GetHostmask(), cmdType, target, message)
		ch.Broadcast(msgText, c)

		h.logger.Debug("Channel message", "from", c.GetNickname(), "channel", target)
	} else {
		// Private message to user
		targetClient := h.clients.GetClient(target)
		if targetClient == nil {
			h.sendNumeric(c, ERR_NOSUCHNICK, target+" :No such nick/channel")
			return nil
		}

		msgText := fmt.Sprintf(":%s %s %s :%s", c.GetHostmask(), cmdType, target, message)
		targetClient.Send(msgText)

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

	h.logger.Info("Topic changed", "channel", channelName, "by", c.GetNickname())

	return nil
}

// sendNamesList sends the NAMES list for a channel
func (h *Handler) sendNamesList(c *client.Client, ch *channel.Channel) {
	nicks := ch.GetMemberNicks()
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
		default:
			h.sendNumeric(c, ERR_UNKNOWNMODE, string(modeChar)+" :is unknown mode char to me")
		}
	}

	// Broadcast mode change
	if changes != "" {
		ch.BroadcastAll(fmt.Sprintf(":%s MODE %s %s", c.GetHostmask(), channelName, changes))
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
	flags := "H" // H = here (not away)
	if target.HasMode('o') {
		flags += "*" // IRC operator
	}
	if ch != nil && ch.IsOperator(target) {
		flags += "@" // Channel operator
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

	h.logger.Info("User invited to channel", "channel", channelName, "target", targetNick, "by", c.GetNickname())

	return nil
}
