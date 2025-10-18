# Phase 7: IRC Server Linking - Design Document

## Overview

Server linking enables multiple IRC servers to connect together forming a network where users on different servers can communicate as if they're on the same server. This is how IRC networks like Freenode, EFnet, and others operate.

## Goals

- Connect multiple IRC servers into a network
- Synchronize users, channels, and messages across servers
- Handle network splits gracefully
- Maintain data consistency across the network
- Provide redundancy and load distribution

## Architecture

### Network Topology

We'll implement a **hybrid hub-leaf** model:

```
         [Hub Server A]
         /     |      \
        /      |       \
   [Leaf 1] [Leaf 2] [Hub B]
                        |
                     [Leaf 3]
```

**Server Types:**
- **Hub**: Can connect to multiple servers (hubs or leaves)
- **Leaf**: Can only connect to one hub (cannot link other servers)

**Benefits:**
- Prevents routing loops
- Easier to manage than full mesh
- Scales well
- Clear hierarchy

### Protocol Choice: TS6

We'll implement a **simplified TS6-like protocol** (used by Charybdis, ircd-ratbox):

**Why TS6?**
- ✅ Well-documented and proven
- ✅ Handles nick/channel collisions
- ✅ Efficient burst mode
- ✅ Timestamp-based conflict resolution
- ✅ UID (Unique ID) system

**What we'll implement:**
- Server handshake (PASS, CAPAB, SERVER)
- User propagation (UID, NICK, QUIT)
- Channel propagation (SJOIN, MODE, TOPIC, KICK)
- Message routing (PRIVMSG, NOTICE)
- Server management (SQUIT, CONNECT, LINKS)

## Data Structures

### Server ID (SID)

Each server gets a unique 3-character alphanumeric ID:

```
Format: [0-9][A-Z0-9][A-Z0-9]
Examples: 0AA, 1BC, 9ZZ
```

### User ID (UID)

Each user gets a globally unique ID:

```
Format: [SID][AAAAAA]
Examples: 0AAAAAAAA, 1BCAAAAAB
Components:
  - First 3 chars: Server ID
  - Last 6 chars: User counter on that server
```

### Server Structure

```go
type LinkedServer struct {
    SID         string           // Server ID (3 chars)
    Name        string           // Server name
    Description string           // Server description
    Conn        net.Conn        // Connection
    IsHub       bool            // Can link other servers?
    Uplink      *LinkedServer   // Parent server
    Downlinks   []*LinkedServer // Child servers
    Distance    int             // Hops from this server
    Users       map[string]*RemoteUser
    Channels    map[string]*RemoteChannel
    LastPing    time.Time
    Version     string
}

type RemoteUser struct {
    UID      string
    Nick     string
    User     string
    Host     string
    RealName string
    Server   *LinkedServer
    Modes    string
    Away     string
    Channels map[string]bool
}

type RemoteChannel struct {
    Name      string
    TS        int64              // Channel timestamp
    Modes     string
    Topic     string
    TopicTime int64
    TopicBy   string
    Members   map[string]string  // UID -> modes (@, +, etc)
}
```

## Protocol Messages

### 1. Server Handshake

**Direction:** Initiating → Receiving

```
Step 1: PASS <password> TS 6 <SID>
Step 2: CAPAB :ENCAP EX IE KLN KNOCK TB UNKLN
Step 3: SERVER <name> <hopcount> :<description>
```

**Example:**
```
PASS mySecretPass123 TS 6 0AA
CAPAB :QS EX CHW IE KLN KNOCK TB UNKLN
SERVER hub1.example.net 1 :Main Hub Server
```

**Receiving server responds:**
```
PASS anotherSecretPass TS 6 1BB
CAPAB :QS EX CHW IE KLN KNOCK TB UNKLN
SERVER hub2.example.net 1 :Secondary Hub Server
SVINFO 6 6 0 :1729200000
```

### 2. User Introduction (Burst)

**Format:**
```
UID <nick> <hopcount> <ts> <modes> <user> <host> <IP> <UID> :<realname>
```

**Example:**
```
:0AA UID alice 1 1729200000 +i alice localhost 127.0.0.1 0AAAAAAAA :Alice Wonderland
:0AA UID bob 1 1729200100 +o bob localhost 127.0.0.1 0AAAAAAAB :Bob the Operator
```

**On nick change:**
```
:0AAAAAAAA NICK alice2 :1729200200
```

**On quit:**
```
:0AAAAAAAA QUIT :Client exited
```

### 3. Channel State (SJOIN)

**Format:**
```
:SID SJOIN <TS> <channel> <modes> :<members>
```

**Members format:** `@UID` (op), `+UID` (voice), `UID` (normal)

**Example:**
```
:0AA SJOIN 1729200000 #test +nt :@0AAAAAAAA +0AAAAAAAB 0AAAAAAAC
```

Means:
- Channel #test created at timestamp 1729200000
- Modes: +n +t
- Members: @alice (op), +bob (voice), charlie (normal)

### 4. Channel Operations

**JOIN:**
```
:0AAAAAAAA JOIN #test
or
:0AA SJOIN 1729200000 #test + :0AAAAAAAA
```

**PART:**
```
:0AAAAAAAA PART #test :Goodbye
```

**KICK:**
```
:0AAAAAAAA KICK #test 0AAAAAAAB :Bad behavior
```

**MODE:**
```
:0AAAAAAAA MODE #test +o 0AAAAAAAB
:0AAAAAAAA TMODE 1729200000 #test +k secretpass
```

**TOPIC:**
```
:0AAAAAAAA TOPIC #test :Welcome to the channel!
or
:0AA TB #test 1729200000 alice!alice@host :Welcome!
```

### 5. Messages

**PRIVMSG/NOTICE:**
```
:0AAAAAAAA PRIVMSG #test :Hello everyone!
:0AAAAAAAA PRIVMSG 1BBAAAAAA :Private message
:0AAAAAAAA NOTICE 1BBAAAAAA :This is a notice
```

### 6. Server Management

**SQUIT (Server Quit):**
```
:0AA SQUIT 1BB :Server is shutting down
```

**PING/PONG:**
```
PING :0AA
:1BB PONG 1BB :0AA
```

## Timestamp-Based Conflict Resolution

### Problem: Nick Collisions

Two users try to use the same nick on different servers:

```
Server A: alice (TS: 1000)
Server B: alice (TS: 1100)
```

**Resolution:** Lower timestamp wins!
- alice on Server A keeps nick (TS 1000)
- alice on Server B gets force-renamed to Guest12345 (TS 1100)

### Problem: Channel Timestamp (TS)

Two servers have different channel states:

```
Server A: #test (TS: 1000, modes: +nt, ops: @alice)
Server B: #test (TS: 1100, modes: +i, ops: @bob)
```

**Resolution:** Lower timestamp wins!
- Server A's state is authoritative (TS 1000)
- Server B adopts Server A's modes and ops
- Bob loses ops, alice keeps ops

**Why this works:**
- Older channel = more authoritative
- Prevents mode wars
- Deterministic resolution

## Connection Flow

### 1. Initial Connection

```
Client → Server A                      Server A → Server B
  |                                           |
  | NICK alice                                | PASS secret TS 6 0AA
  | USER alice ...                            | CAPAB :...
  | (registration)                            | SERVER hub1 1 :Hub
  |                                           |
  |← Welcome messages                         |← PASS secret2 TS 6 1BB
  |                                           |← CAPAB :...
  |                                           |← SERVER hub2 1 :Hub
  |                                           |← SVINFO ...
  |                                           |
  |                                           | (BURST MODE)
  |                                           |← UID bob 1 ... :Bob
  |                                           |← UID charlie 1 ... 
  |                                           |← SJOIN ... #test ...
  |                                           |
  |                                           | (END BURST)
  |                                           |← PING :1BB
  |                                           |
  | JOIN #test                                |→ PONG 0AA :1BB
  |← You join #test                           |
  |                                           |→ :0AAAAAAAA JOIN #test
  |                                           |
  | PRIVMSG #test :Hi!                        |→ :0AAAAAAAA PRIVMSG #test :Hi!
  |                                           |
  |                                           | (broadcast to Server B's users)
  |                                           | bob sees: <alice> Hi!
```

### 2. Message Routing

When alice on Server A sends to bob on Server B:

```
alice (Server A) → Server A → Server B → bob (Server B)
```

Server A logic:
1. Receive: PRIVMSG bob :Hello
2. Lookup: bob's UID = 1BBAAAAAA
3. Find: bob is on Server B (SID 1BB)
4. Route: :0AAAAAAAA PRIVMSG 1BBAAAAAA :Hello → Server B

Server B logic:
1. Receive: :0AAAAAAAA PRIVMSG 1BBAAAAAA :Hello
2. Lookup: 1BBAAAAAA is local user bob
3. Deliver: :alice!alice@host PRIVMSG bob :Hello

### 3. Network Split Handling

When Server B disconnects:

```
Before:
Server A ← → Server B
  alice        bob

After split:
Server A        Server B
  alice        bob (isolated)

Server A actions:
- Detect disconnect (ping timeout or socket close)
- Send SQUIT to all other servers
- Remove all Server B users from channels
- Send QUIT for each lost user
- Clean up state

Server B actions:
- Detect disconnect
- Continue operating independently
- Users see: "Net split: ServerA ServerB"

When reconnect:
- Full burst again
- Resolve any conflicts
- Users see: "Net join: ServerA ServerB"
```

## Configuration

### config.yaml

```yaml
server:
  name: "hub1.example.net"
  sid: "0AA"              # Server ID (unique!)
  description: "Main Hub Server"
  is_hub: true            # Can link other servers

linking:
  enabled: true
  port: 7777              # Server-to-server port
  
  # Servers allowed to connect
  links:
    - name: "hub2.example.net"
      sid: "1BB"
      host: "192.168.1.100"
      port: 7777
      password: "linkpass123"
      auto_connect: true
      is_hub: true
    
    - name: "leaf1.example.net"
      sid: "2CC"
      host: "192.168.1.101"
      port: 7777
      password: "leafpass456"
      auto_connect: false
      is_hub: false

  # Link authentication
  password: "myLinkPassword"  # Accept connections with this pass
  
  # Ping settings
  ping_frequency: 60
  ping_timeout: 180
```

## Implementation Plan

### Phase 7.1: Foundation (Week 1)

1. **Server ID system**
   - Generate/assign SIDs
   - Implement UID generation
   - Update Client to use UIDs

2. **Link listener**
   - Create server-to-server TCP listener
   - Implement link authentication
   - Basic handshake (PASS, SERVER)

3. **Server structure**
   - LinkedServer data structure
   - Server registry
   - Connection management

### Phase 7.2: Burst Mode (Week 2)

4. **User burst**
   - Send UID for all local users
   - Receive and store remote users
   - Build global user map

5. **Channel burst**
   - Send SJOIN for all channels
   - Receive and merge channel state
   - Timestamp conflict resolution

6. **State synchronization**
   - Ensure consistency
   - Handle burst completion

### Phase 7.3: Live Propagation (Week 3)

7. **User events**
   - Propagate NICK changes
   - Propagate QUIT
   - Handle remote user updates

8. **Channel events**
   - Propagate JOIN/PART
   - Propagate MODE/TOPIC/KICK
   - Update remote channel state

9. **Message routing**
   - Route PRIVMSG across servers
   - Route NOTICE across servers
   - Efficient path finding

### Phase 7.4: Advanced Features (Week 4)

10. **Split handling**
    - Detect disconnects
    - Clean up remote state
    - Rejoin synchronization

11. **Server commands**
    - CONNECT (link servers)
    - SQUIT (disconnect servers)
    - LINKS (show topology)
    - MAP (visualize network)

12. **Testing & Documentation**
    - Multi-server test setup
    - Integration tests
    - Network administration guide

## Testing Strategy

### Local Testing

```bash
# Terminal 1: Server A (hub)
./bin/ircd -config config-serverA.yaml

# Terminal 2: Server B (hub)
./bin/ircd -config config-serverB.yaml

# Terminal 3: Connect Server B to Server A
telnet localhost 7777
PASS linkpass123 TS 6 1BB
CAPAB :QS EX
SERVER hub2.example.net 1 :Hub 2

# Terminal 4: Client on Server A
nc localhost 6667
NICK alice
USER alice 0 * :Alice
JOIN #test

# Terminal 5: Client on Server B
nc localhost 6668
NICK bob
USER bob 0 * :Bob
JOIN #test

# Both users should see each other!
```

### Integration Tests

```bash
./tests/test_server_linking.sh

Tests:
1. Server handshake
2. User burst
3. Channel burst
4. Cross-server PRIVMSG
5. Cross-server JOIN/PART
6. Nick collision resolution
7. Channel TS resolution
8. Network split
9. Network rejoin
10. 3-server topology
```

## Security Considerations

1. **Authentication**
   - Strong link passwords
   - Optional TLS for server links
   - Password hashing
   - IP whitelist

2. **DoS Prevention**
   - Rate limit link connections
   - Validate server commands
   - Limit burst size
   - Connection flood protection

3. **Data Validation**
   - Validate SIDs and UIDs
   - Check timestamp sanity
   - Prevent injection attacks
   - Sanitize server names

## Performance

### Optimizations

1. **Efficient routing**
   - Build routing table
   - Cache remote user locations
   - Minimize message copies

2. **Batch updates**
   - Combine multiple MODEs
   - Batch user introductions
   - Compress channel states

3. **Memory management**
   - Limit remote user cache
   - Clean up stale data
   - Efficient data structures

### Scalability

```
Estimated capacity:
- 10 linked servers
- 10,000 users per server
- 100,000 total network users
- 50,000 channels
- 1000 messages/second
```

## Documentation

Files to create:
- `docs/SERVER_LINKING.md` - User guide
- `docs/LINKING_PROTOCOL.md` - Protocol spec
- `docs/NETWORK_ADMIN.md` - Admin guide
- `tests/test_server_linking.sh` - Test script

## Success Metrics

✅ Two servers can link and stay connected
✅ Users on different servers see each other in channels
✅ Messages route correctly across servers
✅ Splits are detected and handled
✅ Rejoins sync state correctly
✅ Nick/channel collisions resolve properly
✅ Network is stable under load

## Timeline

- **Week 1:** Foundation + handshake
- **Week 2:** Burst mode + state sync
- **Week 3:** Live propagation + routing
- **Week 4:** Advanced features + testing

**Total:** ~4 weeks for full implementation

## Next Steps

1. Review and approve this design
2. Set up development environment with multiple server configs
3. Start with Phase 7.1: Foundation
4. Iterative development and testing

Ready to start building? 🚀
