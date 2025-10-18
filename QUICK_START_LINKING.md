# Quick Start: Server Linking

## Choose Your Role

### 🏢 I Want to Run a HUB Server
(Other people will connect their leaf servers to me)

1. **Edit** `config/config-hub.yaml`:
   ```yaml
   linking:
     enabled: true
     port: 7000              # Server-to-server port
     server_id: "001"        # Hub uses 001
     password: "CHANGE_ME"   # Pick a strong password
   ```

2. **Open firewall**:
   ```bash
   sudo ufw allow 7000/tcp
   ```

3. **Start server**:
   ```bash
   ./ircd -config config/config-hub.yaml
   ```

4. **Share with leaf admins**:
   - Your address: `your.domain.com` or `203.0.113.42`
   - Link port: `7000`
   - Password: `CHANGE_ME`
   - Your SID: `001`
   - Your name: `hub.yournetwork.com`

✅ **Done!** Leaf servers can now connect to you.

📚 **Full guide**: [docs/SERVER_LINKING_SETUP.md#scenario-1-running-a-hub-server](docs/SERVER_LINKING_SETUP.md#scenario-1-running-a-hub-server)

---

### 🍃 I Want to Connect a LEAF to Existing Hub
(Someone else is running the hub)

1. **Get details** from hub admin:
   - Hub address: `hub.example.com`
   - Hub link port: `7000`
   - Link password: `shared_secret`
   - Hub SID: `001`
   - Hub name: `hub.example.com`

2. **Edit** `config/config-leaf.yaml`:
   ```yaml
   linking:
     enabled: true
     server_id: "002"           # NOT 001! Use 002, 003, etc.
     password: "shared_secret"  # Must match hub!
     links:
       - name: "hub.example.com"    # Hub's name
         sid: "001"                  # Hub's SID
         host: "hub.example.com"     # Hub's address
         port: 7000                  # Hub's link port
         password: "shared_secret"   # Same password!
         auto_connect: true
         is_hub: true
   ```

3. **Start server**:
   ```bash
   ./ircd -config config/config-leaf.yaml
   ```

4. **Verify connection** in logs:
   ```
   ✅ "Server link established name=hub.example.com"
   ✅ "Burst sent" and "Burst received"
   ```

✅ **Done!** Your leaf is connected to the network.

📚 **Full guide**: [docs/SERVER_LINKING_SETUP.md#scenario-2-connecting-a-leaf-to-remote-hub](docs/SERVER_LINKING_SETUP.md#scenario-2-connecting-a-leaf-to-remote-hub)

---

## Test It Works

**On hub**, connect a client:
```bash
telnet hub_address 6667
NICK Alice
USER alice 0 * :Alice
JOIN #test
```

**On leaf**, connect another client:
```bash
telnet leaf_address 6667
NICK Bob
USER bob 0 * :Bob
JOIN #test
```

**Expected result**:
- ✅ Alice sees Bob join
- ✅ Bob sees Alice already in channel
- ✅ They can message each other
- ✅ NAMES shows both users

---

## Troubleshooting

### Can't connect?
- Check firewall: `sudo ufw status`
- Test port: `telnet hub_ip 7000`
- Verify hub is running: `netstat -tlnp | grep 7000`

### Wrong password?
- Passwords must match EXACTLY
- Check for typos and trailing spaces
- Both servers need same password

### Users can't see each other?
- Check "Burst sent/received" in logs
- Restart both servers
- Verify linking.enabled is true on BOTH

📚 **Full troubleshooting**: [docs/SERVER_LINKING_SETUP.md#troubleshooting](docs/SERVER_LINKING_SETUP.md#troubleshooting)

---

## Need More Help?

- 📖 **Complete guide**: [docs/SERVER_LINKING_SETUP.md](docs/SERVER_LINKING_SETUP.md)
- ✅ **Verified examples**: [CROSS_SERVER_VERIFIED.md](CROSS_SERVER_VERIFIED.md)
- 🔧 **Config examples**: `config/config-hub.yaml` and `config/config-leaf.yaml`

**Your distributed IRC network is ready!** 🎉
