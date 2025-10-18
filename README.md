# IRC Server (ircd)

A modern, production-ready IRC server implementation written in Go, following RFC 1459 specifications.

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Coverage](https://img.shields.io/badge/coverage-75%25-green)
![Go Version](https://img.shields.io/badge/go-1.21%2B-blue)
![License](https://img.shields.io/badge/license-MIT-blue)

## 🚀 Features

### Core IRC Functionality
- ✅ **23 IRC Commands** - NICK, USER, JOIN, PART, PRIVMSG, NOTICE, QUIT, PING, PONG, NAMES, TOPIC, MODE, KICK, WHO, WHOIS, LIST, INVITE, OPER, AWAY, USERHOST, ISON
- ✅ **Multi-channel Support** - Create and manage multiple chat rooms
- ✅ **User Management** - Nickname registration, hostmask tracking, away status
- ✅ **Channel Operators** - First user becomes operator, grant/revoke operator status
- ✅ **User & Channel Modes** - +i (invisible), +o (operator), +m (moderated), +n (no external), +t (topic protection), +b (ban), +k (key), +v (voice)
- ✅ **Server Operators** - OPER command with bcrypt authentication
- ✅ **Presence System** - AWAY, USERHOST, ISON commands
- ✅ **WebSocket Support** - Browser-based IRC clients (port 8080)

### Security & Stability
- 🔒 **TLS/SSL Encryption** - Secure connections on port 7000
- 🛡️ **Rate Limiting** - Prevent flooding (5 msg/sec, burst of 10)
- ✅ **Input Validation** - RFC-compliant message parsing
- 🔐 **Ban Lists** - Per-channel user banning
- ⚡ **Concurrent Connection Handling** - Goroutine-per-client architecture

### Administration
- 👮 **Operator Commands** - MODE, KICK for channel management
- 📝 **Comprehensive Logging** - Structured logging with levels
- 🔍 **User Information** - WHO and WHOIS commands for user details
- 📋 **Channel Listing** - LIST command to browse channels
- 📨 **Invitations** - INVITE users to channels

### Distributed Network (NEW! ✨)
- 🌐 **Server Linking** - Connect multiple IRC servers into one network
- 🔗 **Hub & Leaf Topology** - Scalable distributed architecture
- 🔄 **Real-time Sync** - Users on any server can see and message users on all other servers
- 📡 **TS6 Protocol** - Industry-standard server-to-server protocol
- ⚡ **Zero-Config Discovery** - Automatic network state synchronization

### Production Ready
- 🐳 **Docker Support** - Docker Compose with optional Prometheus/Grafana monitoring
- 🔧 **systemd Integration** - Service file with security hardening
- 📊 **Test Coverage** - 75% coverage in critical packages (security 98.5%, parser 81.6%, channel 75.7%, commands 66.9%)
- 📚 **Comprehensive Documentation** - Deployment guides, phase documentation, troubleshooting
- ✅ **Integration Tested** - Tested with irssi, weechat, hexchat

## Project Structure

```
ircd/
├── cmd/
│   └── ircd/                 # Main entry point
│       └── main.go
├── internal/
│   ├── server/               # TCP server and connection handling
│   ├── client/               # Client state management
│   ├── channel/              # Channel (room) management
│   ├── parser/               # IRC protocol parser
│   ├── commands/             # IRC command handlers
│   ├── security/             # Rate limiting, TLS, validation
│   ├── logger/               # Structured logging
│   └── webclient/            # Web client and WebSocket support
├── config/
│   └── config.yaml           # Server configuration
├── pkg/
│   └── utils/                # Shared utilities
├── tests/                    # Integration tests
├── go.mod
└── README.md
```

## Development Phases

### ✅ Phase 0: Setup & Infrastructure (Complete)
- [x] Project structure
- [x] Basic TCP server
- [x] Logger
- [x] Client management
- [x] Parser
- [x] Channel management

### ✅ Phase 1: IRC Protocol Foundation (Complete)
- [x] NICK, USER commands
- [x] PING/PONG keepalive
- [x] Client registration flow
- [x] QUIT handling

### ✅ Phase 2: Channels & Messaging (Complete)
- [x] JOIN, PART commands
- [x] PRIVMSG, NOTICE
- [x] Channel broadcasting
- [x] TOPIC, NAMES commands

### ✅ Phase 3: Security & Stability (Complete)
- [x] TLS/SSL support (port 7000)
- [x] Rate limiting (token bucket algorithm)
- [x] Input sanitization and validation
- [x] Connection timeouts and keepalive
- [x] Flood protection
- [x] YAML configuration loading

### ✅ Phase 4: Administration & Operator Commands (Complete)
- [x] User modes (+i invisible, +w wallops, +o operator)
- [x] Channel modes (+n no external, +t topic protection, +i invite-only, +m moderated, +o operator, +b ban)
- [x] MODE command (user and channel)
- [x] KICK command with permission checks
- [x] Channel operator privileges
- [x] Ban list management with add/remove

### ✅ Phase 5: Testing & Deployment (Complete)
- [x] Unit tests (66.9% commands, 75.7% channel, 81.6% parser, 98.5% security)
- [x] Integration tests (Phase 2, 3, 4 scenarios)
- [x] Docker Compose setup with Prometheus/Grafana
- [x] systemd service with security hardening
- [x] Production deployment guide
- [x] Deployment testing and validation

### ✅ Phase 6: Advanced Features (Complete)
- [x] WHO command (list users in channels with flags)
- [x] WHOIS command (detailed user information with idle time)
- [x] LIST command (list channels with topics and counts)
- [x] INVITE command (invite users to channels)
- [x] Channel keys (+k mode) for password-protected channels
- [x] Voice mode (+v) for speaking in moderated channels
- [x] OPER command for server operator authentication (bcrypt)
- [x] AWAY command for away status with messages
- [x] USERHOST command for user@host information
- [x] ISON command for online presence checking
- [x] Enhanced WHO with away status (G/H flags)
- [x] Enhanced WHOIS with away messages
- [x] PRIVMSG away notifications
- [x] **WebSocket Support** - Browser-based IRC clients (port 8080)
  - Origin validation with wildcard patterns
  - TLS/SSL support (WSS)
  - Health check endpoint
  - Interactive HTML test client

**Total Commands: 23** | **Channel Modes: 8** | **User Modes: 3** | **Protocols: TCP, TLS, WebSocket**

### ✅ Phase 7: Server Linking & Distributed IRC (Complete) ⭐ NEW!
- [x] **Server-to-Server Protocol** (TS6)
  - Server authentication and handshake
  - Unique server IDs (SID) and user IDs (UID)
  - Hub and leaf topology support
  - Auto-connect configuration
- [x] **Burst Mode Synchronization**
  - Complete state transfer on link
  - SJOIN for channels with members
  - User and channel state propagation
- [x] **Message Routing & Propagation**
  - Cross-server PRIVMSG and NOTICE
  - JOIN, PART, QUIT propagation
  - NICK changes across network
  - MODE, TOPIC, KICK, INVITE routing
- [x] **Network State Management**
  - Remote user tracking with UIDs
  - Remote channel membership
  - Network-wide NAMES lists
  - Clean state cleanup on QUIT/disconnect
- [x] **Verified Working** ✅
  - Users on hub can see/message users on leaf
  - Full bidirectional communication
  - Real-time synchronization
  - Production-ready distributed network

**🚀 See [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md) for complete setup guide!**

### 🎯 Completed Phases
- ✅ Phase 1-5: Core IRC functionality (commands, channels, security)
- ✅ Phase 6: Advanced features (WebSocket, web client, operators)
- ✅ Phase 7: **Server linking for distributed IRC networks** (TS6 protocol) ⭐ NEW!
  - Full multi-server support with hub/leaf topology
  - Cross-server user visibility and messaging
  - See [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md) for setup

### 🚀 Future Enhancements
- **Services integration** (NickServ, ChanServ, MemoServ)
- **Advanced security** (SASL authentication, certificate fingerprinting)
- **Message history** (ZNC-style bouncer features)
- **Server-to-server TLS** (encrypted server links)
- **IRCv3 capabilities** (message tags, account tracking)
- **Mesh topology** (multi-hop server routing)

## Getting Started

### Quick Start (Single Server)

#### Prerequisites

- Go 1.21 or higher

#### Installation

```bash
# Clone the repository
git clone https://github.com/supamanluva/ircd.git
cd ircd

# Build the server
make build

# Generate TLS certificates (for testing)
./generate_cert.sh

# Run the server
./bin/ircd -config config/config.yaml
```

### Configuration

Edit `config/config.yaml` to customize server settings:

```yaml
server:
  name: "IRCServer"
  host: "0.0.0.0"
  port: 6667              # Plaintext port
  tls:
    enabled: true
    port: 7000            # TLS/SSL port
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
  
  max_clients: 1000
  timeout_seconds: 300
  ping_interval_seconds: 60
  
  rate_limit:
    enabled: true
    messages_per_second: 5
    burst: 10

# WebSocket support for browser-based IRC clients
websocket:
  enabled: true
  host: "0.0.0.0"
  port: 8080              # WebSocket port
  allowed_origins:
    - "*"                 # Allow all origins (restrict in production!)
```

### Testing with a Client

**Traditional IRC Clients:**

```bash
# Connect via plaintext (testing only)
telnet localhost 6667

# Or use netcat
nc localhost 6667

# Connect via TLS (recommended)
openssl s_client -connect localhost:7000

# Or use a proper IRC client with TLS
irssi -c localhost -p 7000 --tls
weechat -r "/server add local localhost/7000 -ssl"
```

**Browser-based WebSocket Client:**

```bash
# Start the server
./bin/ircd

# Open tests/websocket_client.html in your browser
# Or serve it via HTTP:
python3 -m http.server 8000
# Navigate to: http://localhost:8000/tests/websocket_client.html
```

### 🌐 Distributed Network Setup (Multi-Server)

Want to run multiple servers in a network where users can talk across servers?

**📋 Quick Start**: See [QUICK_START_LINKING.md](QUICK_START_LINKING.md) for one-page setup guide!

**Scenario 1: You're running the HUB** and letting others connect their leaf servers:
- See [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md#scenario-1-running-a-hub-server)
- Configure your hub to accept server links
- Share connection details with leaf admins

**Scenario 2: You're connecting a LEAF** to someone else's hub:
- See [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md#scenario-2-connecting-a-leaf-to-remote-hub)
- Get hub connection details from the hub admin
- Configure your leaf to auto-connect

**Quick Example - Leaf connecting to Hub:**

```yaml
# Your leaf config
linking:
  enabled: true
  server_id: "002"              # Unique ID (not 001!)
  password: "shared_secret"     # Must match hub
  links:
    - name: "irc.example.com"   # Hub's name
      sid: "001"                 # Hub's ID
      host: "hub.example.com"    # Hub's address
      port: 7000                 # Hub's link port
      password: "shared_secret"  # Same password
      auto_connect: true         # Connect automatically
      is_hub: true
```

Start your leaf, and users can connect to either server and see everyone! 🎉

**Full guide:** [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md)

## Security

### TLS/SSL Setup

For production, use proper certificates:

```bash
# Let's Encrypt example
certbot certonly --standalone -d irc.yourdomain.com

# Update config.yaml with certificate paths
server:
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt/live/irc.yourdomain.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/irc.yourdomain.com/privkey.pem"
```

### Rate Limiting

The server includes built-in flood protection:
- Token bucket algorithm (5 msg/sec, burst of 10)
- Per-client rate limits
- Automatic disconnection on violations

### Input Validation

All user input is validated and sanitized:
- Nicknames: ASCII letters, digits, special chars only
- Channel names: Must start with # or &
- Messages: Control characters stripped

See [docs/PHASE3_SECURITY.md](docs/PHASE3_SECURITY.md) for details.

## Architecture Decisions

### Concurrency Model
- Each client connection runs in its own goroutine
- Channels use goroutines for message broadcasting
- Thread-safe access via `sync.RWMutex`
- Message passing via Go channels for inter-goroutine communication

### Message Flow
```
Client → Parser → Command Handler → Channel/Client Manager → Broadcast
```

### State Management
- Immutable message structs
- Client state protected by mutexes
- Send queue per client to prevent blocking

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
# Development build
go build ./cmd/ircd

# Production build with optimizations
go build -ldflags="-s -w" -o bin/ircd ./cmd/ircd
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - See LICENSE file for details

## References

- [RFC 1459 - Internet Relay Chat Protocol](https://tools.ietf.org/html/rfc1459)
- [RFC 2812 - IRC Client Protocol](https://tools.ietf.org/html/rfc2812)
- [Modern IRC Documentation](https://modern.ircdocs.horse/)
