#!/bin/bash

# Phase 7.4 Integration Test: Complete Server Linking and Message Routing
# Tests all features: PRIVMSG/NOTICE routing, user state propagation,
# channel state propagation, and SQUIT handling

# Don't exit on error - we want to count all test results
set +e

echo "=== Phase 7.4: Integration Test ==="
echo "Testing complete server linking and message routing functionality"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
HUB_PORT=6667
LEAF1_PORT=6668
LEAF2_PORT=6669
HUB_LINK_PORT=7000
LEAF1_LINK_PORT=7001
LEAF2_LINK_PORT=7002

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((TESTS_FAILED++))
    fi
}

# Build the server
echo "Building ircd..."
go build -o /tmp/ircd_test cmd/ircd/main.go
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
  port: ${HUB_LINK_PORT}
  server_id: "001"
  description: "Hub Server"
  password: "linkpass"
  links:
    - name: "leaf1.example.com"
      host: "127.0.0.1"
      port: ${LEAF1_LINK_PORT}
      password: "linkpass"
      auto_connect: false
    - name: "leaf2.example.com"
      host: "127.0.0.1"
      port: ${LEAF2_LINK_PORT}
      password: "linkpass"
      auto_connect: false

operators:
  - name: "admin"
    password: "\$2a\$10\$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

# Leaf1 server config
cat > /tmp/leaf1_config.yaml << EOF
server:
  name: "leaf1.example.com"
  host: "127.0.0.1"
  port: ${LEAF1_PORT}

linking:
  enabled: true
  host: "0.0.0.0"
  port: ${LEAF1_LINK_PORT}
  server_id: "002"
  description: "Leaf Server 1"
  links:
    - name: "hub.example.com"
      sid: "001"
      host: "127.0.0.1"
      port: ${HUB_LINK_PORT}
      password: "linkpass"
      auto_connect: true

operators:
  - name: "admin"
    password: "\$2a\$10\$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

# Leaf2 server config
cat > /tmp/leaf2_config.yaml << EOF
server:
  name: "leaf2.example.com"
  host: "127.0.0.1"
  port: ${LEAF2_PORT}

linking:
  enabled: true
  host: "0.0.0.0"
  port: ${LEAF2_LINK_PORT}
  server_id: "003"
  description: "Leaf Server 2"
  links:
    - name: "hub.example.com"
      sid: "001"
      host: "127.0.0.1"
      port: ${HUB_LINK_PORT}
      password: "linkpass"
      auto_connect: true

operators:
  - name: "admin"
    password: "\$2a\$10\$N9qo8uLOickgx2ZMRZoMye.bJuLLLdMJ8gV7R5Y6M8a4mHdHOIxKu"
EOF

echo -e "${GREEN}Configuration files created${NC}"
echo

# Clean up any existing processes
pkill -f "/tmp/ircd_test" || true
sleep 1

# Start servers
echo "Starting servers..."
/tmp/ircd_test -config /tmp/hub_config.yaml > /tmp/hub.log 2>&1 &
HUB_PID=$!
sleep 2

/tmp/ircd_test -config /tmp/leaf1_config.yaml > /tmp/leaf1.log 2>&1 &
LEAF1_PID=$!
sleep 2

/tmp/ircd_test -config /tmp/leaf2_config.yaml > /tmp/leaf2.log 2>&1 &
LEAF2_PID=$!
sleep 2

echo -e "${GREEN}Servers started (Hub: $HUB_PID, Leaf1: $LEAF1_PID, Leaf2: $LEAF2_PID)${NC}"
echo

# Function to cleanup on exit
cleanup() {
    echo
    echo "Cleaning up..."
    kill $HUB_PID $LEAF1_PID $LEAF2_PID 2>/dev/null || true
    sleep 1
    pkill -f "/tmp/ircd_test" || true
    echo "Cleanup complete"
}
trap cleanup EXIT

# Wait for servers to start
echo "Waiting for servers to initialize..."
sleep 3

# Test 1: Server Linking
echo -e "${BLUE}=== Test 1: Server Linking ===${NC}"
echo "Waiting for auto-connect..."
sleep 5

echo "Checking if servers linked..."
if grep -q "Server link established" /tmp/hub.log && grep -q "Server link established" /tmp/leaf1.log; then
    test_result 0 "Hub and Leaf1 linked successfully"
else
    test_result 1 "Hub and Leaf1 failed to link"
fi

if grep -q "Received burst from.*leaf2" /tmp/hub.log || grep -q "Server link established.*leaf2" /tmp/hub.log; then
    test_result 0 "Hub and Leaf2 linked successfully"
else
    test_result 1 "Hub and Leaf2 failed to link"
fi
echo

# Test 2: User Registration and JOIN
echo -e "${BLUE}=== Test 2: User Registration and JOIN Propagation ===${NC}"

echo "Connecting Alice to Hub..."
(
    echo "NICK Alice"
    echo "USER alice 0 * :Alice User"
    sleep 3
    echo "JOIN #test"
    sleep 60
) | nc 127.0.0.1 ${HUB_PORT} > /tmp/alice.out 2>&1 &
ALICE_PID=$!

sleep 2

echo "Connecting Bob to Leaf1..."
(
    echo "NICK Bob"
    echo "USER bob 0 * :Bob User"
    sleep 3
    echo "JOIN #test"
    sleep 60
) | nc 127.0.0.1 ${LEAF1_PORT} > /tmp/bob.out 2>&1 &
BOB_PID=$!

sleep 2

echo "Connecting Charlie to Leaf2..."
(
    echo "NICK Charlie"
    echo "USER charlie 0 * :Charlie User"
    sleep 3
    echo "JOIN #test"
    sleep 60
) | nc 127.0.0.1 ${LEAF2_PORT} > /tmp/charlie.out 2>&1 &
CHARLIE_PID=$!

sleep 10

# Check if JOIN was propagated
# Look for channel creation or users joining in logs
if grep -iq "#test\|JOIN #test" /tmp/hub.log /tmp/leaf1.log /tmp/leaf2.log; then
    test_result 0 "JOIN propagation working"
else
    test_result 1 "JOIN propagation not detected"
fi
echo

# Test 3: PRIVMSG Routing
echo -e "${BLUE}=== Test 3: PRIVMSG/NOTICE Routing ===${NC}"

echo "Alice sends PRIVMSG to Bob (cross-server)..."
{
    sleep 1
    echo "NICK Alice2"
    echo "USER alice2 0 * :Alice2 User"
    sleep 2
    echo "PRIVMSG Bob :Hello from Hub!"
    sleep 3
    echo "QUIT :Done"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/alice2.out 2>&1 &

sleep 5

if grep -iq "RoutePrivmsg\|handleLinkPrivmsg" /tmp/hub.log || grep -iq "RoutePrivmsg\|handleLinkPrivmsg" /tmp/leaf1.log; then
    test_result 0 "PRIVMSG routing attempted"
else
    test_result 1 "PRIVMSG routing not detected"
fi
echo

# Test 4: Channel Message Routing
echo -e "${BLUE}=== Test 4: Channel Message Routing ===${NC}"

echo "Dave on Hub sends message to #test channel..."
{
    sleep 1
    echo "NICK Dave"
    echo "USER dave 0 * :Dave User"
    sleep 2
    echo "JOIN #test"
    sleep 2
    echo "PRIVMSG #test :Hello everyone!"
    sleep 3
    echo "QUIT :Done"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/dave.out 2>&1 &

sleep 6

if grep -iq "RouteChannelMessage\|channel message" /tmp/hub.log; then
    test_result 0 "Channel message routing working"
else
    test_result 1 "Channel message routing not detected"
fi
echo

# Test 5: TOPIC Propagation
echo -e "${BLUE}=== Test 5: TOPIC Propagation ===${NC}"

echo "Eve on Leaf1 sets topic..."
{
    sleep 1
    echo "NICK Eve"
    echo "USER eve 0 * :Eve User"
    sleep 2
    echo "JOIN #test"
    sleep 2
    echo "TOPIC #test :New Topic"
    sleep 3
    echo "QUIT :Done"
} | nc 127.0.0.1 ${LEAF1_PORT} > /tmp/eve.out 2>&1 &

sleep 6

if grep -iq "PropagateTopic\|Delivered remote TOPIC" /tmp/leaf1.log || grep -iq "PropagateTopic\|Delivered remote TOPIC" /tmp/hub.log; then
    test_result 0 "TOPIC propagation working"
else
    test_result 1 "TOPIC propagation not detected"
fi
echo

# Test 6: MODE Propagation
echo -e "${BLUE}=== Test 6: MODE Propagation ===${NC}"

echo "Frank on Hub sets channel mode..."
{
    sleep 1
    echo "NICK Frank"
    echo "USER frank 0 * :Frank User"
    sleep 2
    echo "JOIN #test"
    sleep 2
    echo "MODE #test +m"
    sleep 3
    echo "QUIT :Done"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/frank.out 2>&1 &

sleep 6

if grep -iq "PropagateMode\|Delivered remote MODE" /tmp/hub.log; then
    test_result 0 "MODE propagation working"
else
    test_result 1 "MODE propagation not detected"
fi
echo

# Test 7: NICK Change Propagation
echo -e "${BLUE}=== Test 7: NICK Change Propagation ===${NC}"

echo "Grace on Leaf2 changes nick..."
{
    sleep 1
    echo "NICK Grace"
    echo "USER grace 0 * :Grace User"
    sleep 2
    echo "JOIN #test"
    sleep 2
    echo "NICK Gracie"
    sleep 3
    echo "QUIT :Done"
} | nc 127.0.0.1 ${LEAF2_PORT} > /tmp/grace.out 2>&1 &

sleep 6

if grep -iq "PropagateNick\|Delivered remote NICK" /tmp/leaf2.log || grep -iq "PropagateNick\|Delivered remote NICK" /tmp/hub.log; then
    test_result 0 "NICK propagation working"
else
    test_result 1 "NICK propagation not detected"
fi
echo

# Test 8: QUIT Propagation
echo -e "${BLUE}=== Test 8: QUIT Propagation ===${NC}"

echo "Henry on Hub quits..."
{
    sleep 1
    echo "NICK Henry"
    echo "USER henry 0 * :Henry User"
    sleep 2
    echo "JOIN #test"
    sleep 2
    echo "QUIT :Goodbye!"
} | nc 127.0.0.1 ${HUB_PORT} > /tmp/henry.out 2>&1 &

sleep 5

if grep -iq "PropagateQuit\|Delivered remote QUIT" /tmp/hub.log; then
    test_result 0 "QUIT propagation working"
else
    test_result 1 "QUIT propagation not detected"
fi
echo

# Test 9: Error Handling - Check for crashes
echo -e "${BLUE}=== Test 9: Error Handling ===${NC}"

if ps -p $HUB_PID > /dev/null && ps -p $LEAF1_PID > /dev/null && ps -p $LEAF2_PID > /dev/null; then
    test_result 0 "All servers still running"
else
    test_result 1 "One or more servers crashed"
fi

# Check for panics or fatal errors
if grep -iq "panic\|fatal" /tmp/hub.log /tmp/leaf1.log /tmp/leaf2.log; then
    test_result 1 "Found panic or fatal errors in logs"
else
    test_result 0 "No panics or fatal errors"
fi
echo

# Test 10: Network State Consistency
echo -e "${BLUE}=== Test 10: Network State Consistency ===${NC}"

# Check server counts
HUB_SERVERS=$(grep -c "Server link established" /tmp/hub.log || echo "0")
if [ "$HUB_SERVERS" -ge 2 ]; then
    test_result 0 "Hub has connections to both leaf servers"
else
    test_result 1 "Hub missing expected server connections"
fi

# Check for burst completion
if grep -q "Burst sent" /tmp/hub.log && grep -q "Burst received" /tmp/hub.log; then
    test_result 0 "Burst exchange completed successfully"
else
    test_result 1 "Burst exchange incomplete"
fi
echo

# Summary
echo "=== Test Summary ==="
echo -e "Tests passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests failed: ${RED}${TESTS_FAILED}${NC}"
echo
echo "Log files:"
echo "  Hub:   /tmp/hub.log"
echo "  Leaf1: /tmp/leaf1.log"
echo "  Leaf2: /tmp/leaf2.log"
echo
echo "Client outputs: /tmp/*.out"
echo
echo "Useful debugging commands:"
echo "  grep -i 'error\|warn' /tmp/hub.log"
echo "  grep -i 'propagate' /tmp/*.log"
echo "  grep -i 'link\|burst' /tmp/hub.log"
echo

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}=== ALL TESTS PASSED ===${NC}"
    exit 0
else
    echo -e "${RED}=== SOME TESTS FAILED ===${NC}"
    exit 1
fi
