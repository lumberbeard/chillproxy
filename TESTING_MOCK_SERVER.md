# Chillproxy Testing Guide - Mock Server Setup

## Overview

This guide walks you through testing **chillproxy** end-to-end using a mock Chillstreams API server.

---

## Prerequisites

✅ **Required**:
- Node.js installed (for mock server)
- Go installed (for chillproxy)
- TorBox API key (for real torrent testing)

✅ **Already Complete**:
- Phase 1 & 1.5 code implemented
- chillproxy compiles successfully

---

## Step 1: Configure TorBox API Key

### Option A: Environment Variable (Recommended)
```powershell
# Set TorBox API key
$env:TORBOX_API_KEY = "your_torbox_api_key_here"
```

### Option B: Edit Mock Server File
Open `mock-chillstreams-api.js` and edit line 12:
```javascript
apiKey: 'your_torbox_api_key_here'
```

---

## Step 2: Start Mock Chillstreams API

```powershell
# Install dependencies (if needed)
cd C:\chillproxy
npm install express

# Start mock server
node mock-chillstreams-api.js
```

**Expected Output**:
```
🚀 Mock Chillstreams API Server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📡 Listening on: http://localhost:3000
🔗 Endpoints:
   POST /api/v1/internal/pool/get-key
   POST /api/v1/internal/pool/log-usage
   GET  /api/v1/internal/pool/stats
   GET  /health
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Ready for chillproxy testing!
```

**Keep this terminal window open!**

---

## Step 3: Configure Chillproxy Environment

Open a **new PowerShell window** and configure chillproxy:

```powershell
cd C:\chillproxy

# Copy phase 3 environment (has Chillstreams integration enabled)
Copy-Item .env.phase3 .env -Force

# Or manually set these variables in .env:
# ENABLE_CHILLSTREAMS_AUTH=true
# CHILLSTREAMS_API_URL=http://localhost:3000
# CHILLSTREAMS_API_KEY=mock-secret-key
```

**Verify `.env` has**:
```bash
ENABLE_CHILLSTREAMS_AUTH=true
CHILLSTREAMS_API_URL=http://localhost:3000
CHILLSTREAMS_API_KEY=mock-secret-key
STREMTHRU_PORT=8080
STREMTHRU_FEATURE=+stremio-torz
```

---

## Step 4: Build and Start Chillproxy

```powershell
# Build chillproxy
go build -o chillproxy.exe .

# Start chillproxy
.\chillproxy.exe
```

**Expected Output**:
```
INFO  starting stremthru server on :8080
INFO  features enabled: stremio-torz
INFO  chillstreams auth: enabled
INFO  chillstreams api: http://localhost:3000
```

**Keep this terminal window open too!**

---

## Step 5: Test the Integration

### Test 1: Health Check

**New PowerShell window**:
```powershell
# Test chillproxy health
Invoke-WebRequest -Uri http://localhost:8080/health

# Test mock API health
Invoke-WebRequest -Uri http://localhost:3000/health
```

### Test 2: Create Test Config

```powershell
# Create a test config with Chillstreams auth
$config = @{
    stores = @(
        @{
            c = "tb"
            t = ""
            auth = "test-user-uuid-12345"
        }
    )
} | ConvertTo-Json -Compress

# Base64 encode the config
$bytes = [System.Text.Encoding]::UTF8.GetBytes($config)
$base64 = [Convert]::ToBase64String($bytes)

Write-Host "Config (base64): $base64"
```

### Test 3: Request Manifest

```powershell
# Request manifest from chillproxy
$manifestUrl = "http://localhost:8080/stremio/torz/$base64/manifest.json"
Invoke-WebRequest -Uri $manifestUrl | Select-Object StatusCode, Content
```

**Expected**:
- Status: 200 OK
- Content: JSON manifest with Torz addon info

### Test 4: Request Stream (Real Test!)

```powershell
# Test with a popular movie (e.g., The Matrix)
# IMDb ID: tt0133093
$streamUrl = "http://localhost:8080/stremio/torz/$base64/stream/movie/tt0133093.json"

Invoke-WebRequest -Uri $streamUrl | Select-Object StatusCode, Content
```

**What Should Happen**:

1. **Chillproxy receives request**
2. **Calls mock API**: `POST /api/v1/internal/pool/get-key`
   - Mock API returns TorBox pool key
3. **Chillproxy uses pool key** to call TorBox API
4. **Returns stream results** to you

**Check Mock Server Terminal**:
```
📥 GET-KEY REQUEST: {
  userId: 'test-user-uuid-12345',
  deviceId: 'abc123...',
  action: 'check-cache',
  hash: '...'
}
✅ Pool key assigned
```

**Check Chillproxy Terminal**:
```
INFO  processing stream request
INFO  fetching pool key from chillstreams
INFO  pool key received: pool-key-1
INFO  checking torbox cache
INFO  streams found: 5
```

---

## Step 6: View Stats

```powershell
# View mock API statistics
Invoke-WebRequest -Uri http://localhost:3000/api/v1/internal/pool/stats | 
  ConvertFrom-Json | 
  ConvertTo-Json -Depth 10
```

**Shows**:
- Device assignments
- Usage logs
- Pool key info

---

## Troubleshooting

### Mock Server Not Starting

**Error**: `Cannot find module 'express'`

**Fix**:
```powershell
cd C:\chillproxy
npm install express
```

### Chillproxy Can't Connect to Mock API

**Error**: `failed to get pool key from chillstreams: connection refused`

**Fix**:
1. Verify mock server is running on port 3000
2. Check `.env` has: `CHILLSTREAMS_API_URL=http://localhost:3000`
3. Restart chillproxy

### TorBox Returns 401 Unauthorized

**Error**: `Please provide a valid API token`

**Fix**:
1. Set real TorBox API key in mock server
2. Restart mock server
3. Try request again

### No Streams Returned

**Possible Causes**:
- Invalid IMDb ID (try: `tt0133093` for The Matrix)
- TorBox API key invalid
- Movie not cached on TorBox

**Debug**:
```powershell
# Check TorBox API directly
Invoke-WebRequest -Uri "https://api.torbox.app/v1/api/torrents/checkcached?hash=test" `
  -Headers @{"Authorization" = "Bearer YOUR_TORBOX_KEY"}
```

---

## Success Criteria

✅ **Phase 1.5 Testing Complete When**:

1. ✅ Mock server starts on port 3000
2. ✅ Chillproxy starts on port 8080
3. ✅ Manifest request succeeds (200 OK)
4. ✅ Stream request succeeds (200 OK)
5. ✅ Mock server logs show pool key assignment
6. ✅ Chillproxy logs show TorBox integration
7. ✅ Usage logging succeeds

---

## What's Next After Testing?

Once mock server tests pass:

### Phase 2: Build Real Chillstreams API

**Tasks**:
1. Implement `POST /api/v1/internal/pool/get-key` in Chillstreams
2. Implement `POST /api/v1/internal/pool/log-usage` in Chillstreams
3. Connect to PostgreSQL (torbox_pool tables)
4. Implement pool key rotation
5. Implement device tracking
6. Deploy and test with production Chillstreams

### Phase 3: Production Deployment

**Tasks**:
1. Docker compose setup (chillstreams + chillproxy)
2. Environment configuration
3. SSL/TLS setup
4. Monitoring and logging
5. Load testing

---

## Quick Reference

### Ports
- Mock API: `http://localhost:3000`
- Chillproxy: `http://localhost:8080`

### Test User
- UUID: `test-user-uuid-12345`
- Can be any valid UUID format

### Test Movie
- The Matrix: `tt0133093`
- Inception: `tt1375666`
- Interstellar: `tt0816692`

---

**Status**: Ready to test! Follow Step 1 and proceed through the guide.

