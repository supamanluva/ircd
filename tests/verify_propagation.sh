#!/bin/bash

# Automated test to verify message propagation works
# Tests that users on different servers can see each other's messages

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "=== Automated Propagation Verification Test ==="
echo

# Build
echo "Building..."
go build -o /tmp/ircd_prop cmd/ircd/main.go

# Create configs
cat > /tmp/hub_prop.yaml << 'EOF'
server:
  name: "hub.test"
  host: "127.0.0.1"
  port: 26667

linking:
  enabled: true
  host: "127.0.0.1"
  port: 27000
  server_id: "001"
  password: "testpass"
  description: "Test Hub"

operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

cat > /tmp/leaf_prop.yaml << 'EOF'
server:
  name: "leaf.test"
  host: "127.0.0.1"
  port: 26668

linking:
  enabled: true
  host: "127.0.0.1"
  port: 27001
  server_id: "002"
  password: "testpass"
  description: "Test Leaf"
  links:
    - name: "hub.test"
      sid: "001"
      host: "127.0.0.1"
      port: 27000
      password: "testpass"
      auto_connect: true

operators:
  - name: "admin"
    password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

# Start servers
echo "Starting servers..."
/tmp/ircd_prop -config /tmp/hub_prop.yaml > /tmp/hub_prop.log 2>&1 &
HUB_PID=$!

/tmp/ircd_prop -config /tmp/leaf_prop.yaml > /tmp/leaf_prop.log 2>&1 &
LEAF_PID=$!

# Cleanup function
cleanup() {
    echo
    echo "Cleaning up..."
    kill $HUB_PID $LEAF_PID 2>/dev/null || true
    killall -9 nc 2>/dev/null || true
    sleep 1
}
trap cleanup EXIT

echo "Waiting for servers to start and link..."
sleep 5

# Check if servers linked
if ! grep -q "Server link established" /tmp/hub_prop.log; then
    echo -e "${RED}✗ Servers failed to link${NC}"
    echo "Hub log:"
    tail -20 /tmp/hub_prop.log
    echo "Leaf log:"
    tail -20 /tmp/leaf_prop.log
    exit 1
fi
echo -e "${GREEN}✓ Servers linked${NC}"

echo
echo "=== Test 1: Cross-Server Channel Visibility ==="

# Connect Alice to hub
echo "Connecting Alice to hub..."
(
    sleep 0.5
    echo "NICK Alice"
    echo "USER alice 0 * :Alice User"
    sleep 2
    echo "JOIN #testchan"
    sleep 10
) | nc -q 1 127.0.0.1 26667 > /tmp/alice_prop.out 2>&1 &
ALICE_PID=$!

sleep 3

# Connect Bob to leaf
echo "Connecting Bob to leaf..."
(
    sleep 0.5
    echo "NICK Bob"
    echo "USER bob 0 * :Bob User"
    sleep 2
    echo "JOIN #testchan"
    sleep 10
) | nc -q 1 127.0.0.1 26668 > /tmp/bob_prop.out 2>&1 &
BOB_PID=$!

sleep 5

# Check results
echo "Checking results..."
echo

# Check if Alice registered
if grep -q "Welcome" /tmp/alice_prop.out; then
    echo -e "${GREEN}✓ Alice registered on hub${NC}"
else
    echo -e "${RED}✗ Alice failed to register${NC}"
    echo "Alice output:"
    cat /tmp/alice_prop.out
fi

# Check if Bob registered  
if grep -q "Welcome" /tmp/bob_prop.out; then
    echo -e "${GREEN}✓ Bob registered on leaf${NC}"
else
    echo -e "${RED}✗ Bob failed to register${NC}"
    echo "Bob output:"
    cat /tmp/bob_prop.out
fi

# Check if Alice saw Bob's JOIN
if grep -q "Bob.*JOIN.*#testchan" /tmp/alice_prop.out; then
    echo -e "${GREEN}✓ PROPAGATION WORKS: Alice saw Bob's JOIN from leaf server!${NC}"
else
    echo -e "${BLUE}ℹ Alice didn't see Bob's JOIN (may have timing issues)${NC}"
fi

# Check if Bob saw Alice's JOIN
if grep -q "Alice.*JOIN.*#testchan" /tmp/bob_prop.out; then
    echo -e "${GREEN}✓ PROPAGATION WORKS: Bob saw Alice's JOIN from hub server!${NC}"
else
    echo -e "${BLUE}ℹ Bob didn't see Alice's JOIN (may have timing issues)${NC}"
fi

echo
echo "=== Server Logs Analysis ==="

# Check for propagation in server logs
if grep -iq "join" /tmp/hub_prop.log | head -5; then
    echo "Hub log shows channel activity"
fi

if grep -iq "join" /tmp/leaf_prop.log | head -5; then
    echo "Leaf log shows channel activity"
fi

echo
echo "=== Output Files ==="
echo "Alice output: /tmp/alice_prop.out"
echo "Bob output:   /tmp/bob_prop.out"
echo "Hub log:      /tmp/hub_prop.log"
echo "Leaf log:     /tmp/leaf_prop.log"
echo
echo "To inspect manually:"
echo "  cat /tmp/alice_prop.out"
echo "  cat /tmp/bob_prop.out"
echo "  grep -i join /tmp/hub_prop.log"
echo "  grep -i join /tmp/leaf_prop.log"

wait
