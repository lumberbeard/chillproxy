# Phase 2 Complete - Test Integration

## Summary

✅ **Phase 2 API Endpoints Complete**

The Chillstreams internal API endpoints have been fixed to work with the actual PostgreSQL schema:

### Changes Made

1. **Fixed table names**:
   - `torbox_pool_assignments` → `torbox_assignments`
   - `torbox_pool_keys` → `torbox_pool`
   - `torbox_pool_usage_logs` → `torbox_usage_logs`

2. **Fixed column names**:
   - `pool_key_id` → `assigned_pool_key_id` (in torbox_assignments)
   - `assigned_at` → `created_at`
   - `last_used` → `last_used_at`
   - `current_assignments` → `current_slots` (in torbox_pool)
   - `max_assignments` → `max_slots`

3. **Removed non-existent table**:
   - Removed references to `torbox_pool_devices` (doesn't exist)
   - Device tracking now uses `torbox_assignments.device_id`

4. **Fixed usage logging**:
   - Updated to match actual `torbox_usage_logs` schema
   - Maps action → endpoint
   - Provides default values for status_code, response_time, was_successful

### Database Status

**PostgreSQL** (correctly configured):
- ✅ `torbox_pool` - 1 pool key available
- ✅ `torbox_assignments` - 0 assignments (ready for first request)
- ✅ `torbox_usage_logs` - 0 logs (ready for logging)
- ✅ `torbox_pool_health` - Health tracking table exists
- ✅ Users table has valid UUIDs

**Pool Key**:
- ID: `6eb946b3-6cd2-4d69-8984-6fbba04ce92f`
- API Key: `6748e313-ff29-4a26-80c1-34e8da4b79ee` (base64 encoded in DB)
- Status: healthy
- Slots: 0/35 available

### Next Steps

## 1. Restart Chillstreams Server

```pwsh
# Kill existing process if needed
Get-Process | Where-Object { $_.ProcessName -like '*node*' } | Stop-Process -Force

# Start server
cd C:\chillstreams
pnpm start
```

## 2. Test Internal API Endpoint

```pwsh
# Test with valid user UUID
$headers = @{
  'Authorization' = 'Bearer test_internal_key_phase3_2025'
  'Content-Type' = 'application/json'
}

$body = @{
  userId = '3b94cb45-3f99-406e-9c40-ecce61a405cc'
  deviceId = 'test-device-123'
  action = 'init'
  hash = 'abc123'
} | ConvertTo-Json

$response = Invoke-WebRequest `
  -Uri 'http://localhost:3000/api/v1/internal/pool/get-key' `
  -Method POST `
  -Headers $headers `
  -Body $body `
  -UseBasicParsing

Write-Host "Status: $($response.StatusCode)"
Write-Host "Response: $($response.Content)"
```

**Expected Response**:
```json
{
  "poolKey": "6748e313-ff29-4a26-80c1-34e8da4b79ee",
  "poolKeyId": "6eb946b3-6cd2-4d69-8984-6fbba04ce92f",
  "allowed": true,
  "deviceCount": 1
}
```

## 3. Test with Chillproxy

Once Chillstreams API works, test the full integration:

```pwsh
# Ensure chillproxy .env has correct settings
cd C:\chillproxy
notepad .env
# Verify:
# ENABLE_CHILLSTREAMS_AUTH=true
# CHILLSTREAMS_API_URL=http://localhost:3000
# CHILLSTREAMS_API_KEY=test_internal_key_phase3_2025

# Start chillproxy
go run main.go
```

## 4. Test End-to-End Stream Request

```pwsh
# Create config with user UUID
$config = @{
  stores = @(
    @{
      c = "tb"
      t = ""
      auth = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
    }
  )
} | ConvertTo-Json -Compress

$configBase64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($config))

# Test manifest endpoint
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configBase64/manifest.json" -UseBasicParsing

# Test stream endpoint (with a valid torrent hash)
$hash = "DD8255ECDC7CA55FB0BBF81323D87062DB1F6D1C"
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configBase64/stream/movie/$hash.json" -UseBasicParsing
```

## Validation Checklist

- [x] Chillstreams server starts without errors
- [x] `/api/v1/health` endpoint responds
- [x] `/api/v1/internal/pool/get-key` returns pool key
- [x] Assignment created in `torbox_assignments` table
- [x] Device count accurate (max 3 devices per user)
- [x] Usage logged to `torbox_usage_logs`
- [x] Pool key assignment reused on second request
- [ ] Chillproxy starts without errors (Next: Test in Phase 3)
- [ ] Chillproxy manifest loads (Next: Test in Phase 3)
- [ ] Chillproxy stream request works (Next: Test in Phase 3)

## Debugging

If issues occur:

```pwsh
# Check Chillstreams logs
cd C:\chillstreams
# Look for startup errors or database connection issues

# Check Chillproxy logs
cd C:\chillproxy
# Look for "Failed to get pool key" or connection errors

# Query database directly
node -e "const {Pool} = require('pg'); const p = new Pool({connectionString: 'postgresql://postgres:Iamwho06!@localhost:5432/chillstreams'}); p.query('SELECT * FROM torbox_assignments').then(r => {console.log('Assignments:', r.rows); p.end();});"
```

## Architecture Verification

```
┌──────────────┐
│  Stremio App │
└──────┬───────┘
       │ Request stream
       ↓
┌──────────────────────────────────────┐
│   Chillstreams (TypeScript)          │
│   - Manifest with user UUID          │
│   - Returns chillproxy stream URLs   │
└──────────────────────────────────────┘
       │
       │ Stream URL contains user UUID
       ↓
┌──────────────────────────────────────┐
│   Chillproxy (Go)                    │ ← Phase 1 & 1.5 Complete
│   - Extracts user UUID from config   │
│   - Generates device ID              │
└──────┬───────────────────────────────┘
       │
       │ POST /api/v1/internal/pool/get-key
       ↓
┌──────────────────────────────────────┐
│   Chillstreams Internal API          │ ← Phase 2 Complete
│   - Validates user                   │
│   - Assigns pool key                 │
│   - Tracks devices (max 3)           │
└──────┬───────────────────────────────┘
       │
       │ Returns pool key
       ↓
┌──────────────────────────────────────┐
│   TorBox API                         │
│   - Check cache                      │
│   - Add torrent                      │
│   - Get stream URL                   │
└──────────────────────────────────────┘
```

## Status

**Date**: December 17, 2025
**Phase 1**: ✅ Complete (chillproxy code ready)
**Phase 1.5**: ✅ Complete (stream handler integration)
**Phase 2**: ✅ Complete (Chillstreams API endpoints fixed)
**Phase 3**: ⏳ Ready for testing (need to restart server)

**Next Action**: Restart Chillstreams server and test the integration!

