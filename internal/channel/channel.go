package channel

import (
	"sync"
	"time"

	"github.com/supamanluva/ircd/internal/client"
)

// Channel represents an IRC channel (chat room)
type Channel struct {
	name      string
	topic     string
	key       string                     // channel key for +k mode
	createdAt time.Time
	members   map[string]*client.Client // nickname -> client
	operators map[string]bool            // nickname -> is operator
	voiced    map[string]bool            // nickname -> has voice (+v)
	modes     map[rune]bool              // channel modes (i, m, n, t, etc.)
	banList   []string                   // ban masks (nick!user@host patterns)
	mu        sync.RWMutex
}

// New creates a new channel
func New(name string) *Channel {
	ch := &Channel{
		name:      name,
		createdAt: time.Now(),
		members:   make(map[string]*client.Client),
		operators: make(map[string]bool),
		voiced:    make(map[string]bool),
		modes:     make(map[rune]bool),
		banList:   make([]string, 0),
	}
	
	// Set default modes
	ch.modes['n'] = true // no external messages
	ch.modes['t'] = true // only ops can set topic
	
	return ch
}

// GetName returns the channel name
func (ch *Channel) GetName() string {
	return ch.name
}

// GetTopic returns the channel topic
func (ch *Channel) GetTopic() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.topic
}

// SetTopic sets the channel topic
func (ch *Channel) SetTopic(topic string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.topic = topic
}

// AddMember adds a client to the channel
func (ch *Channel) AddMember(c *client.Client) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	
	nick := c.GetNickname()
	ch.members[nick] = c
	
	// First member becomes operator
	if len(ch.members) == 1 {
		ch.operators[nick] = true
	}
}

// RemoveMember removes a client from the channel
func (ch *Channel) RemoveMember(c *client.Client) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	
	nick := c.GetNickname()
	delete(ch.members, nick)
	delete(ch.operators, nick)
	delete(ch.voiced, nick)
}

// HasMember checks if a client is in the channel
func (ch *Channel) HasMember(c *client.Client) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	nick := c.GetNickname()
	_, exists := ch.members[nick]
	return exists
}

// GetMembers returns a slice of all members
func (ch *Channel) GetMembers() []*client.Client {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	members := make([]*client.Client, 0, len(ch.members))
	for _, c := range ch.members {
		members = append(members, c)
	}
	return members
}

// GetMemberCount returns the number of members
func (ch *Channel) GetMemberCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.members)
}

// IsOperator checks if a client is an operator
func (ch *Channel) IsOperator(c *client.Client) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	nick := c.GetNickname()
	return ch.operators[nick]
}

// SetOperator sets or unsets operator status for a client
func (ch *Channel) SetOperator(c *client.Client, isOp bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	
	nick := c.GetNickname()
	if isOp {
		ch.operators[nick] = true
	} else {
		delete(ch.operators, nick)
	}
}

// IsVoiced checks if a client has voice
func (ch *Channel) IsVoiced(c *client.Client) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	nick := c.GetNickname()
	return ch.voiced[nick]
}

// SetVoice sets or unsets voice status for a client
func (ch *Channel) SetVoice(c *client.Client, hasVoice bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	
	nick := c.GetNickname()
	if hasVoice {
		ch.voiced[nick] = true
	} else {
		delete(ch.voiced, nick)
	}
}

// CanSpeak checks if a client can speak in a moderated channel
func (ch *Channel) CanSpeak(c *client.Client) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	// If not moderated, everyone can speak
	if !ch.modes['m'] {
		return true
	}
	
	nick := c.GetNickname()
	// Operators and voiced users can speak in moderated channels
	return ch.operators[nick] || ch.voiced[nick]
}

// Broadcast sends a message to all members except the sender
func (ch *Channel) Broadcast(message string, sender *client.Client) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	senderNick := ""
	if sender != nil {
		senderNick = sender.GetNickname()
	}
	
	for nick, c := range ch.members {
		if nick != senderNick {
			c.Send(message)
		}
	}
}

// BroadcastFrom sends a message to all members including the sender
func (ch *Channel) BroadcastAll(message string) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	for _, c := range ch.members {
		c.Send(message)
	}
}

// IsEmpty returns true if the channel has no members
func (ch *Channel) IsEmpty() bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.members) == 0
}

// GetMemberNicks returns a slice of all member nicknames
func (ch *Channel) GetMemberNicks() []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	nicks := make([]string, 0, len(ch.members))
	for nick := range ch.members {
		// Prefix operators with @, voiced with +
		if ch.operators[nick] {
			nicks = append(nicks, "@"+nick)
		} else if ch.voiced[nick] {
			nicks = append(nicks, "+"+nick)
		} else {
			nicks = append(nicks, nick)
		}
	}
	return nicks
}

// SetMode sets or unsets a channel mode
func (ch *Channel) SetMode(mode rune, enabled bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if enabled {
		ch.modes[mode] = true
	} else {
		delete(ch.modes, mode)
	}
}

// HasMode checks if a channel has a specific mode
func (ch *Channel) HasMode(mode rune) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.modes[mode]
}

// GetModes returns a string representation of channel modes
func (ch *Channel) GetModes() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	modes := ""
	for mode := range ch.modes {
		modes += string(mode)
	}
	if modes == "" {
		return ""
	}
	return "+" + modes
}

// AddBan adds a ban mask to the channel
func (ch *Channel) AddBan(mask string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	
	// Check if already banned
	for _, ban := range ch.banList {
		if ban == mask {
			return
		}
	}
	ch.banList = append(ch.banList, mask)
}

// RemoveBan removes a ban mask from the channel
func (ch *Channel) RemoveBan(mask string) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	
	for i, ban := range ch.banList {
		if ban == mask {
			ch.banList = append(ch.banList[:i], ch.banList[i+1:]...)
			return true
		}
	}
	return false
}

// GetBanList returns a copy of the ban list
func (ch *Channel) GetBanList() []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	bans := make([]string, len(ch.banList))
	copy(bans, ch.banList)
	return bans
}

// IsBanned checks if a hostmask matches any ban
func (ch *Channel) IsBanned(hostmask string) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	for _, ban := range ch.banList {
		if matchMask(ban, hostmask) {
			return true
		}
	}
	return false
}

// matchMask checks if a mask matches a hostmask
// Supports * (any sequence) and ? (any single char)
func matchMask(mask, hostmask string) bool {
	// Simple implementation - can be enhanced with proper wildcard matching
	if mask == hostmask {
		return true
	}
	
	// TODO: Implement proper wildcard matching with * and ?
	// For now, exact match only
	return false
}

// GetMemberByNick returns a member by nickname
func (ch *Channel) GetMemberByNick(nick string) *client.Client {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.members[nick]
}

// SetKey sets the channel key (password) for +k mode
func (ch *Channel) SetKey(key string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.key = key
}

// GetKey returns the channel key
func (ch *Channel) GetKey() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.key
}

// CheckKey validates if the provided key matches the channel key
func (ch *Channel) CheckKey(providedKey string) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	
	// If no key is set, allow entry
	if ch.key == "" {
		return true
	}
	
	return ch.key == providedKey
}
