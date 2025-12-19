# NEXT STEPS: Prowlarr + Chillproxy Integration

**Date**: December 17, 2025  
**Current Status**: All architectural issues resolved  
**What's Remaining**: Practical integration testing

---

## Executive Summary for You

**You were completely right**:
1. ✅ Stremthru HAS built-in torrent indexing (`/stremio/torz/`)
2. ✅ It just needs an indexer backend (Prowlarr is perfect)
3. ✅ Prowlarr is simpler and faster than Jackett
4. ✅ No credentials needed in user manifest
5. ✅ All secrets stay server-side

**What you need to do now**:
1. Install Prowlarr (docker or binary - 5 min)
2. Enable 5 indexers (YTS, EZTV, RARBG, TPB, TG - 2 min)
3. Get Torznab URL and API key from Prowlarr (1 min)
4. Test Chillproxy with Prowlarr backend (5 min)
5. Verify pool key system still works (5 min)

**Total time**: ~15-20 minutes to full integration

---

## Step 1: Install Prowlarr

### Option A: Docker (Recommended)

```powershell
# Run Prowlarr in Docker
docker run -d `
  --name prowlarr `
  -p 9696:9696 `
  -e PUID=1000 `
  -e PGID=1000 `
  -v prowlarr_config:/config `
  lscr.io/linuxserver/prowlarr:latest

# Wait 10 seconds for startup
Start-Sleep -Seconds 10

# Verify it's running
Invoke-WebRequest -Uri http://localhost:9696 -UseBasicParsing
```

### Option B: Binary

```powershell
# Download from prowlarr.com
# Extract to C:\Prowlarr
# Run: C:\Prowlarr\Prowlarr.exe

# Then open: http://localhost:9696
```

**Verify**: Open browser to `http://localhost:9696` - you should see Prowlarr UI

---

## Step 2: Configure Prowlarr

**In Prowlarr UI** (`http://localhost:9696`):

1. **Settings** → **Indexers** → **Add Indexers**
2. **Search and enable these 5 only**:
   - ✅ YTS (movies, good quality)
   - ✅ EZTV (TV, very reliable)
   - ✅ RARBG (both, high quality)
   - ✅ The Pirate Bay (both, good coverage)
   - ✅ TorrentGalaxy (both, modern)

3. **Disable everything else** (to keep it fast)

4. **Settings** → **General**:
   - Copy your **API Key** (save this)

5. **Settings** → **Apps**:
   - Copy the **Torznab URL** (save this)

**You should have**:
```
Torznab URL: http://localhost:9696/api/v2.0/indexers/all/results/torznab
API Key: [32-character string]
```

---

## Step 3: Test Prowlarr Directly

Verify Prowlarr is working:

```powershell
# Test indexer search
$apiKey = "YOUR_PROWLARR_API_KEY"
$query = "breaking+bad"

$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=tvsearch&q=$query&apikey=$apiKey"

$result = Invoke-WebRequest -Uri $url -UseBasicParsing
Write-Host $result.StatusCode
Write-Host $result.Content

# Should return XML with torrent results
# If not, check indexers are enabled and API key is correct
```

**Expected**: XML feed with torrent results for "Breaking Bad"

---

## Step 4: Test Chillproxy + Prowlarr

### Create config JSON

```powershell
# Create the config object
$config = @{
  stores = @(
    @{
      c = "tb"
      t = ""
      auth = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
    }
  )
  indexers = @(
    @{
      url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab"
      apiKey = "YOUR_PROWLARR_API_KEY"
    }
  )
}

# Verify JSON is valid
$jsonString = $config | ConvertTo-Json
Write-Host "JSON is valid:"
Write-Host $jsonString

# Base64 encode it
$jsonBytes = [System.Text.Encoding]::UTF8.GetBytes($jsonString)
$configBase64 = [Convert]::ToBase64String($jsonBytes)

Write-Host "Base64 encoded config:"
Write-Host $configBase64

# Save for testing
$configBase64 | Out-File config.txt
```

### Test Chillproxy manifest

```powershell
# Read the base64 config
$configBase64 = Get-Content config.txt

# Test manifest endpoint
$manifestUrl = "http://localhost:8080/stremio/torz/$configBase64/manifest.json"
Write-Host "Testing: $manifestUrl"

$manifest = Invoke-WebRequest -Uri $manifestUrl -UseBasicParsing
Write-Host "Status: $($manifest.StatusCode)"
Write-Host "Response:"
Write-Host $manifest.Content | ConvertFrom-Json | ConvertTo-Json -Depth 3
```

**Expected**: JSON manifest with `resources` and `catalogs` arrays

---

## Step 5: Test Stream Search

```powershell
$configBase64 = Get-Content config.txt

# Test stream endpoint for Breaking Bad S01E01
$streamUrl = "http://localhost:8080/stremio/torz/$configBase64/stream/series/tt0903747:1:1.json"
Write-Host "Testing: $streamUrl"

$streams = Invoke-WebRequest -Uri $streamUrl -UseBasicParsing
Write-Host "Status: $($streams.StatusCode)"

$response = $streams.Content | ConvertFrom-Json
Write-Host "Number of streams found: $($response.streams.Count)"

# Show first 3 streams
if ($response.streams.Count -gt 0) {
  Write-Host "First 3 streams:"
  $response.streams[0..2] | ForEach-Object {
    Write-Host "  - $($_.title)"
    Write-Host "    URL: $($_.url -replace '(.{50}).*', '$1...')"
  }
} else {
  Write-Host "ERROR: No streams found!"
  Write-Host "Response: $($response | ConvertTo-Json)"
}
```

**Expected**: JSON with array of streams containing magnet links from Prowlarr results

---

## Step 6: Verify Pool Key System

```powershell
# Check if torbox_assignments table has the user
$pool = @"
SELECT user_id, pool_key_id, assigned_at FROM torbox_assignments 
WHERE user_id = '3b94cb45-3f99-406e-9c40-ecce61a405cc';
"@

Write-Host "Check assignments table:"
Write-Host $pool

# Check if any usage was logged
$usage = @"
SELECT user_id, action, hash, timestamp FROM torbox_usage_logs 
WHERE user_id = '3b94cb45-3f99-406e-9c40-ecce61a405cc'
ORDER BY timestamp DESC 
LIMIT 5;
"@

Write-Host "Check usage logs:"
Write-Host $usage
```

**Expected**: 
- Assignment row for user UUID
- Usage log entries for recent stream searches

---

## Troubleshooting

### Problem: Prowlarr not responding

```powershell
# Check if running
Get-Process | Where-Object { $_.ProcessName -like '*prowlarr*' }

# Check if port open
Get-NetTCPConnection -LocalPort 9696 -ErrorAction SilentlyContinue

# Restart if needed
docker stop prowlarr
docker start prowlarr
```

### Problem: Chillproxy returns 0 streams

1. Verify Prowlarr URL and API key are correct
2. Test Prowlarr directly with query parameter
3. Check indexers are enabled in Prowlarr
4. Check Chillproxy logs for errors

```powershell
# Test Prowlarr directly
$apiKey = "YOUR_KEY"
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=tvsearch&q=test&apikey=$apiKey"
Invoke-WebRequest -Uri $url -UseBasicParsing
```

### Problem: Pool key not working

1. Verify user exists in Chillstreams: `SELECT * FROM users WHERE uuid = '3b94cb45-...'`
2. Verify pool key exists: `SELECT * FROM torbox_pool LIMIT 1`
3. Verify assignment exists: `SELECT * FROM torbox_assignments WHERE user_id = '3b94cb45-...'`
4. Test Pool API directly:

```powershell
$headers = @{
  'Authorization' = 'Bearer test_internal_key_phase3_2025'
  'Content-Type' = 'application/json'
}

$body = @{
  userId = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
  deviceId = "test-device-123"
  action = "stream"
  hash = "test-hash"
} | ConvertTo-Json

Invoke-WebRequest -Uri "http://localhost:3000/api/v1/internal/pool/get-key" `
  -Method POST `
  -Headers $headers `
  -Body $body `
  -UseBasicParsing
```

---

## Validation Checklist

After each step, verify:

### After Prowlarr Install
- [ ] Prowlarr running on `http://localhost:9696`
- [ ] 5 indexers enabled (YTS, EZTV, RARBG, TPB, TG)
- [ ] API Key copied
- [ ] Torznab URL copied

### After Chillproxy Config
- [ ] JSON is valid (no syntax errors)
- [ ] Base64 encoding successful
- [ ] Auth UUID is correct
- [ ] Prowlarr URL is correct
- [ ] Prowlarr API key is correct

### After Manifest Test
- [ ] Status code is 200
- [ ] Response contains `resources` array
- [ ] Response contains `catalogs` array
- [ ] ID prefix contains `tt` (IMDb) or anime ID types

### After Stream Test
- [ ] Status code is 200
- [ ] Response contains `streams` array
- [ ] At least 1 stream returned
- [ ] Streams have `title`, `url`, `created` fields
- [ ] URLs start with `magnet:` or `http:`

### After Pool Key Test
- [ ] User exists in `torbox_assignments`
- [ ] Pool key is assigned
- [ ] Usage logs show recent activity
- [ ] Device tracking shows correct device ID

---

## Full Integration Test

Once everything works individually, run end-to-end:

```powershell
Write-Host "=== FULL INTEGRATION TEST ===" -ForegroundColor Green

# 1. Manifest
$config = @{ ... } | ConvertTo-Json
$configB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($config))
$r = Invoke-WebRequest "http://localhost:8080/stremio/torz/$configB64/manifest.json"
Write-Host "[✓] Manifest: $($r.StatusCode)" 

# 2. Stream search
$r = Invoke-WebRequest "http://localhost:8080/stremio/torz/$configB64/stream/series/tt0903747:1:1.json"
Write-Host "[✓] Streams: $($r.StatusCode), Found: $($r.Content | ConvertFrom-Json | % { $_.streams.Count }) results"

# 3. Pool assignment
$r = Invoke-WebRequest "http://localhost:3000/api/v1/internal/pool/get-key" -Method POST -Headers @{Authorization="Bearer test_internal_key_phase3_2025"} -Body (@{userId="3b94..."; deviceId="test"; action="test"} | ConvertTo-Json)
Write-Host "[✓] Pool key: $($r.StatusCode), Allowed: $($r.Content | ConvertFrom-Json | % { $_.allowed })"

Write-Host "=== ALL TESTS PASSED ===" -ForegroundColor Green
```

---

## What Happens Next

Once this works:

1. ✅ Chillproxy searches via Prowlarr (built-in indexing)
2. ✅ Results are checked against TorBox pool
3. ✅ Users get instant streams (cached) or wait (uncached)
4. ✅ All using shared pool keys (no user credentials)
5. ✅ Usage is logged for analytics

**Then you can**:
- Deploy to production
- Add more users
- Monitor usage logs
- Implement analytics dashboard
- Add device revocation features

---

## Quick Reference

**Key URLs**:
- Prowlarr: `http://localhost:9696`
- Chillproxy: `http://localhost:8080`
- Chillstreams: `http://localhost:3000`

**Key IDs**:
- Test User UUID: `3b94cb45-3f99-406e-9c40-ecce61a405cc`
- Test Device ID: `test-device-123`

**Key Endpoints**:
- Manifest: `/stremio/torz/{config}/manifest.json`
- Stream: `/stremio/torz/{config}/stream/{type}/{id}.json`
- Pool API: `/api/v1/internal/pool/get-key` (POST)

**Key Credentials**:
- Prowlarr API Key: [from Prowlarr UI]
- Internal API Key: `test_internal_key_phase3_2025`

---

**Status**: Ready for practical testing  
**Estimated Time**: 20 minutes to full integration  
**Next**: Run Step 1 (Install Prowlarr)

Good luck! You're very close to a working system. 🚀

