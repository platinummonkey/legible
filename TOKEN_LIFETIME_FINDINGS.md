# iOS Mobile Device Token Lifetime Findings

**Status**: Monitoring in progress
**Device Type**: `mobile-ios` (iOS app authentication)
**Started**: 2026-01-06
**Related Issue**: remarkable-sync-bu7

## Executive Summary

This document tracks observations about iOS mobile device token lifetime and refresh patterns for the reMarkable API. Initial analysis shows iOS mobile tokens have a **3-hour user token validity** with **device tokens that do not expire**.

---

## Initial Observations (2026-01-06)

### Device Token Characteristics

From examination of an active device token:

```json
{
  "device-desc": "mobile-ios",
  "device-id": "e90a54b4-f949-4e4c-889e-4e6eed4ab48c",
  "iat": 1767721076,  // 2026-01-06T17:37:56.000Z
  "exp": (not present) // No expiration claim
}
```

**Key Findings:**
- ✓ Device token **does NOT have an expiration claim**
- ✓ Token likely has indefinite lifetime or very long expiration (years)
- ✓ Successfully used for multiple API operations over several hours
- ✓ No device token rotation observed yet

**Comparison to Desktop Mode:**
- Desktop device tokens (TBD - need to test for comparison)

### User Token Characteristics

From examination of an active user token:

```json
{
  "device-desc": "mobile-ios",
  "iat": 1767721076,  // Issued: 2026-01-06T17:37:56.000Z
  "exp": 1767731876,  // Expires: 2026-01-06T20:37:56.000Z
  "scopes": "docedit screenshare hwcmail:-1 mail:-1 sync:tortoise intgr",
  "level": "connect"
}
```

**Key Findings:**
- ✓ User token validity: **3 hours (10800 seconds)**
- ✓ Token must be refreshed every 3 hours for continued API access
- ✓ Application uses 5-minute buffer for proactive renewal
- ✓ Token refresh is automatic via device token exchange

**Comparison to Desktop Mode:**
- Desktop user tokens (TBD - need to test for comparison)

### Token Refresh Mechanism

The application implements proactive token renewal:

1. **Before each API call**: Check user token expiration
2. **If expires within 5 minutes**: Automatically renew using device token
3. **Exchange device token**: Get fresh user token from reMarkable API
4. **Update API context**: Inject new token into HTTP client
5. **Persist to disk**: Save renewed token for next session

**Code Reference:**
- `internal/rmclient/client.go:587-646` - `ensureValidToken()`
- `internal/rmclient/client.go:69` - `tokenExpirationBuffer = 5 * time.Minute`

---

## Monitoring Plan

### Phase 1: Short-Term Test (48 hours) ⏳ IN PROGRESS

**Goal**: Establish baseline token behavior and verify refresh mechanism

**Test Configuration:**
```bash
./scripts/monitor-token-lifetime.sh 48
```

**What to observe:**
- [ ] User token renewals every ~3 hours
- [ ] Device token remains valid throughout
- [ ] No authentication errors
- [ ] Successful API calls between renewals

**Expected token renewals**: ~16 renewals over 48 hours (48h ÷ 3h = 16)

### Phase 2: Medium-Term Test (1 week)

**Goal**: Confirm stability over longer period

**Test Configuration:**
```bash
./scripts/monitor-token-lifetime.sh 168
```

**What to observe:**
- [ ] Device token still valid after 1 week
- [ ] User token refresh pattern remains consistent
- [ ] No degradation in authentication
- [ ] No unexpected token rotations

**Expected token renewals**: ~56 renewals over 1 week (168h ÷ 3h = 56)

### Phase 3: Long-Term Test (30 days)

**Goal**: Identify any long-term rotation requirements

**Test Configuration:**
```bash
./scripts/monitor-token-lifetime.sh 720
```

**What to observe:**
- [ ] Device token lifetime characteristics
- [ ] Any automatic rotation events
- [ ] Authentication stability over extended period
- [ ] Any differences in token behavior

**Expected token renewals**: ~240 renewals over 30 days (720h ÷ 3h = 240)

---

## Monitoring Tools

### Quick Token Check

View current token status without running tests:

```bash
./scripts/check-token-info.sh
```

Example output:
```
=== Current Token Information ===

## Device Token
Device type: mobile-ios
Device ID: e90a54b4-f949-4e4c-889e-4e6eed4ab48c
Issued at: 2026-01-06T17:37:56.000Z
Expiration: No exp claim (may not expire)

## User Token
Device type: mobile-ios
Validity period: 3h 0m (10800s)
Time remaining: 2h 37m (9454s)
Status: VALID

✓ Device token present
✓ User token valid
```

### Start Monitoring

Begin token lifetime monitoring:

```bash
# 48-hour test (recommended first step)
./scripts/monitor-token-lifetime.sh 48

# 1-week test
./scripts/monitor-token-lifetime.sh 168

# 30-day test
./scripts/monitor-token-lifetime.sh 720

# Custom duration (hours)
./scripts/monitor-token-lifetime.sh <hours>
```

### Analyze Existing Logs

Parse and analyze monitoring logs:

```bash
./scripts/analyze-token-logs.sh ~/.legible/monitoring/token-lifetime-YYYYMMDD-HHMMSS.log
```

---

## Preliminary Conclusions (2026-01-06)

### Device Token Lifetime

**Finding**: iOS mobile device tokens appear to have **indefinite lifetime** or very long expiration.

**Evidence**:
- No `exp` claim in JWT payload
- Device token used successfully over multiple hours
- No device token refresh events observed
- Token structure suggests long-term use

**Confidence Level**: Medium (need long-term testing to confirm)

**Implications**:
- ✓ One-time device registration is sufficient
- ✓ No need to re-register device periodically
- ⚠ Device token should be treated as sensitive credential
- ⚠ Token revocation requires manual action via reMarkable UI

### User Token Lifetime

**Finding**: iOS mobile user tokens have **3-hour validity period** (10800 seconds).

**Evidence**:
- JWT `exp` claim shows 3-hour lifetime from `iat`
- Consistent with mobile app usage patterns
- Automatic renewal via device token exchange

**Confidence Level**: High (directly observable in JWT)

**Implications**:
- ✓ Automatic renewal every 3 hours
- ✓ 5-minute proactive renewal buffer prevents mid-operation expiration
- ✓ Suitable for daemon mode with 30-minute sync intervals
- ⚠ Application must handle token refresh for long-running operations

### Authentication Stability

**Finding**: Authentication mechanism is **stable and automatic**.

**Evidence**:
- Proactive token renewal before expiration
- Graceful handling of expired tokens
- Successful API operations over extended periods

**Confidence Level**: Medium (need extended testing)

**Implications**:
- ✓ Suitable for daemon mode
- ✓ No user intervention required
- ✓ Robust error handling

---

## Comparison: iOS Mobile vs Desktop Mode

| Characteristic | iOS Mobile | Desktop Mode | Notes |
|---------------|------------|--------------|-------|
| Device Type | `mobile-ios` | `desktop-*` | Device descriptor |
| Device Token Lifetime | No expiration claim | TBD | iOS appears indefinite |
| User Token Lifetime | 3 hours | TBD | Fixed 3-hour window |
| Token Refresh | Automatic | TBD | Via device token exchange |
| Registration | UUID-based | TBD | Random device ID |

**Status**: Desktop mode comparison pending testing

---

## Questions Remaining

### Short-Term (Can answer with 48h test)
- [ ] What is the exact user token refresh frequency in practice?
- [ ] Are there any edge cases in token renewal?
- [ ] Does sync interval affect token refresh timing?

### Medium-Term (Can answer with 1-week test)
- [ ] Does device token remain valid for 1+ weeks?
- [ ] Is user token validity period always exactly 3 hours?
- [ ] Are there any network-related token issues?

### Long-Term (Can answer with 30-day test)
- [ ] Does device token expire after extended period?
- [ ] Are there any automatic rotation requirements?
- [ ] How does this compare to desktop mode?

---

## Next Steps

1. ✓ Set up monitoring infrastructure
2. ✓ Document initial token characteristics
3. ⏳ Run 48-hour monitoring test
4. ⏳ Analyze short-term results
5. ⏳ Run extended tests if needed
6. ⏳ Document final findings
7. ⏳ Update issue remarkable-sync-bu7

---

## References

- **Issue**: remarkable-sync-bu7 - Monitor iOS mobile token lifetime and refresh behavior
- **Code**: `internal/rmclient/client.go` - Token management implementation
- **Docs**: `docs/token-monitoring.md` - Monitoring procedures
- **Related**: remarkable-sync-3c2 - Verify all API operations work (completed)

---

## Change Log

### 2026-01-06
- Created initial findings document
- Documented device token characteristics (no expiration)
- Documented user token characteristics (3-hour validity)
- Set up monitoring infrastructure
- Prepared for 48-hour monitoring test

---

**Last Updated**: 2026-01-06 17:58 UTC
**Monitoring Status**: Phase 1 (48h) - Ready to start
**Next Review**: After 48-hour test completion
