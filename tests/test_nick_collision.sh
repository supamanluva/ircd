#!/bin/bash
# Test script to verify nickname collision prevention

SERVER="localhost"
PORT="6667"

echo "=== Testing Nickname Collision Prevention ==="
echo ""

# Start two clients in parallel trying to use the same nickname
echo "Starting two clients simultaneously with nickname 'testuser'..."

# Client 1
(
  echo "Client 1: Attempting to register as 'testuser'"
  (
    echo "NICK testuser"
    sleep 0.1
    echo "USER test1 0 * :Test User 1"
    sleep 2
  ) | nc $SERVER $PORT > /tmp/client1.log 2>&1 &
) &

# Client 2 - slightly delayed to create race condition
(
  sleep 0.05
  echo "Client 2: Attempting to register as 'testuser'"
  (
    echo "NICK testuser"
    sleep 0.1
    echo "USER test2 0 * :Test User 2"
    sleep 2
  ) | nc $SERVER $PORT > /tmp/client2.log 2>&1 &
) &

# Wait for both clients to finish
sleep 3

echo ""
echo "=== Client 1 Results ==="
cat /tmp/client1.log

echo ""
echo "=== Client 2 Results ==="
cat /tmp/client2.log

echo ""
echo "=== Analysis ==="
if grep -q "433.*Nickname is already in use" /tmp/client1.log || grep -q "433.*Nickname is already in use" /tmp/client2.log; then
    echo "✓ SUCCESS: Nickname collision was detected and prevented!"
else
    echo "✗ FAILURE: Both clients may have registered with the same nickname"
fi

# Cleanup
rm -f /tmp/client1.log /tmp/client2.log

echo ""
echo "=== Test Complete ==="
