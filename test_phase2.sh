#!/bin/bash
# Integration test for IRC server Phase 2 - Channels & Messaging

echo "=== IRC Server Phase 2 Integration Test ==="
echo

# Start server in background
./bin/ircd > /tmp/ircd_phase2.log 2>&1 &
SERVER_PID=$!
echo "Started server with PID $SERVER_PID"
sleep 1

# Test 1: Channel JOIN
echo "Test 1: JOIN channel"
{
  echo "NICK alice"
  echo "USER alice 0 * :Alice"
  sleep 0.3
  echo "JOIN #test"
  sleep 0.3
} | nc -q 1 localhost 6667 > /tmp/test_join.txt 2>&1

echo "Response:"
cat /tmp/test_join.txt | grep -E "(JOIN|TOPIC|NAMES|353|366)"
echo

# Test 2: Multiple users in channel
echo "Test 2: Multiple users and channel messaging"
{
  echo "NICK bob"
  echo "USER bob 0 * :Bob"
  sleep 0.3
  echo "JOIN #test"
  sleep 0.2
  echo "PRIVMSG #test :Hello everyone!"
  sleep 0.3
} | nc -q 1 localhost 6667 > /tmp/test_multi.txt 2>&1 &

sleep 0.5

{
  echo "NICK carol"
  echo "USER carol 0 * :Carol"
  sleep 0.3
  echo "JOIN #test"
  sleep 0.5
} | nc -q 1 localhost 6667 > /tmp/test_carol.txt 2>&1

wait

echo "Bob's view:"
cat /tmp/test_multi.txt | grep -E "(JOIN|PRIVMSG|353)"
echo
echo "Carol's view:"
cat /tmp/test_carol.txt | grep -E "(JOIN|PRIVMSG|bob|353)"
echo

# Test 3: Private messaging
echo "Test 3: Private messaging between users"
{
  echo "NICK dave"
  echo "USER dave 0 * :Dave"
  sleep 0.3
  echo "PRIVMSG eve :Hello Eve!"
  sleep 0.2
} | nc -q 1 localhost 6667 > /tmp/test_dave.txt 2>&1 &

sleep 0.2

{
  echo "NICK eve"
  echo "USER eve 0 * :Eve"
  sleep 0.5
} | nc -q 1 localhost 6667 > /tmp/test_eve.txt 2>&1

wait

echo "Eve received:"
cat /tmp/test_eve.txt | grep PRIVMSG
echo

# Test 4: TOPIC
echo "Test 4: TOPIC command"
{
  echo "NICK frank"
  echo "USER frank 0 * :Frank"
  sleep 0.3
  echo "JOIN #topic-test"
  sleep 0.2
  echo "TOPIC #topic-test :This is the topic"
  sleep 0.2
  echo "TOPIC #topic-test"
  sleep 0.2
} | nc -q 1 localhost 6667 > /tmp/test_topic.txt 2>&1

echo "Response:"
cat /tmp/test_topic.txt | grep TOPIC
echo

# Test 5: PART
echo "Test 5: PART channel"
{
  echo "NICK grace"
  echo "USER grace 0 * :Grace"
  sleep 0.3
  echo "JOIN #part-test"
  sleep 0.2
  echo "PART #part-test :Goodbye!"
  sleep 0.2
} | nc -q 1 localhost 6667 > /tmp/test_part.txt 2>&1

echo "Response:"
cat /tmp/test_part.txt | grep -E "(JOIN|PART)"
echo

# Test 6: Invalid channel name
echo "Test 6: Invalid channel name"
{
  echo "NICK henry"
  echo "USER henry 0 * :Henry"
  sleep 0.3
  echo "JOIN invalid"
  sleep 0.2
} | nc -q 1 localhost 6667 > /tmp/test_invalid.txt 2>&1

echo "Response:"
cat /tmp/test_invalid.txt | grep -E "(403|No such channel)"
echo

# Stop server
echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo
echo "=== Phase 2 Tests Complete ==="
echo "Server log saved to /tmp/ircd_phase2.log"
