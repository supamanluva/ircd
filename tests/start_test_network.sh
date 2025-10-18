#!/bin/bash

# Real client propagation test
# This script starts a 2-server network for manual testing with IRC clients

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== IRC Server Network - Real Client Test ===${NC}"
echo

# Check if servers are already running
if pgrep -f "ircd.*hub_real" > /dev/null; then
    echo -e "${YELLOW}Servers already running. Stopping them first...${NC}"
    pkill -f "ircd.*hub_real" || true
    pkill -f "ircd.*leaf_real" || true
    sleep 2
fi

# Build
echo "Building server..."
go build -o /tmp/ircd_real cmd/ircd/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Build successful${NC}"
echo

# Create configs
echo "Creating configurations..."

cat > /tmp/hub_real.yaml << 'EOF'
server:
  name: "hub.example.net"
  host: "0.0.0.0"
  port: 6667

linking:
  enabled: true
  host: "0.0.0.0"
  port: 7000
  server_id: "001"
  password: "linkpass"
  description: "Hub Server"

operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

cat > /tmp/leaf_real.yaml << 'EOF'
server:
  name: "leaf.example.net"
  host: "0.0.0.0"
  port: 6668

linking:
  enabled: true
  host: "0.0.0.0"
  port: 7001
  server_id: "002"
  password: "linkpass"
  description: "Leaf Server"
  links:
    - name: "hub.example.net"
      sid: "001"
      host: "127.0.0.1"
      port: 7000
      password: "linkpass"
      auto_connect: true

operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

echo -e "${GREEN}✓ Configurations created${NC}"
echo

# Start servers
echo "Starting Hub server (port 6667)..."
/tmp/ircd_real -config /tmp/hub_real.yaml > /tmp/hub_real.log 2>&1 &
HUB_PID=$!
sleep 2

echo "Starting Leaf server (port 6668)..."
/tmp/ircd_real -config /tmp/leaf_real.yaml > /tmp/leaf_real.log 2>&1 &
LEAF_PID=$!
sleep 3

# Check if servers are running
if ! ps -p $HUB_PID > /dev/null; then
    echo -e "${RED}✗ Hub server failed to start${NC}"
    cat /tmp/hub_real.log
    exit 1
fi

if ! ps -p $LEAF_PID > /dev/null; then
    echo -e "${RED}✗ Leaf server failed to start${NC}"
    cat /tmp/leaf_real.log
    exit 1
fi

echo -e "${GREEN}✓ Both servers started${NC}"
echo

# Wait for linking
echo "Waiting for servers to link..."
sleep 2

# Check if linked
if grep -q "Server link established" /tmp/hub_real.log && grep -q "Burst sent" /tmp/hub_real.log; then
    echo -e "${GREEN}✓ Servers linked successfully!${NC}"
else
    echo -e "${YELLOW}⚠ Servers may not be linked yet. Check logs if issues occur.${NC}"
fi

echo
echo -e "${GREEN}=== Servers are ready! ===${NC}"
echo
echo "Hub Server:  127.0.0.1:6667 (PID: $HUB_PID)"
echo "Leaf Server: 127.0.0.1:6668 (PID: $LEAF_PID)"
echo
echo "Logs:"
echo "  Hub:  /tmp/hub_real.log"
echo "  Leaf: /tmp/leaf_real.log"
echo
echo -e "${BLUE}=== Connection Instructions ===${NC}"
echo
echo "Option 1: Using irssi (recommended)"
echo "  Terminal 1: irssi -c 127.0.0.1 -p 6667 -n Alice"
echo "  Terminal 2: irssi -c 127.0.0.1 -p 6668 -n Bob"
echo
echo "Option 2: Using weechat"
echo "  Terminal 1: weechat -r '/server add hub 127.0.0.1/6667 -autoconnect; /connect hub; /nick Alice'"
echo "  Terminal 2: weechat -r '/server add leaf 127.0.0.1/6668 -autoconnect; /connect leaf; /nick Bob'"
echo
echo "Option 3: Using telnet (manual typing)"
echo "  Terminal 1: telnet 127.0.0.1 6667"
echo "    Then type: NICK Alice"
echo "               USER alice 0 * :Alice User"
echo "               JOIN #test"
echo
echo "  Terminal 2: telnet 127.0.0.1 6668"
echo "    Then type: NICK Bob"
echo "               USER bob 0 * :Bob User"
echo "               JOIN #test"
echo
echo -e "${BLUE}=== Test Procedure ===${NC}"
echo
echo "1. Connect both clients to their respective servers"
echo "2. Both join the same channel: /join #test"
echo "3. Send a message from Alice: /msg #test Hello from hub!"
echo "4. Send a message from Bob: /msg #test Hello from leaf!"
echo
echo -e "${GREEN}Expected Result:${NC}"
echo "  - Both users should see each other's JOIN messages"
echo "  - Both users should see all messages in #test"
echo "  - This proves cross-server propagation works!"
echo
echo -e "${YELLOW}=== Press Ctrl+C to stop servers ===${NC}"
echo
echo "To view logs in real-time:"
echo "  tail -f /tmp/hub_real.log"
echo "  tail -f /tmp/leaf_real.log"
echo

# Save PIDs to file for cleanup script
echo "$HUB_PID" > /tmp/ircd_hub.pid
echo "$LEAF_PID" > /tmp/ircd_leaf.pid

# Wait for user to stop
trap 'echo; echo "Stopping servers..."; kill $HUB_PID $LEAF_PID 2>/dev/null; rm -f /tmp/ircd_hub.pid /tmp/ircd_leaf.pid; echo "Servers stopped."; exit 0' INT TERM

# Keep script running
wait
