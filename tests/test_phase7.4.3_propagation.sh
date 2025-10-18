#!/bin/bash

# Phase 7.4.3 Test: User State Propagation (JOIN/PART/QUIT/NICK)
# Tests real-time propagation of user state changes across linked servers

set -e

echo "=== Phase 7.4.3: User State Propagation Test ==="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
HUB_PORT=6667
LEAF_PORT=6668
LINK_PORT=7001

# Build the server
echo "Building ircd..."
go build -o /tmp/ircd cmd/ircd/main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}Build successful${NC}"
echo

# Create config files
echo "Creating configuration files..."

# Hub server config
cat > /tmp/hub_config.yaml << EOF
server:
  name: "hub.example.com"
  host: "127.0.0.1"
  port: ${HUB_PORT}

linking:
  enabled: true
  host: "0.0.0.0"
  port: ${LINK_PORT}
  server_id: "001"
  description: "Hub Server"
  links:
    - name: "leaf.example.com"
      host: "127.0.0.1"
      port: 7002
      password: "linkpass"
      auto_connect: false

operators:
  - name: "admin"
    password: "adminpass"
EOF

# Leaf server config
cat > /tmp/leaf_config.yaml << EOF
server:
  name: "leaf.example.com"
  host: "127.0.0.1"
  port: ${LEAF_PORT}

linking:
  enabled: true
  host: "0.0.0.0"
  port: 7002
  server_id: "002"
  description: "Leaf Server"
  links:
    - name: "hub.example.com"
      host: "127.0.0.1"
      port: ${LINK_PORT}
      password: "linkpass"
      auto_connect: false

operators:
  - name: "admin"
    password: "adminpass"
EOF

echo -e "${GREEN}Configuration files created${NC}"
echo

# Clean up any existing processes
pkill -f "/tmp/ircd" || true
sleep 1

# Start servers
echo "Starting hub server..."
/tmp/ircd -config /tmp/hub_config.yaml > /tmp/hub.log 2>&1 &
HUB_PID=$!
sleep 2

echo "Starting leaf server..."
/tmp/ircd -config /tmp/leaf_config.yaml > /tmp/leaf.log 2>&1 &
LEAF_PID=$!
sleep 2

echo -e "${GREEN}Servers started (Hub PID: $HUB_PID, Leaf PID: $LEAF_PID)${NC}"
echo

# Function to cleanup on exit
cleanup() {
    echo
    echo "Cleaning up..."
    kill $HUB_PID $LEAF_PID 2>/dev/null || true
    sleep 1
    pkill -f "/tmp/ircd" || true
    echo "Cleanup complete"
}
trap cleanup EXIT

# Wait for servers to start
echo "Waiting for servers to initialize..."
sleep 3

# Test 1: JOIN propagation
echo -e "${YELLOW}Test 1: JOIN Propagation${NC}"
echo "Connecting Alice to hub..."
{
    sleep 1
    echo "NICK Alice"
    echo "USER alice 0 * :Alice User"
    sleep 1
    echo "JOIN #test"
    sleep 2
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/alice.out 2>&1 &
ALICE_PID=$!

sleep 3

echo "Connecting Bob to leaf..."
{
    sleep 1
    echo "NICK Bob"
    echo "USER bob 0 * :Bob User"
    sleep 1
    echo "JOIN #test"
    sleep 3
    # Bob should see Alice's JOIN if propagated
    echo "NAMES #test"
    sleep 1
    echo "QUIT :Test complete"
} | nc 127.0.0.1 ${LEAF_PORT} > /tmp/bob.out 2>&1 &
BOB_PID=$!

sleep 5

echo "Checking if Alice's JOIN was propagated to Bob..."
if grep -q "Alice" /tmp/bob.out; then
    echo -e "${GREEN}✓ JOIN propagation working${NC}"
else
    echo -e "${RED}✗ JOIN propagation failed${NC}"
    echo "Bob's output:"
    cat /tmp/bob.out
fi
echo

# Test 2: PART propagation
echo -e "${YELLOW}Test 2: PART Propagation${NC}"
echo "Alice PARTs #test..."
{
    sleep 1
    echo "PART #test :Goodbye"
    sleep 2
    echo "QUIT :Done"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/alice_part.out 2>&1 &

sleep 3

echo "Checking hub logs for PART propagation..."
if grep -q "Propagate.*PART" /tmp/hub.log; then
    echo -e "${GREEN}✓ PART propagation attempted${NC}"
else
    echo -e "${YELLOW}⚠ No PART propagation logged${NC}"
fi
echo

# Test 3: NICK propagation
echo -e "${YELLOW}Test 3: NICK Propagation${NC}"
echo "Connecting Charlie to hub..."
{
    sleep 1
    echo "NICK Charlie"
    echo "USER charlie 0 * :Charlie User"
    sleep 1
    echo "JOIN #test"
    sleep 1
    echo "NICK Chuck"
    sleep 2
    echo "QUIT :Done"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/charlie.out 2>&1 &

sleep 5

echo "Checking hub logs for NICK propagation..."
if grep -q "Propagate.*NICK" /tmp/hub.log; then
    echo -e "${GREEN}✓ NICK propagation attempted${NC}"
else
    echo -e "${YELLOW}⚠ No NICK propagation logged${NC}"
fi
echo

# Test 4: QUIT propagation
echo -e "${YELLOW}Test 4: QUIT Propagation${NC}"
echo "Connecting Dave to hub and QUITing..."
{
    sleep 1
    echo "NICK Dave"
    echo "USER dave 0 * :Dave User"
    sleep 1
    echo "JOIN #test"
    sleep 1
    echo "QUIT :Leaving now"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/dave.out 2>&1 &

sleep 4

echo "Checking hub logs for QUIT propagation..."
if grep -q "Propagate.*QUIT" /tmp/hub.log; then
    echo -e "${GREEN}✓ QUIT propagation attempted${NC}"
else
    echo -e "${YELLOW}⚠ No QUIT propagation logged${NC}"
fi
echo

# Check server logs for errors
echo -e "${YELLOW}Checking server logs for errors...${NC}"
if grep -i "error\|panic\|fatal" /tmp/hub.log | grep -v "ERROR :Closing"; then
    echo -e "${RED}Hub server errors found${NC}"
else
    echo -e "${GREEN}✓ No hub server errors${NC}"
fi

if grep -i "error\|panic\|fatal" /tmp/leaf.log | grep -v "ERROR :Closing"; then
    echo -e "${RED}Leaf server errors found${NC}"
else
    echo -e "${GREEN}✓ No leaf server errors${NC}"
fi
echo

# Summary
echo "=== Test Summary ==="
echo "Hub log: /tmp/hub.log"
echo "Leaf log: /tmp/leaf.log"
echo "Client outputs: /tmp/*.out"
echo
echo "Check logs with:"
echo "  grep -i 'propagate' /tmp/hub.log"
echo "  grep -i 'propagate' /tmp/leaf.log"
echo "  grep -i 'join\|part\|quit\|nick' /tmp/leaf.log"
echo
echo -e "${GREEN}Phase 7.4.3 test complete${NC}"
