#!/bin/bash

echo "=== Full Cross-Server Test ==="
echo ""

# Clean up old test files
rm -f /tmp/alice_full.txt /tmp/bob_full.txt

# Connect Alice to hub
{
    sleep 1
    echo "NICK Alice"
    sleep 1
    echo "USER alice 0 * :Alice Smith"
    sleep 2
    echo "JOIN #test"
    sleep 3
    echo "PRIVMSG #test :Hello from Alice on hub!"
    sleep 2
    echo "NAMES #test"
    sleep 2
    echo "QUIT :Goodbye"
} | telnet 127.0.0.1 6667 > /tmp/alice_full.txt 2>&1 &

# Connect Bob to leaf (join after Alice)
{
    sleep 4
    echo "NICK Bob"
    sleep 1
    echo "USER bob 0 * :Bob Jones"
    sleep 2
    echo "JOIN #test"
    sleep 2
    echo "PRIVMSG #test :Hello from Bob on leaf!"
    sleep 2
    echo "NAMES #test"
    sleep 2
    echo "QUIT :Goodbye"
} | telnet 127.0.0.1 6668 > /tmp/bob_full.txt 2>&1 &

echo "Waiting for test to complete..."
sleep 18

echo ""
echo "=== Alice's View (Hub Server) ==="
grep -E "JOIN|PRIVMSG|353|Alice|Bob" /tmp/alice_full.txt | grep -v "Looking up"

echo ""
echo "=== Bob's View (Leaf Server) ==="
grep -E "JOIN|PRIVMSG|353|Alice|Bob" /tmp/bob_full.txt | grep -v "Looking up"

echo ""
echo "=== Analysis ==="
echo ""

# Check if Alice saw Bob's JOIN
if grep -q "Bob.*JOIN" /tmp/alice_full.txt; then
    echo "✅ Alice saw Bob's JOIN"
else
    echo "❌ Alice didn't see Bob's JOIN"
fi

# Check if Alice saw Bob's message
if grep -q "Bob.*Hello from Bob" /tmp/alice_full.txt; then
    echo "✅ Alice saw Bob's message"
else
    echo "❌ Alice didn't see Bob's message"
fi

# Check if Bob saw Alice's JOIN
if grep -q "Alice.*JOIN" /tmp/bob_full.txt; then
    echo "✅ Bob saw Alice's JOIN"
else
    echo "❌ Bob didn't see Alice's JOIN"
fi

# Check if Bob saw Alice's message
if grep -q "Alice.*Hello from Alice" /tmp/bob_full.txt; then
    echo "✅ Bob saw Alice's message"
else
    echo "❌ Bob didn't see Alice's message"
fi

# Check if Bob's NAMES shows Alice
if grep "353.*#test" /tmp/bob_full.txt | grep -q "Alice"; then
    echo "✅ Bob's NAMES list includes Alice"
else
    echo "❌ Bob's NAMES list doesn't include Alice"
fi

# Check if Alice's NAMES shows Bob
if grep "353.*#test" /tmp/alice_full.txt | grep -q "Bob"; then
    echo "✅ Alice's NAMES list includes Bob"
else
    echo "❌ Alice's NAMES list doesn't include Bob"
fi

echo ""
echo "Full outputs saved to /tmp/alice_full.txt and /tmp/bob_full.txt"
