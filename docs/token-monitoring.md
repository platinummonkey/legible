# Token Lifetime Monitoring

## Overview

This document describes the tools and procedures for monitoring iOS mobile device token lifetime and refresh behavior in the legible application.

## Background

The reMarkable API uses two types of tokens:
- **Device Token**: Long-lived token obtained during device registration
- **User Token**: Short-lived token obtained by exchanging the device token

This monitoring focuses on understanding the lifetime characteristics and refresh patterns of iOS mobile device tokens (`mobile-ios` device type).

## Monitoring Tools

### Scripts

1. **`scripts/monitor-token-lifetime.sh`**
   - Runs the legible daemon with debug logging
   - Captures token events for a specified duration
   - Saves logs for later analysis

2. **`scripts/analyze-token-logs.sh`**
   - Parses monitoring logs
   - Extracts token lifetime statistics
   - Generates analysis reports

### Usage

#### Basic Monitoring (48 hours)

```bash
# Build the application first
make build

# Run monitoring for 48 hours (default)
./scripts/monitor-token-lifetime.sh
```

#### Custom Duration

```bash
# Run for 7 days (168 hours)
./scripts/monitor-token-lifetime.sh 168

# Run for 30 days (720 hours)
./scripts/monitor-token-lifetime.sh 720
```

#### Manual Analysis

```bash
# Analyze an existing log file
./scripts/analyze-token-logs.sh ~/.legible/monitoring/token-lifetime-20260106-120000.log
```

## Monitoring Goals

### 1. Device Token Lifetime

**Questions to answer:**
- Does the device token expire?
- If so, what is the expiration period?
- Are there any rotation requirements?
- How does it compare to desktop device tokens?

**Data to collect:**
- Initial token acquisition timestamp
- Token length and format
- Any device token refresh events
- Authentication failures related to device token

### 2. User Token Refresh Patterns

**Questions to answer:**
- How frequently does the user token need renewal?
- What is the typical validity period?
- Is the refresh frequency different from desktop mode?
- Are there longer validity periods for mobile tokens?

**Data to collect:**
- User token acquisition timestamps
- Time between renewals
- Token expiration times
- Token validity periods

### 3. Long-Running Behavior

**Questions to answer:**
- Does the application maintain authentication over extended periods?
- Are there any unexpected authentication failures?
- Does automatic token renewal work reliably?

**Data to collect:**
- Authentication errors over time
- Token renewal success rate
- Sync operation success rate

## Test Scenarios

### Short-Term Test (48 hours)

**Purpose:** Establish baseline token behavior

**Duration:** 48 hours continuous operation

**Configuration:**
```bash
./scripts/monitor-token-lifetime.sh 48
```

**Expected observations:**
- Multiple user token renewals
- No authentication errors
- Successful sync operations

### Medium-Term Test (1 week)

**Purpose:** Observe token behavior over longer period with idle time

**Duration:** 7 days (168 hours)

**Configuration:**
```bash
./scripts/monitor-token-lifetime.sh 168
```

**Expected observations:**
- Device token remains valid
- User token renewal pattern is consistent
- No degradation in authentication

### Long-Term Test (30 days)

**Purpose:** Validate long-term stability and identify any rotation requirements

**Duration:** 30 days (720 hours)

**Configuration:**
```bash
./scripts/monitor-token-lifetime.sh 720
```

**Expected observations:**
- Device token lifetime characteristics
- Any long-term token rotation requirements
- Authentication stability over extended period

## Log Format

The monitoring scripts generate JSON-formatted logs with the following key events:

### Device Registration
```json
{
  "level": "info",
  "time": "2026-01-06T12:00:00.000Z",
  "message": "Device registered successfully, received device token",
  "device_id": "uuid",
  "token_length": 123,
  "token_preview": "abcd...wxyz"
}
```

### User Token Renewal
```json
{
  "level": "info",
  "time": "2026-01-06T12:30:00.000Z",
  "message": "User token renewed and saved successfully",
  "expiration": "2026-01-06T13:30:00.000Z",
  "valid_for": "59m59s",
  "user_token_length": 456
}
```

### Token Expiration Warning
```json
{
  "level": "info",
  "time": "2026-01-06T13:25:00.000Z",
  "message": "User token is expired or about to expire, need to renew",
  "expiration": "2026-01-06T13:30:00.000Z",
  "time_until_expiry": "-5m0s"
}
```

## Analysis Output

The analysis script (`analyze-token-logs.sh`) produces a report with:

1. **Device Registration**: Initial device token acquisition
2. **User Token Renewals**: List of all token refresh events
3. **Refresh Pattern Analysis**: Time intervals between renewals
4. **Token Expiration Events**: Warnings and expiration handling
5. **Sync Operations**: Count of sync cycles
6. **Authentication Errors**: Any failures or issues
7. **Summary**: Overall statistics and typical token validity

### Example Analysis Output

```
=== Token Lifetime Analysis ===

## Device Registration
(Already registered - no new registration in this monitoring period)

## User Token Renewals
Total renewals: 48

Renewal events:
  - Time: 2026-01-06T12:00:00.000Z
    Expiration: 2026-01-06T13:00:00.000Z
    Valid for: 59m59s
    Token length: 456

  - Time: 2026-01-06T13:00:00.000Z
    Expiration: 2026-01-06T14:00:00.000Z
    Valid for: 59m59s
    Token length: 456

## Refresh Pattern Analysis
Time between renewals:
  1h 0m (3600s)
  1h 0m (3600s)
  1h 0m (3600s)

## Summary
Monitoring period: 2026-01-06T12:00:00.000Z to 2026-01-08T12:00:00.000Z
Token renewals: 48
Typical token validity: 59m59s
```

## Current Implementation Details

### Token Expiration Handling

The application implements proactive token renewal with a 5-minute buffer:

```go
// From internal/rmclient/client.go
const tokenExpirationBuffer = 5 * time.Minute
```

**Behavior:**
- Tokens are checked before each API call
- If a token expires within 5 minutes, it's proactively renewed
- This prevents mid-operation token expiration

### Token Renewal Flow

1. **Check token expiration**: `isTokenExpired()` checks JWT exp claim
2. **Renew if needed**: `renewUserToken()` exchanges device token for new user token
3. **Update API context**: New token is injected into HTTP client
4. **Save token**: Renewed token is persisted to disk

### Logging Verbosity

Token lifecycle events are logged at different levels:

- **INFO**: Token renewals, expiration warnings, authentication success
- **DEBUG**: Token details (length, preview), API calls, parsing
- **ERROR**: Authentication failures, token renewal failures

Use `--log-level debug` for detailed monitoring.

## Success Criteria

### Documented Token Expiration Behavior
- [ ] Device token lifetime documented
- [ ] User token lifetime documented
- [ ] Expiration patterns identified

### Measured Refresh Frequency
- [ ] Typical token validity period measured
- [ ] Refresh frequency documented
- [ ] Comparison with desktop mode (if available)

### No Unexpected Authentication Failures
- [ ] Authentication remains stable over test period
- [ ] Automatic renewal works reliably
- [ ] No token-related errors

### Comparison Data vs Desktop Mode
- [ ] iOS mobile vs desktop token characteristics
- [ ] Any differences in validity periods
- [ ] Any differences in refresh patterns

## Deliverable

A document (`TOKEN_LIFETIME_FINDINGS.md`) with:

1. **Device Token Characteristics**
   - Lifetime (does it expire?)
   - Format and length
   - Any rotation requirements

2. **User Token Characteristics**
   - Typical validity period
   - Refresh frequency
   - Expiration handling

3. **Long-Running Behavior**
   - Stability over 30+ days
   - Authentication reliability
   - Any issues encountered

4. **Comparison with Desktop Mode**
   - Differences in token lifetime
   - Differences in refresh patterns
   - Any iOS-specific considerations

5. **Recommendations**
   - Best practices for token management
   - Suggested monitoring intervals
   - Any code improvements identified

## Next Steps

1. Run initial 48-hour test to establish baseline
2. Analyze results and document initial findings
3. Run extended tests (1 week, 30 days) if needed
4. Document comprehensive findings
5. Update issue remarkable-sync-bu7 with results

## Related Issues

- **remarkable-sync-bu7**: Monitor iOS mobile token lifetime and refresh behavior
- **remarkable-sync-3c2**: Verify all API operations work with tablet tokens (completed)
- **remarkable-sync-kpp**: Document iOS mobile authentication approach
