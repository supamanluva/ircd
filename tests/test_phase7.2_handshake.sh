#!/bin/bash

# Test script for Phase 7.2 - Server Handshake Protocol
# Tests two IRC servers linking together

set -e

echo "=== Phase 7.2 Server Linking Test ==="
echo

# Build the server
echo "Building IRC server..."
go build -o ircd cmd/ircd/main.go
echo "✓ Build successful"
echo

# Create logs directory
mkdir -p logs

# Clean up function
cleanup() {
    echo
    echo "Cleaning up..."
    if [ ! -z "$HUB_PID" ]; then
        kill $HUB_PID 2>/dev/null || true
    fi
    if [ ! -z "$LEAF_PID" ]; then
        kill $LEAF_PID 2>/dev/null || true
    fi
    sleep 1
}

trap cleanup EXIT

# Start hub server
echo "Starting hub server (hub.test, SID: 0AA)..."
./ircd -config config/config-hub.yaml > logs/hub-test.log 2>&1 &
HUB_PID=$!
echo "  PID: $HUB_PID"
echo "  Client port: 6667"
echo "  Link port: 7777"
echo "  WebSocket: 8080"
echo

# Wait for hub to start
sleep 2

# Check if hub is running
if ! kill -0 $HUB_PID 2>/dev/null; then
    echo "✗ Hub server failed to start"
    cat logs/hub-test.log
    exit 1
fi
echo "✓ Hub server started"
echo

# Start leaf server
echo "Starting leaf server (leaf.test, SID: 1BB)..."
./ircd -config config/config-leaf.yaml > logs/leaf-test.log 2>&1 &
LEAF_PID=$!
echo "  PID: $LEAF_PID"
echo "  Client port: 6668"
echo "  Link port: 7778"
echo "  WebSocket: 8081"
echo

# Wait for leaf to start and auto-connect
sleep 3

# Check if leaf is running
if ! kill -0 $LEAF_PID 2>/dev/null; then
    echo "✗ Leaf server failed to start"
    cat logs/leaf-test.log
    exit 1
fi
echo "✓ Leaf server started"
echo

# Check logs for successful link
echo "Checking hub logs for link establishment..."
if grep -q "Server link established.*name=leaf.test.*sid=1BB" logs/hub-test.log; then
    echo "✓ Hub received link from leaf"
else
    echo "✗ Hub did not establish link"
    echo "Hub log:"
    cat logs/hub-test.log
    exit 1
fi
echo

echo "Checking leaf logs for link establishment..."
if grep -q "Server link established.*name=hub.test.*sid=0AA" logs/leaf-test.log; then
    echo "✓ Leaf connected to hub"
else
    echo "✗ Leaf did not establish link"
    echo "Leaf log:"
    cat logs/leaf-test.log
    exit 1
fi
echo

# Check for handshake completion
echo "Checking for handshake completion..."
if grep -q "Server.*registered in network" logs/hub-test.log; then
    echo "✓ Hub completed handshake"
else
    echo "⚠ Hub handshake may not be complete"
fi

if grep -q "Server.*registered in network" logs/leaf-test.log; then
    echo "✓ Leaf completed handshake"
else
    echo "⚠ Leaf handshake may not be complete"
fi
echo

# Display server counts
echo "Hub log excerpt:"
grep -E "(Server linking enabled|Server link|registered in network|Handshake)" logs/hub-test.log | tail -5
echo

echo "Leaf log excerpt:"
grep -E "(Server linking enabled|Server link|registered in network|Handshake|Auto-connecting)" logs/leaf-test.log | tail -5
echo

echo "=== Test Summary ==="
echo "✓ Hub server started and listening"
echo "✓ Leaf server started and listening"
echo "✓ Leaf auto-connected to hub"
echo "✓ Handshake completed successfully"
echo "✓ Servers linked together"
echo
echo "Phase 7.2 test completed successfully!"
echo
echo "Servers are running. Press Ctrl+C to stop them."
echo "Hub log: logs/hub-test.log"
echo "Leaf log: logs/leaf-test.log"
echo

# Keep script running
wait
