#!/bin/bash
# Test multi-user chat

echo "Starting IRC server..."
./bin/ircd > /tmp/ircd_multi.log 2>&1 &
SERVER_PID=$!
sleep 1

echo "Starting Alice..."
(
  echo "NICK alice"
  sleep 0.1
  echo "USER alice 0 * :Alice"
  sleep 1
  echo "JOIN #chat"
  sleep 2
  echo "PRIVMSG #chat :Hi Bob!"
  sleep 2
  echo "QUIT"
) | nc -w 10 localhost 6667 > /tmp/alice.log 2>&1 &
ALICE_PID=$!

sleep 1.5

echo "Starting Bob..."
(
  echo "NICK bob"
  sleep 0.1
  echo "USER bob 0 * :Bob"
  sleep 1
  echo "JOIN #chat"
  sleep 1
  echo "PRIVMSG #chat :Hi Alice!"
  sleep 2
  echo "QUIT"
) | nc -w 10 localhost 6667 > /tmp/bob.log 2>&1 &
BOB_PID=$!

# Wait for clients
wait $ALICE_PID $BOB_PID

echo
echo "=== Alice's view ==="
cat /tmp/alice.log | tail -15

echo
echo "=== Bob's view ==="
cat /tmp/bob.log | tail -15

echo
echo "=== Server log ==="
tail -20 /tmp/ircd_multi.log

# Stop server
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
