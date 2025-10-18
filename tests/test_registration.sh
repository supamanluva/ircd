#!/bin/bash

# Test user registration and UID propagation

echo "=== Testing User Registration and UID Propagation ==="

# Connect Alice to hub
echo "Connecting Alice to hub..."
{
    sleep 1
    echo "NICK Alice"
    sleep 1
    echo "USER alice 0 * :Alice Smith"
    sleep 3
    echo "QUIT :Done testing"
} | telnet 127.0.0.1 6667 > /tmp/alice_reg.txt 2>&1 &

sleep 6

echo ""
echo "=== Hub Log (Alice registration) ==="
grep -i "alice\|assigned uid\|propagat\|attempting to add" /tmp/hub_real.log | tail -20

echo ""
echo "=== Leaf Log (UID propagation) ==="
grep -i "alice\|remote user\|uid" /tmp/leaf_real.log | tail -20

echo ""
echo "=== Alice's Output ==="
cat /tmp/alice_reg.txt

echo ""
echo "Done!"
