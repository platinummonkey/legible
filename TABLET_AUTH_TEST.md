# iOS Mobile Authentication Manual Test Plan

## Status: Updated to iOS Mobile Approach

**Previous Attempt:** Tablet authentication with "remarkable" failed during testing.
**Current Approach:** iOS mobile authentication with "mobile-ios" and proper UUID.

### Completed Automated Verifications

1. ✅ Code compiles successfully with tablet registration changes
2. ✅ Registration flow triggers on first run (no token)
3. ✅ Correct URL is displayed: `https://my.remarkable.com/device/remarkable?showOtp=true`
4. ✅ Prompt appears for one-time code input
5. ✅ Input validation works (expects 8-character code)

### Manual Testing Required

The following steps require a real reMarkable account and must be completed manually:

#### Step 1: Clean Slate Test
```bash
# Remove any existing token
rm -f ~/.legible/token.json

# Run legible to trigger registration
./bin/legible sync
```

**Expected Output:**
```
INFO	Visit https://my.remarkable.com/device/mobile-ios?showOtp=true to get a one-time code
Enter one-time code: _
```

#### Step 2: Get OTP Code
1. Open browser and visit: `https://my.remarkable.com/device/mobile-ios?showOtp=true`
2. Log in with your reMarkable account
3. Note the 8-character one-time code displayed

#### Step 3: Complete Registration
1. Enter the 8-character code at the prompt
2. Press Enter

**Expected Success Output:**
```
INFO	Registering device with one-time code...
INFO	✓ Device registered successfully
INFO	✓ Device token saved
INFO	Initializing API client with new device token...
INFO	=== Successfully authenticated with new device token ===
```

#### Step 4: Verify Token Storage
```bash
# Check token file was created
ls -la ~/.legible/token.json

# Verify permissions are restrictive
# Should show: -rw------- (0600)

# Check token contents (optional)
cat ~/.legible/token.json
```

**Expected token.json format:**
```json
{
  "device_token": "...",
  "user_token": "..."
}
```

#### Step 5: Test Token Persistence
```bash
# Run sync again - should NOT prompt for registration
./bin/legible sync
```

**Expected:**
- NO registration prompt
- Sync proceeds using existing token
- User token auto-renewed if needed

#### Step 6: Verify API Operations
```bash
# Test listing documents (requires documents in account)
./bin/legible sync

# Check logs for:
# - Successful authentication
# - Document listing
# - No authentication errors
```

#### Step 7: Test Token Renewal (Optional - Long Test)
```bash
# Run daemon mode for extended period
./bin/legible daemon

# Monitor logs for:
# - User token auto-renewal messages
# - No authentication failures
# - Time between renewals
```

### Success Criteria

- [ ] OTP code generation works at tablet URL
- [ ] Device token successfully acquired
- [ ] Token saved to `~/.legible/token.json` with 0600 permissions
- [ ] User token can be obtained from device token
- [ ] No errors in registration flow
- [ ] Token persists across application restarts
- [ ] API operations work (list, download documents)
- [ ] User token auto-renewal works

### Failure Scenarios to Check

1. **Invalid OTP Code**
   - Enter wrong code, verify error message
   - Should allow retry

2. **API Rejection**
   - If API rejects "remarkable" device type, document error
   - If API rejects "remarkable" device ID, document error

3. **Token Renewal Failure**
   - Monitor logs for renewal errors
   - Check if device token works for extended period

### Observations to Document

1. **Device Token Lifetime**
   - Does it expire? When?
   - Compare to desktop token behavior (if known)

2. **User Token Refresh Frequency**
   - How often does it refresh?
   - Is it different from desktop mode?

3. **API Behavior Differences**
   - Any different responses or errors?
   - Different rate limits or permissions?

4. **reMarkable Account Dashboard**
   - Check if device appears in account settings
   - How is it labeled/identified?
   - Can it coexist with actual tablet?

---

## Results Template

After completing manual testing, document results here:

### Test Date: ________

**Registration:**
- [ ] Success / [ ] Failure
- Notes:

**Token Persistence:**
- [ ] Success / [ ] Failure
- Notes:

**API Operations:**
- [ ] Success / [ ] Failure
- Notes:

**Issues Encountered:**
-

**Observations:**
-

**Recommendation:**
- [ ] Proceed with tablet auth
- [ ] Revert to desktop auth
- [ ] Needs more investigation

---

## Next Steps After Manual Testing

Based on test results:

1. **If Successful:**
   - Close remarkable-sync-q95
   - Start remarkable-sync-3c2 (API verification)
   - Continue with remaining tasks

2. **If Issues Found:**
   - Document specific error messages
   - Create new task for fixes
   - Consider remarkable-sync-x5q (device ID research)

3. **If API Rejects Tablet Auth:**
   - Investigate device ID format (remarkable-sync-x5q)
   - Consider alternative approaches
   - May need to add config toggle (remarkable-sync-4l1)
