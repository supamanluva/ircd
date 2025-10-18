# Server Linking Quick Reference

## SID (Server ID) Format
```
[0-9][A-Z0-9][A-Z0-9]

Examples:
  0AA  - Hub server 1
  1BB  - Hub server 2
  2CC  - Leaf server 1
  9ZZ  - Maximum SID value
```

## UID (User ID) Format
```
[SID][AAAAAA]

Examples:
  0AAAAAAAA  - First user on server 0AA
  0AAAAAAAB  - Second user on server 0AA
  1BB000001  - User on server 1BB
  
Format: SID (3 chars) + Base36 counter (6 chars)
```

## Network Port Layout
```
6667  - IRC client connections (TCP)
7000  - IRC client connections (TLS)
8080  - WebSocket connections
7777  - Server linking (default)
```

## Configuration Structure

```yaml
linking:
  enabled: true              # Enable server linking
  host: "0.0.0.0"           # Bind address
  port: 7777                # Link port
  server_id: "0AA"          # This server's SID
  description: "IRC Hub"     # Server description
  password: "linkpass"       # Password for incoming links
  
  links:
    - name: "hub.example.net"
      sid: "1BB"
      host: "10.0.0.1"
      port: 7777
      password: "linkpass"
      auto_connect: true    # Connect on startup
      is_hub: true          # Can link other servers
```

## Data Structures

### Network
```go
type Network struct {
    LocalSID   string                    // Our SID
    LocalName  string                    // Our server name
    Servers    map[string]*Server        // SID -> Server
    Users      map[string]*RemoteUser    // UID -> User
    Channels   map[string]*RemoteChannel // Name -> Channel
    NickToUID  map[string]string         // Nick -> UID
    UIDCounter uint32                    // UID counter
}
```

### Server
```go
type Server struct {
    SID         string           // Server ID
    Name        string           // Server name
    Conn        net.Conn        // Connection
    IsHub       bool            // Hub server?
    Uplink      *Server         // Parent server
    Downlinks   []*Server       // Child servers
    Distance    int             // Hops from local
    Users       map[string]*RemoteUser
    Channels    map[string]*RemoteChannel
}
```

### RemoteUser
```go
type RemoteUser struct {
    UID        string              // Unique ID
    Nick       string              // Nickname
    User       string              // Username
    Host       string              // Hostname
    Server     *Server            // Which server
    Modes      string              // User modes
    Channels   map[string]bool     // Joined channels
    Timestamp  int64               // Nick timestamp
}
```

### RemoteChannel
```go
type RemoteChannel struct {
    Name       string              // Channel name
    TS         int64              // Timestamp
    Modes      string             // Channel modes
    Topic      string             // Topic
    Members    map[string]string  // UID -> modes
}
```

## API Functions

### Network Management
```go
// Create network
net := linking.NewNetwork("0AA", "hub.example.net")

// Generate UIDs
uid := net.GenerateUID()  // "0AAAAAAAA"

// Servers
net.AddServer(srv)
net.RemoveServer("1BB")
srv, ok := net.GetServer("1BB")
count := net.GetServerCount()

// Users
net.AddUser(user)
net.RemoveUser("0AAAAAAAA")
net.UpdateNick("0AAAAAAAA", "NewNick", timestamp)
user, ok := net.GetUserByUID("0AAAAAAAA")
user, ok := net.GetUserByNick("Alice")
count := net.GetUserCount()

// Channels
net.AddChannel(ch)  // Auto-merges with TS conflict resolution
ch, ok := net.GetChannel("#test")
count := net.GetChannelCount()
```

### SID/UID Operations
```go
// Generate random SID
sid, err := linking.GenerateSID()  // "5K9"

// Generate specific SID (for testing)
sid := linking.GenerateSpecificSID(0, 0, 0)  // "000"

// Validate
valid := linking.ValidateSID("0AA")   // true
valid := linking.ValidateUID("0AAAAAAAA")  // true
```

### Server Linking
```go
// Start link listener (port 7777)
err := srv.StartLinkListener()

// Connect to remote server
err := srv.ConnectToServer(linkCfg)

// Auto-connect to configured servers
srv.AutoConnect()
```

## Conflict Resolution

### Nick Collisions
**Rule:** Lower timestamp wins

```
Server 1: NICK Alice TS=1000
Server 2: NICK Alice TS=2000

Result: Server 1's user keeps "Alice"
        Server 2's user gets force-renamed
```

### Channel TS Conflicts
**Rules:**
- Older TS wins (overwrites modes)
- Same TS: merge members

```
Server 1: SJOIN #test TS=500 modes=+s
Server 2: SJOIN #test TS=1000 modes=+nt

Result: TS=500, modes=+s (older TS wins)
```

```
Server 1: SJOIN #test TS=1000 modes=+nt users=@Alice
Server 2: SJOIN #test TS=1000 modes=+nt users=@Bob

Result: TS=1000, modes=+nt, users=@Alice,@Bob (merge)
```

## Network Topology

### Hub-Leaf Model
```
       [Hub 0AA]
      /    |    \
     /     |     \
[Leaf]  [Hub]  [Leaf]
 1BB     2CC     3DD
         /\
        /  \
    [Leaf] [Leaf]
     4EE    5FF
```

**Rules:**
- Hubs can link to multiple servers
- Leaves connect to exactly one hub
- No loops (tree structure)

## State Synchronization

### Server Connect Sequence
```
1. TCP connection established
2. PASS authentication
3. CAPAB capability negotiation
4. SERVER registration
5. SVINFO version exchange
6. BURST state sync
7. Normal operation
```

### Server Disconnect
```
1. Connection lost
2. SQUIT propagated
3. Remove server from Network
4. Remove all users from that server
5. Update channel member lists
6. Remove empty channels
```

## Testing

### Run Tests
```bash
cd internal/linking
go test -v
```

### Expected Output
```
=== RUN   TestGenerateSID
--- PASS: TestGenerateSID (0.00s)
=== RUN   TestValidateSID
--- PASS: TestValidateSID (0.00s)
...
PASS
ok      github.com/supamanluva/ircd/internal/linking    0.003s
```

## Development Status

✅ **Phase 7.1: Foundation** (COMPLETE)
- Core data structures
- SID/UID generation
- Network state management
- Configuration support
- Server integration
- Test suite

🔄 **Phase 7.2: Handshake** (NEXT)
- PASS command
- CAPAB command
- SERVER command
- SVINFO command

⏳ **Phase 7.3: Burst Mode**
- UID command
- SJOIN command
- State synchronization

⏳ **Phase 7.4: Routing**
- Message propagation
- PRIVMSG/NOTICE routing
- JOIN/PART/QUIT/NICK propagation
- SQUIT handling

## Common Issues

### SID Already In Use
```
Error: server with SID 0AA already exists
```
**Solution:** Each server needs a unique SID in config

### Invalid SID Format
```
Error: invalid SID: AA0
```
**Solution:** First char must be digit (0-9)

### Nick Collision
```
Error: nick collision: Alice already exists with older timestamp
```
**Solution:** This is expected - timestamp resolution working correctly

### Port Already In Use
```
Error: failed to start link listener: address already in use
```
**Solution:** Change linking.port in config or stop conflicting service

## Resources

- **Design Document:** `docs/PHASE7_DESIGN.md`
- **Phase 7.1 Summary:** `docs/PHASE7.1_SUMMARY.md`
- **Source Code:** `internal/linking/`
- **Tests:** `internal/linking/network_test.go`
- **Configuration:** `config/config.yaml`

## Example: Two-Server Network

### Server A Configuration (config-serverA.yaml)
```yaml
linking:
  enabled: true
  server_id: "0AA"
  host: "0.0.0.0"
  port: 7777
  description: "Hub Server"
  password: "linkpass123"
  links:
    - name: "serverB.test"
      sid: "1BB"
      host: "localhost"
      port: 7778
      password: "linkpass123"
      auto_connect: true
      is_hub: false
```

### Server B Configuration (config-serverB.yaml)
```yaml
linking:
  enabled: true
  server_id: "1BB"
  host: "0.0.0.0"
  port: 7778
  description: "Leaf Server"
  password: "linkpass123"
  links: []  # Leaf connects to hub only
```

### Start Servers
```bash
# Terminal 1
./ircd -config config-serverA.yaml

# Terminal 2
./ircd -config config-serverB.yaml
```

---

**Last Updated:** Phase 7.1 Complete
**Commit:** 454b015
