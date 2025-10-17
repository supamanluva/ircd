# Phase 6: Advanced Features

## Overview
Phase 6 extends the IRC server with advanced IRC commands, WebSocket support for web clients, and enhanced functionality for a more complete IRC experience.

## Goals
1. Implement advanced IRC commands (WHO, WHOIS, INVITE, LIST)
2. Add WebSocket support for browser-based clients
3. Implement channel keys and voice mode
4. Add OPER command for server operators
5. Improve error handling and edge cases
6. Optional: Server federation and REST API

## Timeline
**Estimated Duration:** 3-4 days

---

## 📋 Feature List

### 1. Advanced IRC Commands (Day 1)

#### WHO Command
**Purpose:** List users matching a pattern or in a channel

**Syntax:**
```
WHO <mask>
WHO #channel
```

**RFC 1459 Replies:**
- `352 RPL_WHOREPLY`: `<channel> <user> <host> <server> <nick> <flags> :<hopcount> <realname>`
- `315 RPL_ENDOFWHO`: `<name> :End of WHO list`

**Implementation:**
- Search by channel name
- Search by nickname pattern (wildcards)
- Show flags: H=here, G=gone, *=ircop, @=chanop, +=voice
- Return user info: nick, user, host, server, realname

#### WHOIS Command
**Purpose:** Get detailed information about a user

**Syntax:**
```
WHOIS <nickname>
WHOIS <nickname> <nickname>
```

**RFC 1459 Replies:**
- `311 RPL_WHOISUSER`: `<nick> <user> <host> * :<realname>`
- `312 RPL_WHOISSERVER`: `<nick> <server> :<server info>`
- `317 RPL_WHOISIDLE`: `<nick> <seconds> :seconds idle`
- `318 RPL_ENDOFWHOIS`: `<nick> :End of WHOIS list`
- `319 RPL_WHOISCHANNELS`: `<nick> :<channels>`
- `401 ERR_NOSUCHNICK`: `<nickname> :No such nick/channel`

**Implementation:**
- Show nickname, username, hostname, realname
- Show channels user is in
- Show idle time
- Show if user is an operator
- Handle non-existent users

#### INVITE Command
**Purpose:** Invite a user to an invite-only channel

**Syntax:**
```
INVITE <nickname> <channel>
```

**RFC 1459 Replies:**
- `341 RPL_INVITING`: `<channel> <nick>`
- `401 ERR_NOSUCHNICK`: `<nickname> :No such nick/channel`
- `442 ERR_NOTONCHANNEL`: `<channel> :You're not on that channel`
- `443 ERR_USERONCHANNEL`: `<user> <channel> :is already on channel`
- `482 ERR_CHANOPRIVSNEEDED`: `<channel> :You're not channel operator`

**Implementation:**
- Check if inviter is on channel
- Check if inviter is operator
- Check if target exists and is not already on channel
- Send INVITE notification to target
- Add target to channel invite list (if +i mode)

#### LIST Command
**Purpose:** List all channels or channels matching pattern

**Syntax:**
```
LIST
LIST <channel>
LIST <channel>,<channel>
```

**RFC 1459 Replies:**
- `321 RPL_LISTSTART`: `Channel :Users Name`
- `322 RPL_LIST`: `<channel> <# visible> :<topic>`
- `323 RPL_LISTEND`: `:End of LIST`

**Implementation:**
- List all channels (unless +s secret mode)
- Show user count
- Show topic
- Support filtering by channel name

### 2. Channel Enhancements (Day 1-2)

#### Channel Key (+k mode)
**Purpose:** Password-protect channels

**Implementation:**
- `MODE #channel +k <key>` - Set key
- `MODE #channel -k` - Remove key
- `JOIN #channel <key>` - Join with key
- `475 ERR_BADCHANNELKEY` - Wrong or missing key

**Channel Structure Update:**
```go
type Channel struct {
    // ... existing fields ...
    key string  // Channel password
}
```

#### Voice Mode (+v)
**Purpose:** Allow users to speak in moderated channels

**Implementation:**
- `MODE #channel +v <nick>` - Give voice
- `MODE #channel -v <nick>` - Remove voice
- Users with +v can speak in +m (moderated) channels
- Show + prefix in NAMES list

**Client/Channel Updates:**
- Track voiced users in channel
- Check voice status before allowing messages in +m channels

### 3. Server Operators (Day 2)

#### OPER Command
**Purpose:** Gain server operator privileges

**Syntax:**
```
OPER <username> <password>
```

**RFC 1459 Replies:**
- `381 RPL_YOUREOPER`: `:You are now an IRC operator`
- `461 ERR_NEEDMOREPARAMS`: `<command> :Not enough parameters`
- `464 ERR_PASSWDMISMATCH`: `:Password incorrect`

**Implementation:**
- Load operator credentials from config
- Hash passwords (bcrypt)
- Set +o user mode on success
- Log operator authentication attempts

**Config Addition:**
```yaml
operators:
  - username: admin
    password: $2a$10$... # bcrypt hash
  - username: moderator
    password: $2a$10$...
```

#### Server Operator Commands
- `KILL <nick> <reason>` - Disconnect user
- `REHASH` - Reload configuration
- `DIE` - Shutdown server (with confirmation)
- `RESTART` - Restart server (with confirmation)

### 4. WebSocket Support (Day 2-3)

#### WebSocket Bridge
**Purpose:** Allow browser-based IRC clients to connect

**Implementation:**
```go
// internal/websocket/handler.go
type WSHandler struct {
    upgrader websocket.Upgrader
    server   *server.Server
}

func (h *WSHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
    // Upgrade HTTP to WebSocket
    conn, err := h.upgrader.Upgrade(w, r, nil)
    
    // Create WebSocket wrapper that implements net.Conn
    wsConn := NewWSConn(conn)
    
    // Pass to existing IRC server
    h.server.HandleClient(wsConn)
}
```

**WebSocket Connection Wrapper:**
```go
type WSConn struct {
    conn *websocket.Conn
    reader io.Reader
}

// Implement net.Conn interface
func (w *WSConn) Read(b []byte) (n int, err error)
func (w *WSConn) Write(b []byte) (n int, err error)
func (w *WSConn) Close() error
// ... other net.Conn methods
```

**Server Configuration:**
```yaml
websocket:
  enabled: true
  port: 8080
  path: /irc
  tls:
    enabled: true
    cert: certs/server.crt
    key: certs/server.key
  origin_check: true
  allowed_origins:
    - https://example.com
    - https://chat.example.com
```

**Features:**
- HTTPS endpoint for WebSocket connections
- Origin verification for security
- IP-based rate limiting
- Session management
- Automatic reconnection support

#### Simple Web Client (Optional)
Create basic HTML/JS client in `web/` directory:
```html
<!DOCTYPE html>
<html>
<head>
    <title>IRC Web Client</title>
</head>
<body>
    <div id="chat"></div>
    <input id="input" type="text" />
    <script src="client.js"></script>
</body>
</html>
```

### 5. Additional Commands (Day 3)

#### AWAY Command
**Purpose:** Mark user as away

**Syntax:**
```
AWAY :<message>
AWAY
```

**Replies:**
- `306 RPL_NOWAWAY`: `:You have been marked as being away`
- `305 RPL_UNAWAY`: `:You are no longer marked as being away`
- `301 RPL_AWAY`: `<nick> :<away message>` (when messaging away user)

#### USERHOST Command
**Purpose:** Get hostname information for users

**Syntax:**
```
USERHOST <nickname> [<nickname>...]
```

**Reply:**
- `302 RPL_USERHOST`: `:[<nick>=<flag><user>@<host>]`

#### ISON Command
**Purpose:** Check if users are online

**Syntax:**
```
ISON <nickname> [<nickname>...]
```

**Reply:**
- `303 RPL_ISON`: `:[<nick> [<nick>...]]`

### 6. Enhanced Features (Day 3-4)

#### Channel Limit (+l mode)
- `MODE #channel +l <limit>` - Set user limit
- `471 ERR_CHANNELISFULL` when limit reached

#### Ban Masks with Wildcards
- Support wildcard patterns in ban masks
- `*!*@*.example.com`
- `nick!*@*`
- `*!user@host`

#### Channel Exceptions (+e mode)
- Exception list for bans
- `MODE #channel +e <mask>` - Add exception
- Users matching exception can join despite bans

#### Invite Exceptions (+I mode)
- Exception list for invite-only channels
- `MODE #channel +I <mask>` - Add invite exception
- Users matching pattern can join +i channels

---

## 🏗️ Implementation Plan

### Day 1: Advanced Commands
**Morning:**
- [ ] Implement WHO command
- [ ] Add WHO reply formatting
- [ ] Test WHO with channels and patterns

**Afternoon:**
- [ ] Implement WHOIS command
- [ ] Add all WHOIS replies (user, server, idle, channels)
- [ ] Test WHOIS edge cases

**Evening:**
- [ ] Implement INVITE command
- [ ] Handle invite-only channels
- [ ] Test invite permissions

### Day 2: Channel Features & Operators
**Morning:**
- [ ] Implement channel keys (+k mode)
- [ ] Add voice mode (+v)
- [ ] Update NAMES to show voice prefix

**Afternoon:**
- [ ] Implement OPER command
- [ ] Add operator authentication
- [ ] Implement KILL command

**Evening:**
- [ ] Add REHASH command
- [ ] Test operator privileges
- [ ] Update documentation

### Day 3: WebSocket Support
**Morning:**
- [ ] Create websocket package
- [ ] Implement WebSocket handler
- [ ] Create net.Conn wrapper for WebSocket

**Afternoon:**
- [ ] Add WebSocket configuration
- [ ] Implement origin checking
- [ ] Test WebSocket connections

**Evening:**
- [ ] Create simple web client (optional)
- [ ] Test browser connections
- [ ] Add WebSocket documentation

### Day 4: Additional Commands & Polish
**Morning:**
- [ ] Implement LIST command
- [ ] Implement AWAY command
- [ ] Implement USERHOST command

**Afternoon:**
- [ ] Implement ISON command
- [ ] Add channel limit (+l mode)
- [ ] Add ban/exception wildcards

**Evening:**
- [ ] Write comprehensive tests
- [ ] Update documentation
- [ ] Performance testing

---

## 📊 Success Criteria

### Functionality
- [ ] All advanced commands implemented and tested
- [ ] WebSocket connections work from browser
- [ ] Channel keys and voice modes functional
- [ ] Server operators can authenticate and use privileged commands
- [ ] All new features have unit tests

### Performance
- [ ] WebSocket adds <10ms latency vs direct TCP
- [ ] Server handles 1000+ concurrent connections
- [ ] Commands respond within 50ms

### Documentation
- [ ] All new commands documented
- [ ] WebSocket setup guide created
- [ ] Operator manual written
- [ ] API documentation updated

### Testing
- [ ] Unit tests for all new commands
- [ ] Integration tests with web client
- [ ] Load testing with WebSocket connections
- [ ] Security testing (origin validation, authentication)

---

## 🔐 Security Considerations

### WebSocket Security
- HTTPS/WSS only (no unencrypted WebSocket)
- Origin header validation
- CSRF token support (optional)
- Rate limiting per IP
- Connection limits per IP

### Operator Security
- Bcrypt password hashing
- Failed login attempt limiting
- Operator action logging
- Two-factor authentication (stretch goal)

### Channel Security
- Key length validation (max 23 chars per RFC)
- Ban mask validation
- Invite list size limits
- Exception list size limits

---

## 🧪 Testing Strategy

### Unit Tests
```go
// WHO command tests
func TestHandleWho(t *testing.T)
func TestWhoChannelMembers(t *testing.T)
func TestWhoPatternMatching(t *testing.T)

// WHOIS command tests
func TestHandleWhois(t *testing.T)
func TestWhoisMultipleUsers(t *testing.T)
func TestWhoisNonExistentUser(t *testing.T)

// WebSocket tests
func TestWebSocketUpgrade(t *testing.T)
func TestWebSocketOriginValidation(t *testing.T)
func TestWebSocketIRCCommands(t *testing.T)
```

### Integration Tests
```bash
# Test with real IRC client over WebSocket
node websocket_client_test.js

# Test operator authentication
./test_oper.sh

# Test channel keys
./test_channel_keys.sh
```

---

## 📚 Documentation Deliverables

1. **COMMANDS.md** - Complete command reference
2. **WEBSOCKET.md** - WebSocket setup and usage guide
3. **OPERATORS.md** - Operator manual and commands
4. **MODES.md** - Complete mode reference (user and channel)
5. **API.md** - WebSocket API documentation
6. **PHASE6_ADVANCED_FEATURES.md** - This document

---

## 🚀 Deployment Updates

### Configuration Changes
```yaml
# config.yaml additions
server:
  name: IRCServer
  network: MyNetwork

operators:
  - username: admin
    password: $2a$10$...
  - username: moderator
    password: $2a$10$...

websocket:
  enabled: true
  port: 8080
  path: /irc
  tls:
    enabled: true
    cert: certs/server.crt
    key: certs/server.key
  allowed_origins:
    - https://example.com
```

### New Ports
- **6667** - Standard IRC (TCP)
- **7000** - Secure IRC (TLS)
- **8080** - WebSocket (HTTPS/WSS)

### Firewall Updates
```bash
# UFW
sudo ufw allow 8080/tcp comment 'IRC WebSocket'

# firewalld
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

---

## Next Steps After Phase 6

### Phase 7 (Future)
- Server-to-server federation (linking IRC servers)
- Services integration (NickServ, ChanServ)
- REST API for server statistics
- Admin web dashboard
- Enhanced logging and analytics
- SASL authentication
- Client capabilities negotiation (CAP)

---

**Status:** Ready to Begin  
**Priority:** High  
**Complexity:** Medium-High  
**Dependencies:** Phase 5 complete ✅
