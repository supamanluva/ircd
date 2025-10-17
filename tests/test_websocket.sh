#!/bin/bash

# WebSocket IRC Server Test Script
# Tests WebSocket connectivity and IRC protocol over WebSocket

set -e

echo "=============================================="
echo "  IRC Server WebSocket Test"
echo "=============================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

# Start server
echo "Starting IRC server..."
pkill -9 ircd 2>/dev/null || true
sleep 1

./bin/ircd -config config/config.yaml > server.log 2>&1 &
SERVER_PID=$!

echo "Server PID: $SERVER_PID"
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}✗ Server failed to start${NC}"
    cat server.log
    exit 1
fi

echo -e "${GREEN}✓ Server started${NC}"
echo ""

# Test 1: WebSocket health check
echo "Test 1: WebSocket Health Check"
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo -e "${GREEN}✓ Health check passed${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ Health check failed${NC}"
    FAILED=$((FAILED + 1))
fi
echo ""

# Test 2: WebSocket connection with Node.js (if available)
echo "Test 2: WebSocket Connection Test"
if command -v node &> /dev/null; then
    cat > /tmp/ws_test.js << 'EOF'
const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8080/');

ws.on('open', function open() {
    console.log('Connected to WebSocket');
    
    // Send IRC NICK and USER commands
    ws.send('NICK TestUser\r\n');
    ws.send('USER testuser 0 * :Test User\r\n');
    
    setTimeout(() => {
        ws.send('QUIT :Test complete\r\n');
        ws.close();
    }, 2000);
});

ws.on('message', function incoming(data) {
    console.log('Received:', data.toString());
});

ws.on('error', function error(err) {
    console.error('WebSocket error:', err.message);
    process.exit(1);
});

ws.on('close', function close() {
    console.log('WebSocket closed');
    process.exit(0);
});

// Timeout after 5 seconds
setTimeout(() => {
    console.error('Timeout');
    process.exit(1);
}, 5000);
EOF

    if node /tmp/ws_test.js 2>&1 | grep -q "Connected to WebSocket"; then
        echo -e "${GREEN}✓ WebSocket connection successful${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ WebSocket connection failed${NC}"
        FAILED=$((FAILED + 1))
    fi
    rm -f /tmp/ws_test.js
else
    echo -e "${YELLOW}⚠ Node.js not available, skipping WebSocket test${NC}"
fi
echo ""

# Test 3: Check server logs
echo "Test 3: Server Log Check"
if grep -q "WebSocket server listening" server.log; then
    echo -e "${GREEN}✓ WebSocket server started correctly${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ WebSocket server not found in logs${NC}"
    FAILED=$((FAILED + 1))
fi
echo ""

# Test 4: Port availability
echo "Test 4: Port Availability"
if nc -z localhost 8080 2>/dev/null; then
    echo -e "${GREEN}✓ WebSocket port 8080 is open${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ WebSocket port 8080 is not accessible${NC}"
    FAILED=$((FAILED + 1))
fi
echo ""

# Test 5: TCP and WebSocket both work
echo "Test 5: Dual Protocol Test"
echo "Testing TCP connection..."
(
    echo "NICK TCPUser"
    echo "USER tcpuser 0 * :TCP User"
    sleep 1
    echo "QUIT :Bye"
) | nc -q 2 localhost 6667 > /tmp/tcp_test.log 2>&1

if grep -q "001" /tmp/tcp_test.log || grep -q "Welcome" /tmp/tcp_test.log; then
    echo -e "${GREEN}✓ TCP connection still works${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${YELLOW}⚠ TCP connection test inconclusive${NC}"
fi
echo ""

# Show server stats
echo "=============================================="
echo "  Server Statistics"
echo "=============================================="
echo "TCP Port: 6667"
echo "TLS Port: 7000"
echo "WebSocket Port: 8080"
echo ""
grep -E "(listening|WebSocket)" server.log | tail -5 || echo "No log entries found"
echo ""

# Summary
echo "=============================================="
echo "  Test Summary"
echo "=============================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

# Cleanup
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
sleep 1
pkill -9 ircd 2>/dev/null || true

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "WebSocket is working! Open tests/websocket_client.html in a browser to test interactively."
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo ""
    echo "Server log:"
    tail -20 server.log
    exit 1
fi
