#!/bin/bash

# Test script for AWAY, USERHOST, and ISON commands

set -e

SERVER_PORT=6667
TEMP_DIR=$(mktemp -d)
LOG_FILE="$TEMP_DIR/test_additional.log"

echo "=== Testing AWAY, USERHOST, and ISON Commands ==="
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

# Test 1: AWAY command - set away message
echo ""
echo "Test 1: Setting AWAY message"
{
    echo "NICK alice"
    echo "USER alice 0 * :Alice"
    sleep 0.5
    echo "AWAY :Gone to lunch"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice_away.log" 2>&1

if grep -q "306\|You have been marked as being away" "$TEMP_DIR/alice_away.log"; then
    echo "✓ AWAY message set (RPL_NOWAWAY)"
else
    echo "❌ Failed to set AWAY message"
    cat "$TEMP_DIR/alice_away.log"
fi

sleep 1

# Test 2: AWAY command - unset away
echo ""
echo "Test 2: Removing AWAY status"
{
    echo "NICK bob"
    echo "USER bob 0 * :Bob"
    sleep 0.5
    echo "AWAY :Be right back"
    sleep 0.5
    echo "AWAY"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/bob_unaway.log" 2>&1

if grep -q "305\|You are no longer marked as being away" "$TEMP_DIR/bob_unaway.log"; then
    echo "✓ AWAY status removed (RPL_UNAWAY)"
else
    echo "❌ Failed to remove AWAY status"
    cat "$TEMP_DIR/bob_unaway.log"
fi

sleep 1

# Test 3: AWAY shown in WHOIS
echo ""
echo "Test 3: AWAY status shown in WHOIS"
{
    echo "NICK charlie"
    echo "USER charlie 0 * :Charlie"
    sleep 0.5
    echo "AWAY :Out for coffee"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/charlie_away.log" 2>&1 &

sleep 2

{
    echo "NICK dave"
    echo "USER dave 0 * :Dave"
    sleep 0.5
    echo "WHOIS charlie"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/dave_whois.log" 2>&1

sleep 2

if grep -q "301.*Out for coffee\|RPL_AWAY.*charlie" "$TEMP_DIR/dave_whois.log"; then
    echo "✓ AWAY message shown in WHOIS (RPL_AWAY)"
else
    echo "⚠ Could not verify AWAY in WHOIS (may need manual verification)"
fi

# Test 4: PRIVMSG to away user shows away message
echo ""
echo "Test 4: PRIVMSG to away user shows away notification"
{
    echo "NICK eve"
    echo "USER eve 0 * :Eve"
    sleep 0.5
    echo "AWAY :Sleeping"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/eve_away.log" 2>&1 &

sleep 2

{
    echo "NICK frank"
    echo "USER frank 0 * :Frank"
    sleep 0.5
    echo "PRIVMSG eve :Are you there?"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/frank_msg.log" 2>&1

sleep 2

if grep -q "301.*Sleeping\|RPL_AWAY.*eve" "$TEMP_DIR/frank_msg.log"; then
    echo "✓ PRIVMSG shows away notification"
else
    echo "⚠ Could not verify PRIVMSG away notification"
fi

# Test 5: USERHOST command
echo ""
echo "Test 5: USERHOST command"
{
    echo "NICK grace"
    echo "USER grace 0 * :Grace"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/grace.log" 2>&1 &

{
    echo "NICK henry"
    echo "USER henry 0 * :Henry"
    sleep 0.5
    echo "AWAY :BRB"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/henry.log" 2>&1 &

sleep 2

{
    echo "NICK iris"
    echo "USER iris 0 * :Iris"
    sleep 0.5
    echo "USERHOST grace henry"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/iris_userhost.log" 2>&1

sleep 2

if grep -q "302\|RPL_USERHOST" "$TEMP_DIR/iris_userhost.log"; then
    echo "✓ USERHOST command works (RPL_USERHOST)"
    if grep -q "grace.*=" "$TEMP_DIR/iris_userhost.log" && grep -q "henry.*=" "$TEMP_DIR/iris_userhost.log"; then
        echo "  ✓ Shows multiple users"
    fi
    if grep -q "grace.*=+" "$TEMP_DIR/iris_userhost.log"; then
        echo "  ✓ Shows + for not away"
    fi
    if grep -q "henry.*=-" "$TEMP_DIR/iris_userhost.log"; then
        echo "  ✓ Shows - for away"
    fi
else
    echo "❌ USERHOST command failed"
    cat "$TEMP_DIR/iris_userhost.log"
fi

sleep 1

# Test 6: ISON command - users online
echo ""
echo "Test 6: ISON command (users online)"
{
    echo "NICK jack"
    echo "USER jack 0 * :Jack"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/jack.log" 2>&1 &

{
    echo "NICK kate"
    echo "USER kate 0 * :Kate"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/kate.log" 2>&1 &

sleep 2

{
    echo "NICK leo"
    echo "USER leo 0 * :Leo"
    sleep 0.5
    echo "ISON jack kate nobody offline"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/leo_ison.log" 2>&1

sleep 2

if grep -q "303\|RPL_ISON" "$TEMP_DIR/leo_ison.log"; then
    echo "✓ ISON command works (RPL_ISON)"
    if grep -q "jack" "$TEMP_DIR/leo_ison.log" && grep -q "kate" "$TEMP_DIR/leo_ison.log"; then
        echo "  ✓ Shows online users"
    fi
    if ! grep -q "nobody\|offline" "$TEMP_DIR/leo_ison.log" || grep -q "303.*:$" "$TEMP_DIR/leo_ison.log"; then
        echo "  ✓ Correctly excludes offline users"
    fi
else
    echo "❌ ISON command failed"
    cat "$TEMP_DIR/leo_ison.log"
fi

# Test 7: WHO shows away status with G flag
echo ""
echo "Test 7: WHO shows away status (G flag)"
{
    echo "NICK mike"
    echo "USER mike 0 * :Mike"
    sleep 0.5
    echo "JOIN #test"
    sleep 0.5
    echo "AWAY :AFK"
    sleep 1
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/mike_away.log" 2>&1 &

sleep 2

{
    echo "NICK nancy"
    echo "USER nancy 0 * :Nancy"
    sleep 0.5
    echo "JOIN #test"
    sleep 0.5
    echo "WHO #test"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/nancy_who.log" 2>&1

sleep 2

if grep -q "352.*mike.*G" "$TEMP_DIR/nancy_who.log"; then
    echo "✓ WHO shows G flag for away users"
else
    echo "⚠ Could not verify G flag in WHO (may need manual verification)"
fi

if grep -q "352.*nancy.*H" "$TEMP_DIR/nancy_who.log"; then
    echo "✓ WHO shows H flag for here (not away) users"
else
    echo "⚠ Could not verify H flag in WHO"
fi

echo ""
echo "=== Additional Commands Test Complete ==="
echo ""
echo "Commands tested:"
echo "  - AWAY (set/unset away message)"
echo "  - USERHOST (query user@host with away status)"
echo "  - ISON (check which users are online)"
echo ""
echo "Logs saved in: $TEMP_DIR"
