#!/bin/bash

# Quick manual test to verify message propagation works
# This test demonstrates that users on different servers can see each other

set -e

echo "=== Quick Propagation Test ==="
echo "This will start 2 servers and verify cross-server visibility"
echo

# Build
echo "Building..."
go build -o /tmp/ircd_quick cmd/ircd/main.go

# Create configs
cat > /tmp/hub_quick.yaml << 'EOF'
server:
  name: "hub.local"
  host: "127.0.0.1"
  port: 16667

linking:
  enabled: true
  host: "127.0.0.1"
  port: 17000
  server_id: "001"
  password: "test123"
  description: "Hub Server"

operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

cat > /tmp/leaf_quick.yaml << 'EOF'
server:
  name: "leaf.local"
  host: "127.0.0.1"
  port: 16668

linking:
  enabled: true
  host: "127.0.0.1"
  port: 17001
  server_id: "002"
  password: "test123"
  description: "Leaf Server"
  links:
    - name: "hub.local"
      sid: "001"
      host: "127.0.0.1"
      port: 17000
      password: "test123"
      auto_connect: true

operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

# Start servers
echo "Starting hub server..."
/tmp/ircd_quick -config /tmp/hub_quick.yaml > /tmp/hub_quick.log 2>&1 &
HUB_PID=$!
sleep 2

echo "Starting leaf server..."
/tmp/ircd_quick -config /tmp/leaf_quick.yaml > /tmp/leaf_quick.log 2>&1 &
LEAF_PID=$!
sleep 3

# Cleanup function
cleanup() {
    echo
    echo "Cleaning up..."
    kill $HUB_PID $LEAF_PID 2>/dev/null || true
    sleep 1
}
trap cleanup EXIT

# Check if servers linked
echo "Checking if servers linked..."
if grep -q "Server link established" /tmp/hub_quick.log && grep -q "Burst sent" /tmp/hub_quick.log; then
    echo "✓ Servers linked successfully!"
else
    echo "✗ Servers failed to link"
    echo "Hub log:"
    tail -20 /tmp/hub_quick.log
    echo "Leaf log:"
    tail -20 /tmp/leaf_quick.log
    exit 1
fi

echo
echo "=== Manual Test Instructions ==="
echo
echo "Servers are running:"
echo "  Hub:  127.0.0.1:16667"
echo "  Leaf: 127.0.0.1:16668"
echo
echo "In Terminal 1, connect to HUB:"
echo "  telnet 127.0.0.1 16667"
echo "  NICK Alice"
echo "  USER alice 0 * :Alice"
echo "  JOIN #test"
echo "  PRIVMSG #test :Hello from hub!"
echo
echo "In Terminal 2, connect to LEAF:"
echo "  telnet 127.0.0.1 16668"
echo "  NICK Bob"
echo "  USER bob 0 * :Bob"
echo "  JOIN #test"
echo
echo "Expected behavior:"
echo "  - Bob should see Alice's JOIN message"
echo "  - Bob should see Alice's message in #test"
echo "  - Alice should see Bob's JOIN message"
echo "  - Messages sent by either should appear on both servers"
echo
echo "Press Ctrl+C when done testing..."

# Keep running
wait
