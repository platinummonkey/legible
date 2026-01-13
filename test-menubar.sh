#!/usr/bin/env bash
#
# Menu Bar Application Test Suite
# Tests daemon lifecycle management and menu bar functionality
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

cleanup() {
    info "Cleaning up test processes..."
    pkill -9 -f "legible-menubar" 2>/dev/null || true
    pkill -9 -f "legible daemon" 2>/dev/null || true
    sleep 1
}

# Run cleanup on exit
trap cleanup EXIT

echo "========================================"
echo "  Menu Bar Application Test Suite"
echo "========================================"
echo ""

# Pre-test cleanup
cleanup

# Test 1: Daemon Auto-Launch
echo "Test 1: Daemon Auto-Launch"
echo "----------------------------"
info "Starting menu bar app..."

./dist/legible-menubar --daemon-addr http://localhost:8081 > /tmp/menubar-test1.log 2>&1 &
MENUBAR_PID=$!
sleep 4

# Check if daemon was launched
DAEMON_PID=$(pgrep -f "legible daemon --health-addr localhost:8081" || true)
if [ -n "$DAEMON_PID" ]; then
    pass "Daemon process launched automatically (PID: $DAEMON_PID)"
else
    fail "Daemon process not found"
fi

# Check if menu bar app is running
if ps -p $MENUBAR_PID > /dev/null 2>&1; then
    pass "Menu bar app is running (PID: $MENUBAR_PID)"
else
    fail "Menu bar app exited unexpectedly"
fi

# Check daemon API
API_RESPONSE=$(curl -s http://localhost:8081/status 2>/dev/null || echo "")
if echo "$API_RESPONSE" | jq -e '.state' > /dev/null 2>&1; then
    STATE=$(echo "$API_RESPONSE" | jq -r '.state')
    pass "Daemon API responding (state: $STATE)"
else
    fail "Daemon API not responding or invalid JSON"
fi

# Cleanup
kill $MENUBAR_PID 2>/dev/null || true
sleep 2

echo ""

# Test 2: Daemon Shutdown on Menu Bar Exit
echo "Test 2: Daemon Shutdown on Menu Bar Exit"
echo "----------------------------------------"

./dist/legible-menubar --daemon-addr http://localhost:8081 > /tmp/menubar-test2.log 2>&1 &
MENUBAR_PID=$!
sleep 3

DAEMON_PID=$(pgrep -f "legible daemon --health-addr localhost:8081" || true)
if [ -n "$DAEMON_PID" ]; then
    info "Daemon running (PID: $DAEMON_PID)"
else
    fail "Daemon not started"
fi

# Kill menu bar app
info "Stopping menu bar app..."
kill $MENUBAR_PID 2>/dev/null || true
sleep 3

# Check if daemon was also stopped
if pgrep -f "legible daemon --health-addr localhost:8081" > /dev/null 2>&1; then
    fail "Daemon still running after menu bar exit"
else
    pass "Daemon stopped gracefully with menu bar app"
fi

echo ""

# Test 3: No Auto-Launch Flag
echo "Test 3: No Auto-Launch Flag"
echo "---------------------------"

./dist/legible-menubar --no-auto-launch --daemon-addr http://localhost:8081 > /tmp/menubar-test3.log 2>&1 &
MENUBAR_PID=$!
sleep 3

# Check that daemon was NOT launched
DAEMON_PID=$(pgrep -f "legible daemon --health-addr localhost:8081" || true)
if [ -z "$DAEMON_PID" ]; then
    pass "--no-auto-launch flag prevents daemon launch"
else
    fail "Daemon was launched despite --no-auto-launch flag"
    kill $DAEMON_PID 2>/dev/null || true
fi

kill $MENUBAR_PID 2>/dev/null || true
sleep 1

echo ""

# Test 4: Daemon Crash Recovery
echo "Test 4: Daemon Crash Recovery"
echo "-----------------------------"

./dist/legible-menubar --daemon-addr http://localhost:8081 > /tmp/menubar-test4.log 2>&1 &
MENUBAR_PID=$!
sleep 3

DAEMON_PID=$(pgrep -f "legible daemon --health-addr localhost:8081" || true)
if [ -n "$DAEMON_PID" ]; then
    info "Daemon running (PID: $DAEMON_PID)"

    # Kill daemon to simulate crash
    info "Simulating daemon crash..."
    kill -9 $DAEMON_PID 2>/dev/null || true
    sleep 7  # Wait for health check + restart delay

    # Check if daemon was restarted
    NEW_DAEMON_PID=$(pgrep -f "legible daemon --health-addr localhost:8081" || true)
    if [ -n "$NEW_DAEMON_PID" ] && [ "$NEW_DAEMON_PID" != "$DAEMON_PID" ]; then
        pass "Daemon auto-restarted after crash (new PID: $NEW_DAEMON_PID)"
    else
        fail "Daemon not restarted after crash"
    fi
else
    fail "Daemon not started initially"
fi

kill $MENUBAR_PID 2>/dev/null || true
sleep 2

echo ""

# Test 5: Binary Detection
echo "Test 5: Binary Detection"
echo "-----------------------"

# Test should find legible in PATH or relative location
if grep -q "Daemon manager configured" /tmp/menubar-test1.log 2>/dev/null; then
    DAEMON_PATH=$(grep "Starting daemon manager" /tmp/menubar-test1.log | grep -o 'daemon_path[^[:space:]]*' | head -1)
    info "Daemon binary detected: $DAEMON_PATH"
    pass "Binary detection working"
else
    warn "Could not verify binary detection from logs"
fi

echo ""

# Summary
echo "========================================"
echo "  Test Summary"
echo "========================================"
echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
echo -e "${RED}Failed:${NC} $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
