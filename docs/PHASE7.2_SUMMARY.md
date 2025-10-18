# Phase 7.2 Server Handshake Protocol - Complete ✅

## Summary
Successfully implemented the complete TS6-style server-to-server handshake protocol, allowing IRC servers to authenticate and establish links.

## What Was Built

### 1. Protocol Messages (`internal/linking/protocol.go` - 408 lines)

#### Message Structure
```go
type Message struct {
    Source  string   // Source SID or UID
    Command string   // Command name
    Params  []string // Parameters
}
```

#### Protocol Commands Implemented

**PASS** - Authentication
```
Format: PASS <password> TS <version> <SID>
Example: PASS linkpass123 TS 6 0AA
```
- Authenticates with password
- Specifies TS protocol version
- Provides Server ID (SID)

**CAPAB** - Capability Negotiation
```
Format: CAPAB :<capabilities>
Example: CAPAB :QS EX CHW IE KLN UNKLN ENCAP
```
- Negotiates server capabilities
- Default capabilities: QS, EX, CHW, IE, KLN, UNKLN, ENCAP, SERVICES, EUID, EOPMOD, MLOCK

**SERVER** - Server Registration
```
Format: SERVER <name> <hopcount> :<description>
Example: SERVER hub.test 1 :Test IRC Hub Server
```
- Registers server name
- Provides description

**SVINFO** - Version Information
```
Format: SVINFO <TS_version> <min_TS_version> <current_time>
Example: SVINFO 6 6 1729268400
```
- Exchange TS protocol versions
- Synchronize server time
- Warn if time delta > 60 seconds

**UID** - User Introduction (for Phase 7.3)
```
Format: :<SID> UID <nick> <hop> <ts> <modes> <user> <host> <ip> <uid> :<realname>
Example: :0AA UID Alice 1 1234567890 +i alice host.com 1.2.3.4 0AAAAAAAA :Alice User
```

**SJOIN** - Channel Synchronization (for Phase 7.3)
```
Format: :<SID> SJOIN <ts> <channel> <modes> :<members>
Example: :0AA SJOIN 1234567890 #test +nt :@0AAAAAAAA +0AAAAAAAB 0AAAAAAAC
```

**SQUIT** - Server Disconnect
```
Format: :<source> SQUIT <server> :<reason>
Example: :0AA SQUIT 1BB :Connection lost
```

**PING/PONG** - Keepalive
```
Format: :<source> PING <target>
Format: :<source> PONG <target>
```

**ERROR** - Error Messages
```
Format: ERROR :<reason>
Example: ERROR :Invalid password
```

#### Build/Parse Functions
```go
// Building messages
BuildPASS(password, sid) *Message
BuildCAPAB(capabilities) *Message
BuildSERVER(name, hopcount, description) *Message
BuildSVINFO() *Message
BuildUID(...) *Message
BuildSJOIN(...) *Message
BuildPING(source, target) *Message
BuildPONG(source, target) *Message
BuildSQUIT(source, server, reason) *Message
BuildERROR(reason) *Message

// Parsing messages
ParseMessage(line) (*Message, error)
ParsePASS(msg) (password, sid, version, error)
ParseCAPAB(msg) ([]string, error)
ParseSERVER(msg) (name, hopcount, description, error)
ParseSVINFO(msg) (tsVersion, minVersion, serverTime, error)
ParseUID(msg) (*RemoteUser, error)
ParseSJOIN(msg) (channel, ts, modes, members, error)
ParseSQUIT(msg) (server, reason, error)
```

### 2. Handshake Implementation (`internal/linking/handshake.go` - 364 lines)

#### Link Structure
```go
type Link struct {
    conn           net.Conn
    server         *Server
    state          LinkState
    reader         *bufio.Reader
    writer         *bufio.Writer
    receivedPass   bool
    receivedCapab  bool
    receivedServer bool
    receivedSVINFO bool
    remoteSID      string
    remoteName     string
    capabilities   []string
    closed         chan struct{}
}
```

#### Link States
```go
const (
    LinkStateConnected   // Initial connection
    LinkStatePassRecv    // Received PASS
    LinkStateCapabRecv   // Received CAPAB
    LinkStateServerRecv  // Received SERVER
    LinkStateRegistered  // Fully registered
)
```

#### HandshakeServer (Incoming Connection)
Receives connection from remote server:
1. **Read PASS** from remote
   - Validate password
   - Check TS version compatibility
   - Verify SID doesn't conflict
2. **Read CAPAB** from remote
   - Store capabilities
3. **Read SERVER** from remote
   - Get server name and description
   - Create Server object
4. **Send our handshake**
   - Send PASS, CAPAB, SERVER, SVINFO
5. **Read SVINFO** from remote
   - Verify TS version compatibility
   - Check time delta
6. **Complete** - Set state to Registered

#### HandshakeClient (Outgoing Connection)
Initiates connection to remote server:
1. **Send our handshake**
   - Send PASS, CAPAB, SERVER, SVINFO
2. **Read PASS** from remote
   - Verify expected SID
   - Check TS version
3. **Read CAPAB** from remote
   - Store capabilities
4. **Read SERVER** from remote
   - Verify expected server name
   - Create Server object
5. **Read SVINFO** from remote
   - Verify TS version compatibility
6. **Complete** - Set state to Registered

#### I/O Functions
```go
link.ReadMessage() (*Message, error)   // Read protocol message
link.WriteMessage(*Message) error      // Send protocol message
link.Close() error                     // Close connection
link.IsClosed() bool                   // Check if closed
link.IsRegistered() bool               // Check if handshake complete
link.GetServer() *Server               // Get registered server
```

### 3. Server Integration

#### Updated `internal/server/linking.go`

**handleLinkConnection()** - Server side
```go
// 1. Create Link from net.Conn
link := linking.NewLink(conn)

// 2. Perform handshake
err := link.HandshakeServer(s.network, s.config.LinkPassword)

// 3. Get registered server
server := link.GetServer()

// 4. Add to network
s.network.AddServer(server)
```

**ConnectToServer()** - Client side
```go
// 1. Connect to remote
conn, err := net.Dial("tcp", addr)

// 2. Create Link
link := linking.NewLink(conn)

// 3. Perform handshake
err = link.HandshakeClient(s.network, password, remoteSID, remoteName)

// 4. Get registered server
server := link.GetServer()

// 5. Mark as hub if configured
server.IsHub = linkCfg.IsHub

// 6. Add to network
s.network.AddServer(server)
```

### 4. Configuration Files

#### Hub Server (`config/config-hub.yaml`)
```yaml
server:
  name: "hub.test"
  port: 6667

linking:
  enabled: true
  port: 7777
  server_id: "0AA"
  description: "Test IRC Hub Server"
  password: "linkpass123"
  links: []  # No auto-connect, waits for leaf
```

#### Leaf Server (`config/config-leaf.yaml`)
```yaml
server:
  name: "leaf.test"
  port: 6668

linking:
  enabled: true
  port: 7778
  server_id: "1BB"
  description: "Test IRC Leaf Server"
  password: "linkpass123"
  links:
    - name: "hub.test"
      sid: "0AA"
      host: "127.0.0.1"
      port: 7777
      password: "linkpass123"
      auto_connect: true
      is_hub: true
```

### 5. Testing

#### Unit Tests (`internal/linking/protocol_test.go` - 382 lines)
**22 tests, all passing:**
- TestParseMessage
- TestMessageString
- TestBuildParsePASS
- TestBuildParseCAPAB
- TestBuildParseSERVER
- TestBuildParseSVINFO
- TestBuildParseUID
- TestBuildParseSJOIN
- TestBuildParseSQUIT
- TestParsePASSInvalid (4 subtests)

```bash
$ cd internal/linking && go test -v
=== RUN   TestParseMessage
--- PASS: TestParseMessage (0.00s)
...
PASS
ok      github.com/supamanluva/ircd/internal/linking    0.003s
```

#### Integration Test (`tests/test_phase7.2_handshake.sh` - 106 lines)
Automated end-to-end test:
1. Build IRC server
2. Start hub server (SID: 0AA, port 7777)
3. Start leaf server (SID: 1BB, port 7778)
4. Leaf auto-connects to hub
5. Verify handshake completion
6. Check server registration in network

**Test Output:**
```
=== Phase 7.2 Server Linking Test ===
✓ Build successful
✓ Hub server started
✓ Leaf server started
✓ Hub received link from leaf
✓ Leaf connected to hub
✓ Hub completed handshake
✓ Leaf completed handshake
✓ Servers linked together
Phase 7.2 test completed successfully!
```

### 6. Handshake Flow Diagram

```
Leaf Server (1BB)              Hub Server (0AA)
    |                              |
    |--- TCP Connect ------------->|
    |                              |
    |--- PASS linkpass TS 6 1BB -->|  Authenticate
    |--- CAPAB :QS EX CHW... ----->|  Capabilities
    |--- SERVER leaf.test 1... --->|  Register name
    |--- SVINFO 6 6 <time> ------->|  Version info
    |                              |
    |<-- PASS linkpass TS 6 0AA ---|  Authenticate
    |<-- CAPAB :QS EX CHW... ------|  Capabilities
    |<-- SERVER hub.test 1... -----|  Register name
    |<-- SVINFO 6 6 <time> --------|  Version info
    |                              |
    [Registered]                [Registered]
    |                              |
    network.Servers["0AA"] = hub   network.Servers["1BB"] = leaf
    |                              |
```

## Validation and Security

### Password Authentication
- Both sides must provide correct password
- Mismatch results in ERROR and connection close

### SID Validation
```go
// Format: [0-9][A-Z0-9][A-Z0-9]
ValidateSID("0AA")  // true
ValidateSID("AAA")  // false (first char must be digit)
```

### SID Uniqueness
- Check SID != local SID
- Check SID not already linked
- Reject connection on conflict

### TS Version Compatibility
```go
// Must support TS6
if version < MinTSVersion {
    return error
}
if tsVersion < MinTSVersion || minVersion > TS6Version {
    return error
}
```

### Server Name Verification (Client Side)
```go
// Verify we connected to expected server
if name != remoteName {
    return fmt.Errorf("server name mismatch")
}
```

### Time Delta Warning
```go
timeDelta := time.Now().Unix() - serverTime
if timeDelta < -60 || timeDelta > 60 {
    log.Warning("Server time delta: %d seconds", timeDelta)
}
```

## Current Capabilities

Default negotiated capabilities:
- **QS** - Quit Storm (batch QUIT messages)
- **EX** - Exceptions (ban exceptions)
- **CHW** - Channel WHO
- **IE** - Invite Exceptions
- **KLN** - K-line (network bans)
- **UNKLN** - Remove K-line
- **ENCAP** - Encapsulated commands
- **SERVICES** - Services support
- **EUID** - Extended UID
- **EOPMOD** - Op moderation
- **MLOCK** - Mode lock

## Log Output Example

### Hub Server Log
```
[2025-10-18 12:37:22] INFO: Starting IRC Server version=0.1.0
[2025-10-18 12:37:22] INFO: Server linking enabled with SID: 0AA
[2025-10-18 12:37:22] INFO: Starting IRC server address=0.0.0.0:6667
[2025-10-18 12:37:22] INFO: Server listening address=0.0.0.0:6667
[2025-10-18 12:37:22] INFO: Server link listener started on address=0.0.0.0:7777
[2025-10-18 12:37:24] INFO: Incoming link connection from address=127.0.0.1:58060
[2025-10-18 12:37:24] INFO: Link connection handler started for address=127.0.0.1:58060
[2025-10-18 12:37:24] INFO: Server link established name=leaf.test sid=1BB address=127.0.0.1:58060
[2025-10-18 12:37:24] INFO: Server registered in network name=leaf.test total_servers=1
```

### Leaf Server Log
```
[2025-10-18 12:37:24] INFO: Starting IRC Server version=0.1.0
[2025-10-18 12:37:24] INFO: Server linking enabled with SID: 1BB
[2025-10-18 12:37:24] INFO: Starting IRC server address=0.0.0.0:6668
[2025-10-18 12:37:24] INFO: Server listening address=0.0.0.0:6668
[2025-10-18 12:37:24] INFO: Server link listener started on address=0.0.0.0:7778
[2025-10-18 12:37:24] INFO: Auto-connecting to name=hub.test
[2025-10-18 12:37:24] INFO: Attempting to connect to server name=hub.test sid=0AA address=127.0.0.1:7777
[2025-10-18 12:37:24] INFO: Connected, starting handshake address=127.0.0.1:7777
[2025-10-18 12:37:24] INFO: Server link established name=hub.test sid=0AA
[2025-10-18 12:37:24] INFO: Server registered in network name=hub.test total_servers=1
```

## Files Created/Modified

### New Files (1362 lines total)
1. `internal/linking/protocol.go` (408 lines)
   - Message parsing and formatting
   - All protocol command builders/parsers
   
2. `internal/linking/protocol_test.go` (382 lines)
   - 22 comprehensive unit tests
   
3. `internal/linking/handshake.go` (364 lines)
   - Link struct and state machine
   - HandshakeServer and HandshakeClient
   
4. `config/config-hub.yaml` (48 lines)
   - Hub server configuration
   
5. `config/config-leaf.yaml` (54 lines)
   - Leaf server configuration with auto-connect
   
6. `tests/test_phase7.2_handshake.sh` (106 lines)
   - Automated integration test script

### Modified Files
1. `internal/server/linking.go`
   - Updated handleLinkConnection() to use handshake
   - Updated ConnectToServer() to use handshake
   - Fixed logger format (key-value pairs)

## Next Steps: Phase 7.3 Burst Mode

Once the handshake completes, servers need to synchronize state:

### Tasks
1. **Send Burst**
   - Send UID messages for all local users
   - Send SJOIN messages for all channels
   - Send burst completion marker

2. **Receive Burst**
   - Handle incoming UID messages
   - Handle incoming SJOIN messages
   - Detect burst completion
   - Build global user/channel maps

3. **Burst Handler**
   - Queue messages during burst
   - Process burst atomically
   - Update Network state
   - Log burst statistics

### Protocol Already Ready
- UID command: Implemented in protocol.go
- SJOIN command: Implemented in protocol.go
- BuildUID/ParseUID: Ready
- BuildSJOIN/ParseSJOIN: Ready

Just need to:
- Iterate over local clients → send UID
- Iterate over channels → send SJOIN
- Handle received UID → add to Network
- Handle received SJOIN → merge channels

## Testing Phase 7.2

### Manual Test
```bash
# Terminal 1 - Hub server
./ircd -config config/config-hub.yaml

# Terminal 2 - Leaf server  
./ircd -config config/config-leaf.yaml

# Check logs
tail -f logs/hub.log
tail -f logs/leaf.log
```

### Automated Test
```bash
./tests/test_phase7.2_handshake.sh
```

### Expected Results
- ✓ Both servers start successfully
- ✓ Leaf auto-connects to hub
- ✓ Handshake completes (PASS, CAPAB, SERVER, SVINFO)
- ✓ Servers register in each other's network
- ✓ total_servers=1 on both sides

## Commit

**Commit Hash:** `2457577`

**Commit Message:** "Phase 7.2: Implement server handshake protocol"

**Stats:**
- 7 files changed
- 1,513 insertions
- 29 deletions
- 6 new files created

**Repository:** https://github.com/supamanluva/ircd

---

*Phase 7.2 completed successfully! Servers can now authenticate and link. Ready to implement burst mode in Phase 7.3.*
