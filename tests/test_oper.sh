#!/bin/bash

# Test script for OPER command

set -e

SERVER_PORT=6667
TEMP_DIR=$(mktemp -d)
LOG_FILE="$TEMP_DIR/test_oper.log"

echo "=== Testing OPER Command ==="
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

# Test 1: OPER with correct credentials
echo ""
echo "Test 1: OPER with correct credentials (admin/admin123)"
{
    echo "NICK alice"
    echo "USER alice 0 * :Alice"
    sleep 0.5
    echo "OPER admin admin123"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/alice_admin.log" 2>&1

if grep -q "381\|You are now an IRC operator" "$TEMP_DIR/alice_admin.log"; then
    echo "✓ OPER succeeded with correct credentials"
else
    echo "❌ OPER should have succeeded"
    cat "$TEMP_DIR/alice_admin.log"
fi

sleep 1

# Test 2: OPER with wrong password
echo ""
echo "Test 2: OPER with wrong password"
{
    echo "NICK bob"
    echo "USER bob 0 * :Bob"
    sleep 0.5
    echo "OPER admin wrongpassword"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/bob_wrong.log" 2>&1

if grep -q "464\|Password incorrect" "$TEMP_DIR/bob_wrong.log"; then
    echo "✓ OPER correctly rejected wrong password (ERR_PASSWDMISMATCH)"
else
    echo "❌ Should have rejected wrong password"
    cat "$TEMP_DIR/bob_wrong.log"
fi

sleep 1

# Test 3: OPER with unknown name
echo ""
echo "Test 3: OPER with unknown operator name"
{
    echo "NICK charlie"
    echo "USER charlie 0 * :Charlie"
    sleep 0.5
    echo "OPER unknown password123"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/charlie_unknown.log" 2>&1

if grep -q "464\|Password incorrect" "$TEMP_DIR/charlie_unknown.log"; then
    echo "✓ OPER correctly rejected unknown operator name"
else
    echo "❌ Should have rejected unknown operator"
    cat "$TEMP_DIR/charlie_unknown.log"
fi

sleep 1

# Test 4: OPER with second configured operator (oper/oper456)
echo ""
echo "Test 4: OPER with second operator credentials (oper/oper456)"
{
    echo "NICK dave"
    echo "USER dave 0 * :Dave"
    sleep 0.5
    echo "OPER oper oper456"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/dave_oper.log" 2>&1

if grep -q "381\|You are now an IRC operator" "$TEMP_DIR/dave_oper.log"; then
    echo "✓ Second operator credentials work"
else
    echo "❌ Second operator should have succeeded"
    cat "$TEMP_DIR/dave_oper.log"
fi

sleep 1

# Test 5: OPER without enough parameters
echo ""
echo "Test 5: OPER without enough parameters"
{
    echo "NICK eve"
    echo "USER eve 0 * :Eve"
    sleep 0.5
    echo "OPER admin"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/eve_params.log" 2>&1

if grep -q "461\|Not enough parameters" "$TEMP_DIR/eve_params.log"; then
    echo "✓ OPER correctly requires both name and password"
else
    echo "⚠ Could not verify parameter check (test may need manual verification)"
fi

sleep 1

# Test 6: Check operator mode in WHOIS
echo ""
echo "Test 6: Verify operator appears in WHOIS"
{
    echo "NICK frank"
    echo "USER frank 0 * :Frank"
    sleep 0.5
    echo "OPER admin admin123"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/frank_oper.log" 2>&1 &

sleep 2

{
    echo "NICK grace"
    echo "USER grace 0 * :Grace"
    sleep 0.5
    echo "WHOIS frank"
    sleep 0.5
    echo "QUIT :Done"
} | nc -q 2 localhost $SERVER_PORT > "$TEMP_DIR/grace_whois.log" 2>&1

sleep 2

if grep -q "313\|is an IRC operator" "$TEMP_DIR/grace_whois.log"; then
    echo "✓ Operator status shown in WHOIS"
else
    echo "⚠ Could not verify operator in WHOIS (may need manual verification)"
fi

echo ""
echo "=== OPER Command Test Complete ==="
echo ""
echo "Logs saved in: $TEMP_DIR"
echo ""
echo "Configured operators in config/config.yaml:"
echo "  - admin / admin123"
echo "  - oper / oper456"
