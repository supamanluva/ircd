#!/bin/bash

# Simple demonstration of cross-server message propagation
# Uses telnet with manual interaction

echo "=== IRC Cross-Server Propagation Demo ==="
echo
echo "Servers are running:"
echo "  Hub:  127.0.0.1:6667"
echo "  Leaf: 127.0.0.1:6668"
echo
echo "Servers are linked: $(grep -c 'Server link established' /tmp/hub_real.log) connection(s)"
echo
echo "=== Automated Test ===" 
echo

# Test with automated clients
echo "Connecting Alice to Hub (6667)..."
(
    sleep 1
    echo "NICK Alice"
    sleep 0.5
    echo "USER alice 0 * :Alice User"
    sleep 3
    echo "JOIN #demo"
    sleep 2
    echo "PRIVMSG #demo :Hello from the Hub server!"
    sleep 3
    echo "QUIT :Test complete"
) | telnet 127.0.0.1 6667 > /tmp/alice_demo.txt 2>&1 &
ALICE_PID=$!

sleep 4

echo "Connecting Bob to Leaf (6668)..."
(
    sleep 1
    echo "NICK Bob"
    sleep 0.5
    echo "USER bob 0 * :Bob User"  
    sleep 3
    echo "JOIN #demo"
    sleep 2
    echo "PRIVMSG #demo :Hello from the Leaf server!"
    sleep 3
    echo "NAMES #demo"
    sleep 2
    echo "QUIT :Test complete"
) | telnet 127.0.0.1 6668 > /tmp/bob_demo.txt 2>&1 &
BOB_PID=$!

echo "Waiting for test to complete..."
sleep 15

echo
echo "=== Results ==="
echo

# Check Alice's output
echo "--- Alice's perspective (on Hub) ---"
if grep -q "001.*Welcome" /tmp/alice_demo.txt; then
    echo "✓ Alice registered successfully"
else
    echo "✗ Alice failed to register"
fi

if grep -q "JOIN.*#demo" /tmp/alice_demo.txt | head -1; then
    echo "✓ Alice joined #demo"
fi

if grep -i "bob.*JOIN" /tmp/alice_demo.txt; then
    echo "✓ Alice SAW Bob's JOIN from leaf server!"
    grep -i "bob.*JOIN" /tmp/alice_demo.txt | head -1
else
    echo "✗ Alice didn't see Bob's JOIN"
fi

if grep -q "Bob.*PRIVMSG.*#demo.*Hello from the Leaf" /tmp/alice_demo.txt; then
    echo "✓ Alice SAW Bob's message from leaf server!"
    grep "Bob.*PRIVMSG.*#demo" /tmp/alice_demo.txt | head -1
else
    echo "✗ Alice didn't see Bob's message"
fi

echo
echo "--- Bob's perspective (on Leaf) ---"
if grep -q "001.*Welcome" /tmp/bob_demo.txt; then
    echo "✓ Bob registered successfully"
else
    echo "✗ Bob failed to register"
fi

if grep -q "JOIN.*#demo" /tmp/bob_demo.txt | head -1; then
    echo "✓ Bob joined #demo"
fi

if grep -i "alice.*JOIN" /tmp/bob_demo.txt; then
    echo "✓ Bob SAW Alice's JOIN from hub server!"
    grep -i "alice.*JOIN" /tmp/bob_demo.txt | head -1
else
    echo "✗ Bob didn't see Alice's JOIN"
fi

if grep -q "Alice.*PRIVMSG.*#demo.*Hello from the Hub" /tmp/bob_demo.txt; then
    echo "✓ Bob SAW Alice's message from hub server!"
    grep "Alice.*PRIVMSG.*#demo" /tmp/bob_demo.txt | head -1
else
    echo "✗ Bob didn't see Alice's message"
fi

if grep -q "353.*#demo.*Alice.*Bob" /tmp/bob_demo.txt || grep -q "353.*#demo.*Bob.*Alice" /tmp/bob_demo.txt; then
    echo "✓ NAMES shows both users in channel!"
    grep "353.*#demo" /tmp/bob_demo.txt
else
    echo "ℹ NAMES output:"
    grep -i "353\|#demo" /tmp/bob_demo.txt || echo "  (no NAMES reply found)"
fi

echo
echo "=== Server Logs ===" 
echo
echo "Messages in hub log:"
grep -i "alice\|bob\|#demo\|join\|privmsg" /tmp/hub_real.log | tail -10

echo
echo "Messages in leaf log:"
grep -i "alice\|bob\|#demo\|join\|privmsg" /tmp/leaf_real.log | tail -10

echo
echo "=== Full client outputs available at: ==="
echo "  Alice: /tmp/alice_demo.txt"
echo "  Bob:   /tmp/bob_demo.txt"
echo
echo "To view:"
echo "  cat /tmp/alice_demo.txt"
echo "  cat /tmp/bob_demo.txt"
