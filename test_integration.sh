#!/bin/bash
# Integration test for IRC server Phase 1

echo "=== IRC Server Phase 1 Integration Test ==="
echo

# Start server in background
./bin/ircd > /tmp/ircd_test.log 2>&1 &
SERVER_PID=$!
echo "Started server with PID $SERVER_PID"
sleep 1

# Test 1: Registration flow
echo "Test 1: Client Registration Flow"
{
  echo "NICK alice"
  echo "USER alice 0 * :Alice Wonderland"
  sleep 0.3
  echo "PING :test123"
  sleep 0.3
} | nc -q 1 localhost 6667 > /tmp/test1_output.txt 2>&1

echo "Server response:"
cat /tmp/test1_output.txt
echo

# Test 2: Invalid nickname
echo "Test 2: Invalid Nickname"
{
  echo "NICK 123invalid"
  sleep 0.2
} | nc -q 1 localhost 6667 > /tmp/test2_output.txt 2>&1

echo "Server response:"
cat /tmp/test2_output.txt
echo

# Test 3: QUIT command
echo "Test 3: QUIT Command"
{
  echo "NICK bob"
  echo "USER bob 0 * :Bob Smith"
  sleep 0.2
  echo "QUIT :Leaving now"
  sleep 0.2
} | nc -q 1 localhost 6667 > /tmp/test3_output.txt 2>&1

echo "Server response:"
cat /tmp/test3_output.txt
echo

# Stop server
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo
echo "=== Tests Complete ==="
echo "Server log saved to /tmp/ircd_test.log"
