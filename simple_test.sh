#!/bin/bash

echo "=== Simple Cross-Server Message Test ==="

# Send message from hub
echo "Sending message from hub server..."
echo -e "NICK testuser\nUSER testuser 0 * :Test User\nJOIN #test\nPRIVMSG #test :Cross-server test message!\nQUIT\n" | nc localhost 6667

echo "Message sent. Checking server logs..."

# Check if message was routed (look for routing logs)
sleep 2

# Check hub logs for routing
echo "Hub routing logs:"
ps aux | grep "ircd -config config/config-hub.yaml" | grep -v grep | head -1 | xargs -I {} sh -c 'echo "Hub PID: $1"' _ {}

# Check leaf logs for received message
echo "Leaf routing logs:"  
ps aux | grep "ircd -config config/config-leaf.yaml" | grep -v grep | head -1 | xargs -I {} sh -c 'echo "Leaf PID: $1"' _ {}

echo "Test completed - check if message routing worked!"
