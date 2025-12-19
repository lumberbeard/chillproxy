# Prowlarr + Chillproxy Integration - Quick Start

**Your API Key**: `f963a60693dd49a08ff75188f9fc72d2`  
**Date**: December 18, 2025

---

## What You Have

✅ Prowlarr installed on `http://localhost:9696`  
✅ Prowlarr API key: `f963a60693dd49a08ff75188f9fc72d2`  
✅ Indexers enabled (YTS, EZTV, RARBG, TPB, TorrentGalaxy)  
✅ Chillproxy running on `http://localhost:8080`  
✅ Chillstreams running on `http://localhost:3000`

---

## Your Configuration (Base64 Encoded)

```
eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0=
```

---

## Your Manifest URL

```
http://localhost:8080/stremio/torz/eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0=/manifest.json
```

---

## Testing - Copy and Paste These Commands

### Test 1: Check Prowlarr is Running
```powershell
Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 3
```
**Expected**: Status code 200

---

### Test 2: Test Prowlarr Search
```powershell
$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=tvsearch&q=breaking+bad&season=1&ep=1&apikey=$apiKey"
$r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 10
Write-Host "Status: $($r.StatusCode)"
Write-Host "Has results: $($r.Content.Contains('<item>'))"
```
**Expected**: Status 200 and contains `<item>` tags (XML with torrents)

---

### Test 3: Check Chillstreams is Running
```powershell
Invoke-WebRequest -Uri "http://localhost:3000/api/v1/health" -UseBasicParsing -TimeoutSec 3
```
**Expected**: Status code 200

---

### Test 4: Check Chillproxy is Running
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 3
```
**Expected**: Status code 200

---

### Test 5: Get Chillproxy Manifest
```powershell
$config = "eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0="
$r = Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$config/manifest.json" -UseBasicParsing -TimeoutSec 5
$manifest = $r.Content | ConvertFrom-Json
Write-Host "Manifest: $($manifest.name)"
Write-Host "Resources: $($manifest.resources.Count)"
```
**Expected**: Status 200 and manifest loaded with addon name

---

### Test 6: Search for Content (The Main Test!)
```powershell
$config = "eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0="
$r = Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$config/stream/series/tt0903747:1:1.json" -UseBasicParsing -TimeoutSec 20
$streams = $r.Content | ConvertFrom-Json
Write-Host "Found $($streams.streams.Count) results"
if ($streams.streams.Count -gt 0) {
    Write-Host "First result: $($streams.streams[0].title)"
}
```
**Expected**: Status 200 and found multiple stream results (15-50 results typical)

---

### Test 7: Verify Pool Key System
```powershell
$headers = @{
    'Authorization' = 'Bearer test_internal_key_phase3_2025'
    'Content-Type' = 'application/json'
}
$body = @{
    userId = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
    deviceId = "test-device-123"
    action = "test"
    hash = "testhash"
} | ConvertTo-Json
$r = Invoke-WebRequest -Uri "http://localhost:3000/api/v1/internal/pool/get-key" -Method POST -Headers $headers -Body $body -UseBasicParsing -TimeoutSec 5
$result = $r.Content | ConvertFrom-Json
Write-Host "Allowed: $($result.allowed)"
Write-Host "Pool Key ID: $($result.poolKeyId)"
```
**Expected**: Status 200 and `allowed: true`

---

## Summary of Your Setup

| Component | URL | Status |
|-----------|-----|--------|
| Prowlarr | http://localhost:9696 | Running ✅ |
| Prowlarr API Key | f963a60693dd49a08ff75188f9fc72d2 | Ready ✅ |
| Chillproxy | http://localhost:8080 | Running ✅ |
| Chillstreams | http://localhost:3000 | Running ✅ |
| Manifest URL | See above | Ready ✅ |

---

## Next Steps

1. **Run all tests above** in PowerShell to verify everything is working
2. **If Test 6 returns results**, you're ready to use Stremio!
3. **Add manifest to Stremio**:
   - Open Stremio
   - Settings → Add-ons → Enter Manifest URL
   - Search for content

---

## If Something Fails

**Prowlarr search returning 0 results?**
- Check indexers are enabled: http://localhost:9696/settings/indexers
- Make sure YTS, EZTV, RARBG are marked as enabled
- Try searching in Prowlarr UI manually first

**Chillproxy returning error?**
- Check Chillstreams is running: `Invoke-WebRequest http://localhost:3000/api/v1/health`
- Check Chillproxy is running: `Invoke-WebRequest http://localhost:8080/health`
- Check logs for detailed error messages

**Pool key not working?**
- Verify user exists in Chillstreams database
- Verify pool key exists
- Check internal API key is correct

---

## You're All Set! 🎉

Everything is configured and ready to test. Run the tests above and let me know what happens!


