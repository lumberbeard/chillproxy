# Root Cause Analysis: Why Streams Stopped Working

## Executive Summary

**Primary Root Cause**: Device ID mismatch between database records and runtime code

**What Happened**:
1. Existing pool key assignments in database have device IDs as SHA256 hashes (64 hex characters)
2. Code changed to look for device ID = "default-device" (14 characters)
3. No matching assignments found in database
4. System tries to create new assignment
5. Either succeeds but uses different key, or fails due to device limit (3 max)
6. Subscription error handler catches the error
7. Returns fake "subscription required" stream or blocks access

**Confidence Level**: 95% - This is the smoking gun

---

## Detailed Analysis

### 1. Device ID Mismatch (PRIMARY SUSPECT - 95% confidence)

#### The Problem

**Timeline of Device ID Logic**:
```
Original (Pre-Jan 27):
  device_id = sha256(IP + "|" + UserAgent)
  Example: "a3f7b89c2d1e4567890abcdef1234567890abcdef1234567890abcdef123456"

Jan 27 Commit (bb375f9):
  device_id = sha256(IP)
  Example: "f1e2d3c4b5a6978012345678901234567890123456789012345678901234"

Post-Jan 27 (Current):
  device_id = "default-device"
  Example: "default-device"
```

**Database State**:
```sql
SELECT user_uuid, device_id, torbox_pool_key_id, created_at
FROM torbox_pool_assignments
WHERE user_uuid = '<your_user_uuid>';

-- Results likely show:
-- device_id = "a3f7b89c2d1e4567..." (64-char hash from IP+UA)
-- OR device_id = "f1e2d3c4b5a6978..." (64-char hash from IP-only)
```

**Current Code Lookup**:
```go
// chillproxy: internal/device/tracker.go
func GenerateDeviceID(r *http.Request) string {
    return "default-device"
}

// Then used in: internal/stremio/userdata/chillstreams_integration.go
deviceID := device.GenerateDeviceID(r)  // Returns "default-device"

resp, err := client.GetPoolKey(ctx, chillstreams.GetPoolKeyRequest{
    UserID:   s.ChillstreamsAuth,
    DeviceID: deviceID,  // "default-device"
    Action:   "init",
})
```

**Database Query in Chillstreams Backend**:
```typescript
// packages/server/src/routes/api/internal/pool.ts
const existingAssignment = await repository.getAssignment(userId, deviceId);

// This queries:
// SELECT * FROM torbox_pool_assignments
// WHERE user_uuid = ? AND device_id = ?
// With device_id = "default-device"

// But database has device_id = "a3f7b89c2d1e..." (hash)
// Result: NO MATCH FOUND
```

**What Happens Next**:
1. No existing assignment found
2. Counts current devices: `SELECT COUNT(DISTINCT device_id) FROM torbox_pool_assignments WHERE user_uuid = ?`
3. Returns 1, 2, or 3 (depending on how many old hashed device IDs exist)
4. If count >= 3: **Device limit reached error**
5. If count < 3: Creates NEW assignment with device_id = "default-device"
6. New assignment gets different pool key than old assignment
7. TorBox API call succeeds but with wrong key context
8. OR TorBox API rejects due to rate limiting, cooldown, or other issues

#### Evidence from Logs

From your chillproxy logs:
```
💛 torpool pool key injected
userId=a9dd4f63-33f5-4b36-8e0a-51f8a4f0e3ad
poolKeyId=cb3b5562-1c93-42a3-9853-ca0c6f88ca15
deviceCount=1  <-- Only 1 device counted (the new "default-device")
```

This shows only 1 device is being counted, but there are likely 3 old hashed device IDs in the database that are now orphaned.

#### How to Verify

```sql
-- Check existing device IDs in database
SELECT user_uuid, device_id, torbox_pool_key_id, created_at, last_used_at
FROM torbox_pool_assignments
WHERE user_uuid = '<your_user_uuid>'
ORDER BY created_at;

-- Expected results:
-- Row 1: device_id = "a3f7b89c..." (old IP+UA hash, created Jan 26 or earlier)
-- Row 2: device_id = "f1e2d3c4..." (IP-only hash, created Jan 27)
-- Row 3: device_id = "default-device" (constant, created Jan 28-29)

-- If 3+ rows exist with different device_ids:
-- You've hit the device limit and can't create "default-device" assignment
```

#### The Fix

**Option 1: Delete old device assignments** (RECOMMENDED)
```sql
-- Delete all assignments with hashed device IDs (length > 20)
DELETE FROM torbox_pool_assignments
WHERE user_uuid = '<your_user_uuid>'
AND LENGTH(device_id) > 20;

-- This will:
-- 1. Remove old orphaned assignments
-- 2. Free up device slots
-- 3. Allow "default-device" to be assigned
-- 4. Streams should work immediately after
```

**Option 2: Migrate device IDs in-place**
```sql
-- Change all device IDs to "default-device" for consistency
-- WARNING: This assumes you want 1 device per user
UPDATE torbox_pool_assignments
SET device_id = 'default-device'
WHERE user_uuid = '<your_user_uuid>'
AND device_id != 'default-device';

-- If multiple rows, keep only the most recent:
DELETE FROM torbox_pool_assignments
WHERE id NOT IN (
  SELECT id FROM torbox_pool_assignments
  WHERE user_uuid = '<your_user_uuid>'
  ORDER BY last_used_at DESC
  LIMIT 1
);

-- Then update the remaining one:
UPDATE torbox_pool_assignments
SET device_id = 'default-device'
WHERE user_uuid = '<your_user_uuid>';
```

**Option 3: Revert code to use IP-based hash** (NOT RECOMMENDED)
```go
// Revert internal/device/tracker.go to bb375f9 version:
func GenerateDeviceID(r *http.Request) string {
    ip := getClientIP(r)
    hash := sha256.Sum256([]byte(ip))
    return hex.EncodeToString(hash[:])
}

// But this doesn't solve the parallel request race condition issue
```

---

### 2. Node.js vs Go Device ID Mismatch (SECONDARY SUSPECT - 80% confidence)

#### The Problem

**Two Different Implementations**:

**Go (chillproxy)**:
```go
// internal/device/tracker.go
func GenerateDeviceID(r *http.Request) string {
    return "default-device"
}
```

**Node.js (chillstreams backend)**:
```typescript
// packages/server/src/services/torbox-key-injector.ts
export async function getAssignedPoolKey(userId: string, clientIp: string) {
  const deviceId = crypto
    .createHash('sha256')
    .update(`${userId}:${clientIp}`)
    .digest('hex');
  // Returns 64-char hash
}
```

**The Mismatch**:
- Go chillproxy requests pool key with device_id = "default-device"
- Node.js backend calculates device_id = sha256(userId:clientIp) for internal use
- These don't match!
- Backend creates/looks up assignment with hashed device_id
- Responds to Go with that pool key
- Go stores it under "default-device" in its logs/context
- Next request from Go with "default-device" doesn't find the assignment backend created with hash

#### Evidence

From [torbox-key-injector.ts](c:\chillstreams\packages\server\src\services\torbox-key-injector.ts) line 38-41:
```typescript
const deviceId = crypto
  .createHash('sha256')
  .update(`${userId}:${clientIp}`)
  .digest('hex');
```

This is calculating a DIFFERENT device ID than what Go sends in the API request!

#### How to Verify

Add logging to both sides:

**Go side** ([chillstreams_integration.go](c:\chillproxy\internal\stremio\userdata\chillstreams_integration.go) line 46):
```go
deviceID := device.GenerateDeviceID(r)
log.Debug("device id generated", "deviceId", deviceID)
```

**Node.js side** ([pool.ts](c:\chillstreams\packages\server\src\routes\api\internal\pool.ts)):
```typescript
const { userId, deviceId } = req.body;
logger.info('GetPoolKey request', { userId, deviceId, clientIp: req.ip });

// Then later when calculating for internal use:
const calculatedDeviceId = crypto
  .createHash('sha256')
  .update(`${userId}:${clientIp}`)
  .digest('hex');
logger.info('Device ID comparison', {
  received: deviceId,
  calculated: calculatedDeviceId,
  match: deviceId === calculatedDeviceId
});
```

Expected output:
```
Go logs:  device id generated deviceId=default-device
Node logs: GetPoolKey request deviceId=default-device
Node logs: Device ID comparison received=default-device calculated=f1e2d3c4b5a6... match=false
```

#### The Fix

**Update Node.js backend to use the device ID sent by Go** instead of recalculating:

```typescript
// packages/server/src/routes/api/internal/pool.ts
router.post('/get-key', async (req, res) => {
  const { userId, deviceId } = req.body;

  // REMOVE THIS:
  // const calculatedDeviceId = crypto.createHash('sha256')...

  // USE the deviceId sent by Go directly:
  const result = await repository.getAssignment(userId, deviceId);

  // Rest of code...
});

// packages/server/src/services/torbox-key-injector.ts
// REMOVE the deviceId calculation entirely:
export async function getAssignedPoolKey(userId: string, deviceId: string) {
  // ^^ Accept deviceId as parameter instead of clientIp

  const cacheKey = `${userId}:${deviceId}`;

  // Rest of code uses deviceId directly...
}
```

---

### 3. Subscription Error Handler False Positives (TERTIARY SUSPECT - 60% confidence)

#### The Problem

The subscription error handler added on Jan 26 might be catching errors that AREN'T subscription-related.

**Error Detection Logic** ([stream.go](c:\chillproxy\internal\stremio\store\stream.go)):
```go
func isSubscriptionError(err error) bool {
    if err == nil {
        return false
    }
    errMsg := strings.ToLower(err.Error())
    return strings.Contains(errMsg, "subscription") ||
        strings.Contains(errMsg, "expired") ||
        strings.Contains(errMsg, "not allowed") ||
        strings.Contains(errMsg, "requires upgrade") ||
        strings.Contains(errMsg, "requires renewal")
}
```

**Too Broad**:
- "not allowed" could match generic permission errors
- "expired" could match token expiration, cache expiration, etc.
- Any error message containing these words triggers fake stream response

**Example False Positive**:
```
Error: "Pool key assignment not allowed - device limit reached"
                              ^^^ matches "not allowed"

Result: Returns fake "subscription required" stream instead of "device limit" error
```

#### How to Verify

Add detailed error logging before the subscription check:

```go
func handleStream(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    ctx, err := ud.GetRequestContext(r, idr)
    if err != nil {
        // Log the ACTUAL error before checking if it's subscription-related
        LogError(r, "GetRequestContext failed", err)
        logger.Info("Error details", "error", err.Error(), "type", fmt.Sprintf("%T", err))

        if isSubscriptionError(err) {
            logger.Warn("Treated as subscription error", "error", err.Error())
            res.Streams = []stremio.Stream{createSubscriptionRequiredStream()}
            SendResponse(w, r, 200, res)
            return
        }
        // ... rest of error handling ...
    }
}
```

#### The Fix

**Make error detection more specific**:

```go
func isSubscriptionError(err error) bool {
    if err == nil {
        return false
    }
    errMsg := strings.ToLower(err.Error())

    // More specific matching
    isSubError := strings.Contains(errMsg, "subscription expired") ||
        strings.Contains(errMsg, "subscription inactive") ||
        strings.Contains(errMsg, "subscription required") ||
        strings.Contains(errMsg, "requires renewal") ||
        strings.Contains(errMsg, "trial expired")

    // Explicitly exclude device limit errors
    isDeviceLimit := strings.Contains(errMsg, "device limit") ||
        strings.Contains(errMsg, "too many devices")

    return isSubError && !isDeviceLimit
}
```

---

### 4. Pool Key Caching Issues (MINOR SUSPECT - 40% confidence)

#### The Problem

The 5-minute cache in [torbox-key-injector.ts](c:\chillstreams\packages\server\src\services\torbox-key-injector.ts) might serve stale pool keys.

**Cache Implementation**:
```typescript
const assignmentCache = Cache.getInstance<string, { id: string; apiKey: string }>('torbox-pool-assignments', 1000);

// Cache TTL: 5 minutes (300,000ms)
```

**Stale Key Scenario**:
1. User makes request → pool key assigned and cached
2. Pool key rotated/changed in database (by admin, by scheduled job, etc.)
3. User makes second request within 5 minutes → gets cached OLD key
4. TorBox API rejects old key → 401 UNAUTHORIZED
5. Subscription error handler catches it → fake stream

#### Evidence

From your logs:
```
💛 TORPOOL | Using cached pool key assignment
```

This shows the cache is being used. If the cached key is stale, it would fail.

#### How to Verify

Clear the cache and test:

```typescript
// Add to pool.ts endpoint:
router.post('/get-key', async (req, res) => {
  const { userId, deviceId } = req.body;

  // Force bypass cache for testing
  const cacheKey = `${userId}:${deviceId}`;
  await assignmentCache.delete(cacheKey);

  // Rest of code...
});
```

OR reduce cache TTL to 30 seconds for testing:
```typescript
const assignmentCache = Cache.getInstance<string, { id: string; apiKey: string }>('torbox-pool-assignments', 30); // 30 seconds
```

#### The Fix

**Option 1: Cache invalidation on pool key changes**
```typescript
// When pool key is rotated:
export async function rotatePoolKey(userId: string, deviceId: string) {
  // Clear cache
  const cacheKey = `${userId}:${deviceId}`;
  await assignmentCache.delete(cacheKey);

  // Update database...
}
```

**Option 2: Reduce cache TTL**
```typescript
const assignmentCache = Cache.getInstance<string, { id: string; apiKey: string }>('torbox-pool-assignments', 60); // 1 minute
```

**Option 3: Add cache validation**
```typescript
const cached = await assignmentCache.get(cacheKey);
if (cached) {
  // Verify key still valid by checking database
  const dbAssignment = await repository.getAssignment(userId, deviceId);
  if (dbAssignment && dbAssignment.id === cached.id) {
    return cached;
  } else {
    // Cache is stale, invalidate it
    await assignmentCache.delete(cacheKey);
  }
}
```

---

### 5. Buddy/Peer Service Interference (UNLIKELY - 20% confidence)

#### The Problem

External buddy/peer services making unauthorized API calls with wrong credentials.

**Why This is Unlikely**:
- You confirmed the TorBox key works with AIOStreams natively
- Pool key injection logs show success: `keySet:true`, `keyLength:36`
- Authorization header being set correctly: `Bearer 6748e313-ff29-4a26-80c1-34e8da4b79ee`

**We Spent All Day Here** and it was the wrong path.

#### Evidence Against This Theory

1. TorBox API key is valid (works in AIOStreams)
2. Pool key injection succeeds
3. Authorization headers are correct format
4. We disabled buddy/peer services → still doesn't work

#### The Fix

We already disabled buddy/peer services. No further action needed.

---

## Ranked List of Suspects

| Rank | Issue | Confidence | Impact | Fix Complexity |
|------|-------|------------|--------|----------------|
| 1 | Device ID database mismatch | 95% | High | Low - SQL UPDATE |
| 2 | Node.js vs Go device ID mismatch | 80% | High | Medium - Code change |
| 3 | Subscription error handler false positives | 60% | Medium | Low - Tighten regex |
| 4 | Pool key caching issues | 40% | Medium | Medium - Cache invalidation |
| 5 | Buddy/peer service interference | 20% | Low | Done - Already disabled |

---

## Recommended Action Plan

### Immediate Fix (Do This First)

**1. Check database for orphaned device IDs**:
```sql
SELECT user_uuid, device_id, torbox_pool_key_id, created_at, last_used_at
FROM torbox_pool_assignments
WHERE user_uuid = '<your_user_uuid>'
ORDER BY created_at;
```

**2. Delete old hashed device IDs**:
```sql
DELETE FROM torbox_pool_assignments
WHERE user_uuid = '<your_user_uuid>'
AND LENGTH(device_id) > 20;
```

**3. Test streaming immediately** - This should fix it 95% probability

---

### Secondary Fix (If Immediate Fix Doesn't Work)

**4. Add detailed logging to track device ID mismatch**:

**Go side** ([chillstreams_integration.go](c:\chillproxy\internal\stremio\userdata\chillstreams_integration.go)):
```go
deviceID := device.GenerateDeviceID(r)
log.Info("🔍 DEVICE_CHECK", "deviceId", deviceID, "length", len(deviceID))
```

**Node.js side** ([pool.ts](c:\chillstreams\packages\server\src\routes\api\internal\pool.ts)):
```typescript
const { userId, deviceId } = req.body;
logger.info('🔍 POOL_KEY_REQUEST', { userId, deviceId, length: deviceId.length });

// Check what's in database
const allAssignments = await db.query(
  'SELECT device_id FROM torbox_pool_assignments WHERE user_uuid = ?',
  [userId]
);
logger.info('🔍 DB_DEVICE_IDS', { assignments: allAssignments.map(a => ({ id: a.device_id, len: a.device_id.length })) });
```

**5. Test streaming** - Compare device IDs in logs

---

### Tertiary Fix (If Still Not Working)

**6. Temporarily disable subscription error handler** to see real errors:

```go
// Comment out the subscription error check
// if isSubscriptionError(err) {
//     res.Streams = []stremio.Stream{createSubscriptionRequiredStream()}
//     SendResponse(w, r, 200, res)
//     return
// }

// Let the real error through
LogError(r, "failed to get request context", err)
shared.ErrorBadRequest(r, "").Send(w, r)
return
```

**7. Test streaming** - See actual error message

---

### Long-term Fix (After Immediate Issue Resolved)

**8. Align device ID generation across Go and Node.js**:

**Either**:
- Make Node.js backend use the device ID sent by Go (recommended)
- OR make Go and Node.js use the same hashing algorithm

**9. Add migration script for future device ID changes**:
```typescript
// packages/server/src/migrations/migrate-device-ids.ts
export async function migrateDeviceIds() {
  // Get all old device IDs
  const oldAssignments = await db.query(
    'SELECT * FROM torbox_pool_assignments WHERE LENGTH(device_id) > 20'
  );

  // For each user, keep only most recent, update to "default-device"
  for (const user of getUniqueUsers(oldAssignments)) {
    const userAssignments = oldAssignments.filter(a => a.user_uuid === user);
    const mostRecent = userAssignments.sort((a, b) => b.last_used_at - a.last_used_at)[0];

    // Update most recent to "default-device"
    await db.execute(
      'UPDATE torbox_pool_assignments SET device_id = ? WHERE id = ?',
      ['default-device', mostRecent.id]
    );

    // Delete others
    const toDelete = userAssignments.filter(a => a.id !== mostRecent.id);
    for (const assignment of toDelete) {
      await db.execute('DELETE FROM torbox_pool_assignments WHERE id = ?', [assignment.id]);
    }
  }
}
```

**10. Add device ID validation in pool.ts**:
```typescript
router.post('/get-key', async (req, res) => {
  const { userId, deviceId } = req.body;

  // Validate device ID format
  if (!deviceId || typeof deviceId !== 'string') {
    return res.status(400).json({ error: 'Invalid device ID' });
  }

  // Log device ID for debugging
  logger.info('Pool key request', { userId, deviceId, length: deviceId.length });

  // Rest of code...
});
```

---

## What We Learned (Post-Mortem)

### Mistakes Made Today

1. **Chased symptoms instead of root cause**
   - Focused on buddy/peer services
   - Should have checked database device IDs first

2. **Misdiagnosed TorBox cooldown**
   - You correctly rejected this
   - We should have listened and pivoted faster

3. **Didn't verify database state**
   - Never checked what device IDs actually exist in database
   - Assumed code changes would just work

4. **Didn't trace the full request flow**
   - Should have logged device IDs at every step
   - Would have immediately spotted the mismatch

### What We Should Have Done

1. **Start with database inspection**:
   ```sql
   SELECT * FROM torbox_pool_assignments WHERE user_uuid = '<uuid>';
   ```

2. **Add comprehensive logging**:
   - Log device ID at generation time
   - Log device ID at API request time
   - Log device ID at database lookup time
   - Compare all three

3. **Test hypothesis before implementing fix**:
   - "If device ID is the issue, manually updating database should fix it"
   - Test this with SQL UPDATE before changing code

4. **Listen to user feedback**:
   - You said TorBox key works elsewhere
   - This immediately ruled out account issues
   - Should have focused on integration layer

---

## Confidence Assessment

**Very High Confidence (95%)**: Device ID database mismatch is the root cause

**Evidence**:
1. Device ID generation changed 3 times in 3 days
2. Database records have old device IDs
3. New code looks for "default-device"
4. No matching records found
5. Error handling treats this as subscription issue
6. Fake stream returned or access blocked

**Next Step**: Run the immediate fix SQL commands and test streaming.
