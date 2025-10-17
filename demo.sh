#!/bin/bash
# Comprehensive demo of IRC server capabilities

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║        IRC Server - Phase 1 & 2 Complete Demonstration       ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo

# Start server
echo "🚀 Starting IRC server..."
./bin/ircd > /tmp/ircd_demo.log 2>&1 &
SERVER_PID=$!
sleep 1

echo "✅ Server started (PID: $SERVER_PID)"
echo

# Demo scenario
echo "📋 Demo Scenario:"
echo "   - Alice connects and creates #general channel"
echo "   - Bob joins and they chat"
echo "   - Alice sets a topic"
echo "   - Carol joins the conversation"
echo "   - Bob leaves"
echo "   - Everyone quits gracefully"
echo
echo "Press Enter to start demo..."
read

echo "════════════════════════════════════════════════════════════════"
echo "👤 Alice: Connecting..."
(
  echo "NICK alice"
  sleep 0.1
  echo "USER alice 0 * :Alice Wonderland"
  sleep 0.8
  echo "JOIN #general"
  sleep 2
  echo "TOPIC #general :Welcome to the general channel!"
  sleep 2
  echo "PRIVMSG #general :Hi everyone! This is a test of our IRC server."
  sleep 3
  echo "PRIVMSG #general :Bob, how are you?"
  sleep 3
  echo "PRIVMSG #general :Welcome Carol!"
  sleep 2
  echo "QUIT :Time to go!"
) | nc -w 20 localhost 6667 > /tmp/alice_demo.log 2>&1 &
ALICE_PID=$!

sleep 2

echo "👤 Bob: Connecting..."
(
  echo "NICK bob"
  sleep 0.1
  echo "USER bob 0 * :Bob Smith"
  sleep 0.8
  echo "JOIN #general"
  sleep 1
  echo "PRIVMSG #general :Hey Alice! I'm doing great, thanks for asking!"
  sleep 3
  echo "PRIVMSG #general :Hi Carol, welcome!"
  sleep 2
  echo "PART #general :Gotta run, see you later!"
  sleep 1
  echo "QUIT"
) | nc -w 20 localhost 6667 > /tmp/bob_demo.log 2>&1 &
BOB_PID=$!

sleep 3

echo "👤 Carol: Connecting..."
(
  echo "NICK carol"
  sleep 0.1
  echo "USER carol 0 * :Carol Johnson"
  sleep 0.8
  echo "JOIN #general"
  sleep 2
  echo "PRIVMSG #general :Hello everyone! Thanks for the warm welcome!"
  sleep 2
  echo "NAMES #general"
  sleep 1
  echo "QUIT :Bye all!"
) | nc -w 15 localhost 6667 > /tmp/carol_demo.log 2>&1 &
CAROL_PID=$!

# Wait for all clients
wait $ALICE_PID $BOB_PID $CAROL_PID

echo
echo "════════════════════════════════════════════════════════════════"
echo "📊 Results:"
echo "════════════════════════════════════════════════════════════════"
echo

echo "📝 Alice's perspective:"
echo "────────────────────────────────────────────────────────────────"
cat /tmp/alice_demo.log | tail -20
echo

echo "📝 Bob's perspective:"
echo "────────────────────────────────────────────────────────────────"
cat /tmp/bob_demo.log | tail -20
echo

echo "📝 Carol's perspective:"
echo "────────────────────────────────────────────────────────────────"
cat /tmp/carol_demo.log | tail -15
echo

echo "════════════════════════════════════════════════════════════════"
echo "📈 Server Statistics:"
echo "════════════════════════════════════════════════════════════════"
echo "Total connections: $(grep "New connection" /tmp/ircd_demo.log | wc -l)"
echo "Registrations: $(grep "Client registered" /tmp/ircd_demo.log | wc -l)"
echo "Channels created: $(grep "Channel created" /tmp/ircd_demo.log | wc -l)"
echo "Channel joins: $(grep "Client joined channel" /tmp/ircd_demo.log | wc -l)"
echo "Messages sent: $(grep "PRIVMSG" /tmp/alice_demo.log /tmp/bob_demo.log /tmp/carol_demo.log 2>/dev/null | wc -l)"
echo "Disconnections: $(grep "Client disconnected" /tmp/ircd_demo.log | wc -l)"
echo

# Stop server
echo "🛑 Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo "✅ Demo complete!"
echo
echo "Full logs available in /tmp/*_demo.log"
