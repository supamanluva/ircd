# Server Linking: Visual Setup Guide

## 🎯 What Do You Want To Do?

```
┌─────────────────────────────────────────────────────────────┐
│  I want to run the MAIN SERVER (hub)                        │
│  and let other people connect their servers to me           │
│                                                              │
│  → Go to: "HUB SETUP" below                                 │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Someone else is running the main server                    │
│  and I want to connect MY server to their network           │
│                                                              │
│  → Go to: "LEAF SETUP" below                                │
└─────────────────────────────────────────────────────────────┘
```

---

## 🏢 HUB SETUP
### You're running the central server

```
          ┌──────────────┐
          │  YOUR HUB    │ ← You run this
          │  (port 6667) │
          └──────┬───────┘
                 │
         ┌───────┴────────┐
         │                │
    Other people      Other people
    connect their     connect their
    leaf servers      leaf servers
```

### Step 1: Edit `config/config-hub.yaml`

```yaml
server:
  name: "irc.yournetwork.com"    # ← Your server's name
  port: 6667                      # ← Clients connect here

linking:
  enabled: true
  port: 7000                      # ← LEAF SERVERS connect here
  server_id: "001"                # ← Always use "001" for hub
  password: "super_secret_pass"   # ← CHANGE THIS! You'll share this
```

### Step 2: Start Your Hub

```bash
./ircd -config config/config-hub.yaml
```

### Step 3: Share These Details With Leaf Admins

```
┌─────────────────────────────────────────────────┐
│ LEAF ADMINS: Connect your server with:         │
│                                                 │
│ Hub Address:   irc.yournetwork.com              │
│                (or 203.0.113.42)                │
│                                                 │
│ Link Port:     7000                             │
│                                                 │
│ Password:      super_secret_pass                │
│                                                 │
│ Hub SID:       001                              │
│                                                 │
│ Hub Name:      irc.yournetwork.com              │
└─────────────────────────────────────────────────┘
```

### Step 4: Done! ✅

When leaf admins configure their servers with your details, they'll automatically connect!

**Check logs for**: `"Server link established name=<leaf-server-name>"`

---

## 🍃 LEAF SETUP
### You're connecting to someone else's hub

```
         ┌──────────────┐
         │  THEIR HUB   │ ← Someone else runs this
         └──────┬───────┘
                │
         ┌──────┴──────┐
         │             │
    ┌────▼────┐   ┌───▼────┐
    │ YOUR    │   │ Other  │
    │ LEAF    │   │ leafs  │
    └─────────┘   └────────┘
```

### Step 1: Get Hub Connection Details

**Contact the hub admin** and ask for:

```
┌─────────────────────────────────────────────────┐
│ HUB ADMIN: I need these details to connect:    │
│                                                 │
│ [ ] Hub address (IP or domain)                 │
│ [ ] Link port                                  │
│ [ ] Link password                              │
│ [ ] Hub server ID (SID)                        │
│ [ ] Hub server name                            │
└─────────────────────────────────────────────────┘
```

### Step 2: Edit `config/config-leaf.yaml`

Replace the example values with what the hub admin gave you:

```yaml
server:
  name: "leaf.yourname.com"       # ← YOUR server's name (NOT the hub's!)
  port: 6667                       # ← YOUR client port

linking:
  enabled: true
  server_id: "002"                 # ← NOT "001"! Use 002, 003, etc.
  password: "super_secret_pass"    # ← Must match hub's password
  
  links:
    - name: "irc.yournetwork.com"  # ← Hub's server name
      sid: "001"                    # ← Hub's server ID
      host: "203.0.113.42"          # ← Hub's address
      port: 7000                    # ← Hub's link port
      password: "super_secret_pass" # ← Hub's password (same!)
      auto_connect: true
      is_hub: true
```

### Step 3: Start Your Leaf

```bash
./ircd -config config/config-leaf.yaml
```

### Step 4: Verify Connection ✅

Check your logs for:

```
✅ "Auto-connecting to name=irc.yournetwork.com"
✅ "Connected, starting handshake"
✅ "Server link established name=irc.yournetwork.com"
✅ "Burst sent" and "Burst received"
```

If you see these, **you're connected!** 🎉

---

## 🧪 Test It Works

### On the HUB, connect a client:
```bash
telnet hub.address.com 6667
NICK Alice
USER alice 0 * :Alice
JOIN #test
PRIVMSG #test :Hello from hub!
```

### On YOUR LEAF, connect a client:
```bash
telnet your.server.com 6667
NICK Bob
USER bob 0 * :Bob
JOIN #test
PRIVMSG #test :Hello from leaf!
```

### Expected Results:
```
✅ Alice sees: "Bob JOIN #test"
✅ Alice sees: "Hello from leaf!"
✅ Bob sees: "Alice" in NAMES list
✅ Bob sees: "Hello from hub!"
```

**If you see all of these → SUCCESS! Users can talk across servers!** 🎊

---

## 🔧 Common Issues

### "Connection refused"
```
❌ Problem: Can't connect to hub
✅ Fix: 
   1. Check hub is running: ssh to hub, run: netstat -tlnp | grep 7000
   2. Check firewall: sudo ufw allow 7000/tcp
   3. Verify hub address is correct
```

### "Invalid password"
```
❌ Problem: Passwords don't match
✅ Fix:
   1. Check for typos
   2. Both servers need EXACT same password
   3. Check for trailing spaces in YAML
   4. Leaf needs password in TWO places:
      - linking.password
      - linking.links[0].password
```

### "Server ID conflict"
```
❌ Problem: Two servers with same SID
✅ Fix:
   1. Hub should use "001"
   2. Leaf should use "002", "003", etc.
   3. Each server needs UNIQUE SID
```

### Users can't see each other
```
❌ Problem: Connection works but users invisible
✅ Fix:
   1. Check logs for "Burst sent" and "Burst received"
   2. Verify linking.enabled: true on BOTH servers
   3. Restart both servers
```

---

## 📚 More Help

**Quick Start**: [QUICK_START_LINKING.md](QUICK_START_LINKING.md)  
**Complete Guide**: [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md)  
**Working Examples**: [CROSS_SERVER_VERIFIED.md](CROSS_SERVER_VERIFIED.md)

---

## Decision Tree

```
START: Do you own/control the main hub server?
  │
  ├─ YES → Follow "HUB SETUP" above
  │         │
  │         └─ Share connection details with others
  │
  └─ NO  → Follow "LEAF SETUP" above
            │
            └─ Get connection details from hub admin
```

**Both setups take ~5 minutes!** ⏱️
