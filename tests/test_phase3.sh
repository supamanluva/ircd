#!/bin/bash
# Phase 3 Integration Test: Security & Stability
# Tests: Rate limiting, flood protection, timeouts, TLS

set -e

SERVER_HOST="localhost"
SERVER_PORT=6667
TLS_PORT=7000

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Phase 3 Integration Test ===${NC}"
echo "Testing: Rate limiting, flood protection, timeouts, TLS"
echo ""

# Start server
echo -e "${YELLOW}Starting server...${NC}"
pkill -9 ircd 2>/dev/null || true
sleep 1
bin/ircd &
SERVER_PID=$!
sleep 2

# Verify server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}✗ Server failed to start${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Server started (PID: $SERVER_PID)${NC}"

# Test 1: Basic connection and registration
echo -e "${YELLOW}Test 1: Basic connection${NC}"
{
    sleep 0.5
    echo "NICK alice"
    echo "USER alice 0 * :Alice Test"
    sleep 1
    echo "QUIT :Test complete"
} | nc -w 3 $SERVER_HOST $SERVER_PORT > /tmp/test_phase3_basic.out 2>&1 &
sleep 3

if grep -q "001.*alice.*Welcome" /tmp/test_phase3_basic.out; then
    echo -e "${GREEN}✓ Basic connection and registration works${NC}"
else
    echo -e "${RED}✗ Basic connection failed${NC}"
    cat /tmp/test_phase3_basic.out
fi

# Test 2: Rate limiting (flood protection)
echo -e "${YELLOW}Test 2: Rate limiting${NC}"
{
    sleep 0.5
    echo "NICK flood_test"
    echo "USER flood 0 * :Flood Test"
    sleep 1
    # Send 20 messages rapidly (should trigger rate limit)
    for i in {1..20}; do
        echo "PRIVMSG #test :Message $i"
    done
    sleep 1
    echo "QUIT :Flood test done"
} | nc -w 5 $SERVER_HOST $SERVER_PORT > /tmp/test_phase3_flood.out 2>&1 &
sleep 6

# Check if connection was closed due to rate limiting
if grep -q "ERROR.*Closing" /tmp/test_phase3_flood.out || 
   grep -q "Excess Flood" /tmp/test_phase3_flood.out ||
   [ $(wc -l < /tmp/test_phase3_flood.out) -lt 25 ]; then
    echo -e "${GREEN}✓ Rate limiting appears to be working${NC}"
else
    echo -e "${YELLOW}⚠ Rate limiting may need tuning (check logs)${NC}"
fi

# Test 3: Ping/Pong timeout protection
echo -e "${YELLOW}Test 3: Ping/timeout handling${NC}"
{
    sleep 0.5
    echo "NICK pinger"
    echo "USER pinger 0 * :Pinger Test"
    sleep 1
    # Don't send PONG, wait for server to ping us
    sleep 5
} | nc -w 8 $SERVER_HOST $SERVER_PORT > /tmp/test_phase3_ping.out 2>&1 &
sleep 9

if grep -q "PING" /tmp/test_phase3_ping.out; then
    echo -e "${GREEN}✓ Server sends PING messages${NC}"
else
    echo -e "${YELLOW}⚠ No PING received (may need longer wait)${NC}"
fi

# Test 4: Input validation
echo -e "${YELLOW}Test 4: Input validation${NC}"
{
    sleep 0.5
    # Try invalid nickname with special chars
    echo "NICK test!@#$%"
    sleep 0.5
    echo "NICK validnick"
    echo "USER valid 0 * :Valid User"
    sleep 1
    # Try to join invalid channel
    echo "JOIN invalid_channel"
    sleep 0.5
    # Try valid channel
    echo "JOIN #valid"
    sleep 1
    echo "QUIT"
} | nc -w 5 $SERVER_HOST $SERVER_PORT > /tmp/test_phase3_validation.out 2>&1 &
sleep 6

if grep -q "Erroneous Nickname" /tmp/test_phase3_validation.out; then
    echo -e "${GREEN}✓ Input validation rejects invalid nicknames${NC}"
else
    echo -e "${YELLOW}⚠ Input validation may not be rejecting invalid input${NC}"
fi

# Test 5: TLS connection (if enabled)
echo -e "${YELLOW}Test 5: TLS support${NC}"
if [ -f "certs/server.crt" ] && [ -f "certs/server.key" ]; then
    # Try TLS connection
    (echo "NICK tlsuser"; sleep 0.5; echo "USER tls 0 * :TLS Test"; sleep 2; echo "QUIT") | \
        timeout 5 openssl s_client -connect $SERVER_HOST:$TLS_PORT -quiet > /tmp/test_phase3_tls.out 2>&1 || true
    sleep 1
    
    if grep -q "001" /tmp/test_phase3_tls.out || grep -q "Welcome" /tmp/test_phase3_tls.out; then
        echo -e "${GREEN}✓ TLS connection successful${NC}"
    else
        echo -e "${YELLOW}⚠ TLS may not be fully configured (check output below)${NC}"
        cat /tmp/test_phase3_tls.out
    fi
else
    echo -e "${YELLOW}⚠ TLS certificates not found${NC}"
fi

# Cleanup
echo -e "${YELLOW}Cleaning up...${NC}"
kill $SERVER_PID 2>/dev/null || true
sleep 1
pkill -9 ircd 2>/dev/null || true

echo ""
echo -e "${GREEN}=== Phase 3 Testing Complete ===${NC}"
echo "Check server logs for detailed rate limiting and security events"
echo ""
echo "To test TLS manually:"
echo "  openssl s_client -connect localhost:7000"
echo ""
echo "To monitor rate limiting:"
echo "  tail -f logs/ircd.log | grep -i 'rate\\|flood\\|limit'"
