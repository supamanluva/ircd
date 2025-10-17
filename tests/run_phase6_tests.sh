#!/bin/bash

# Comprehensive test runner for Phase 6 features

set -e

echo "================================================================"
echo "       IRC Server - Phase 6 Integration Test Suite"
echo "================================================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

TEMP_DIR=$(mktemp -d)
TEST_RESULTS="$TEMP_DIR/test_results.log"
FAILED_TESTS=0
PASSED_TESTS=0
TOTAL_TESTS=0

echo "Test directory: $TEMP_DIR"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up test environment..."
    pkill -9 ircd 2>/dev/null || true
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Function to run a test
run_test() {
    local test_name="$1"
    local test_script="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -e "${BLUE}[$TOTAL_TESTS]${NC} Running: ${YELLOW}$test_name${NC}"
    
    if bash "$test_script" > "$TEMP_DIR/${test_name}.log" 2>&1; then
        echo -e "    ${GREEN}✓ PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo "PASS: $test_name" >> "$TEST_RESULTS"
    else
        echo -e "    ${RED}✗ FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo "FAIL: $test_name" >> "$TEST_RESULTS"
        echo "    Log: $TEMP_DIR/${test_name}.log"
    fi
    
    # Kill any lingering processes
    pkill -9 ircd 2>/dev/null || true
    sleep 1
}

echo "================================================================"
echo "  Phase 6 Feature Tests"
echo "================================================================"
echo ""

# Build the server first
echo "Building IRC server..."
if make build > "$TEMP_DIR/build.log" 2>&1; then
    echo -e "${GREEN}✓ Build successful${NC}"
    echo ""
else
    echo -e "${RED}✗ Build failed!${NC}"
    cat "$TEMP_DIR/build.log"
    exit 1
fi

# Test 1: Channel Keys
if [ -f "tests/test_channel_keys.sh" ]; then
    run_test "Channel Keys (+k mode)" "tests/test_channel_keys.sh"
else
    echo -e "${YELLOW}⚠ test_channel_keys.sh not found${NC}"
fi

# Test 2: Voice Mode
if [ -f "tests/test_voice_mode.sh" ]; then
    run_test "Voice Mode (+v)" "tests/test_voice_mode.sh"
else
    echo -e "${YELLOW}⚠ test_voice_mode.sh not found${NC}"
fi

# Test 3: OPER Command
if [ -f "tests/test_oper.sh" ]; then
    run_test "OPER Command" "tests/test_oper.sh"
else
    echo -e "${YELLOW}⚠ test_oper.sh not found${NC}"
fi

# Test 4: Additional Commands
if [ -f "tests/test_additional_commands.sh" ]; then
    run_test "AWAY/USERHOST/ISON Commands" "tests/test_additional_commands.sh"
else
    echo -e "${YELLOW}⚠ test_additional_commands.sh not found${NC}"
fi

# Test 5: Phase 2 (regression)
if [ -f "tests/test_phase2.sh" ]; then
    run_test "Phase 2 Regression" "tests/test_phase2.sh"
else
    echo -e "${YELLOW}⚠ test_phase2.sh not found${NC}"
fi

# Test 6: Phase 3 (regression)
if [ -f "tests/test_phase3.sh" ]; then
    run_test "Phase 3 Regression" "tests/test_phase3.sh"
else
    echo -e "${YELLOW}⚠ test_phase3.sh not found${NC}"
fi

# Test 7: Phase 4 (regression)
if [ -f "tests/test_phase4.sh" ]; then
    run_test "Phase 4 Regression" "tests/test_phase4.sh"
else
    echo -e "${YELLOW}⚠ test_phase4.sh not found${NC}"
fi

echo ""
echo "================================================================"
echo "  Test Summary"
echo "================================================================"
echo ""
echo -e "Total Tests:  ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed:       ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:       ${RED}$FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Phase 6 is complete and all features are working correctly."
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo ""
    echo "Failed tests:"
    grep "FAIL:" "$TEST_RESULTS" || true
    echo ""
    echo "Review the logs in: $TEMP_DIR"
    exit 1
fi
