#!/bin/bash
# Monitor iOS mobile device token lifetime and refresh patterns
#
# This script runs the legible daemon with debug logging and tracks
# token acquisition and refresh events. It's designed for long-running
# observation (48 hours, 1 week, or 30 days).
#
# Usage:
#   ./scripts/monitor-token-lifetime.sh [duration_hours]
#
# Example:
#   ./scripts/monitor-token-lifetime.sh 48  # Run for 48 hours

set -e

# Configuration
DURATION_HOURS="${1:-48}"  # Default: 48 hours
LOG_DIR="${HOME}/.legible/monitoring"
LOG_FILE="${LOG_DIR}/token-lifetime-$(date +%Y%m%d-%H%M%S).log"
ANALYSIS_FILE="${LOG_DIR}/token-analysis-$(date +%Y%m%d-%H%M%S).txt"
SYNC_INTERVAL="30m"  # Sync every 30 minutes

# Calculate end time
END_TIME=$(date -u -v+${DURATION_HOURS}H +%s 2>/dev/null || date -u -d "+${DURATION_HOURS} hours" +%s)

echo "=== Token Lifetime Monitoring ==="
echo "Start time: $(date)"
echo "Duration: ${DURATION_HOURS} hours"
echo "End time: $(date -u -r ${END_TIME} 2>/dev/null || date -u -d @${END_TIME})"
echo "Log file: ${LOG_FILE}"
echo "Sync interval: ${SYNC_INTERVAL}"
echo ""

# Create log directory
mkdir -p "${LOG_DIR}"

# Write monitoring metadata
cat > "${LOG_FILE}.meta" <<EOF
{
  "start_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "duration_hours": ${DURATION_HOURS},
  "sync_interval": "${SYNC_INTERVAL}",
  "device_type": "mobile-ios",
  "purpose": "Monitor iOS mobile device token lifetime and refresh patterns"
}
EOF

echo "Starting daemon with debug logging..."
echo "Press Ctrl+C to stop monitoring early"
echo ""

# Trap handler for graceful shutdown
cleanup() {
    echo ""
    echo "=== Monitoring stopped at $(date) ==="
    echo "Analyzing logs..."

    # Run analysis script if it exists
    if [ -f "scripts/analyze-token-logs.sh" ]; then
        bash scripts/analyze-token-logs.sh "${LOG_FILE}" > "${ANALYSIS_FILE}"
        echo "Analysis saved to: ${ANALYSIS_FILE}"
        echo ""
        echo "=== Summary ==="
        cat "${ANALYSIS_FILE}"
    else
        echo "Analysis script not found. Log file saved to: ${LOG_FILE}"
    fi

    exit 0
}

trap cleanup SIGINT SIGTERM

# Run daemon with debug logging
# Using tee to write to both file and stdout
timeout ${DURATION_HOURS}h ./bin/legible daemon \
    --interval ${SYNC_INTERVAL} \
    --log-level debug \
    --no-ocr 2>&1 | tee -a "${LOG_FILE}" || cleanup

# If timeout completes naturally, run cleanup
cleanup
