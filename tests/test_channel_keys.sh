#!/bin/bash

# Test script for channel keys (+k mode)

set -e

SERVER_PORT=6667
TEMP_DIR=$(mktemp -d)
LOG_FILE="$TEMP_DIR/test_keys.log"

echo "=== Testing Channel Keys (+k mode) ==="
echo "Test directory: $TEMP_DIR"

# Start the server in the background
echo "Starting IRC server..."
./bin/ircd > "$TEMP_DIR/server.log" 2>&1 &
SERVER_PID=$!
sleep 2

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    cat "$TEMP_DIR/server.log"
    exit 1
fi

echo "✓ Server started (PID: $SERVER_PID)"

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 1
    pkill -9 ircd 2>/dev/null || true
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Test 1: Create channel and set key
echo ""
echo "Test 1: Setting channel key with MODE +k"
{
    echo "NICK alice"
    echo "USER alice 0 * :Alice"
    sleep 0.5
    echo "JOIN #secret"
    sleep 0.5
    echo "MODE #secret +k mypassword"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice.log" 2>&1

if grep -q "MODE #secret +k" "$TEMP_DIR/alice.log"; then
    echo "✓ Channel key set successfully"
else
    echo "❌ Failed to set channel key"
    cat "$TEMP_DIR/alice.log"
fi

sleep 1

# Test 2: Join with correct key
echo ""
echo "Test 2: Joining channel with correct key"
{
    echo "NICK bob"
    echo "USER bob 0 * :Bob"
    sleep 0.5
    echo "JOIN #secret mypassword"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/bob_good.log" 2>&1 &

sleep 2

if grep -q "JOIN #secret" "$TEMP_DIR/bob_good.log"; then
    echo "✓ Joined channel with correct key"
else
    echo "❌ Failed to join with correct key"
    cat "$TEMP_DIR/bob_good.log"
fi

sleep 1

# Test 3: Join with wrong key
echo ""
echo "Test 3: Joining channel with wrong key"
{
    echo "NICK charlie"
    echo "USER charlie 0 * :Charlie"
    sleep 0.5
    echo "JOIN #secret wrongpassword"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/charlie.log" 2>&1 &

sleep 2

if grep -q "475" "$TEMP_DIR/charlie.log" || grep -q "Cannot join channel" "$TEMP_DIR/charlie.log"; then
    echo "✓ Correctly rejected wrong key (ERR_BADCHANNELKEY)"
else
    echo "❌ Should have rejected wrong key"
    cat "$TEMP_DIR/charlie.log"
fi

sleep 1

# Test 4: Join without key
echo ""
echo "Test 4: Joining channel without providing key"
{
    echo "NICK dave"
    echo "USER dave 0 * :Dave"
    sleep 0.5
    echo "JOIN #secret"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/dave.log" 2>&1 &

sleep 2

if grep -q "475" "$TEMP_DIR/dave.log" || grep -q "Cannot join channel" "$TEMP_DIR/dave.log"; then
    echo "✓ Correctly rejected join without key"
else
    echo "❌ Should have rejected join without key"
    cat "$TEMP_DIR/dave.log"
fi

sleep 1

# Test 5: Remove key with MODE -k
echo ""
echo "Test 5: Removing channel key with MODE -k"
{
    echo "NICK alice2"
    echo "USER alice2 0 * :Alice2"
    sleep 0.5
    echo "JOIN #public"
    sleep 0.5
    echo "MODE #public +k testkey"
    sleep 0.5
    echo "MODE #public -k"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice2.log" 2>&1

if grep -q "MODE #public -k" "$TEMP_DIR/alice2.log"; then
    echo "✓ Channel key removed successfully"
else
    echo "❌ Failed to remove channel key"
    cat "$TEMP_DIR/alice2.log"
fi

sleep 1

# Test 6: Join after key removed
echo ""
echo "Test 6: Joining channel after key removed"
{
    echo "NICK eve"
    echo "USER eve 0 * :Eve"
    sleep 0.5
    echo "JOIN #public"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/eve.log" 2>&1 &

sleep 2

if grep -q "JOIN #public" "$TEMP_DIR/eve.log"; then
    echo "✓ Joined channel after key removed"
else
    echo "❌ Failed to join after key removed"
    cat "$TEMP_DIR/eve.log"
fi

echo ""
echo "=== Channel Keys Test Complete ==="
echo ""
echo "Logs saved in: $TEMP_DIR"
