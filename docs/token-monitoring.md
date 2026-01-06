# Token Lifetime Monitoring

## Overview

This document describes how to monitor iOS mobile device token lifetime and refresh behavior in the legible application using built-in CLI commands.

## Background

The reMarkable API uses two types of tokens:
- **Device Token**: Long-lived token obtained during device registration
- **User Token**: Short-lived token obtained by exchanging the device token

This monitoring focuses on understanding the lifetime characteristics and refresh patterns of iOS mobile device tokens (`mobile-ios` device type).

## Quick Start

### Check Current Token Status

```bash
legible token info
```

This displays:
- Device token information (type, ID, expiration if any)
- User token information (validity period, time remaining, status)
- Overall authentication status

### Monitor Token Renewals

Run daemon with monitoring enabled:

```bash
# Enable monitoring with console output
legible daemon --monitor-tokens --interval 30m

# Save statistics to a file
legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats.json \
  --interval 30m

# Full monitoring with debug logging
legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats.json \
  --interval 30m \
  --log-level debug \
  --no-ocr
```

Press Ctrl+C to stop the daemon and view statistics summary.

---

## Token Info Command

The `legible token info` command displays current authentication token status without making any API calls.

### Usage

```bash
legible token info [flags]
```

### Flags

- `--json`: Output in JSON format (for programmatic use)
- `--verbose`: Show full token details (future)

### Example Output

```
=== Authentication Token Information ===

Token file: /Users/username/.legible/token.json
Last modified: 2026-01-06 11:37:56

## Device Token

Device type: mobile-ios
Device ID: e90a54b4-f949-4e4c-889e-4e6eed4ab48c
Issued at: 2026-01-06T17:37:56Z
Expiration: No exp claim (does not expire)
Status: valid (indefinite)
Token length: 387 characters

## User Token

Device type: mobile-ios
Scopes: docedit screenshare hwcmail:-1 mail:-1 sync:tortoise intgr
Issued at: 2026-01-06T17:37:56Z
Expires at: 2026-01-06T20:37:56Z
Validity period: 3h 0m
Time remaining: 2h 37m
Status: valid
Token length: 945 characters

=== Summary ===

✓ Device token present
✓ User token valid

Ready to use. Run 'legible sync' or 'legible daemon' to start syncing.
```

### Token Status Indicators

- ✓ Device token present - Token exists and is valid
- ✓ User token valid - Token is not expired
- ⚠ User token expired - Token will be renewed on next API call
- ⚠ No user token - Token will be obtained on first API call
- ✗ No device token found - Need to run `legible auth register`

---

## Daemon Token Monitoring

The daemon command has built-in token monitoring capabilities that track renewal events and provide statistics.

### Monitoring Flags

**`--monitor-tokens`**
- Enables token renewal tracking
- Records renewal events with timestamps
- Displays statistics summary on exit

**`--token-stats-file <path>`**
- Path to save token statistics in JSON format
- Requires `--monitor-tokens` flag
- Statistics updated after each renewal
- Example: `~/.legible/token-stats.json`

### Usage Examples

#### Basic Monitoring (Console Only)

```bash
legible daemon --monitor-tokens --interval 30m
```

Tracks renewals and displays summary when daemon stops.

#### Save Statistics to File

```bash
legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats.json \
  --interval 30m
```

Statistics are saved to JSON file after each renewal.

#### 48-Hour Monitoring Test

```bash
timeout 48h legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats-48h.json \
  --interval 30m \
  --log-level debug \
  --no-ocr
```

Runs for 48 hours, logs token events, saves statistics.

#### Long-Term Monitoring (1 Week)

```bash
nohup legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats-1week.json \
  --interval 1h \
  --log-level info \
  --no-ocr \
  > ~/.legible/daemon-1week.log 2>&1 &
```

Runs in background for extended monitoring.

---

## Token Statistics

### Console Output

When the daemon exits, token statistics are automatically displayed:

```
=== Token Renewal Statistics ===

Total renewals: 16
First renewal: 2026-01-06T12:00:00Z
Last renewal: 2026-01-08T12:00:00Z
Monitoring duration: 2d 0h 0m
Average interval: 3h 0m

Recent renewals:
  12. 2026-01-08T09:00:00Z (after 3h 0m)
  13. 2026-01-08T10:00:00Z (after 3h 0m)
  14. 2026-01-08T11:00:00Z (after 3h 0m)
  15. 2026-01-08T12:00:00Z (after 3h 0m)
  16. 2026-01-08T13:00:00Z (after 3h 0m)

Statistics file: /Users/username/.legible/token-stats.json
```

### JSON Statistics File

If `--token-stats-file` is specified, statistics are saved in JSON format:

```json
{
  "start_time": "2026-01-06T12:00:00Z",
  "last_update": "2026-01-08T12:00:00Z",
  "renewal_count": 16,
  "renewal_events": [
    {
      "timestamp": "2026-01-06T12:00:00Z",
      "token_type": "user"
    },
    {
      "timestamp": "2026-01-06T15:00:00Z",
      "token_type": "user",
      "time_since_last": "3h 0m"
    }
  ],
  "average_interval": "3h 0m"
}
```

### Real-Time Log Events

With `--log-level debug`, token renewal events are logged:

```json
{
  "level": "info",
  "time": "2026-01-06T12:00:00Z",
  "message": "Token renewal tracked",
  "token_type": "user",
  "renewal_count": 1,
  "valid_for": "3h0m0s"
}

{
  "level": "info",
  "time": "2026-01-06T15:00:00Z",
  "message": "Token renewal tracked",
  "token_type": "user",
  "renewal_count": 2,
  "valid_for": "3h0m0s",
  "time_since_last": "3h0m0s"
}
```

---

## Monitoring Goals

### 1. Device Token Lifetime

**Questions to answer:**
- Does the device token expire?
- If so, what is the expiration period?
- Are there any rotation requirements?
- How does it compare to desktop device tokens?

**How to monitor:**
```bash
# Check initial state
legible token info

# Run for extended period
legible daemon --monitor-tokens --interval 1h

# Check again after 1+ weeks
legible token info
```

### 2. User Token Refresh Patterns

**Questions to answer:**
- How frequently does the user token need renewal?
- What is the typical validity period?
- Is the refresh frequency consistent?

**How to monitor:**
```bash
# Run with statistics tracking
legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats.json \
  --interval 30m \
  --log-level debug

# Analyze average_interval in stats file
cat ~/.legible/token-stats.json | jq .average_interval
```

### 3. Long-Running Behavior

**Questions to answer:**
- Does authentication remain stable over extended periods?
- Are there any unexpected failures?
- Does automatic renewal work reliably?

**How to monitor:**
```bash
# Run for 30 days
nohup legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats-30d.json \
  --interval 1h \
  > ~/.legible/daemon-30d.log 2>&1 &

# Check logs for errors
grep -i error ~/.legible/daemon-30d.log
```

---

## Monitoring Workflow

### Phase 1: Initial Assessment

1. **Check current token state**
   ```bash
   legible token info
   ```

2. **Document baseline characteristics**
   - Device token: expiration status
   - User token: validity period, time remaining

### Phase 2: Short-Term Monitoring (48 hours)

1. **Start monitoring**
   ```bash
   timeout 48h legible daemon \
     --monitor-tokens \
     --token-stats-file ~/.legible/token-stats-48h.json \
     --interval 30m \
     --log-level debug \
     --no-ocr
   ```

2. **Review statistics**
   - Expected: ~16 renewals (48h ÷ 3h)
   - Check average interval
   - Verify no errors

3. **Analyze results**
   ```bash
   cat ~/.legible/token-stats-48h.json | jq
   ```

### Phase 3: Extended Monitoring (1+ weeks)

1. **Start long-term monitoring**
   ```bash
   nohup legible daemon \
     --monitor-tokens \
     --token-stats-file ~/.legible/token-stats-1week.json \
     --interval 1h \
     --log-level info \
     --no-ocr \
     > ~/.legible/daemon-1week.log 2>&1 &
   ```

2. **Periodic checks**
   ```bash
   # Check if still running
   pgrep -f "legible daemon"

   # View recent renewals
   tail -20 ~/.legible/daemon-1week.log | grep "Token renewal"

   # Check statistics
   cat ~/.legible/token-stats-1week.json | jq .renewal_count
   ```

3. **Final analysis**
   - Device token still valid?
   - User token refresh pattern consistent?
   - Any authentication errors?

---

## Troubleshooting

### No statistics displayed

**Issue**: Daemon exits but no statistics shown

**Solutions**:
- Ensure `--monitor-tokens` flag is set
- Check that daemon ran long enough for at least one renewal
- User tokens renew every ~3 hours

### Token expired warning

**Issue**: `legible token info` shows expired user token

**Solution**: This is normal - token will be automatically renewed on next API call:
```bash
legible sync  # Triggers renewal
legible token info  # Should now show valid token
```

### Statistics file not found

**Issue**: `--token-stats-file` path doesn't exist

**Solution**:
- Directory is created automatically
- Check file permissions
- Use absolute path or `~/` prefix

---

## Best Practices

### For Short-Term Testing (48h-1week)

```bash
# Use timeout for automatic termination
timeout 48h legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats-$(date +%Y%m%d).json \
  --interval 30m \
  --log-level debug \
  --no-ocr
```

### For Long-Term Monitoring (1+ weeks)

```bash
# Use nohup for background execution
nohup legible daemon \
  --monitor-tokens \
  --token-stats-file ~/.legible/token-stats-longterm.json \
  --interval 1h \
  --log-level info \
  --no-ocr \
  > ~/.legible/daemon-$(date +%Y%m%d).log 2>&1 &

# Save PID for later
echo $! > ~/.legible/daemon.pid
```

### Stopping Long-Term Monitoring

```bash
# Graceful stop (displays statistics)
kill -SIGTERM $(cat ~/.legible/daemon.pid)

# Force stop if needed
kill -9 $(cat ~/.legible/daemon.pid)
```

---

## Related Documentation

- **Token Findings**: `TOKEN_LIFETIME_FINDINGS.md` - Observed token behavior
- **Issue**: remarkable-sync-bu7 - Monitor iOS mobile token lifetime
- **Auth Guide**: `docs/authentication.md` - Authentication setup

---

## Success Criteria

### Documented Token Expiration Behavior
- [ ] Device token lifetime documented
- [ ] User token lifetime documented
- [ ] Expiration patterns identified

### Measured Refresh Frequency
- [ ] Typical token validity period measured
- [ ] Refresh frequency documented
- [ ] Consistency over time verified

### Verified Stability
- [ ] Authentication remains stable over test period
- [ ] Automatic renewal works reliably
- [ ] No token-related errors

### Statistics Collected
- [ ] Token statistics JSON files saved
- [ ] Renewal count and intervals recorded
- [ ] Long-term behavior documented

---

**Last Updated**: 2026-01-06
**CLI Version**: v1.1.3+
