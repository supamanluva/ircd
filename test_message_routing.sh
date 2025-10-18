#!/bin/bash

# Test cross-server message routing
echo "Testing cross-server message routing..."

# Start hub user in background
(echo -e "NICK hubuser\nUSER hubuser 0 * :Hub User\nJOIN #test\nPRIVMSG #test :Hello from hub!\n" | nc localhost 6667) &

# Wait a bit then start leaf user
sleep 2
(echo -e "NICK leafuser\nUSER leafuser 0 * :Leaf User\nJOIN #test\nPRIVMSG #test :Hello from leaf!\n" | nc localhost 6668) &

# Wait for both to complete
wait
echo "Test completed"
