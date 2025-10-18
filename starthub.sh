#!/bin/bash
# Start IRC Hub Server

# Build if binary doesn't exist
if [ ! -f "./ircd" ]; then
    echo "Building ircd..."
    go build -o ircd cmd/ircd/main.go
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
fi

# Check if config exists
if [ ! -f "config/config-hub.yaml" ]; then
    echo "Error: config/config-hub.yaml not found!"
    echo "Please create hub configuration first."
    exit 1
fi

# Kill any existing ircd processes
echo "Stopping any existing ircd processes..."
pkill -f "ircd -config" || true
sleep 1

# Start hub server
echo "Starting IRC Hub Server..."
echo "Config: config/config-hub.yaml"
echo "Logs: /tmp/ircd_hub.log"
echo ""
echo "Server will listen on:"
echo "  - IRC (plaintext): :6667"
echo "  - IRC (TLS):       :7000"
echo "  - WebSocket:       :8080"
echo "  - Server Links:    :7777"
echo ""

./ircd -config config/config-hub.yaml > /tmp/ircd_hub.log 2>&1 &
HUB_PID=$!

echo "Hub server started with PID: $HUB_PID"
echo ""
echo "To view logs:    tail -f /tmp/ircd_hub.log"
echo "To stop server:  pkill -f 'ircd -config' or kill $HUB_PID"
echo ""
echo "Connect with: irssi -c localhost -p 6667"
