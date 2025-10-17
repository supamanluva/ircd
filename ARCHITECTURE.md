# IRC Server Architecture

## Overview

This document outlines the architectural decisions and design patterns used in the IRC server implementation.

## Design Principles

1. **Separation of Concerns** - Each module has a single, well-defined responsibility
2. **Concurrency Safety** - Thread-safe operations using goroutines and mutexes
3. **Graceful Degradation** - Server continues operating even when individual clients fail
4. **Protocol Compliance** - Follows RFC 1459 and RFC 2812 specifications
5. **Security First** - Rate limiting, input validation, and TLS support built-in

## Module Architecture

### Core Modules

```
┌─────────────────────────────────────────────────────┐
│                    Main (cmd/ircd)                  │
│          Orchestrates startup and shutdown          │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                  Server (internal/server)           │
│     - TCP listener                                  │
│     - Connection handler                            │
│     - Client registry                               │
│     - Channel registry                              │
└─────────────────────────────────────────────────────┘
         │                                    │
         ▼                                    ▼
┌──────────────────────┐          ┌──────────────────────┐
│  Client              │          │  Channel             │
│  (internal/client)   │◄────────►│  (internal/channel)  │
│  - Connection        │          │  - Member list       │
│  - State             │          │  - Topic             │
│  - Send queue        │          │  - Broadcasting      │
└──────────────────────┘          └──────────────────────┘
         │
         ▼
┌──────────────────────┐
│  Parser              │
│  (internal/parser)   │
│  - IRC message       │
│    parsing           │
└──────────────────────┘
         │
         ▼
┌──────────────────────┐
│  Commands            │
│  (internal/commands) │
│  - Command handlers  │
│  - Numeric replies   │
└──────────────────────┘
```

## Concurrency Model

### Client Goroutines

Each client connection runs in its own goroutine:

```go
func (s *Server) handleClient(conn net.Conn) {
    // Each client gets:
    // 1. Main goroutine for reading and processing
    // 2. Send worker goroutine for writing
}
```

**Benefits:**
- Non-blocking I/O for each client
- Isolated failures (one client crash doesn't affect others)
- Natural flow control via buffered channels

### Message Flow

```
Client Connection
    │
    ├─► Read Goroutine ─────────────┐
    │   - Reads from TCP socket      │
    │   - Parses IRC messages        │
    │   - Routes to command handlers │
    │                                │
    │                                ▼
    │                        Command Handler
    │                                │
    │                                ├─► Update client state
    │                                ├─► Update channel state
    │                                └─► Queue messages
    │                                        │
    └─► Send Goroutine ◄────────────────────┘
        - Receives from sendQueue
        - Writes to TCP socket
```

### Synchronization Strategy

1. **Client State** - Protected by `sync.RWMutex`
   - Multiple readers, single writer
   - Lock held for minimal time

2. **Channel State** - Protected by `sync.RWMutex`
   - Member list modifications are serialized
   - Broadcasting uses read lock

3. **Server State** - Protected by `sync.RWMutex`
   - Client registry
   - Channel registry

4. **Message Passing** - Buffered channels
   - Each client has a send queue (buffered channel)
   - Non-blocking sends with select
   - Prevents slow clients from blocking fast ones

## Message Processing Pipeline

```
┌──────────────┐
│ Raw TCP Data │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│    Parser    │  Extracts: prefix, command, params
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Validation  │  Checks: command exists, params valid
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Handler    │  Executes command logic
└──────┬───────┘
       │
       ├─► Update State
       ├─► Queue Responses
       └─► Broadcast to Channels
```

## State Management

### Client State

```go
type Client struct {
    // Immutable after creation
    conn     net.Conn
    connType ConnectionType
    
    // Mutable, protected by mutex
    nickname  string
    username  string
    realname  string
    channels  map[string]bool
    registered bool
    
    // Concurrency primitives
    mu           sync.RWMutex
    sendQueue    chan string
    disconnected bool
}
```

**Access Patterns:**
- Read-heavy workload (checking nickname, channels)
- Write happens during registration and channel join/part
- Use `RWMutex` to optimize for reads

### Channel State

```go
type Channel struct {
    // Immutable
    name      string
    createdAt time.Time
    
    // Mutable, protected by mutex
    topic     string
    members   map[string]*Client
    operators map[string]bool
    
    // Concurrency primitives
    mu sync.RWMutex
}
```

**Broadcasting Strategy:**
- Take read lock
- Iterate members
- Queue message to each client's send queue
- Release lock quickly

## Error Handling

### Panic Recovery

```go
defer func() {
    if r := recover(); r != nil {
        log.Error("Panic recovered", "error", r)
        // Cleanup and graceful degradation
    }
}()
```

**Applied at:**
- Client handler goroutines
- Send worker goroutines
- Command handlers

### Error Types

1. **Protocol Errors** - Send numeric reply, continue
2. **Connection Errors** - Disconnect client, cleanup
3. **Internal Errors** - Log, attempt recovery
4. **Configuration Errors** - Fail fast at startup

## Security Architecture

### Input Validation

```
Raw Input
    │
    ├─► Length Check (max 512 bytes per RFC)
    ├─► Character Validation (no control chars)
    ├─► Command Validation (known commands only)
    └─► Parameter Validation (correct count, format)
```

### Rate Limiting

**Per-Client Token Bucket:**
- X messages per second
- Burst capacity of Y messages
- Exceeding limit results in throttling or kick

**Implementation:**
```go
type RateLimiter struct {
    tokens      float64
    maxTokens   float64
    refillRate  float64
    lastRefill  time.Time
    mu          sync.Mutex
}
```

### TLS Support

- Optional TLS on separate port (6697)
- Use standard library `crypto/tls`
- Support modern cipher suites
- Certificate validation

## Web Client Integration

### Architecture

```
Browser
    │
    ├─► HTTPS + WebSocket
    │
    ▼
WebSocket Handler
    │
    ├─► Upgrade HTTP to WebSocket
    ├─► Create Virtual IRC Client
    │   (implements same Client interface)
    │
    ▼
IRC Core (existing server logic)
```

### Session Management

**IP-Based Restriction:**
```go
type SessionManager struct {
    sessions map[string]*Session  // IP -> Session
    mu       sync.RWMutex
}

func (sm *SessionManager) Connect(ip string, client *Client) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    if existingSession, exists := sm.sessions[ip]; exists {
        // Disconnect existing session
        existingSession.Disconnect()
    }
    
    sm.sessions[ip] = &Session{client: client}
    return nil
}
```

## Performance Considerations

### Memory

- **Client State**: ~1KB per client
- **Send Queue**: 100 messages × ~512 bytes = 50KB per client
- **1000 clients**: ~51MB (acceptable)

### CPU

- **Goroutines**: 1000 clients = 2000 goroutines (manageable)
- **Context Switching**: Goroutines are lightweight
- **Mutex Contention**: Minimized by using RWMutex and short critical sections

### Network

- **Buffer Sizes**: 8KB read, 8KB write per connection
- **TCP Tuning**: Keep-alive, Nagle's algorithm control
- **Timeout Management**: Read/write deadlines prevent hung connections

## Scalability Strategy

### Current Design (Single Instance)

**Limits:**
- 10,000 - 100,000 concurrent clients (depends on hardware)
- Single point of failure
- Vertical scaling only

### Future: Horizontal Scaling

**Option 1: Server Federation**
```
Client ──► Server A ◄──┐
Client ──► Server B ◄──┼─► Inter-server Protocol
Client ──► Server C ◄──┘
```

**Option 2: Shared State (Redis)**
```
Client ──► Server A ──┐
Client ──► Server B ──┼─► Redis (channels, users)
Client ──► Server C ──┘
```

## Testing Strategy

### Unit Tests
- Parser: Message parsing edge cases
- Client: State management
- Channel: Member operations
- Commands: Handler logic

### Integration Tests
- Connect multiple clients
- Join channels
- Send messages
- Handle disconnects

### Load Tests
- Concurrent connections
- Message throughput
- Memory usage over time

## Deployment

### Systemd Service

```ini
[Unit]
Description=IRC Server
After=network.target

[Service]
Type=simple
User=ircd
ExecStart=/usr/local/bin/ircd -config /etc/ircd/config.yaml
Restart=always

[Install]
WantedBy=multi-user.target
```

### Docker

- Multi-stage build for small image
- Non-root user
- Health checks
- Volume for logs and config

### Monitoring

**Metrics to Track:**
- Connected clients
- Active channels
- Messages per second
- Memory usage
- Goroutine count
- Connection errors

## Future Enhancements

1. **Persistence**
   - SQLite for user registration
   - Channel topic persistence
   - Message history (optional)

2. **Services**
   - NickServ (nickname registration)
   - ChanServ (channel management)
   - MemoServ (offline messages)

3. **Federation**
   - Server-to-server protocol
   - Distributed channels
   - Global user directory

4. **REST API**
   - Admin interface
   - Statistics
   - User management

5. **Plugin System**
   - Custom commands
   - Event hooks
   - Moderation tools
