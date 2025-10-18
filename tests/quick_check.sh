#!/bin/bash

# Quick demonstration using simple echo commands
# This shows what SHOULD happen when propagation works

echo "=== Quick Propagation Check ==="
echo

# Connect Alice to hub
echo "Connecting Alice to hub..."
{
    sleep 0.5
    echo "NICK Alice"
    echo "USER alice 0 * :Alice Test"
    sleep 4
    echo "JOIN #test" 
    sleep 2
    echo "PRIVMSG #test :Hello from hub!"
    sleep 2
} | nc -q 1 127.0.0.1 6667 > /tmp/quick_alice.txt 2>&1 &

sleep 2

# Connect Bob to leaf
echo "Connecting Bob to leaf..."
{
    sleep 0.5
    echo "NICK Bob"
    echo "USER bob 0 * :Bob Test"
    sleep 4
    echo "JOIN #test"
    sleep 2
    echo "PRIVMSG #test :Hello from leaf!"
    sleep 2
} | nc -q 1 127.0.0.1 6668 > /tmp/quick_bob.txt 2>&1 &

sleep 12

echo
echo "=== Alice's output ==="
cat /tmp/quick_alice.txt | grep -E "Welcome|JOIN|PRIVMSG|Bob"

echo  
echo "=== Bob's output ==="
cat /tmp/quick_bob.txt | grep -E "Welcome|JOIN|PRIVMSG|Alice"

echo
echo "=== Hub log ==="
grep -i "alice\|bob\|propagat\|remote user\|uid" /tmp/hub_real.log | tail -15

echo
echo "=== Leaf log ==="
grep -i "alice\|bob\|propagat\|remote user\|uid" /tmp/leaf_real.log | tail -15
