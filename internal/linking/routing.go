package linking

import (
	"fmt"
	"sync"
)

// LinkRegistry manages active server-to-server connections
type LinkRegistry struct {
	links map[string]*Link // SID -> Link
	mu    sync.RWMutex
}

// NewLinkRegistry creates a new link registry
func NewLinkRegistry() *LinkRegistry {
	return &LinkRegistry{
		links: make(map[string]*Link),
	}
}

// AddLink registers a new active link
func (lr *LinkRegistry) AddLink(sid string, link *Link) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	
	if _, exists := lr.links[sid]; exists {
		return fmt.Errorf("link for server %s already exists", sid)
	}
	
	lr.links[sid] = link
	return nil
}

// RemoveLink removes a link from the registry
func (lr *LinkRegistry) RemoveLink(sid string) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	delete(lr.links, sid)
}

// GetLink returns a link by server ID
func (lr *LinkRegistry) GetLink(sid string) (*Link, bool) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	link, ok := lr.links[sid]
	return link, ok
}

// GetAllLinks returns all active links
func (lr *LinkRegistry) GetAllLinks() []*Link {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	
	links := make([]*Link, 0, len(lr.links))
	for _, link := range lr.links {
		links = append(links, link)
	}
	return links
}

// GetLinkCount returns the number of active links
func (lr *LinkRegistry) GetLinkCount() int {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return len(lr.links)
}

// MessageRouter handles routing messages to remote servers
type MessageRouter struct {
	network  *Network
	registry *LinkRegistry
}

// NewMessageRouter creates a new message router
func NewMessageRouter(network *Network, registry *LinkRegistry) *MessageRouter {
	return &MessageRouter{
		network:  network,
		registry: registry,
	}
}

// RouteToUser routes a message to a specific user by UID
// Returns error if user not found or unable to send
func (mr *MessageRouter) RouteToUser(sourceUID, targetUID string, msg *Message) error {
	// Check if target is local (should be handled by caller, but verify)
	if target, ok := mr.network.GetUserByUID(targetUID); ok {
		// Target is a remote user
		targetServer := target.Server
		if targetServer == nil {
			return fmt.Errorf("target user %s has no server", targetUID)
		}
		
		// Get link to target server
		link, ok := mr.registry.GetLink(targetServer.SID)
		if !ok {
			return fmt.Errorf("no link to server %s for user %s", targetServer.SID, targetUID)
		}
		
		// Send message to remote server
		return link.WriteMessage(msg)
	}
	
	return fmt.Errorf("user %s not found", targetUID)
}

// RouteToServer routes a message to a specific server by SID
func (mr *MessageRouter) RouteToServer(sid string, msg *Message) error {
	link, ok := mr.registry.GetLink(sid)
	if !ok {
		return fmt.Errorf("no link to server %s", sid)
	}
	
	return link.WriteMessage(msg)
}

// BroadcastToServers sends a message to all linked servers
// except the one specified by exceptSID (to prevent loops)
func (mr *MessageRouter) BroadcastToServers(msg *Message, exceptSID string) error {
	links := mr.registry.GetAllLinks()
	
	var errs []error
	for _, link := range links {
		// Skip the server we're excluding (typically the source)
		if link.GetServer() != nil && link.GetServer().SID == exceptSID {
			continue
		}
		
		if err := link.WriteMessage(msg); err != nil {
			errs = append(errs, fmt.Errorf("failed to send to %s: %v", 
				link.GetServer().Name, err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("broadcast errors: %v", errs)
	}
	
	return nil
}

// RouteToChannelServers sends a message to all servers that have users in a channel
// except the server specified by exceptSID
func (mr *MessageRouter) RouteToChannelServers(channelName string, msg *Message, exceptSID string) error {
	// Get the channel from network
	channel, ok := mr.network.GetChannel(channelName)
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}
	
	// Get unique server IDs from channel members
	serverSIDs := make(map[string]bool)
	for uid := range channel.Members {
		if user, ok := mr.network.GetUserByUID(uid); ok {
			if user.Server != nil {
				serverSIDs[user.Server.SID] = true
			}
		}
	}
	
	// Send to each server (except source)
	var errs []error
	for sid := range serverSIDs {
		if sid == exceptSID {
			continue
		}
		
		link, ok := mr.registry.GetLink(sid)
		if !ok {
			continue // Server may be local or link may be down
		}
		
		if err := link.WriteMessage(msg); err != nil {
			errs = append(errs, fmt.Errorf("failed to send to %s: %v", sid, err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("channel routing errors: %v", errs)
	}
	
	return nil
}

// GetServerForUID returns the server ID that hosts a given UID
func (mr *MessageRouter) GetServerForUID(uid string) (string, error) {
	user, ok := mr.network.GetUserByUID(uid)
	if !ok {
		return "", fmt.Errorf("user %s not found", uid)
	}
	
	if user.Server == nil {
		return "", fmt.Errorf("user %s has no server", uid)
	}
	
	return user.Server.SID, nil
}

// IsUserLocal checks if a user is on the local server
func (mr *MessageRouter) IsUserLocal(uid string) bool {
	// Extract SID from UID (first 3 characters)
	if len(uid) < 3 {
		return false
	}
	
	uidSID := uid[:3]
	return uidSID == mr.network.LocalSID
}
