# Complete Summary of Recent Changes (Past Week)

## Git Commit Timeline

### January 26, 2026 (Monday) - Phase 5: Subscription Error Handling
**Commit**: d7e009b
**Files Changed**: [internal/stremio/store/stream.go](internal/stremio/store/stream.go)

**What Changed**:
- Added `isSubscriptionError()` helper function to detect subscription-related errors
- Added `createSubscriptionRequiredStream()` to generate fake "Subscription Required" stream
- Modified error handling in `handleStream()` function across 4 different code paths:
  - StremThru Store ID flow
  - IMDB ID flow (with parallel store processing)
  - Anime flow (with parallel store processing)
  - All error branches now check for subscription errors first

**Purpose**: When pool key validation fails due to subscription issues, return user-friendly fake stream instead of error page

**Error Detection Logic**:
```go
strings.Contains(errMsg, "subscription") ||
strings.Contains(errMsg, "expired") ||
strings.Contains(errMsg, "not allowed") ||
strings.Contains(errMsg, "requires upgrade") ||
strings.Contains(errMsg, "requires renewal")
```

**Fake Stream Response**:
```go
Name:        "🔒 Subscription Required"
Title:       "Your subscription has expired"
Description: "Click here to renew your Chillstreams subscription..."
ExternalURL: "https://app.chillstreams.com/account?action=renew"
```

---

### January 27, 2026 (Tuesday) - Device ID Generation Fix
**Commit**: bb375f9
**Files Changed**: [internal/device/tracker.go](internal/device/tracker.go)

**What Changed**:
- Changed device ID generation from `sha256(IP + "|" + UserAgent)` to `sha256(IP)`
- Commented out User-Agent from hash calculation
- Added TODO comment about localStorage-based device fingerprinting

**Purpose**: Prevent parallel requests from same browser being treated as different devices

**Original Code**:
```go
func GenerateDeviceID(r *http.Request) string {
    ip := getClientIP(r)
    ua := r.Header.Get("User-Agent")
    hash := sha256.Sum256([]byte(ip + "|" + ua))
    return hex.EncodeToString(hash[:])
}
```

**Changed To**:
```go
func GenerateDeviceID(r *http.Request) string {
    ip := getClientIP(r)
    // ua := r.Header.Get("User-Agent")
    // Use IP-only for now to avoid parallel request issues
    hash := sha256.Sum256([]byte(ip))
    return hex.EncodeToString(hash[:])
}
```

**Issue Addressed**: User was consuming all 3 device slots from single browser due to race conditions with different User-Agent strings in parallel requests

---

### January 27-29, 2026 (Post-Commit) - Further Device ID Simplification
**Files Changed**: [internal/device/tracker.go](internal/device/tracker.go)
**Status**: UNCOMMITTED (modified after bb375f9)

**What Changed**:
- Changed from IP-only hash to constant `"default-device"`
- Completely eliminated IP-based device fingerprinting
- Removed all hashing logic

**Purpose**: IP addresses proved unreliable due to IPv4/IPv6 switching, proxy headers, load balancing

**Current Code**:
```go
func GenerateDeviceID(r *http.Request) string {
    // Return constant device ID to ensure stability
    // This effectively gives each user ONE device assignment for the pool
    return "default-device"
}
```

**Critical Impact**:
- All existing pool key assignments in database have hashed device IDs
- New requests look for device ID = "default-device"
- Old assignments become orphaned and unreachable

---

### January 27-29, 2026 (Post-Commit) - Pool Key Caching & Race Condition Prevention
**Files Changed**: [c:\chillstreams\packages\server\src\services\torbox-key-injector.ts](c:\chillstreams\packages\server\src\services\torbox-key-injector.ts)
**Status**: UNCOMMITTED (modified recently)

**What Changed**:
- Added `assignmentCache` with 5-minute TTL
- Added `lastUsedUpdateCache` to prevent DB spam
- Added `inflightRequests` Map to track concurrent requests
- Changed from getAssignedPoolKey() directly calling DB to using cache-first approach

**Purpose**: Prevent race conditions when multiple parallel requests arrive for same user+device

**Key Changes**:
```typescript
// Cache for pool key assignments (5 minute TTL)
const assignmentCache = Cache.getInstance<string, { id: string; apiKey: string }>('torbox-pool-assignments', 1000);

// In-flight request tracker to prevent race conditions
const inflightRequests = new Map<string, Promise<{ id: string; apiKey: string } | null>>();

export async function getAssignedPoolKey(userId: string, clientIp: string) {
  const deviceId = crypto.createHash('sha256').update(`${userId}:${clientIp}`).digest('hex');
  const cacheKey = `${userId}:${deviceId}`;

  // Check cache first (fast path)
  const cached = await assignmentCache.get(cacheKey);
  if (cached) {
    return cached;
  }

  // Check if there's already an in-flight request
  const existingRequest = inflightRequests.get(cacheKey);
  if (existingRequest) {
    return existingRequest;
  }

  // Create new request and mark as in-flight
  const requestPromise = getAssignedPoolKeyInternal(userId, deviceId, cacheKey);
  inflightRequests.set(cacheKey, requestPromise);

  try {
    return await requestPromise;
  } finally {
    inflightRequests.delete(cacheKey);
  }
}
```

**Critical Difference**: Node.js backend still uses IP-based device ID hashing (`sha256(userId:clientIp)`), while Go chillproxy now uses "default-device" constant

---

### January 29, 2026 (Today) - Buddy/Peer Service Configuration
**Files Changed**: [~/chillstreams-app/docker-compose.prod.yml](~/chillstreams-app/docker-compose.prod.yml)
**Status**: Multiple iterations attempting to disable external services

**What Changed** (multiple iterations):
1. Set `STREMTHRU_BUDDY_URI: "disabled"` → Failed (invalid protocol)
2. Removed `STREMTHRU_BUDDY_URI` → Failed (defaulted to peer service)
3. Set `STREMTHRU_BUDDY_URI: "http://disabled"` → Failed (DNS lookup)
4. Set `STREMTHRU_PEER_URI: ""` → Failed (empty = not set)
5. Set `STREMTHRU_PEER_URI: "http://localhost:1"` → Partial success
6. Set both `STREMTHRU_BUDDY_URI` and `STREMTHRU_PEER_URI` to `"http://localhost:1"` → Current state

**Purpose**: Disable external buddy/peer services that were making unauthorized API calls

**Current Configuration**:
```yaml
environment:
  STREMTHRU_PEER_FLAG_LAZY: "true"
  STREMTHRU_BUDDY_URI: "http://localhost:1"
  STREMTHRU_PEER_URI: "http://localhost:1"
```

**Issue**: This entire approach was likely chasing symptoms rather than root cause

---

## Database Schema (No Changes)

The `torbox_pool_assignments` table structure remains:
```sql
CREATE TABLE torbox_pool_assignments (
  id UUID PRIMARY KEY,
  user_uuid UUID NOT NULL,
  device_id TEXT NOT NULL,
  torbox_pool_key_id UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  last_used_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(user_uuid, device_id)
);
```

**Critical Point**:
- Existing records have device_id values that are SHA256 hashes (64 hex characters)
- New code looks for device_id = "default-device" (14 characters)
- No matching records found → tries to create new assignment

---

## Authentication & Encryption (No Changes)

The following systems remain unchanged:
- Password hashing (bcryptjs)
- JWT token generation
- HttpOnly cookie settings
- Stripe integration
- Subscription validation logic
- Encryption key handling
- OAuth flows (Trakt, TMDB, TVDB)

---

## Key Configuration Variables (Unchanged)

From [c:\chillproxy\internal\config\config.go](c:\chillproxy\internal\config\config.go):
- `CHILLSTREAMS_API_URL`: "http://chillstreams-app:3000"
- `CHILLSTREAMS_API_KEY`: test_internal_key_phase3_2025
- `ENABLE_CHILLSTREAMS_AUTH`: "true"
- `TORBOX_API_KEY`: (environment variable)
- `TORBOX_MAX_DEVICES_PER_USER`: 3 (default)

---

## Summary of ALL Files Modified

| File | Date | Type | Status |
|------|------|------|--------|
| internal/stremio/store/stream.go | Jan 26 | Go | Committed (d7e009b) |
| internal/device/tracker.go | Jan 27 | Go | Committed (bb375f9) then modified |
| internal/device/tracker.go | Jan 27-29 | Go | Uncommitted ("default-device") |
| packages/server/src/services/torbox-key-injector.ts | Jan 27-29 | TypeScript | Uncommitted (caching) |
| docker-compose.prod.yml | Jan 29 | YAML | Modified (buddy/peer disable) |

---

## What Was NOT Changed

- Database schema
- Database records (assignments still have old device IDs)
- Subscription validation logic
- Password/authentication flows
- Encryption/decryption logic
- Pool key format validation (36 characters)
- Device limit enforcement (3 devices)
- Chillstreams API endpoints
- TorBox API client code (Authorization header logic)
