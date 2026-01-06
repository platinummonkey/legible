#!/bin/bash
# Display current token information
#
# This script decodes and displays information about the current
# authentication tokens without making any API calls.

set -e

TOKEN_FILE="${HOME}/.legible/token.json"

if [ ! -f "${TOKEN_FILE}" ]; then
    echo "Error: Token file not found: ${TOKEN_FILE}"
    echo "Run './dist/legible auth register' to authenticate first."
    exit 1
fi

echo "=== Current Token Information ==="
echo ""

# Extract tokens
DEVICE_TOKEN=$(jq -r '.device_token' "${TOKEN_FILE}")
USER_TOKEN=$(jq -r '.user_token' "${TOKEN_FILE}")

echo "Token file: ${TOKEN_FILE}"
echo "Last modified: $(date -r "${TOKEN_FILE}" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || stat -c '%y' "${TOKEN_FILE}" 2>/dev/null)"
echo ""

# Decode device token
echo "## Device Token"
echo ""
if [ -n "${DEVICE_TOKEN}" ] && [ "${DEVICE_TOKEN}" != "null" ]; then
    # Check if we have node for JWT decoding
    if command -v node &> /dev/null; then
        node -e "
            const jwt = '${DEVICE_TOKEN}';
            const payload = JSON.parse(Buffer.from(jwt.split('.')[1], 'base64').toString());
            console.log('Device type:', payload['device-desc'] || 'unknown');
            console.log('Device ID:', payload['device-id'] || 'unknown');
            if (payload.iat) {
                console.log('Issued at:', new Date(payload.iat*1000).toISOString());
            }
            if (payload.exp) {
                console.log('Expires at:', new Date(payload.exp*1000).toISOString());
                const now = Math.floor(Date.now()/1000);
                const remaining = payload.exp - now;
                if (remaining > 0) {
                    const days = Math.floor(remaining / 86400);
                    const hours = Math.floor((remaining % 86400) / 3600);
                    console.log('Time remaining:', days + 'd', hours + 'h');
                } else {
                    console.log('Status: EXPIRED');
                }
            } else {
                console.log('Expiration: No exp claim (may not expire)');
            }
        "
    else
        echo "Device token present (length: ${#DEVICE_TOKEN})"
        echo "(Install node.js to decode JWT details)"
    fi
else
    echo "No device token found"
fi

echo ""
echo "## User Token"
echo ""

if [ -n "${USER_TOKEN}" ] && [ "${USER_TOKEN}" != "null" ]; then
    if command -v node &> /dev/null; then
        node -e "
            const jwt = '${USER_TOKEN}';
            const payload = JSON.parse(Buffer.from(jwt.split('.')[1], 'base64').toString());
            const now = Math.floor(Date.now()/1000);

            console.log('Device type:', payload['device-desc'] || 'unknown');
            console.log('Scopes:', payload['scopes'] || 'unknown');

            if (payload.iat) {
                console.log('Issued at:', new Date(payload.iat*1000).toISOString());
            }
            if (payload.exp) {
                console.log('Expires at:', new Date(payload.exp*1000).toISOString());

                const validFor = payload.exp - payload.iat;
                const hours = Math.floor(validFor / 3600);
                const mins = Math.floor((validFor % 3600) / 60);
                console.log('Validity period:', hours + 'h', mins + 'm', '(' + validFor + 's)');

                const remaining = payload.exp - now;
                if (remaining > 0) {
                    const remHours = Math.floor(remaining / 3600);
                    const remMins = Math.floor((remaining % 3600) / 60);
                    console.log('Time remaining:', remHours + 'h', remMins + 'm', '(' + remaining + 's)');
                    console.log('Status: VALID');
                } else {
                    console.log('Status: EXPIRED (needs renewal)');
                }
            } else {
                console.log('Expiration: No exp claim');
            }
        "
    else
        echo "User token present (length: ${#USER_TOKEN})"
        echo "(Install node.js to decode JWT details)"
    fi
else
    echo "No user token found (will be obtained on first API call)"
fi

echo ""
echo "=== Summary ==="
echo ""

if command -v node &> /dev/null; then
    # Check overall authentication status
    if [ -n "${DEVICE_TOKEN}" ] && [ "${DEVICE_TOKEN}" != "null" ]; then
        echo "✓ Device token present"

        if [ -n "${USER_TOKEN}" ] && [ "${USER_TOKEN}" != "null" ]; then
            # Check if user token is expired
            IS_EXPIRED=$(node -e "const jwt='${USER_TOKEN}'; const payload=JSON.parse(Buffer.from(jwt.split('.')[1],'base64').toString()); const now=Math.floor(Date.now()/1000); console.log(payload.exp && payload.exp < now ? 'true' : 'false');" 2>/dev/null || echo "unknown")

            if [ "${IS_EXPIRED}" = "true" ]; then
                echo "⚠ User token expired (will be renewed on next API call)"
            else
                echo "✓ User token valid"
            fi
        else
            echo "⚠ No user token (will be obtained on first API call)"
        fi

        echo ""
        echo "Ready for monitoring. Run: ./scripts/monitor-token-lifetime.sh [hours]"
    else
        echo "✗ No device token found"
        echo "Run './dist/legible auth register' to authenticate first"
    fi
else
    echo "Install node.js to see detailed token information"
fi
