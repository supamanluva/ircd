# Server Linking Setup Guide

This guide explains how to set up a distributed IRC network with multiple servers.

## 📖 Table of Contents
1. [Network Topology](#network-topology)
2. [Scenario 1: Running a Hub Server](#scenario-1-running-a-hub-server)
3. [Scenario 2: Connecting a Leaf to Remote Hub](#scenario-2-connecting-a-leaf-to-remote-hub)
4. [Configuration Reference](#configuration-reference)
5. [Verification](#verification)
6. [Troubleshooting](#troubleshooting)

---

## Network Topology

A typical IRC network has:
- **Hub Server**: Central server that accepts connections from leaf servers
- **Leaf Server**: Connects to a hub server and serves IRC clients

```
         ┌─────────────┐
         │   Hub       │ ← Clients connect here
         │ irc.net     │
         │ (0.0.0.0)   │
         └─────┬───────┘
               │
       ┌───────┴────────┐
       │                │
   ┌───▼───┐      ┌────▼────┐
   │ Leaf  │      │  Leaf   │
   │ eu    │      │  us     │
   └───────┘      └─────────┘
    ↑                 ↑
Clients            Clients
```

**Key Concept**: Clients can connect to ANY server, and they can all see and message each other!

---

## Scenario 1: Running a Hub Server

**Use Case**: You want to run the central hub server and let others connect their leaf servers to you.

### Step 1: Configure Your Hub

Create or edit `config/hub.yaml`:

```yaml
# IRC Server Configuration - HUB

server:
  name: "irc.yournetwork.com"    # Your public server name
  host: "0.0.0.0"                # Listen on all interfaces
  port: 6667                      # Client connections
  
  max_clients: 1000
  timeout_seconds: 300
  ping_interval_seconds: 60

# TLS for client connections (optional but recommended)
  tls:
    enabled: true
    port: 6697                    # Standard IRC TLS port
    cert_file: "/etc/letsencrypt/live/irc.yournetwork.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/irc.yournetwork.com/privkey.pem"

# Server operators
operators:
  - name: "admin"
    password: "$2a$10$..."        # Use bcrypt hash (see below)

# Logging
logging:
  level: "info"
  file: "logs/hub.log"
  console: true

# Server Linking Configuration - THIS IS IMPORTANT
linking:
  enabled: true
  host: "0.0.0.0"                # Listen for server links on all interfaces
  port: 7000                      # Port for server-to-server connections
  server_id: "001"                # Unique 3-char ID (use 001 for hub)
  description: "Main Hub Server"
  password: "YourSecureServerPassword123"  # CHANGE THIS!
  
  # No links section needed - leaf servers will connect TO you
```

### Step 2: Firewall Configuration

Open the necessary ports:

```bash
# Client connections (plaintext)
sudo ufw allow 6667/tcp

# Client connections (TLS) - if enabled
sudo ufw allow 6697/tcp

# Server-to-server links - CRITICAL!
sudo ufw allow 7000/tcp
```

### Step 3: Start the Hub

```bash
./ircd -config config/hub.yaml
```

### Step 4: Share Connection Info with Leaf Admins

Tell people who want to connect leaf servers:

```
Server Linking Information:
- Hub Address: irc.yournetwork.com (or your IP: 203.0.113.42)
- Link Port: 7000
- Link Password: YourSecureServerPassword123
- Hub SID: 001
- Hub Name: irc.yournetwork.com

They should configure their leaf to auto-connect to you.
```

**Security Note**: The link password should be shared securely (not in public channels!). Use a strong password.

---

## Scenario 2: Connecting a Leaf to Remote Hub

**Use Case**: Someone else is running a hub, and you want to connect your leaf server to their network.

### Step 1: Get Hub Connection Details

Contact the hub administrator and get:
- ✅ Hub address (domain or IP)
- ✅ Link port (usually 7000)
- ✅ Link password
- ✅ Hub server ID (SID)
- ✅ Hub server name

### Step 2: Configure Your Leaf

Create or edit `config/leaf.yaml`:

```yaml
# IRC Server Configuration - LEAF

server:
  name: "eu.yournetwork.com"     # Your leaf's name (must be different from hub!)
  host: "0.0.0.0"                # Listen for clients
  port: 6667                      # Clients connect here
  
  max_clients: 1000
  timeout_seconds: 300
  ping_interval_seconds: 60

# TLS for clients (optional)
  tls:
    enabled: false                # Can be different from hub

# Server operators
operators:
  - name: "admin"
    password: "$2a$10$..."

# Logging
logging:
  level: "info"
  file: "logs/leaf.log"
  console: true

# Server Linking Configuration - CONNECT TO HUB
linking:
  enabled: true
  host: "0.0.0.0"
  port: 7001                      # YOUR link port (different from hub!)
  server_id: "002"                # Unique ID (NOT 001, that's the hub!)
  description: "EU Leaf Server"
  password: "YourSecureServerPassword123"  # Must match hub's password!
  
  # THIS IS THE CRITICAL PART - Configure the hub connection
  links:
    - name: "irc.yournetwork.com"           # Hub's server name
      sid: "001"                             # Hub's server ID
      host: "203.0.113.42"                   # Hub's IP or hostname
      port: 7000                             # Hub's link port
      password: "YourSecureServerPassword123" # Hub's link password
      auto_connect: true                     # Automatically connect on startup
      is_hub: true                           # This is a hub server
```

### Step 3: Firewall Configuration (Outbound)

Your leaf needs to make outbound connection to the hub:

```bash
# Allow clients to connect to YOUR server
sudo ufw allow 6667/tcp

# Your server link port (for when hub connects back, if needed)
sudo ufw allow 7001/tcp
```

**Note**: Outbound connections (your leaf → hub) usually work by default. You're mainly opening ports for incoming client connections.

### Step 4: Start the Leaf

```bash
./ircd -config config/leaf.yaml
```

Watch the logs - you should see:

```
[INFO] Auto-connecting to name=irc.yournetwork.com
[INFO] Connected, starting handshake address=203.0.113.42:7000
[INFO] Server link established name=irc.yournetwork.com sid=001
[INFO] Burst sent name=irc.yournetwork.com users=0 channels=0
[INFO] Link established, starting message handler
```

---

## Configuration Reference

### Required Unique Values

Each server in the network must have:

1. **Unique Server Name**: `server.name`
   ```yaml
   server:
     name: "hub.example.com"    # Hub
     name: "leaf1.example.com"  # Leaf 1
     name: "leaf2.example.com"  # Leaf 2
   ```

2. **Unique Server ID (SID)**: `linking.server_id`
   - 3 alphanumeric characters
   - Convention: `001` for hub, `002`, `003`, etc. for leafs
   ```yaml
   linking:
     server_id: "001"  # Hub
     server_id: "002"  # Leaf 1
     server_id: "003"  # Leaf 2
   ```

3. **Different Client Ports** (if running on same machine):
   ```yaml
   server:
     port: 6667   # Hub
     port: 6668   # Leaf 1
     port: 6669   # Leaf 2
   ```

4. **Different Link Ports** (if running on same machine):
   ```yaml
   linking:
     port: 7000   # Hub
     port: 7001   # Leaf 1
     port: 7002   # Leaf 2
   ```

### Link Password Security

**Generate a secure password**:
```bash
openssl rand -base64 32
```

**Must match** on both hub and leaf:
- Hub: `linking.password`
- Leaf: `linking.password` AND `linking.links[0].password`

### Auto-Connect Behavior

**Hub Server**:
```yaml
linking:
  links: []    # Empty or omitted - just waits for leafs to connect
```

**Leaf Server**:
```yaml
linking:
  links:
    - name: "hub.example.com"
      auto_connect: true    # Connects automatically on startup
      is_hub: true         # Treat as hub
```

---

## Verification

### Check Link Status

After starting, verify the link is established:

```bash
# Check logs
tail -f logs/hub.log    # On hub
tail -f logs/leaf.log   # On leaf

# Look for these messages:
# Hub sees: "Server link established name=leaf.example.com"
# Leaf sees: "Server link established name=hub.example.com"
# Both see: "Burst sent" and "Burst received"
```

### Test Cross-Server Communication

**Terminal 1** - Connect to hub:
```bash
telnet hub.example.com 6667
NICK Alice
USER alice 0 * :Alice Smith
JOIN #test
PRIVMSG #test :Hello from hub!
```

**Terminal 2** - Connect to leaf:
```bash
telnet leaf.example.com 6667
NICK Bob
USER bob 0 * :Bob Jones
JOIN #test
PRIVMSG #test :Hello from leaf!
```

**Expected Result**:
- Alice sees Bob's messages ✅
- Bob sees Alice's messages ✅
- NAMES shows both users ✅

---

## Troubleshooting

### Link Not Connecting

**Symptom**: Leaf shows "Connection refused" or timeout

**Solutions**:
1. Check hub is running: `netstat -tlnp | grep 7000`
2. Check firewall: `sudo ufw status`
3. Verify hub's `linking.host` is `0.0.0.0` not `127.0.0.1`
4. Test connectivity: `telnet hub_ip 7000`

### Wrong Password

**Symptom**: "Invalid link password" in logs

**Solutions**:
1. Verify passwords match EXACTLY
2. Check for trailing spaces in YAML
3. Ensure both use `linking.password`
4. Leaf also needs password in `linking.links[0].password`

### SID Conflict

**Symptom**: "Server ID already in use"

**Solutions**:
1. Each server needs unique `linking.server_id`
2. Typical: `001` (hub), `002` (leaf1), `003` (leaf2)
3. Must be exactly 3 characters

### Messages Not Crossing Servers

**Symptom**: Users can't see each other across servers

**Solutions**:
1. Check "Burst sent" and "Burst received" in logs
2. Verify `linking.enabled: true` on BOTH servers
3. Restart both servers to re-sync
4. Check network state: Look for "Remote user registered" messages

### Port Already in Use

**Symptom**: "bind: address already in use"

**Solutions**:
1. Different client ports: 6667, 6668, 6669
2. Different link ports: 7000, 7001, 7002
3. Kill old process: `pkill ircd` or `sudo systemctl stop ircd`
4. Check what's using port: `sudo lsof -i :6667`

---

## Quick Reference

### Hub Checklist
- ✅ Unique server name and SID
- ✅ `linking.enabled: true`
- ✅ `linking.host: "0.0.0.0"` (accept connections)
- ✅ `linking.port: 7000` (or your chosen port)
- ✅ Strong `linking.password`
- ✅ Firewall allows link port
- ✅ `links:` section empty or omitted

### Leaf Checklist
- ✅ Unique server name and SID (different from hub!)
- ✅ `linking.enabled: true`
- ✅ `linking.password` matches hub
- ✅ `links:` section configured with:
  - Hub name, SID, host, port
  - Same password as hub
  - `auto_connect: true`
  - `is_hub: true`
- ✅ Firewall allows outbound to hub port

### Success Indicators
```
✅ "Server link established" in logs
✅ "Burst sent" and "Burst received" 
✅ "Network state total_servers=X"
✅ Users on different servers can message each other
✅ NAMES shows users from all servers
```

---

## Example: 3-Server Network

**Network**: irc.example.com with EU and US leafs

### Hub (irc.example.com)
```yaml
server:
  name: "irc.example.com"
  port: 6667
linking:
  enabled: true
  port: 7000
  server_id: "001"
  password: "network_secret_pass"
```

### EU Leaf (eu.example.com)
```yaml
server:
  name: "eu.example.com"
  port: 6667
linking:
  enabled: true
  port: 7001
  server_id: "002"
  password: "network_secret_pass"
  links:
    - name: "irc.example.com"
      sid: "001"
      host: "203.0.113.42"  # Hub IP
      port: 7000
      password: "network_secret_pass"
      auto_connect: true
      is_hub: true
```

### US Leaf (us.example.com)
```yaml
server:
  name: "us.example.com"
  port: 6667
linking:
  enabled: true
  port: 7002
  server_id: "003"
  password: "network_secret_pass"
  links:
    - name: "irc.example.com"
      sid: "001"
      host: "203.0.113.42"  # Hub IP
      port: 7000
      password: "network_secret_pass"
      auto_connect: true
      is_hub: true
```

**Result**: 3 servers, all interconnected. Users can connect to ANY server and communicate with everyone!

---

## Need Help?

- Check logs: `tail -f logs/*.log`
- Verify config: `./ircd -config config/yourconfig.yaml -validate`
- Test manually: `telnet server_ip link_port`
- Review: `CROSS_SERVER_VERIFIED.md` for working examples

**Your distributed IRC network is ready!** 🎉
