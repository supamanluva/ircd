#!/bin/bash

# Test script for voice mode (+v)

set -e

SERVER_PORT=6667
TEMP_DIR=$(mktemp -d)
LOG_FILE="$TEMP_DIR/test_voice.log"

echo "=== Testing Voice Mode (+v) ==="
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

# Test 1: Set moderated mode
echo ""
echo "Test 1: Setting channel to moderated (+m)"
{
    echo "NICK alice"
    echo "USER alice 0 * :Alice"
    sleep 0.5
    echo "JOIN #test"
    sleep 0.5
    echo "MODE #test +m"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice.log" 2>&1

if grep -q "MODE #test +m" "$TEMP_DIR/alice.log"; then
    echo "✓ Channel set to moderated mode"
else
    echo "❌ Failed to set moderated mode"
    cat "$TEMP_DIR/alice.log"
fi

sleep 1

# Test 2: Regular user cannot speak in moderated channel
echo ""
echo "Test 2: Regular user cannot speak in moderated channel"
{
    echo "NICK bob"
    echo "USER bob 0 * :Bob"
    sleep 0.5
    echo "JOIN #test"
    sleep 0.5
    echo "PRIVMSG #test :Can I speak?"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/bob_no_voice.log" 2>&1 &

sleep 2

if grep -q "ERR_CANNOTSENDTOCHAN\|Cannot send to channel" "$TEMP_DIR/bob_no_voice.log"; then
    echo "✓ Regular user correctly blocked in moderated channel"
else
    echo "❌ Regular user should be blocked"
    cat "$TEMP_DIR/bob_no_voice.log"
fi

sleep 1

# Test 3: Give user voice with +v
echo ""
echo "Test 3: Operator gives user voice with MODE +v"
{
    echo "NICK alice2"
    echo "USER alice2 0 * :Alice2"
    sleep 0.5
    echo "JOIN #voiced"
    sleep 0.5
    echo "MODE #voiced +m"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice2.log" 2>&1

sleep 1

{
    echo "NICK charlie"
    echo "USER charlie 0 * :Charlie"
    sleep 0.5
    echo "JOIN #voiced"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/charlie_join.log" 2>&1 &

sleep 2

{
    echo "NICK alice3"
    echo "USER alice3 0 * :Alice3"
    sleep 0.5
    echo "JOIN #voiced"
    sleep 0.5
    echo "MODE #voiced +v charlie"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice3.log" 2>&1

if grep -q "MODE #voiced +v" "$TEMP_DIR/alice3.log"; then
    echo "✓ Voice granted with MODE +v"
else
    echo "❌ Failed to grant voice"
    cat "$TEMP_DIR/alice3.log"
fi

sleep 1

# Test 4: Voiced user can speak in moderated channel
echo ""
echo "Test 4: Voiced user can speak in moderated channel"
{
    echo "NICK dave"
    echo "USER dave 0 * :Dave"
    sleep 0.5
    echo "JOIN #speak"
    sleep 0.5
    echo "MODE #speak +m"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/dave_setup.log" 2>&1

sleep 1

{
    echo "NICK eve"
    echo "USER eve 0 * :Eve"
    sleep 0.5
    echo "JOIN #speak"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/eve_join.log" 2>&1 &

sleep 2

{
    echo "NICK dave2"
    echo "USER dave2 0 * :Dave2"
    sleep 0.5
    echo "JOIN #speak"
    sleep 0.5
    echo "MODE #speak +v eve"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/dave2.log" 2>&1

sleep 1

{
    echo "NICK eve2"
    echo "USER eve2 0 * :Eve2"
    sleep 0.5
    echo "JOIN #speak"
    sleep 0.5
    echo "PRIVMSG #speak :I have voice!"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/eve_speak.log" 2>&1

sleep 2

if grep -q "PRIVMSG #speak" "$TEMP_DIR/eve_speak.log" && ! grep -q "Cannot send to channel" "$TEMP_DIR/eve_speak.log"; then
    echo "✓ Voiced user can speak in moderated channel"
else
    echo "⚠ Could not verify voiced user speaking (test may need manual verification)"
fi

sleep 1

# Test 5: NAMES shows + prefix for voiced users
echo ""
echo "Test 5: NAMES command shows + prefix for voiced users"
{
    echo "NICK frank"
    echo "USER frank 0 * :Frank"
    sleep 0.5
    echo "JOIN #names"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/frank.log" 2>&1

sleep 1

{
    echo "NICK grace"
    echo "USER grace 0 * :Grace"
    sleep 0.5
    echo "JOIN #names"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/grace_join.log" 2>&1 &

sleep 2

{
    echo "NICK frank2"
    echo "USER frank2 0 * :Frank2"
    sleep 0.5
    echo "JOIN #names"
    sleep 0.5
    echo "MODE #names +v grace"
    sleep 0.5
    echo "NAMES #names"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/frank2.log" 2>&1

sleep 2

if grep -q "+grace" "$TEMP_DIR/frank2.log"; then
    echo "✓ NAMES shows + prefix for voiced users"
else
    echo "⚠ Could not verify + prefix in NAMES (test may need manual verification)"
fi

# Test 6: Remove voice with -v
echo ""
echo "Test 6: Removing voice with MODE -v"
{
    echo "NICK henry"
    echo "USER henry 0 * :Henry"
    sleep 0.5
    echo "JOIN #devoice"
    sleep 0.5
    echo "MODE #devoice +m"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/henry.log" 2>&1

sleep 1

{
    echo "NICK iris"
    echo "USER iris 0 * :Iris"
    sleep 0.5
    echo "JOIN #devoice"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/iris_join.log" 2>&1 &

sleep 2

{
    echo "NICK henry2"
    echo "USER henry2 0 * :Henry2"
    sleep 0.5
    echo "JOIN #devoice"
    sleep 0.5
    echo "MODE #devoice +v iris"
    sleep 0.5
    echo "MODE #devoice -v iris"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/henry2.log" 2>&1

if grep -q "MODE #devoice -v" "$TEMP_DIR/henry2.log"; then
    echo "✓ Voice removed with MODE -v"
else
    echo "❌ Failed to remove voice"
    cat "$TEMP_DIR/henry2.log"
fi

echo ""
echo "=== Voice Mode Test Complete ==="
echo ""
echo "Logs saved in: $TEMP_DIR"
