# Chillproxy Integration Plan for Chillstreams

**Date**: December 17, 2025  
**Status**: Planning Phase  
**Goal**: Transform StremThru fork (chillproxy) to work with Chillstreams shared pool key system

---


These were generated on creation:

Credentials (Auto-Generated)
Username: st-r7szcnl
Password: LkTsmuDYNwA0x5TbidjKIlvJWGm


## Overview

This document outlines how we'll modify **chillproxy** (forked from StremThru) to integrate with **Chillstreams**, enabling secure shared pool key management for TorBox and other debrid services.

**Project Relationship**:
- **Chillstreams** (TypeScript): Main addon aggregator, user management, pool key assignment
- **Chillproxy** (Go): Debrid service proxy that validates users and fetches streams using pool keys

---

## Current State (Original StremThru)

### How It Works Now

1. **User provides their own API key** in the manifest URL config:
```
https://stremthru.example.com/stremio/torz/{base64_config}/manifest.json

config = {
  "stores": [
    {"c": "tb", "t": "users_actual_torbox_api_key"}
  ]
}
```

2. **StremThru directly uses that key** to call TorBox API

3. **Problems**:
   - ❌ API key **visible in manifest URL** (user can extract it)
   - ❌ No centralized key management
   - ❌ No device tracking or usage limits
   - ❌ Users can share manifest URLs (key included)

---

## Target State (Chillstreams Integration)

### How It Will Work

1. **User provides Chillstreams user ID** (not TorBox key):
```
https://chillproxy.example.com/stremio/torz/{base64_config}/manifest.json

config = {
  "stores": [
    {
      "c": "tb",           // TorBox
      "t": "",             // Empty token
      "auth": "user-uuid"  // NEW: Chillstreams user ID
    }
  ]
}
```

2. **Chillproxy calls Chillstreams API** to get assigned pool key:
```
POST http://chillstreams:3000/api/v1/internal/pool/get-key
Body: {
  "userId": "user-uuid",
  "deviceId": "hash(ip+useragent)",
  "action": "check-cache",
  "hash": "torrent_infohash"
}

Response: {
  "poolKey": "actual_torbox_key_from_pool",
  "allowed": true,
  "deviceCount": 2
}
```

3. **Chillproxy uses pool key** to call TorBox API

4. **Chillproxy logs usage** back to Chillstreams:
```
POST http://chillstreams:3000/api/v1/internal/pool/log-usage
Body: {
  "userId": "user-uuid",
  "poolKeyId": "key-123",
  "action": "stream-served",
  "hash": "...",
  "cached": true,
  "bytes": 1500000000
}
```

5. **Benefits**:
   - ✅ TorBox keys **never exposed** to users
   - ✅ Centralized device tracking (max 3 devices per user)
   - ✅ Real-time revocation (disable user → streams stop)
   - ✅ Usage analytics (who's using what, when)
   - ✅ Pool key rotation without user awareness

---

## Implementation Steps

### Phase 1: Core Modifications (Week 1)

#### 1.1 Update Config Schema

**File**: `internal/stremio/torz/config.go`

**Change**:
```go
// OLD
type StoreConfig struct {
    Code  string `json:"c"`  // "tb" = TorBox
    Token string `json:"t"`  // User's API key
}

// NEW
type StoreConfig struct {
    Code  string `json:"c"`     // "tb" = TorBox
    Token string `json:"t"`     // Legacy (keep for backward compat)
    Auth  string `json:"auth"`  // NEW: Chillstreams user UUID
}
```

**Validation**:
```go
func (c *StoreConfig) Validate() error {
    if c.Auth == "" && c.Token == "" {
        return errors.New("either 'auth' (user ID) or 't' (token) required")
    }
    if c.Auth != "" {
        // Validate UUID format
        if !isValidUUID(c.Auth) {
            return errors.New("invalid user ID format")
        }
    }
    return nil
}
```

#### 1.2 Create Chillstreams API Client

**New File**: `internal/chillstreams/client.go`

```go
package chillstreams

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    baseURL string
    apiKey  string
    client  *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        baseURL: baseURL,
        apiKey:  apiKey,
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

// GetPoolKey fetches assigned pool key for user
func (c *Client) GetPoolKey(ctx context.Context, req GetPoolKeyRequest) (*GetPoolKeyResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/internal/pool/get-key", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to call chillstreams: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("chillstreams returned %d", resp.StatusCode)
    }

    var result GetPoolKeyResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &result, nil
}

// LogUsage logs pool key usage to Chillstreams
func (c *Client) LogUsage(ctx context.Context, req LogUsageRequest) error {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/internal/pool/log-usage", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("log usage failed: %d", resp.StatusCode)
    }

    return nil
}

type GetPoolKeyRequest struct {
    UserID   string `json:"userId"`
    DeviceID string `json:"deviceId"`
    Action   string `json:"action"`
    Hash     string `json:"hash"`
}

type GetPoolKeyResponse struct {
    PoolKey     string `json:"poolKey"`
    Allowed     bool   `json:"allowed"`
    DeviceCount int    `json:"deviceCount"`
    Message     string `json:"message,omitempty"`
}

type LogUsageRequest struct {
    UserID    string `json:"userId"`
    PoolKeyID string `json:"poolKeyId"`
    Action    string `json:"action"`
    Hash      string `json:"hash"`
    Cached    bool   `json:"cached"`
    Bytes     int64  `json:"bytes"`
}
```

#### 1.3 Add Device Tracking

**New File**: `internal/device/tracker.go`

```go
package device

import (
    "crypto/sha256"
    "encoding/hex"
    "net/http"
    "strings"
)

// GenerateDeviceID creates consistent device ID from IP + User-Agent
func GenerateDeviceID(r *http.Request) string {
    ip := getClientIP(r)
    ua := r.Header.Get("User-Agent")
    
    hash := sha256.Sum256([]byte(ip + "|" + ua))
    return hex.EncodeToString(hash[:])
}

func getClientIP(r *http.Request) string {
    // Check X-Forwarded-For, X-Real-IP headers
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return strings.Split(xff, ",")[0]
    }
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    // Extract IP from RemoteAddr (remove port)
    ip := r.RemoteAddr
    if idx := strings.LastIndex(ip, ":"); idx != -1 {
        ip = ip[:idx]
    }
    return ip
}
```

#### 1.4 Modify Store Initialization

**File**: `store/torbox/store.go`

**Add method to initialize with Chillstreams auth**:
```go
// Add to StoreClientConfig
type StoreClientConfig struct {
    HTTPClient       *http.Client
    UserAgent        string
    ChillstreamsAuth *ChillstreamsAuth  // NEW
}

type ChillstreamsAuth struct {
    UserID   string
    DeviceID string
    Client   *chillstreams.Client
}

// Modify NewStoreClient to accept dynamic key
func (c *StoreClient) InitializeWithAuth(auth *ChillstreamsAuth, hash string) error {
    // Fetch pool key from Chillstreams
    resp, err := auth.Client.GetPoolKey(context.Background(), chillstreams.GetPoolKeyRequest{
        UserID:   auth.UserID,
        DeviceID: auth.DeviceID,
        Action:   "init",
        Hash:     hash,
    })
    if err != nil {
        return fmt.Errorf("failed to get pool key: %w", err)
    }
    if !resp.Allowed {
        return errors.New("user not allowed: " + resp.Message)
    }
    
    // Set API key to pool key
    c.client.apiKey = resp.PoolKey
    return nil
}
```

#### 1.5 Update Stream Handler

**File**: `internal/stremio/torz/stream.go` (or wherever streams are served)

**Intercept stream requests**:
```go
func HandleStream(w http.ResponseWriter, r *http.Request) {
    // Parse config from URL
    config := parseConfigFromPath(r.URL.Path)
    
    // Check if using Chillstreams auth
    if config.Stores[0].Auth != "" {
        // NEW: Use Chillstreams integration
        deviceID := device.GenerateDeviceID(r)
        chillstreamsClient := getChillstreamsClient() // Singleton from env
        
        // Initialize store with auth
        storeConfig := &torbox.StoreClientConfig{
            HTTPClient: http.DefaultClient,
            UserAgent:  "chillproxy",
            ChillstreamsAuth: &torbox.ChillstreamsAuth{
                UserID:   config.Stores[0].Auth,
                DeviceID: deviceID,
                Client:   chillstreamsClient,
            },
        }
        
        storeClient := torbox.NewStoreClient(storeConfig)
        
        // Extract hash from request
        hash := extractHashFromPath(r.URL.Path)
        
        // Initialize with pool key
        if err := storeClient.InitializeWithAuth(storeConfig.ChillstreamsAuth, hash); err != nil {
            http.Error(w, "Failed to initialize: "+err.Error(), http.StatusForbidden)
            return
        }
        
        // Continue with existing stream logic...
        // Check cache, add torrent, get stream URL
        
        // Log usage (async)
        go func() {
            _ = chillstreamsClient.LogUsage(context.Background(), chillstreams.LogUsageRequest{
                UserID: config.Stores[0].Auth,
                Action: "stream-served",
                Hash:   hash,
            })
        }()
    } else {
        // LEGACY: Use token from config
        storeClient := torbox.NewStoreClient(&torbox.StoreClientConfig{
            HTTPClient: http.DefaultClient,
            UserAgent:  "chillproxy",
        })
        storeClient.client.apiKey = config.Stores[0].Token
        
        // Continue with existing logic...
    }
}
```

---

### Phase 2: Configuration & Deployment (Week 2)

#### 2.1 Environment Variables

**Add to chillproxy**:
```bash
# .env
CHILLSTREAMS_API_URL=http://localhost:3000
CHILLSTREAMS_API_KEY=super_secret_internal_key_min_32_chars
```

**Add to chillstreams**:
```bash
# .env
INTERNAL_API_KEY=super_secret_internal_key_min_32_chars
```

#### 2.2 Config Loading

**File**: `internal/config/integration.go` (NEW)

```go
package config

import "os"

var (
    ChillstreamsAPIURL string
    ChillstreamsAPIKey string
)

func init() {
    ChillstreamsAPIURL = os.Getenv("CHILLSTREAMS_API_URL")
    ChillstreamsAPIKey = os.Getenv("CHILLSTREAMS_API_KEY")
    
    if ChillstreamsAPIURL == "" {
        ChillstreamsAPIURL = "http://localhost:3000"
    }
}
```

#### 2.3 Chillstreams API Endpoints

**File** (Chillstreams): `packages/server/src/routes/api/internal/pool.ts`

```typescript
import { Router } from 'express';
import { TorBoxPoolRepository } from '../../../db/repositories/torbox-pool.js';
import { UsersRepository } from '../../../db/repositories/users.js';

const router = Router();

// Middleware to validate internal API key
router.use((req, res, next) => {
  const authHeader = req.headers.authorization;
  const expectedKey = `Bearer ${process.env.INTERNAL_API_KEY}`;
  
  if (authHeader !== expectedKey) {
    return res.status(403).json({ error: 'Unauthorized' });
  }
  
  next();
});

// Get pool key for user
router.post('/pool/get-key', async (req, res) => {
  try {
    const { userId, deviceId, action, hash } = req.body;

    // Validate user exists and is active
    const user = await UsersRepository.findById(userId);
    if (!user || user.status !== 'active') {
      return res.json({ allowed: false, message: 'User not found or inactive' });
    }

    // Get assigned pool key for user
    const assignment = await TorBoxPoolRepository.getAssignmentForUser(userId);
    if (!assignment) {
      return res.json({ allowed: false, message: 'No pool key assigned' });
    }

    // Check device limits (max 3 devices per user)
    const deviceCount = await TorBoxPoolRepository.getDeviceCount(userId);
    const isKnownDevice = await TorBoxPoolRepository.isDeviceRegistered(userId, deviceId);

    if (!isKnownDevice && deviceCount >= 3) {
      return res.json({
        allowed: false,
        message: 'Maximum 3 devices allowed per user'
      });
    }

    // Register new device if needed
    if (!isKnownDevice) {
      await TorBoxPoolRepository.registerDevice(userId, deviceId);
    }

    // Update last used timestamp
    await TorBoxPoolRepository.updateLastUsed(assignment.id);

    res.json({
      allowed: true,
      poolKey: assignment.decryptedKey,
      poolKeyId: assignment.poolKeyId,
      deviceCount: deviceCount + (isKnownDevice ? 0 : 1)
    });
  } catch (error) {
    console.error('Get pool key error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Log usage
router.post('/pool/log-usage', async (req, res) => {
  try {
    const { userId, poolKeyId, action, hash, cached, bytes } = req.body;

    await TorBoxPoolRepository.logUsage({
      userId,
      poolKeyId,
      action,
      hash,
      cached,
      bytes,
      timestamp: new Date()
    });

    res.json({ success: true });
  } catch (error) {
    console.error('Log usage error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
```

---

### Phase 3: Testing & Validation (Week 3)

#### 3.1 Local Testing

**Start Chillstreams**:
```pwsh
cd C:\chillstreams
pnpm build; pnpm start
```

**Start Chillproxy**:
```pwsh
cd C:\chillproxy
$env:CHILLSTREAMS_API_URL="http://localhost:3000"
$env:CHILLSTREAMS_API_KEY="test_internal_key"
go build; .\chillproxy.exe
```

**Test Flow**:
1. Create user in Chillstreams with assigned pool key
2. Generate manifest URL with `auth` field
3. Add to Stremio and test stream playback
4. Verify logs show pool key fetching and usage logging

#### 3.2 Integration Tests

**File**: `internal/chillstreams/client_test.go`

```go
package chillstreams

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetPoolKey(t *testing.T) {
    // Mock Chillstreams API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") != "Bearer test_key" {
            w.WriteHeader(http.StatusForbidden)
            return
        }
        json.NewEncoder(w).Encode(GetPoolKeyResponse{
            Allowed:     true,
            PoolKey:     "test_torbox_key",
            DeviceCount: 1,
        })
    }))
    defer server.Close()

    client := NewClient(server.URL, "test_key")
    resp, err := client.GetPoolKey(context.Background(), GetPoolKeyRequest{
        UserID:   "test-user",
        DeviceID: "device-123",
        Action:   "check-cache",
    })

    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if !resp.Allowed {
        t.Error("Expected allowed=true")
    }
    if resp.PoolKey != "test_torbox_key" {
        t.Errorf("Expected poolKey=test_torbox_key, got %s", resp.PoolKey)
    }
}
```

---

## Database Schema (Chillstreams)

**New Tables**:

```sql
-- Track which pool key is assigned to which user
CREATE TABLE torbox_pool_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  pool_key_id UUID NOT NULL REFERENCES torbox_pool_keys(id),
  assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMP,
  UNIQUE(user_id)
);

-- Track devices per user
CREATE TABLE torbox_pool_devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  device_id VARCHAR(64) NOT NULL,
  first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
  last_seen TIMESTAMP,
  UNIQUE(user_id, device_id)
);

CREATE INDEX idx_pool_devices_user ON torbox_pool_devices(user_id);

-- Usage logs
CREATE TABLE torbox_pool_usage_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID NOT NULL,
  pool_key_id UUID NOT NULL,
  action VARCHAR(50) NOT NULL,
  hash VARCHAR(40),
  cached BOOLEAN,
  bytes BIGINT,
  timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_usage_logs_user ON torbox_pool_usage_logs(user_id, timestamp);
CREATE INDEX idx_usage_logs_key ON torbox_pool_usage_logs(pool_key_id, timestamp);
```

---

## Security Considerations

### 1. Internal API Authentication

- Use strong API key (min 32 characters)
- Validate on every request
- Consider IP whitelisting if both services on same network
- Rotate keys periodically

### 2. Pool Key Storage

- Store encrypted in Chillstreams database
- Decrypt only when needed for API response
- Never log decrypted keys
- Use secure encryption (AES-256-GCM)

### 3. Device Tracking

- Hash `(IP + User-Agent)` for consistent device IDs
- Don't store raw IPs (GDPR compliance)
- Allow users to manage their devices
- Implement device revocation

### 4. Rate Limiting

**Chillproxy**:
- Limit requests per user (10/sec)
- Limit requests per IP (50/sec)

**Chillstreams**:
- Limit `/internal/pool/get-key` calls (100/sec)
- Cache pool key assignments (5 min TTL)

---

## Rollback Plan

If integration fails:

1. **Feature Flag**:
```bash
ENABLE_CHILLSTREAMS_AUTH=false
```

2. **Fallback to Legacy**:
```go
if config.Auth != "" && config.EnableChillstreamsAuth {
    // Try Chillstreams
    poolKey, err := getPoolKey(config.Auth)
    if err != nil {
        log.Warn().Err(err).Msg("Chillstreams unavailable, require token")
        return nil, errors.New("authentication failed")
    }
} else {
    // Use legacy token
    storeClient.apiKey = config.Token
}
```

3. **Dual Mode Support**:
   - Support both `auth` (new) and `t` (legacy)
   - Gradual migration of users

---

## Success Metrics

**Week 1**:
- ✅ Config schema supports `auth` field
- ✅ Chillstreams client implemented
- ✅ Device tracking working
- ✅ Basic integration test passing

**Week 2**:
- ✅ Chillstreams API endpoints deployed
- ✅ Database schema migrated
- ✅ End-to-end test with real TorBox key
- ✅ 10 test users successfully streaming

**Week 3**:
- ✅ Load testing passed (100 concurrent users)
- ✅ Zero key leaks verified
- ✅ Device limits enforced correctly
- ✅ Usage logs populated accurately

---

## Future Enhancements

### Phase 4: Advanced Features

1. **Real-time Revocation**:
   - WebSocket connection for instant user suspension
   - Middleware to check user status on every request

2. **Analytics Dashboard**:
   - Show pool key usage per user
   - Detect abuse patterns
   - Visualize concurrent streams

3. **Multi-Service Support**:
   - Extend to RealDebrid, AllDebrid, etc.
   - Per-service pool management

4. **Automatic Key Rotation**:
   - Rotate pool keys without user disruption
   - Health checks for pool keys

---

## Comparison with Alternatives

| Aspect | Option 1: Direct Keys | Option 4: Self-Host Proxy |
|--------|----------------------|---------------------------|
| **Keys in manifest?** | ✅ Yes - visible | ❌ No - only user ID |
| **Device tracking** | ✅ Yes - in DB | ✅ Yes - full control |
| **Revocation** | ⚠️ Rotate key | ✅ Instant via API |
| **Implementation** | Easy (hours) | Medium (3 weeks) |
| **Security** | Medium | High |
| **Recommendation** | Quick start | Long-term solution |

---

**Status**: 📋 Ready for Implementation  
**Next Step**: Start Phase 1.1 (Update Config Schema)

**Last Updated**: December 17, 2025

