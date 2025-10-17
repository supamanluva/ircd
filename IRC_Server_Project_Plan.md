# IRC Server in Go - Project Plan

## 🧠 Overview

**Goal:** Build a minimal yet functional IRC server written in Go that runs smoothly and securely on a Linux server.  
**Key Features:**
- Multi-user chat over TCP
- Channels (rooms)
- Nickname registration and management
- Command parsing (`JOIN`, `PART`, `PRIVMSG`, etc.)
- Authentication (optional)
- Logging and moderation tools
- TLS support for security
- Extensible plugin structure (optional stretch goal)

---

## 🧱 Project Structure

### 1. Core Modules

| Module | Description |
|--------|--------------|
| `server` | TCP listener, connection handler, startup/shutdown logic |
| `client` | Manages each client’s state (nickname, channels, permissions) |
| `parser` | Parses raw IRC protocol messages into structured commands |
| `channel` | Handles channel creation, user joins, message broadcasts |
| `auth` | Optional: Manages user authentication and registration |
| `commands` | Implements core IRC commands and routing |
| `security` | TLS configuration, rate limiting, input sanitization |
| `logger` | Logs server events and chat activity |
| `config` | YAML or TOML-based configuration loader |
| `tests` | Unit and integration tests |

---

## 🪜 Development Phases

### Phase 1: Setup & Infrastructure
**Goal:** Get a basic TCP server running.

**Tasks:**
- Initialize Go module (`go mod init`)
- Implement a TCP listener on a configurable port
- Handle new client connections
- Send/receive raw messages
- Graceful shutdown handling (signal catching)

**Deliverable:**  
Clients can connect via `telnet` or `netcat` and exchange raw messages.

---

### Phase 2: IRC Protocol Foundation
**Goal:** Implement core IRC protocol behavior.

**Tasks:**
- Parse and handle basic IRC commands:
  - `NICK`, `USER`, `PING`, `PONG`, `QUIT`
- Manage client state (nickname, registration)
- Implement broadcast messaging

**Deliverable:**  
Users can connect, set nicknames, and send global messages.

---

### Phase 3: Channels & Messaging
**Goal:** Support multi-channel chat rooms.

**Tasks:**
- Implement channels as Go structs with member lists
- Commands: `JOIN`, `PART`, `PRIVMSG`, `NOTICE`
- Channel message broadcasting
- Handle topic setting (`TOPIC`) and user lists (`NAMES`)

**Deliverable:**  
Users can join rooms, chat within them, and list who’s online.

---

### Phase 4: Security & Stability
**Goal:** Ensure smooth operation and basic protection.

**Tasks:**
- Add TLS support (optional but recommended)
- Rate limiting (prevent flooding)
- Input sanitization
- Connection timeout/heartbeat handling
- Panic recovery and logging

**Deliverable:**  
Server handles thousands of users safely without crashes or abuse.

---

### Phase 5: Administration & Persistence
**Goal:** Add useful admin and persistence features.

**Tasks:**
- Add operator commands (`KICK`, `BAN`, etc.)
- Add optional user authentication (passwords or simple database)
- Persistent channel topics or logs (file or SQLite)
- Configuration via `config.yaml`

**Deliverable:**  
A production-ready IRC server suitable for small communities.

---

### Phase 6: Testing, Optimization & Deployment
**Goal:** Harden and deploy the server.

**Tasks:**
- Unit tests for all major components
- Integration tests with IRC clients (e.g., `irssi`, `weechat`)
- Benchmark concurrency
- Containerize (Dockerfile)
- Deploy to Linux (systemd service)

**Deliverable:**  
Stable, tested, and monitored IRC server ready for real users.

---

## 🛡️ Security Considerations

- **TLS encryption** for client connections (e.g., via `crypto/tls`)
- **Rate limiting** to prevent spam/flooding
- **Command validation** to prevent malformed input
- **Logging** and **moderation commands** for abuse prevention
- **User authentication** (if enabled)
- **Sandboxing** and **least privilege** (run under dedicated non-root user)

---

## 🧰 Recommended Libraries

| Purpose | Library |
|----------|----------|
| Config parsing | `github.com/spf13/viper` |
| Logging | `go.uber.org/zap` or `github.com/sirupsen/logrus` |
| TLS | Standard library (`crypto/tls`) |
| CLI / Flags | `github.com/spf13/cobra` |
| Unit testing | Standard `testing` package |
| Mocking (optional) | `github.com/stretchr/testify/mock` |

---

## 🧾 Example Directory Layout

```
irc-server/
├── cmd/
│   └── ircd/               # main.go entry point
├── internal/
│   ├── server/
│   ├── client/
│   ├── channel/
│   ├── parser/
│   ├── commands/
│   ├── security/
│   └── logger/
├── config/
│   └── config.yaml
├── pkg/
│   └── utils/
├── tests/
│   └── integration_test.go
├── Dockerfile
└── go.mod
```

---

## 🌐 Web Client Login Feature

**New Goal:** Extend the IRC server to allow users to log in from an HTTPS web client.

**Description:**
- Provide a simple web-based client interface using Go’s `net/http` and WebSocket support.
- Each user connects through a secure HTTPS endpoint that bridges to the IRC core via WebSockets.
- Implement authentication and session management for web users.
- Enforce **one active login per IP address** to prevent abuse or multiple logins from the same host.

**Technical Tasks:**
- Add a new module: `webclient`
- Use Go’s `net/http` with TLS for serving the web UI
- Implement a WebSocket proxy layer that forwards IRC messages to/from connected users
- Maintain IP-based session tracking and disconnect duplicate logins
- Extend configuration file (`config.yaml`) to include web server options (e.g., port, TLS certs, rate limits)
- Optional: Add frontend (HTML + minimal JS) for chat UI

**Deliverable:**  
A secure, browser-accessible web client that connects to the IRC server with single-IP login enforcement.

---

## 🚀 Stretch Goals (Advanced)

- WebSocket bridge (for browser IRC clients)
- Federation between multiple IRC nodes
- Plugin API for custom commands
- REST API for admin stats
- Docker Compose setup with persistent storage
