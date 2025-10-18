# Remote Server Deployment Checklist

## 🎯 Testing Distributed IRC on Remote Servers

This checklist will help you deploy and test your IRC network across real servers.

---

## 📋 Pre-Deployment Checklist

### Server Requirements
- [ ] Two servers available (one for hub, one for leaf)
- [ ] SSH access to both servers
- [ ] Servers can reach each other over network
- [ ] Go 1.21+ installed on both (or Docker)
- [ ] Ports available: 6667 (clients), 7000 (server links)

### Network Information to Collect
- [ ] Hub server IP/domain: `__________________`
- [ ] Leaf server IP/domain: `__________________`
- [ ] Choose link password: `__________________`

---

## 🏢 Hub Server Setup

### 1. Deploy Code
```bash
# SSH into hub server
ssh user@hub-server

# Clone repository
git clone https://github.com/supamanluva/ircd.git
cd ircd

# Build
go build -o ircd cmd/ircd/main.go
# OR use the binary you already built
```

### 2. Configure Hub
```bash
# Edit config
cp config/config-hub.yaml config/production-hub.yaml
nano config/production-hub.yaml
```

**Update these values**:
```yaml
server:
  name: "irc.yourdomain.com"    # Your hub's public name
  host: "0.0.0.0"                # Listen on all interfaces
  port: 6667

linking:
  enabled: true
  host: "0.0.0.0"                # Listen for server connections
  port: 7000
  server_id: "001"               # Hub uses 001
  password: "YOUR_SECURE_PASSWORD"  # CHANGE THIS!
```

### 3. Configure Firewall
```bash
# Allow IRC client connections
sudo ufw allow 6667/tcp

# Allow server-to-server connections (CRITICAL!)
sudo ufw allow 7000/tcp

# Enable firewall if not already
sudo ufw enable
sudo ufw status
```

### 4. Start Hub Server
```bash
# Test run first
./ircd -config config/production-hub.yaml

# If it works, run in background with tmux or screen
tmux new -s ircd
./ircd -config config/production-hub.yaml

# Detach with Ctrl+B, then D
# Reattach later with: tmux attach -t ircd
```

### 5. Verify Hub is Running
```bash
# Check it's listening
netstat -tlnp | grep 6667
netstat -tlnp | grep 7000

# Test client connection locally
telnet localhost 6667
# Type: QUIT
```

### 6. Share Connection Details
Copy this info to send to your leaf server:

```
═══════════════════════════════════════════════
HUB CONNECTION DETAILS

Hub Address:   <hub-server-ip-or-domain>
Link Port:     7000
Link Password: <your-password>
Hub SID:       001
Hub Name:      irc.yourdomain.com
═══════════════════════════════════════════════
```

---

## 🍃 Leaf Server Setup

### 1. Deploy Code
```bash
# SSH into leaf server
ssh user@leaf-server

# Clone repository
git clone https://github.com/supamanluva/ircd.git
cd ircd

# Build
go build -o ircd cmd/ircd/main.go
```

### 2. Configure Leaf
```bash
# Edit config
cp config/config-leaf.yaml config/production-leaf.yaml
nano config/production-leaf.yaml
```

**Update these values** (using info from hub admin):
```yaml
server:
  name: "leaf.yourdomain.com"   # Different from hub!
  host: "0.0.0.0"
  port: 6667

linking:
  enabled: true
  server_id: "002"               # NOT 001! Use 002, 003, etc.
  password: "YOUR_SECURE_PASSWORD"  # Must match hub!
  
  links:
    - name: "irc.yourdomain.com"       # Hub's server name
      sid: "001"                        # Hub's server ID
      host: "<hub-ip-or-domain>"        # Hub's address
      port: 7000                        # Hub's link port
      password: "YOUR_SECURE_PASSWORD"  # Hub's password
      auto_connect: true
      is_hub: true
```

### 3. Configure Firewall
```bash
# Allow IRC client connections
sudo ufw allow 6667/tcp

# Enable firewall
sudo ufw enable
sudo ufw status
```

### 4. Start Leaf Server
```bash
# Test run first
./ircd -config config/production-leaf.yaml

# Check logs for connection
# Should see:
# "Auto-connecting to name=irc.yourdomain.com"
# "Server link established"
# "Burst sent" and "Burst received"
```

### 5. If Connection Succeeds
```bash
# Run in background with tmux
tmux new -s ircd
./ircd -config config/production-leaf.yaml

# Detach with Ctrl+B, then D
```

---

## ✅ Verification Tests

### Test 1: Check Server Link
**On hub server**:
```bash
# Check logs
tail -f logs/hub.log | grep -i "link\|burst"

# Should see:
# "Server link established name=leaf.yourdomain.com"
# "Burst received" from leaf
```

**On leaf server**:
```bash
# Check logs
tail -f logs/leaf.log | grep -i "link\|burst"

# Should see:
# "Server link established name=irc.yourdomain.com"
# "Burst sent" to hub
```

### Test 2: Cross-Server User Communication
**From your computer, connect to HUB**:
```bash
telnet <hub-server-ip> 6667

# Register
NICK Alice
USER alice 0 * :Alice Smith

# Join channel
JOIN #test

# Send message
PRIVMSG #test :Hello from the hub!

# Check who's here
NAMES #test
```

**From another terminal, connect to LEAF**:
```bash
telnet <leaf-server-ip> 6667

# Register
NICK Bob
USER bob 0 * :Bob Jones

# Join same channel
JOIN #test

# Send message
PRIVMSG #test :Hello from the leaf!

# Check who's here
NAMES #test
```

### Expected Results ✅
- [ ] Alice sees Bob's JOIN message
- [ ] Bob sees Alice in NAMES list
- [ ] Alice sees Bob's message "Hello from the leaf!"
- [ ] Bob sees Alice's message "Hello from the hub!"
- [ ] Both NAMES lists show both users

---

## 🐛 Troubleshooting

### Leaf Can't Connect to Hub

**Check 1: Hub is reachable**
```bash
# From leaf server
telnet <hub-ip> 7000
# Should connect. If not, firewall issue.
```

**Check 2: Hub firewall**
```bash
# On hub server
sudo ufw status
# Should show: 7000/tcp ALLOW
```

**Check 3: Passwords match**
```bash
# On both servers, check config
grep -A2 "linking:" config/*.yaml | grep password
# Must be IDENTICAL
```

### Users Can't See Each Other

**Check 1: Link is established**
```bash
# Check logs on both servers
grep "Server link established" logs/*.log
grep "Burst sent\|Burst received" logs/*.log
```

**Check 2: Network state**
```bash
# Look for user propagation in logs
grep "Propagated new user\|Remote user registered" logs/*.log
```

### Server Keeps Disconnecting

**Check 1: Ping/Pong**
```bash
# Check logs for ping failures
grep -i "ping\|pong\|timeout" logs/*.log
```

**Check 2: Network stability**
```bash
# Test connection stability
ping <hub-ip>
mtr <hub-ip>
```

---

## 📊 Monitoring

### Useful Log Monitoring Commands

**On hub**:
```bash
# Watch all activity
tail -f logs/hub.log

# Watch just server linking
tail -f logs/hub.log | grep -i "link\|server\|burst"

# Watch user activity
tail -f logs/hub.log | grep -i "client\|user\|join\|part"
```

**On leaf**:
```bash
# Same commands
tail -f logs/leaf.log
tail -f logs/leaf.log | grep -i "link\|server\|burst"
tail -f logs/leaf.log | grep -i "client\|user\|join\|part"
```

---

## 🔒 Security Checklist

- [ ] Changed default passwords
- [ ] Using strong link password (20+ chars)
- [ ] Firewall configured (only necessary ports open)
- [ ] Servers are patched and up to date
- [ ] SSH key authentication (no password auth)
- [ ] Consider TLS for client connections
- [ ] Monitor logs for unusual activity

---

## 📝 Notes

Keep track of your deployment:

**Hub Server**:
- IP/Domain: `__________________`
- Started at: `__________________`
- Config file: `__________________`

**Leaf Server**:
- IP/Domain: `__________________`
- Started at: `__________________`
- Config file: `__________________`

**Link Password**: `__________________` (store securely!)

**Issues Encountered**:
```
__________________________________________________________
__________________________________________________________
__________________________________________________________
```

**Solutions Applied**:
```
__________________________________________________________
__________________________________________________________
__________________________________________________________
```

---

## ✨ Success Criteria

Your distributed IRC network is working when:

✅ Both servers start without errors
✅ Logs show "Server link established"
✅ Logs show "Burst sent" and "Burst received"
✅ Users on hub can see users on leaf
✅ Users on leaf can see users on hub
✅ Messages are delivered across servers
✅ NAMES lists show all users
✅ No errors in logs during normal operation

---

## 🎉 Next Steps After Successful Test

1. **Document your setup**
   - Save your configs
   - Document any issues you found
   - Note performance observations

2. **Consider productionizing**
   - Set up systemd service (see `docs/DEPLOYMENT.md`)
   - Configure log rotation
   - Set up monitoring/alerts
   - Plan backup strategy

3. **Scale the network**
   - Add more leaf servers
   - Test with more users
   - Monitor resource usage

4. **Share your experience**
   - Did the documentation help?
   - What could be clearer?
   - Any issues found?

---

**Good luck with your deployment!** 🚀

If you encounter issues, check:
- [VISUAL_SETUP_GUIDE.md](../VISUAL_SETUP_GUIDE.md)
- [docs/SERVER_LINKING_SETUP.md](../docs/SERVER_LINKING_SETUP.md)
- [QUICK_START_LINKING.md](../QUICK_START_LINKING.md)
