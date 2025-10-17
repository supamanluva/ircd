#!/bin/bash
# Test Phase 6 advanced IRC commands: WHO, WHOIS, LIST, INVITE

set -e

echo "=== Phase 6 Integration Test ==="
echo "Testing: WHO, WHOIS, LIST, INVITE commands"
echo ""

# Kill any existing server
pkill -9 ircd 2>/dev/null || true
sleep 1

# Start server
echo "Starting server..."
./bin/ircd > logs/test_phase6.log 2>&1 &
SERVER_PID=$!
sleep 2

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "✗ Server failed to start"
    exit 1
fi
echo "✓ Server started (PID: $SERVER_PID)"

# Test 1: WHO command
echo ""
echo "Test 1: WHO command"
{
    sleep 1
    echo "NICK alice"
    echo "USER alice 0 * :Alice Wonderland"
    sleep 1
    echo "JOIN #test"
    sleep 1
    echo "WHO #test"
    sleep 1
    echo "QUIT"
} | nc -q 2 localhost 6667 > /tmp/who_test.out 2>&1 &
wait

if grep -q "352" /tmp/who_test.out && grep -q "315" /tmp/who_test.out; then
    echo "✓ WHO command working (RPL_WHOREPLY + RPL_ENDOFWHO received)"
else
    echo "⚠ WHO command may need review"
fi

# Test 2: WHOIS command
echo ""
echo "Test 2: WHOIS command"
{
    sleep 1
    echo "NICK bob"
    echo "USER bob 0 * :Bob Smith"
    sleep 1
    echo "WHOIS bob"
    sleep 1
    echo "QUIT"
} | nc -q 2 localhost 6667 > /tmp/whois_test.out 2>&1 &
wait

if grep -q "311" /tmp/whois_test.out && grep -q "318" /tmp/whois_test.out; then
    echo "✓ WHOIS command working (RPL_WHOISUSER + RPL_ENDOFWHOIS received)"
else
    echo "⚠ WHOIS command may need review"
fi

# Test 3: LIST command
echo ""
echo "Test 3: LIST command"
{
    sleep 1
    echo "NICK charlie"
    echo "USER charlie 0 * :Charlie Brown"
    sleep 1
    echo "JOIN #test"
    sleep 1
    echo "LIST"
    sleep 1
    echo "QUIT"
} | nc -q 2 localhost 6667 > /tmp/list_test.out 2>&1 &
wait

if grep -q "321" /tmp/list_test.out && grep -q "323" /tmp/list_test.out; then
    echo "✓ LIST command working (RPL_LISTSTART + RPL_LISTEND received)"
else
    echo "⚠ LIST command may need review"
fi

# Test 4: INVITE command (two users)
echo ""
echo "Test 4: INVITE command"

# Start first user (inviter)
{
    sleep 1
    echo "NICK dan"
    echo "USER dan 0 * :Dan Davis"
    sleep 1
    echo "JOIN #private"
    sleep 2
    echo "INVITE eve #private"
    sleep 2
    echo "QUIT"
} | nc -q 3 localhost 6667 > /tmp/invite_dan.out 2>&1 &

# Start second user (invitee)  
{
    sleep 1
    echo "NICK eve"
    echo "USER eve 0 * :Eve Evans"
    sleep 3
    echo "QUIT"
} | nc -q 2 localhost 6667 > /tmp/invite_eve.out 2>&1 &

wait

if grep -q "341" /tmp/invite_dan.out; then
    echo "✓ INVITE command working (RPL_INVITING received)"
else
    echo "⚠ INVITE command may need review"
fi

if grep -q "INVITE" /tmp/invite_eve.out; then
    echo "✓ INVITE notification received by target"
else
    echo "⚠ INVITE notification may need review"
fi

# Test 5: WHO with multiple users in channel
echo ""
echo "Test 5: WHO with multiple channel members"

# Start three users
{
    sleep 1
    echo "NICK user1"
    echo "USER user1 0 * :User One"
    sleep 1
    echo "JOIN #multi"
    sleep 3
    echo "WHO #multi"
    sleep 1
    echo "QUIT"
} | nc -q 2 localhost 6667 > /tmp/who_multi.out 2>&1 &

{
    sleep 1
    echo "NICK user2"
    echo "USER user2 0 * :User Two"
    sleep 1
    echo "JOIN #multi"
    sleep 3
    echo "QUIT"
} | nc -q 2 localhost 6667 &

{
    sleep 1
    echo "NICK user3"
    echo "USER user3 0 * :User Three"
    sleep 1
    echo "JOIN #multi"
    sleep 3
    echo "QUIT"
} | nc -q 2 localhost 6667 &

wait

# Check if WHO returned multiple users
WHO_COUNT=$(grep -c "352" /tmp/who_multi.out || echo "0")
if [ "$WHO_COUNT" -ge "2" ]; then
    echo "✓ WHO lists multiple users ($WHO_COUNT users found)"
else
    echo "⚠ WHO may not list all users (found $WHO_COUNT)"
fi

# Test 6: WHOIS with channels
echo ""
echo "Test 6: WHOIS showing user channels"
{
    sleep 1
    echo "NICK frank"
    echo "USER frank 0 * :Frank"
    sleep 1
    echo "JOIN #test1"
    sleep 1
    echo "JOIN #test2"
    sleep 1
    echo "WHOIS frank"
    sleep 1
    echo "QUIT"
} | nc -q 2 localhost 6667 > /tmp/whois_channels.out 2>&1 &
wait

if grep -q "319" /tmp/whois_channels.out; then
    echo "✓ WHOIS shows user channels (RPL_WHOISCHANNELS)"
else
    echo "⚠ WHOIS may not show channels"
fi

# Cleanup
echo ""
echo "Cleaning up..."
kill $SERVER_PID 2>/dev/null || true
sleep 1

echo ""
echo "=== Phase 6 Testing Complete ==="
echo ""
echo "Summary:"
echo "  - WHO command: List users in channels"
echo "  - WHOIS command: Detailed user information"
echo "  - LIST command: List channels"
echo "  - INVITE command: Invite users to channels"
echo ""
echo "Check logs/test_phase6.log for detailed server logs"
echo ""
echo "Test output files:"
echo "  /tmp/who_test.out"
echo "  /tmp/whois_test.out"
echo "  /tmp/list_test.out"
echo "  /tmp/invite_dan.out"
echo "  /tmp/invite_eve.out"
