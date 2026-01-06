#!/bin/bash
# Analyze token lifetime logs to extract refresh patterns
#
# This script parses debug logs from the legible daemon to extract:
# - Token acquisition timestamps
# - Token renewal timestamps
# - Time between renewals
# - Token expiration times
# - Token characteristics
#
# Usage:
#   ./scripts/analyze-token-logs.sh <log_file>

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <log_file>"
    exit 1
fi

LOG_FILE="$1"

if [ ! -f "${LOG_FILE}" ]; then
    echo "Error: Log file not found: ${LOG_FILE}"
    exit 1
fi

echo "=== Token Lifetime Analysis ==="
echo "Log file: ${LOG_FILE}"
echo ""

# Extract device registration events
echo "## Device Registration"
echo ""
DEVICE_REG=$(grep -i "device registered successfully" "${LOG_FILE}" | head -1 || true)
if [ -n "${DEVICE_REG}" ]; then
    echo "Device registered:"
    echo "${DEVICE_REG}" | grep -o '"time":"[^"]*"' | cut -d'"' -f4 || echo "  (timestamp not found)"
    echo "${DEVICE_REG}" | grep -o '"device_id":"[^"]*"' | cut -d'"' -f4 || echo "  (device_id not found)"
    echo "${DEVICE_REG}" | grep -o '"token_length":[0-9]*' || echo "  (token_length not found)"
else
    echo "No device registration found in logs (device was already registered)"
fi
echo ""

# Extract user token renewals
echo "## User Token Renewals"
echo ""
USER_TOKEN_RENEWALS=$(grep -i "user token renewed successfully" "${LOG_FILE}" || true)
if [ -n "${USER_TOKEN_RENEWALS}" ]; then
    RENEWAL_COUNT=$(echo "${USER_TOKEN_RENEWALS}" | wc -l | tr -d ' ')
    echo "Total renewals: ${RENEWAL_COUNT}"
    echo ""

    echo "Renewal events:"
    echo "${USER_TOKEN_RENEWALS}" | while IFS= read -r line; do
        TIMESTAMP=$(echo "${line}" | grep -o '"time":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        EXPIRATION=$(echo "${line}" | grep -o '"expiration":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        VALID_FOR=$(echo "${line}" | grep -o '"valid_for":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        TOKEN_LEN=$(echo "${line}" | grep -o '"user_token_length":[0-9]*' | grep -o '[0-9]*' || echo "unknown")

        echo "  - Time: ${TIMESTAMP}"
        echo "    Expiration: ${EXPIRATION}"
        echo "    Valid for: ${VALID_FOR}"
        echo "    Token length: ${TOKEN_LEN}"
        echo ""
    done
else
    echo "No user token renewals found in logs"
fi
echo ""

# Calculate time between renewals
echo "## Refresh Pattern Analysis"
echo ""
RENEWAL_TIMES=$(echo "${USER_TOKEN_RENEWALS}" | grep -o '"time":"[^"]*"' | cut -d'"' -f4 || true)
if [ -n "${RENEWAL_TIMES}" ]; then
    echo "Renewal timestamps:"
    echo "${RENEWAL_TIMES}" | nl
    echo ""

    # Convert to seconds and calculate intervals (requires date command with -j or -d)
    PREV_TIME=""
    echo "Time between renewals:"
    echo "${RENEWAL_TIMES}" | while IFS= read -r timestamp; do
        if [ -n "${PREV_TIME}" ]; then
            # Try macOS date format first, then Linux
            CURR_SEC=$(date -j -f "%Y-%m-%dT%H:%M:%S" "$(echo ${timestamp} | cut -d'.' -f1)" +%s 2>/dev/null || \
                       date -d "${timestamp}" +%s 2>/dev/null || echo "0")
            PREV_SEC=$(date -j -f "%Y-%m-%dT%H:%M:%S" "$(echo ${PREV_TIME} | cut -d'.' -f1)" +%s 2>/dev/null || \
                       date -d "${PREV_TIME}" +%s 2>/dev/null || echo "0")

            if [ "${CURR_SEC}" != "0" ] && [ "${PREV_SEC}" != "0" ]; then
                DIFF_SEC=$((CURR_SEC - PREV_SEC))
                DIFF_MIN=$((DIFF_SEC / 60))
                DIFF_HOURS=$((DIFF_MIN / 60))
                DIFF_MIN_REMAINDER=$((DIFF_MIN % 60))

                echo "  ${DIFF_HOURS}h ${DIFF_MIN_REMAINDER}m (${DIFF_SEC}s)"
            fi
        fi
        PREV_TIME="${timestamp}"
    done
else
    echo "Insufficient data to calculate refresh intervals"
fi
echo ""

# Extract token expiration warnings
echo "## Token Expiration Events"
echo ""
EXPIRATION_WARNINGS=$(grep -i "token is expired or about to expire" "${LOG_FILE}" || true)
if [ -n "${EXPIRATION_WARNINGS}" ]; then
    EXPIRATION_COUNT=$(echo "${EXPIRATION_WARNINGS}" | wc -l | tr -d ' ')
    echo "Expiration warnings: ${EXPIRATION_COUNT}"
    echo ""

    echo "Expiration events (first 10):"
    echo "${EXPIRATION_WARNINGS}" | head -10 | while IFS= read -r line; do
        TIMESTAMP=$(echo "${line}" | grep -o '"time":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        EXPIRATION=$(echo "${line}" | grep -o '"expiration":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        TIME_UNTIL=$(echo "${line}" | grep -o '"time_until_expiry":"[^"]*"' | cut -d'"' -f4 || echo "unknown")

        echo "  - Time: ${TIMESTAMP}"
        echo "    Token expires: ${EXPIRATION}"
        echo "    Time until expiry: ${TIME_UNTIL}"
        echo ""
    done
else
    echo "No token expiration warnings found"
fi
echo ""

# Extract sync operations to correlate with token renewals
echo "## Sync Operations"
echo ""
SYNC_START=$(grep -i "starting daemon" "${LOG_FILE}" | head -1 || true)
SYNC_COUNT=$(grep -i "sync cycle" "${LOG_FILE}" | wc -l | tr -d ' ')
echo "Sync cycles: ${SYNC_COUNT}"
echo ""

# Extract any authentication errors
echo "## Authentication Errors"
echo ""
AUTH_ERRORS=$(grep -i "authentication\|auth.*error\|failed to renew" "${LOG_FILE}" | grep -i error || true)
if [ -n "${AUTH_ERRORS}" ]; then
    AUTH_ERROR_COUNT=$(echo "${AUTH_ERRORS}" | wc -l | tr -d ' ')
    echo "Authentication errors: ${AUTH_ERROR_COUNT}"
    echo ""
    echo "Error events (first 5):"
    echo "${AUTH_ERRORS}" | head -5
else
    echo "No authentication errors found"
fi
echo ""

# Summary
echo "## Summary"
echo ""
LOG_START=$(head -1 "${LOG_FILE}" | grep -o '"time":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
LOG_END=$(tail -1 "${LOG_FILE}" | grep -o '"time":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
echo "Monitoring period:"
echo "  Start: ${LOG_START}"
echo "  End: ${LOG_END}"
echo ""

if [ -n "${USER_TOKEN_RENEWALS}" ]; then
    RENEWAL_COUNT=$(echo "${USER_TOKEN_RENEWALS}" | wc -l | tr -d ' ')
    echo "Token renewals: ${RENEWAL_COUNT}"

    # Extract typical validity period
    VALID_FOR_SAMPLE=$(echo "${USER_TOKEN_RENEWALS}" | head -1 | grep -o '"valid_for":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
    echo "Typical token validity: ${VALID_FOR_SAMPLE}"
else
    echo "Token renewals: 0"
fi

echo ""
echo "=== Analysis Complete ==="
