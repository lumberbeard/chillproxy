# Chillproxy Documentation

**Forked from**: [StremThru](https://github.com/MunifTanjim/stremthru)  
**Purpose**: Debrid service proxy with shared pool key management for Chillstreams  
**Status**: Planning Phase - Ready for Implementation

---

## What is Chillproxy?

**Chillproxy** is a Go-based HTTP proxy that sits between Stremio users and debrid services (TorBox, RealDebrid, etc.). It enables **secure, centralized management** of debrid API keys using a shared pool system, eliminating the need for users to expose their personal API keys.

### The Problem We're Solving

**Original StremThru Approach**:
```
User manifest URL contains: {"stores": [{"c": "tb", "t": "user_actual_api_key"}]}
                                                          ^^^^^^^^^^^^^^^^^^^^
                                                          EXPOSED TO USER!
```

**Issues**:
- ❌ Users can extract their API key from the manifest URL
- ❌ Keys can be shared/stolen
- ❌ No centralized device tracking
- ❌ No instant revocation (must rotate key manually)
- ❌ No usage analytics

**Chillproxy + Chillstreams Approach**:
```
User manifest URL contains: {"stores": [{"c": "tb", "auth": "user-uuid-from-chillstreams"}]}
                                                             ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                                             JUST A USER REFERENCE!
```

**Benefits**:
- ✅ **Zero key exposure**: Users never see TorBox API keys
- ✅ **Device tracking**: Max 3 devices per user, enforced automatically
- ✅ **Instant revocation**: Disable user → streams stop immediately
- ✅ **Usage analytics**: Track who's using what, when, and how much
- ✅ **Pool key rotation**: Rotate keys without user disruption
- ✅ **Fair usage**: Distribute load across multiple pool keys

---

## How It Works

### Architecture Overview

```
┌─────────────┐
│   Stremio   │  User clicks stream
│   (User)    │
└──────┬──────┘
       │
       │ 1. Request stream with user UUID
       ▼
┌─────────────────────┐
│   Chillproxy (Go)   │
│                     │
│  ┌───────────────┐  │
│  │ Device Tracker│  │  2. Generate device ID (IP + UA hash)
│  └───────────────┘  │
│         │           │
│         │ 3. Call Chillstreams API
│         ▼           │
│  ┌───────────────┐  │
│  │  CS API Client│──┼─────┐
│  └───────────────┘  │     │
└─────────────────────┘     │
                            │
                            ▼
                ┌───────────────────────┐
                │  Chillstreams (TS)    │
                │                       │
                │  ┌─────────────────┐  │
                │  │ Pool Key Manager│  │
                │  └─────────────────┘  │
                │          │            │
                │          │ 4. Assign pool key
                │          ▼            │
                │  ┌─────────────────┐  │
                │  │   PostgreSQL    │  │
                │  │  (pool_keys,    │  │
                │  │   assignments,  │  │
                │  │   devices)      │  │
                │  └─────────────────┘  │
                └───────────────────────┘
                            │
                            │ 5. Return pool key
                            ▼
                ┌─────────────────────┐
                │   Chillproxy (Go)   │
                │                     │
                │  Uses pool key to:  │
                │  ┌───────────────┐  │
                │  │  TorBox API   │  │  6. Check cache, fetch stream
                │  └───────────────┘  │
                │          │          │
                │          │ 7. Return stream URL
                └──────────┼──────────┘
                           │
                           │ 8. Log usage to Chillstreams
                           ▼
                ┌───────────────────────┐
                │  Usage Logs (async)   │
                │  - Action             │
                │  - Hash               │
                │  - Cached             │
                │  - Bytes transferred  │
                └───────────────────────┘
```

### Request Flow

**Step 1: User Requests Stream**
```
GET /stremio/torz/{base64_config}/stream/{type}/{id}.json

config = {
  "stores": [
    {"c": "tb", "auth": "550e8400-e29b-41d4-a716-446655440000"}
  ]
}
```

**Step 2: Chillproxy Generates Device ID**
```go
deviceID := sha256(clientIP + "|" + userAgent)
// Example: "a7b3c9d1e2f4567890abcdef12345678..."
```

**Step 3: Chillproxy Calls Chillstreams**
```http
POST http://chillstreams:3000/api/v1/internal/pool/get-key
Authorization: Bearer {INTERNAL_API_KEY}

{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "deviceId": "a7b3c9d1e2f4567890abcdef12345678...",
  "action": "check-cache",
  "hash": "torrent_infohash_here"
}
```

**Step 4: Chillstreams Validates & Returns Pool Key**
```json
{
  "allowed": true,
  "poolKey": "actual_torbox_api_key_from_pool",
  "poolKeyId": "pool-key-123",
  "deviceCount": 2
}
```

**Step 5: Chillproxy Uses Pool Key**
```go
torboxClient := torbox.NewStoreClient(&torbox.StoreClientConfig{
    HTTPClient: http.DefaultClient,
})
torboxClient.client.apiKey = poolKey // Use pool key from Chillstreams

// Check if cached
cached, _ := torboxClient.CheckCached(hash)

// Add torrent if needed
torrentID, _ := torboxClient.AddTorrent(magnetLink)

// Get stream URL
streamURL, _ := torboxClient.GetStream(torrentID, fileID)

// Return to user
```

**Step 6: Log Usage (Async)**
```go
go func() {
    chillstreamsClient.LogUsage(context.Background(), chillstreams.LogUsageRequest{
        UserID:    userID,
        PoolKeyID: "pool-key-123",
        Action:    "stream-served",
        Hash:      hash,
        Cached:    true,
        Bytes:     1500000000,
    })
}()
```

---

## Key Features

### 1. **Secure Authentication**
- User UUIDs instead of API keys in manifests
- Internal API key for Chillproxy ↔ Chillstreams communication
- No key exposure to end users

### 2. **Device Tracking**
- Automatic device fingerprinting (IP + User-Agent hash)
- Max 3 devices per user (configurable)
- Device registration on first use
- Device management UI in Chillstreams

### 3. **Pool Key Management**
- Centralized pool of TorBox API keys in Chillstreams
- Dynamic assignment to users
- Load balancing across pool keys
- Automatic rotation without user disruption
- Health monitoring for pool keys

### 4. **Usage Analytics**
- Track every stream request (hash, cached, bytes)
- User-level analytics (top users, heavy usage)
- Pool key usage distribution
- Historical data for optimization

### 5. **Instant Revocation**
- Disable user in Chillstreams → streams stop immediately
- No need to rotate keys manually
- Real-time enforcement on every request

### 6. **Backward Compatibility**
- Supports legacy `t` (token) field for direct API keys
- Gradual migration path from old to new system
- Feature flag to enable/disable Chillstreams integration

---

## Transformation Plan

See **[INTEGRATION_PLAN.md](./INTEGRATION_PLAN.md)** for detailed implementation steps.

### High-Level Changes

#### Phase 1: Core Modifications (Week 1)
1. **Update Config Schema** (`internal/stremio/torz/config.go`)
   - Add `auth` field for user UUID
   - Keep `t` field for backward compatibility
   
2. **Create Chillstreams API Client** (`internal/chillstreams/client.go`)
   - `GetPoolKey()` - Fetch pool key for user
   - `LogUsage()` - Log usage to Chillstreams
   
3. **Add Device Tracking** (`internal/device/tracker.go`)
   - Generate consistent device IDs
   - Extract client IP (handle X-Forwarded-For)
   
4. **Modify Store Initialization** (`store/torbox/store.go`)
   - Support dynamic API key from Chillstreams
   - Initialize with auth instead of static token

5. **Update Stream Handler** (`internal/stremio/torz/stream.go`)
   - Intercept requests
   - Call Chillstreams for pool key
   - Use pool key for TorBox API
   - Log usage asynchronously

#### Phase 2: Chillstreams API (Week 2)
**In Chillstreams repo** (`packages/server/src/routes/api/internal/pool.ts`):

1. **GET Pool Key Endpoint**
   - `POST /api/v1/internal/pool/get-key`
   - Validate user
   - Check device limits
   - Return assigned pool key

2. **Log Usage Endpoint**
   - `POST /api/v1/internal/pool/log-usage`
   - Store usage logs for analytics

3. **Database Schema**
   - `torbox_pool_assignments` - User → Pool Key mapping
   - `torbox_pool_devices` - Device tracking per user
   - `torbox_pool_usage_logs` - Usage logs

#### Phase 3: Testing & Deployment (Week 3)
1. Unit tests for Chillstreams client
2. Integration tests (end-to-end flow)
3. Load testing (100+ concurrent users)
4. Production deployment

---

## Configuration

### Environment Variables

**Chillproxy** (`.env`):
```bash
# Server
PORT=8080
STREMTHRU_BASE_URL=https://chillproxy.example.com

# Database (optional)
DATABASE_URL=postgres://user:pass@localhost/chillproxy
REDIS_URI=redis://localhost:6379

# Chillstreams Integration (NEW)
CHILLSTREAMS_API_URL=http://localhost:3000
CHILLSTREAMS_API_KEY=super_secret_internal_key_min_32_chars

# Features
STREMTHRU_FEATURE_TORZ=true
ENABLE_CHILLSTREAMS_AUTH=true  # Feature flag
```

**Chillstreams** (`.env`):
```bash
# Internal API
INTERNAL_API_KEY=super_secret_internal_key_min_32_chars

# Pool Management
TORBOX_POOL_SIZE=10  # Number of pool keys
TORBOX_MAX_DEVICES_PER_USER=3
```

---

## Development

### Prerequisites
- Go 1.21+
- PostgreSQL 14+ (or SQLite for dev)
- Redis (optional, for caching)
- Node.js 20+ & pnpm (for dashboard)

### Setup

**1. Clone and Install**
```pwsh
cd C:\chillproxy
go mod download
```

**2. Configure Environment**
```pwsh
cp .env.example .env
# Edit .env with your settings
```

**3. Run Locally**
```pwsh
# Build
go build -o chillproxy.exe .

# Run
$env:CHILLSTREAMS_API_URL="http://localhost:3000"
$env:CHILLSTREAMS_API_KEY="test_key"
.\chillproxy.exe
```

**4. Test**
```pwsh
# Run tests
go test ./...

# Test health endpoint
Invoke-WebRequest -Uri "http://localhost:8080/health"
```

### Building Dashboard

```pwsh
cd apps/dash
pnpm install
pnpm build

# Dev server
pnpm dev
```

---

## Testing Strategy

### Unit Tests
- Chillstreams API client (mock HTTP responses)
- Device ID generation (consistent hashing)
- Config parsing with `auth` field

### Integration Tests
- End-to-end: User ID → Pool Key → TorBox → Stream
- Device limit enforcement (max 3 devices)
- Pool key caching and expiration
- Usage logging accuracy

### Load Tests
```pwsh
# Simulate 100 concurrent users
go run scripts/loadtest.go `
  -url "http://localhost:8080/stremio/torz/.../stream/..." `
  -concurrent 100 `
  -duration 60s
```

**Metrics to Monitor**:
- Response times (< 100ms for pool key fetch)
- Error rates (< 0.1%)
- Pool key distribution (even across keys)
- Device tracking accuracy (100%)

---

## Security Considerations

### 1. Internal API Key
- **Strong key**: Min 32 characters, random
- **Not in git**: Use environment variables only
- **Rotation**: Rotate quarterly
- **IP Whitelisting**: If both services on same network

### 2. Pool Key Protection
- **Encrypted storage**: AES-256-GCM in Chillstreams DB
- **No logging**: Never log decrypted keys
- **Rate limiting**: Prevent brute force on `/get-key`
- **Audit trail**: Log all pool key assignments

### 3. Device Tracking
- **Hashing**: Don't store raw IPs (GDPR compliance)
- **Consistent**: Same IP + UA → Same device ID
- **User control**: Allow device revocation in UI

### 4. Rate Limiting
- **Per user**: 10 requests/sec
- **Per IP**: 50 requests/sec
- **Per pool key**: Monitor TorBox rate limits

---

## Deployment

### Docker

**Build Image**:
```pwsh
docker build -t chillproxy:latest .
```

**Run Container**:
```pwsh
docker run -d `
  -p 8080:8080 `
  -e CHILLSTREAMS_API_URL="https://api.chillstreams.com" `
  -e CHILLSTREAMS_API_KEY="prod_secret_key" `
  -e DATABASE_URL="postgres://..." `
  -e REDIS_URI="redis://..." `
  --name chillproxy `
  chillproxy:latest
```

### Docker Compose

**With Chillstreams**:
```yaml
version: '3.8'

services:
  chillstreams:
    image: chillstreams:latest
    ports:
      - "3000:3000"
    environment:
      - INTERNAL_API_KEY=shared_secret_key
      - DATABASE_URL=postgres://...

  chillproxy:
    image: chillproxy:latest
    ports:
      - "8080:8080"
    environment:
      - CHILLSTREAMS_API_URL=http://chillstreams:3000
      - CHILLSTREAMS_API_KEY=shared_secret_key
      - DATABASE_URL=postgres://...
    depends_on:
      - chillstreams
```

---

## Comparison with Alternatives

| Feature | Option 1: Direct Keys | Option 4: Chillproxy |
|---------|----------------------|----------------------|
| **Keys in manifest?** | ✅ Yes - visible | ❌ No - only user ID |
| **Device tracking** | ✅ Yes - in DB | ✅ Yes - full control |
| **Revocation** | ⚠️ Rotate key | ✅ Instant via API |
| **Implementation** | Easy (hours) | Medium (3 weeks) |
| **Security** | Medium | High |
| **Scalability** | Good | Excellent |
| **User experience** | Same | Same |
| **Maintenance** | Low | Medium |
| **Cost** | Low | Medium (extra infra) |

**Recommendation**: Chillproxy for production, long-term solution with best security and control.

---

## Troubleshooting

### Chillproxy Can't Connect to Chillstreams
```pwsh
# Check network connectivity
Invoke-WebRequest -Uri "$env:CHILLSTREAMS_API_URL/health"

# Verify API key
# Should return 403 if wrong key
Invoke-WebRequest -Uri "$env:CHILLSTREAMS_API_URL/api/v1/internal/pool/get-key" `
  -Method POST `
  -Headers @{"Authorization"="Bearer $env:CHILLSTREAMS_API_KEY"} `
  -Body '{"userId":"test","deviceId":"test","action":"test"}'
```

### User Gets "Not Allowed" Error
**Possible causes**:
1. User not found in Chillstreams DB
2. User status is not "active"
3. No pool key assigned to user
4. Device limit exceeded (> 3 devices)

**Debug**:
```sql
-- Check user status
SELECT id, email, status FROM users WHERE id = 'user-uuid';

-- Check pool key assignment
SELECT * FROM torbox_pool_assignments WHERE user_id = 'user-uuid';

-- Check device count
SELECT COUNT(*) FROM torbox_pool_devices WHERE user_id = 'user-uuid';
```

### Pool Key Not Working
**Check in Chillstreams**:
```pwsh
# Test pool key directly with TorBox API
$key = "pool_key_from_db"
Invoke-WebRequest -Uri "https://api.torbox.app/v1/api/user/me" `
  -Headers @{"Authorization"="Bearer $key"}

# Should return user info if key is valid
```

---

## Roadmap

### Phase 4: Advanced Features (Future)

1. **WebSocket for Real-Time Revocation**
   - Keep persistent connection Chillproxy ↔ Chillstreams
   - Instant user suspension without polling

2. **Analytics Dashboard**
   - User-level usage reports
   - Pool key health monitoring
   - Abuse detection patterns

3. **Multi-Service Support**
   - Extend to RealDebrid, AllDebrid, etc.
   - Per-service pool management
   - Automatic failover

4. **Automatic Pool Key Rotation**
   - Weekly key rotation schedule
   - Zero-downtime rotation
   - Health checks before rotation

5. **Geographic Distribution**
   - Deploy chillproxy instances globally
   - Route users to nearest proxy
   - Reduce latency

---

## Contributing

### Before You Start
1. Read `INTEGRATION_PLAN.md` for detailed implementation steps
2. Familiarize yourself with Go codebase structure
3. Understand Chillstreams API contracts
4. Set up local development environment

### Pull Request Guidelines
- ✅ Focus on Chillstreams integration features
- ✅ Add unit tests for new code
- ✅ Update documentation
- ✅ Follow Go best practices
- ✅ Test backward compatibility
- ❌ Don't break existing StremThru functionality

### Code Review Checklist
- [ ] Tests pass (`go test ./...`)
- [ ] No API key leaks in logs
- [ ] Proper error handling
- [ ] Documentation updated
- [ ] Backward compatible (if applicable)

---

## License

Same as upstream StremThru (check LICENSE file).

---

## Support

- **Issues**: GitHub Issues in chillproxy repo
- **Discussions**: GitHub Discussions
- **Documentation**: This folder (`docs/`)

---

## Acknowledgments

- **StremThru**: Original project by [@MunifTanjim](https://github.com/MunifTanjim)
- **Chillstreams**: Integration target by Viren070
- **Community**: Thanks to all contributors

---

**Last Updated**: December 17, 2025  
**Status**: Planning Phase - Ready for Implementation

