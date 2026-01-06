#!/bin/bash
# Quick test of monitoring infrastructure (5 minutes)
#
# This script verifies that the monitoring tools work correctly
# before starting longer-term tests.

set -e

echo "=== Testing Token Monitoring Infrastructure ==="
echo ""

# Check prerequisites
if [ ! -f "dist/legible" ]; then
    echo "Error: legible binary not found. Run 'make build' first."
    exit 1
fi

if [ ! -f "${HOME}/.legible/token.json" ]; then
    echo "Error: No authentication token found. Run './dist/legible auth register' first."
    exit 1
fi

echo "✓ Prerequisites check passed"
echo ""

# Create monitoring directory
LOG_DIR="${HOME}/.legible/monitoring"
mkdir -p "${LOG_DIR}"

echo "Testing monitoring for 5 minutes..."
echo "Log directory: ${LOG_DIR}"
echo ""

# Run monitoring for 5 minutes
LOG_FILE="${LOG_DIR}/test-$(date +%Y%m%d-%H%M%S).log"

echo "Starting daemon with debug logging (5 minute test)..."
timeout 5m ./dist/legible daemon \
    --interval 2m \
    --log-level debug \
    --no-ocr 2>&1 | tee "${LOG_FILE}" || true

echo ""
echo "Test run complete. Analyzing logs..."
echo ""

# Run analysis
if [ -f "scripts/analyze-token-logs.sh" ]; then
    bash scripts/analyze-token-logs.sh "${LOG_FILE}"
else
    echo "Error: analyze-token-logs.sh not found"
    exit 1
fi

echo ""
echo "=== Test Complete ==="
echo "Log file: ${LOG_FILE}"
echo ""
echo "If the analysis shows token events, the monitoring infrastructure is working correctly."
echo "You can now run longer tests with: ./scripts/monitor-token-lifetime.sh [hours]"
