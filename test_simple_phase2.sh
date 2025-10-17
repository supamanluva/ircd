#!/bin/bash
# Simple Phase 2 test

echo "Starting IRC server..."
./bin/ircd > /tmp/ircd_simple.log 2>&1 &
SERVER_PID=$!
sleep 1

echo "Test: Join channel and send message"
(
  sleep 0.2
  echo "NICK alice"
  sleep 0.2
  echo "USER alice 0 * :Alice"
  sleep 0.5
  echo "JOIN #test"
  sleep 0.5
  echo "PRIVMSG #test :Hello world!"
  sleep 0.3
  echo "NAMES #test"
  sleep 0.3
  echo "TOPIC #test :My topic"
  sleep 0.3
  echo "PART #test :Bye"
  sleep 0.3
  echo "QUIT"
) | nc -w 5 localhost 6667

echo
echo "Killing server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo
echo "Server log:"
tail -20 /tmp/ircd_simple.log
