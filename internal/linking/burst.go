package linking

import (
	"fmt"
)

// BurstState tracks the state of a burst operation
type BurstState struct {
	InProgress bool
	UsersRecv  int
	ChansRecv  int
}

// SendBurst sends all local users and channels to a remote server
func (l *Link) SendBurst(network *Network, localClients map[string]interface{}, localChannels map[string]interface{}) error {
	if !l.IsRegistered() {
		return fmt.Errorf("link not registered")
	}
	
	server := l.GetServer()
	if server == nil {
		return fmt.Errorf("no server object")
	}
	
	userCount := 0
	channelCount := 0
	
	// Send all local users as UID messages
	// Note: localClients is a placeholder - actual implementation will use proper client objects
	for nick, clientData := range localClients {
		// In real implementation, we'd extract client details
		// For now, create a stub UID message
		uid := network.GenerateUID()
		msg := BuildUID(
			network.LocalSID,
			nick,
			"user",
			"localhost",
			"127.0.0.1",
			uid,
			"+i",
			"User",
			1729268400, // Placeholder timestamp
		)
		
		if err := l.WriteMessage(msg); err != nil {
			return fmt.Errorf("failed to send UID: %v", err)
		}
		
		userCount++
		_ = clientData // Avoid unused variable error
	}
	
	// Send all channels as SJOIN messages
	for chanName, chanData := range localChannels {
		// In real implementation, we'd get channel members and modes
		members := make(map[string]string)
		
		msg := BuildSJOIN(
			network.LocalSID,
			chanName,
			1729268400, // Placeholder timestamp
			"+nt",
			members,
		)
		
		if err := l.WriteMessage(msg); err != nil {
			return fmt.Errorf("failed to send SJOIN: %v", err)
		}
		
		channelCount++
		_ = chanData // Avoid unused variable error
	}
	
	// Send burst completion marker (using PING as end-of-burst marker)
	eobMsg := BuildPING(network.LocalSID, server.SID)
	if err := l.WriteMessage(eobMsg); err != nil {
		return fmt.Errorf("failed to send end-of-burst: %v", err)
	}
	
	return nil
}

// HandleBurstMessage processes a message received during burst
func (l *Link) HandleBurstMessage(network *Network, msg *Message, burstState *BurstState) error {
	switch msg.Command {
	case "UID":
		// Parse and add remote user
		user, err := ParseUID(msg)
		if err != nil {
			return fmt.Errorf("invalid UID during burst: %v", err)
		}
		
		// Get source server
		sourceSID := msg.Source
		server, ok := network.GetServer(sourceSID)
		if !ok {
			return fmt.Errorf("unknown source server: %s", sourceSID)
		}
		
		user.Server = server
		
		// Add to network
		if err := network.AddUser(user); err != nil {
			return fmt.Errorf("failed to add user %s: %v", user.Nick, err)
		}
		
		burstState.UsersRecv++
		return nil
		
	case "SJOIN":
		// Parse channel
		channel, ts, modes, members, err := ParseSJOIN(msg)
		if err != nil {
			return fmt.Errorf("invalid SJOIN during burst: %v", err)
		}
		
		// Create RemoteChannel
		remoteChan := &RemoteChannel{
			Name:    channel,
			TS:      ts,
			Modes:   modes,
			Members: members,
		}
		
		// Add/merge channel
		network.AddChannel(remoteChan)
		
		// Update user channel membership
		for uid := range members {
			if user, ok := network.GetUserByUID(uid); ok {
				user.mu.Lock()
				user.Channels[channel] = true
				user.mu.Unlock()
			}
		}
		
		burstState.ChansRecv++
		return nil
		
	case "PING":
		// End of burst marker
		burstState.InProgress = false
		
		// Send PONG response
		pong := BuildPONG(network.LocalSID, msg.Source)
		return l.WriteMessage(pong)
		
	case "PONG":
		// Response to our PING - ignore during burst
		return nil
		
	default:
		return fmt.Errorf("unexpected command during burst: %s", msg.Command)
	}
}

// ReceiveBurst receives burst from remote server
func (l *Link) ReceiveBurst(network *Network) (*BurstState, error) {
	burstState := &BurstState{
		InProgress: true,
		UsersRecv:  0,
		ChansRecv:  0,
	}
	
	for burstState.InProgress {
		msg, err := l.ReadMessage()
		if err != nil {
			return burstState, fmt.Errorf("error reading burst: %v", err)
		}
		
		if err := l.HandleBurstMessage(network, msg, burstState); err != nil {
			return burstState, err
		}
	}
	
	return burstState, nil
}

// SendBurstFromClients sends burst from actual client/channel data structures
func (l *Link) SendBurstFromClients(network *Network, getClients func() []BurstClient, getChannels func() []BurstChannel) error {
	if !l.IsRegistered() {
		return fmt.Errorf("link not registered")
	}
	
	server := l.GetServer()
	if server == nil {
		return fmt.Errorf("no server object")
	}
	
	// Send all local users
	clients := getClients()
	for _, client := range clients {
		uid := network.GenerateUID()
		
		msg := BuildUID(
			network.LocalSID,
			client.Nick,
			client.User,
			client.Host,
			client.IP,
			uid,
			client.Modes,
			client.RealName,
			client.Timestamp,
		)
		
		if err := l.WriteMessage(msg); err != nil {
			return fmt.Errorf("failed to send UID for %s: %v", client.Nick, err)
		}
	}
	
	// Send all channels
	channels := getChannels()
	for _, channel := range channels {
		msg := BuildSJOIN(
			network.LocalSID,
			channel.Name,
			channel.TS,
			channel.Modes,
			channel.Members,
		)
		
		if err := l.WriteMessage(msg); err != nil {
			return fmt.Errorf("failed to send SJOIN for %s: %v", channel.Name, err)
		}
	}
	
	// Send end-of-burst marker
	eobMsg := BuildPING(network.LocalSID, server.SID)
	if err := l.WriteMessage(eobMsg); err != nil {
		return fmt.Errorf("failed to send end-of-burst: %v", err)
	}
	
	return nil
}

// BurstClient represents a client for burst synchronization
type BurstClient struct {
	Nick      string
	User      string
	Host      string
	IP        string
	Modes     string
	RealName  string
	Timestamp int64
}

// BurstChannel represents a channel for burst synchronization
type BurstChannel struct {
	Name    string
	TS      int64
	Modes   string
	Members map[string]string // UID -> modes (@, +, etc)
}
