# Additional Commands Implementation (AWAY, USERHOST, ISON)

## Overview
This document covers three utility IRC commands that provide user status and presence information: AWAY, USERHOST, and ISON.

## Commands Implemented

### 1. AWAY Command

**Purpose:** Allows users to mark themselves as away with an optional message.

**Syntax:**
```
AWAY [<message>]
```

**Behavior:**
- **With message:** Sets user as away with the provided message
- **Without message:** Removes away status

**Replies:**
- `RPL_NOWAWAY (306)`: User marked as away
- `RPL_UNAWAY (305)`: User no longer away

**Example:**
```
alice: AWAY :Gone to lunch
Server: 306 alice :You have been marked as being away

alice: AWAY
Server: 305 alice :You are no longer marked as being away
```

### 2. USERHOST Command

**Purpose:** Returns user@host information for specified nicknames, including away and operator status.

**Syntax:**
```
USERHOST <nickname> [<nickname> ...]
```

**Format:** `<nick>[*]=<+|-><user>@<host>`
- `*` = Server operator
- `+` = Not away
- `-` = Away

**Replies:**
- `RPL_USERHOST (302)`: Space-separated list of user information

**Limitations:**
- Maximum 5 nicknames per request (RFC 1459 limit)

**Example:**
```
bob: USERHOST alice charlie
Server: 302 bob :alice=+alice@192.168.1.100 charlie*=-charlie@192.168.1.101
```
(alice is not away, charlie is an operator and away)

### 3. ISON Command

**Purpose:** Checks which of the specified nicknames are currently online.

**Syntax:**
```
ISON <nickname> [<nickname> ...]
```

**Replies:**
- `RPL_ISON (303)`: Space-separated list of online nicknames

**Example:**
```
eve: ISON alice bob charlie dave
Server: 303 eve :alice bob charlie
```
(alice, bob, and charlie are online; dave is not)

## Implementation Details

### Client Structure Updates

**Added field to `internal/client/client.go`:**
```go
type Client struct {
    // ... existing fields ...
    awayMessage string // away message (empty if not away)
}
```

**New methods:**
```go
func (c *Client) SetAway(message string)
func (c *Client) GetAwayMessage() string
func (c *Client) IsAway() bool
```

### Command Handlers (`internal/commands/handler.go`)

#### handleAway()
- Validates registration
- Sets/clears away message
- Sends RPL_NOWAWAY or RPL_UNAWAY
- Logs away status changes

#### handleUserhost()
- Validates parameters
- Processes up to 5 nicknames
- Parses hostmask to extract user@host
- Checks operator and away status
- Formats response with flags
- Sends RPL_USERHOST

#### handleIson()
- Validates parameters
- Checks each nickname for online status
- Builds list of online users
- Sends RPL_ISON (even if empty)

### Integration with Existing Commands

#### WHOIS Enhancement
Shows away message when querying away users:
```
Server: 301 bob alice :Gone to lunch
```

#### PRIVMSG Enhancement
Notifies sender when messaging away users:
```
alice: PRIVMSG bob :Hello
Server: 301 alice bob :Be right back
```
(Note: NOTICE does not trigger away notification per RFC)

#### WHO Enhancement
Shows away status with flags:
- `H` = Here (not away)
- `G` = Gone (away)
- Added voice flag `+` to WHO replies

Example:
```
352 alice #channel bob user host server bob H :0 User
352 alice #channel charlie user host server charlie G* :0 User
```
(bob is here, charlie is gone and is an operator)

## Numeric Reply Codes

All codes already existed in `internal/commands/replies.go`:

```go
RPL_AWAY       = "301" // <nick> :<away message>
RPL_USERHOST   = "302" // :[<reply>{<space><reply>}]
RPL_ISON       = "303" // :[<nick> {<space><nick>}]
RPL_UNAWAY     = "305" // :You are no longer marked as being away
RPL_NOWAWAY    = "306" // :You have been marked as being away
```

## RFC 1459 Compliance

### AWAY (Section 5.1)
✅ Sets/clears away message  
✅ Shows in WHOIS (RPL_AWAY)  
✅ Notifies on PRIVMSG  
✅ Shown in WHO with G flag  

### USERHOST (Section 5.5)
✅ Returns user@host information  
✅ Shows away status (+/-)  
✅ Shows operator status (*)  
✅ Limits to 5 nicknames  

### ISON (Section 5.8)
✅ Returns online nicknames  
✅ Case-insensitive matching  
✅ Always returns RPL_ISON  

## Usage Examples

### Setting Away Status
```
# Set away
alice: AWAY :Out for lunch, back at 2pm
Server: 306 alice :You have been marked as being away

# Check in WHOIS
bob: WHOIS alice
Server: 311 bob alice alice 192.168.1.100 * :User
Server: 301 bob alice :Out for lunch, back at 2pm
Server: 319 bob alice :#general #help
Server: 318 bob alice :End of WHOIS list

# Return from away
alice: AWAY
Server: 305 alice :You are no longer marked as being away
```

### Checking User Status
```
# USERHOST - get detailed info
charlie: USERHOST alice bob charlie
Server: 302 charlie :alice=+alice@192.168.1.100 bob=-bob@192.168.1.101 charlie*=+charlie@192.168.1.102

# ISON - quick online check
dave: ISON alice bob eve frank
Server: 303 dave :alice bob frank
```

### Away Notification in Messages
```
# Alice is away
alice: AWAY :In a meeting

# Bob messages Alice
bob: PRIVMSG alice :Can you help with something?
Server: 301 bob alice :In a meeting
(Message is still delivered to alice)

# NOTICE does not trigger away notification
bob: NOTICE alice :FYI: Server maintenance tonight
(No away notification, per RFC)
```

### WHO with Away Status
```
charlie: WHO #general
Server: 352 charlie #general alice user host server alice H :0 User
Server: 352 charlie #general bob user host server bob G* :0 User
Server: 352 charlie #general charlie user host server charlie H@ :0 User
Server: 315 charlie #general :End of WHO list

# H = Here (not away)
# G = Gone (away)
# * = IRC Operator
# @ = Channel Operator
```

## Testing

Test script: `tests/test_additional_commands.sh`

**Test Cases:**
1. Setting AWAY message (RPL_NOWAWAY)
2. Removing AWAY status (RPL_UNAWAY)
3. AWAY shown in WHOIS
4. PRIVMSG shows away notification
5. USERHOST command with multiple users
6. USERHOST shows away (+/-) and operator (*) flags
7. ISON command shows online users
8. ISON excludes offline users
9. WHO shows G flag for away users
10. WHO shows H flag for present users

Run tests:
```bash
./tests/test_additional_commands.sh
```

## Security & Privacy Considerations

### Away Messages
1. **Public Information:** Away messages are visible to all users
2. **No Length Limit:** Server should limit away message length (current: no limit)
3. **No Sanitization:** Away messages passed as-is (potential for abuse)
4. **Persistence:** Away status cleared on disconnect

### USERHOST
1. **Host Exposure:** Reveals user@host information
2. **Privacy:** Shows away status to anyone
3. **Rate Limiting:** Not implemented (could be abused for user enumeration)

### ISON
1. **Presence Leak:** Reveals online status
2. **No Rate Limit:** Could be used for presence tracking
3. **Bulk Checking:** Allows checking many users at once

**Recommendations:**
- Implement rate limiting for USERHOST and ISON
- Add configurable away message length limit
- Consider privacy modes to hide away status
- Sanitize away messages for special characters

## Performance Considerations

### USERHOST
- **Complexity:** O(n) where n = number of nicknames (max 5)
- **Locking:** Read lock per client lookup
- **String parsing:** Hostmask parsing for each user

### ISON
- **Complexity:** O(n) where n = number of nicknames
- **Locking:** Read lock per client lookup
- **Memory:** Builds list of online users

### AWAY
- **Storage:** Away message stored in memory per user
- **No persistence:** Away status lost on disconnect
- **Notifications:** Additional message on every PRIVMSG to away user

**Optimizations:**
- Away messages not persisted to disk
- No background processing
- Efficient string operations
- Lock-free reads where possible

## Related Files
- `internal/client/client.go` - Client structure with away message
- `internal/commands/handler.go` - Command handlers
- `internal/commands/replies.go` - Numeric reply codes
- `tests/test_additional_commands.sh` - Integration tests

## Future Enhancements
- [ ] Configurable away message length limit
- [ ] Away message sanitization
- [ ] Rate limiting for USERHOST/ISON
- [ ] Privacy mode to hide away status
- [ ] Away time tracking (how long away)
- [ ] Auto-away after idle timeout
- [ ] Persistent away messages
- [ ] Away log (messages received while away)
- [ ] MONITOR command for real-time presence updates
- [ ] WATCH command (alternative to ISON)

## Comparison with Other Commands

### AWAY vs QUIT
- AWAY: Temporary absence, user still connected
- QUIT: Permanent departure, user disconnects

### USERHOST vs WHOIS
- USERHOST: Quick user@host with away/oper flags
- WHOIS: Comprehensive user information

### ISON vs WHO
- ISON: Simple online check, returns nicknames
- WHO: Detailed user list with flags and channels

### MONITOR vs ISON
- ISON: Poll-based, client must query
- MONITOR: Push-based, server notifies of changes (not implemented)
