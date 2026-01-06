# Monitoring Scripts

This directory contains scripts for monitoring and analyzing token lifetime behavior in the legible application.

## Scripts

### `check-token-info.sh`

Display current authentication token information without making API calls.

**Usage:**
```bash
./scripts/check-token-info.sh
```

**Output:**
- Device token details (type, ID, expiration)
- User token details (validity period, time remaining, status)
- Overall authentication status

**Requirements:**
- `jq` for JSON parsing
- `node` for JWT decoding (optional but recommended)

---

### `monitor-token-lifetime.sh`

Run long-term token monitoring with automatic log collection and analysis.

**Usage:**
```bash
./scripts/monitor-token-lifetime.sh [duration_hours]
```

**Examples:**
```bash
# 48-hour monitoring (default)
./scripts/monitor-token-lifetime.sh

# 1-week monitoring
./scripts/monitor-token-lifetime.sh 168

# 30-day monitoring
./scripts/monitor-token-lifetime.sh 720
```

**Features:**
- Runs daemon with debug logging
- Captures token renewal events
- Automatic log analysis on completion
- Graceful shutdown with Ctrl+C

**Output:**
- Log file: `~/.legible/monitoring/token-lifetime-YYYYMMDD-HHMMSS.log`
- Analysis: `~/.legible/monitoring/token-analysis-YYYYMMDD-HHMMSS.txt`
- Metadata: `~/.legible/monitoring/token-lifetime-YYYYMMDD-HHMMSS.log.meta`

---

### `analyze-token-logs.sh`

Parse and analyze token monitoring logs to extract lifetime statistics.

**Usage:**
```bash
./scripts/analyze-token-logs.sh <log_file>
```

**Example:**
```bash
./scripts/analyze-token-logs.sh ~/.legible/monitoring/token-lifetime-20260106-120000.log
```

**Output Sections:**
1. Device Registration - Initial token acquisition
2. User Token Renewals - All refresh events with timestamps
3. Refresh Pattern Analysis - Time intervals between renewals
4. Token Expiration Events - Warnings and expiration handling
5. Sync Operations - Count of sync cycles
6. Authentication Errors - Any failures or issues
7. Summary - Overall statistics

---

### `test-monitoring.sh`

Quick 5-minute test to verify monitoring infrastructure works correctly.

**Usage:**
```bash
./scripts/test-monitoring.sh
```

**Purpose:**
- Verify application builds and authenticates
- Test log capture and analysis
- Validate monitoring scripts before long-term tests

**Output:**
- Short test run with log analysis
- Verification that token events are captured

---

## Prerequisites

### Required
- **legible binary**: Build with `make build`
- **Authentication**: Run `./dist/legible auth register` first
- **jq**: JSON parser (`brew install jq` on macOS)

### Optional
- **node.js**: For JWT decoding in token info display

---

## Monitoring Workflow

### 1. Check Prerequisites
```bash
# Build application
make build

# Verify authentication
./scripts/check-token-info.sh
```

### 2. Test Infrastructure (Optional)
```bash
# 5-minute test run
./scripts/test-monitoring.sh
```

### 3. Start Long-Term Monitoring
```bash
# 48-hour baseline test
./scripts/monitor-token-lifetime.sh 48
```

### 4. Analyze Results
Results are automatically analyzed on completion, or manually:
```bash
./scripts/analyze-token-logs.sh ~/.legible/monitoring/token-lifetime-*.log
```

---

## Understanding the Output

### Token Info Check

```
✓ Device token present     # Device token exists and is valid
✓ User token valid         # User token is not expired
⚠ User token expired       # Token will be renewed on next API call
⚠ No user token            # Token will be obtained on first API call
✗ No device token found    # Need to run 'auth register'
```

### Monitoring Analysis

```
## User Token Renewals
Total renewals: 16         # Number of token refresh events

## Refresh Pattern Analysis
Time between renewals:
  3h 0m (10800s)           # Consistent 3-hour refresh interval

## Summary
Token renewals: 16         # Total over monitoring period
Typical token validity: 3h # Standard validity period
```

---

## Troubleshooting

### "Error: Token file not found"
Run authentication first:
```bash
./dist/legible auth register
```

### "Error: legible binary not found"
Build the application:
```bash
make build
```

### Missing JWT details
Install node.js for JWT decoding:
```bash
brew install node
```

### Script not executable
Make scripts executable:
```bash
chmod +x scripts/*.sh
```

---

## Related Documentation

- **Token Monitoring Guide**: `docs/token-monitoring.md`
- **Token Lifetime Findings**: `TOKEN_LIFETIME_FINDINGS.md`
- **Issue**: remarkable-sync-bu7 - Monitor iOS mobile token lifetime

---

## Notes

- Monitoring logs are stored in `~/.legible/monitoring/`
- Logs are in JSON format for easy parsing
- Use `--log-level debug` for detailed token information
- Monitoring runs with `--no-ocr` to reduce resource usage
- Press Ctrl+C to stop monitoring early (triggers automatic analysis)
