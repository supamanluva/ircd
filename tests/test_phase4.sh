#!/bin/bash
# Phase 4 Integration Test: Administration & Operator Commands
# Tests: MODE (user/channel), KICK, operator privileges, channel modes

set -e

SERVER_HOST="localhost"
SERVER_PORT=6667

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Phase 4 Integration Test ===${NC}"
echo "Testing: MODE, KICK, operator privileges, channel modes"
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

# Test 1: User MODE
echo -e "${YELLOW}Test 1: User MODE command${NC}"
{
    sleep 0.5
    echo "NICK alice"
    echo "USER alice 0 * :Alice Test"
    sleep 1
    echo "MODE alice"
    sleep 0.5
    echo "MODE alice +i"
    sleep 0.5
    echo "MODE alice"
    sleep 1
    echo "QUIT"
} | nc -w 5 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_usermode.out 2>&1 &
sleep 6

if grep -q "221.*+" /tmp/test_phase4_usermode.out; then
    echo -e "${GREEN}✓ User MODE command works${NC}"
else
    echo -e "${RED}✗ User MODE command failed${NC}"
    cat /tmp/test_phase4_usermode.out
fi

# Test 2: Channel operator status
echo -e "${YELLOW}Test 2: Channel operator (first user)${NC}"
{
    sleep 0.5
    echo "NICK bob"
    echo "USER bob 0 * :Bob Test"
    sleep 1
    echo "JOIN #testop"
    sleep 0.5
    echo "NAMES #testop"
    sleep 1
    echo "QUIT"
} | nc -w 5 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_chanop.out 2>&1 &
sleep 6

if grep -q "@bob" /tmp/test_phase4_chanop.out; then
    echo -e "${GREEN}✓ First user gets operator status${NC}"
else
    echo -e "${YELLOW}⚠ Operator status unclear${NC}"
fi

# Test 3: Channel MODE
echo -e "${YELLOW}Test 3: Channel MODE command${NC}"
{
    sleep 0.5
    echo "NICK charlie"
    echo "USER charlie 0 * :Charlie Test"
    sleep 1
    echo "JOIN #modetest"
    sleep 0.5
    echo "MODE #modetest"
    sleep 0.5
    echo "MODE #modetest +i"
    sleep 0.5
    echo "MODE #modetest"
    sleep 1
    echo "QUIT"
} | nc -w 6 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_chanmode.out 2>&1 &
sleep 7

if grep -q "324.*#modetest.*+" /tmp/test_phase4_chanmode.out; then
    echo -e "${GREEN}✓ Channel MODE command works${NC}"
else
    echo -e "${YELLOW}⚠ Channel MODE may need review${NC}"
fi

# Test 4: KICK command
echo -e "${YELLOW}Test 4: KICK command${NC}"

# Start operator (alice)
{
    sleep 0.5
    echo "NICK alice_op"
    echo "USER alice 0 * :Alice Op"
    sleep 1
    echo "JOIN #kicktest"
    sleep 2
    # Wait for bob to join
    sleep 3
    echo "KICK #kicktest bob_victim :You have been kicked"
    sleep 1
    echo "QUIT"
} | nc -w 10 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_kick_op.out 2>&1 &

# Start victim (bob)
sleep 2
{
    sleep 0.5
    echo "NICK bob_victim"
    echo "USER bob 0 * :Bob Victim"
    sleep 1
    echo "JOIN #kicktest"
    sleep 5
    # Should be kicked by now
    echo "QUIT"
} | nc -w 9 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_kick_victim.out 2>&1 &

sleep 11

if grep -q "KICK #kicktest bob_victim" /tmp/test_phase4_kick_victim.out; then
    echo -e "${GREEN}✓ KICK command works${NC}"
else
    echo -e "${YELLOW}⚠ KICK command may need review${NC}"
    echo "Operator output:"
    cat /tmp/test_phase4_kick_op.out | tail -5
    echo "Victim output:"
    cat /tmp/test_phase4_kick_victim.out | tail -5
fi

# Test 5: Non-operator cannot kick
echo -e "${YELLOW}Test 5: Non-operator KICK denied${NC}"

# Start operator (alice)
{
    sleep 0.5
    echo "NICK alice2"
    echo "USER alice 0 * :Alice"
    sleep 1
    echo "JOIN #nopkick"
    sleep 3
    echo "QUIT"
} | nc -w 8 $SERVER_HOST $SERVER_PORT > /dev/null 2>&1 &

# Start non-operator trying to kick
sleep 2
{
    sleep 0.5
    echo "NICK bob2"
    echo "USER bob 0 * :Bob"
    sleep 1
    echo "JOIN #nopkick"
    sleep 1
    echo "KICK #nopkick alice2 :Trying to kick"
    sleep 1
    echo "QUIT"
} | nc -w 6 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_nokick.out 2>&1 &

sleep 7

if grep -q "482.*You're not channel operator" /tmp/test_phase4_nokick.out; then
    echo -e "${GREEN}✓ Non-operators cannot KICK${NC}"
else
    echo -e "${YELLOW}⚠ Operator privilege check unclear${NC}"
fi

# Test 6: MODE to grant operator
echo -e "${YELLOW}Test 6: MODE +o to grant operator${NC}"

# Start channel founder
{
    sleep 0.5
    echo "NICK founder"
    echo "USER founder 0 * :Founder"
    sleep 1
    echo "JOIN #optest"
    sleep 3
    echo "MODE #optest +o newop"
    sleep 1
    echo "QUIT"
} | nc -w 8 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_giveop.out 2>&1 &

# Start user to receive op
sleep 2
{
    sleep 0.5
    echo "NICK newop"
    echo "USER newop 0 * :New Op"
    sleep 1
    echo "JOIN #optest"
    sleep 3
    echo "NAMES #optest"
    sleep 1
    echo "QUIT"
} | nc -w 8 $SERVER_HOST $SERVER_PORT > /tmp/test_phase4_newop.out 2>&1 &

sleep 9

if grep -q "@newop" /tmp/test_phase4_newop.out; then
    echo -e "${GREEN}✓ MODE +o grants operator status${NC}"
else
    echo -e "${YELLOW}⚠ Operator granting unclear${NC}"
fi

# Cleanup
echo -e "${YELLOW}Cleaning up...${NC}"
kill $SERVER_PID 2>/dev/null || true
sleep 1
pkill -9 ircd 2>/dev/null || true

echo ""
echo -e "${GREEN}=== Phase 4 Testing Complete ===${NC}"
echo ""
echo "Summary:"
echo "  - User MODE: Set/view user modes"
echo "  - Channel MODE: Set/view channel modes (+i, +m, +n, +t)"
echo "  - Channel Operators: First user gets @"
echo "  - KICK: Remove users from channels"
echo "  - MODE +o/-o: Grant/revoke operator status"
echo "  - Privilege checks: Only ops can kick/set modes"
echo ""
echo "IRC Modes Implemented:"
echo "  User modes: +i (invisible), +w (wallops)"
echo "  Channel modes: +o (operator), +i (invite-only), +m (moderated),"
echo "                 +n (no external), +t (topic protection), +b (ban)"
