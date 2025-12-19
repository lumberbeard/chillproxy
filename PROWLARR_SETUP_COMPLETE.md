# Prowlarr Setup & Chillproxy Integration Guide

**Date**: December 18, 2025  
**Goal**: Get Prowlarr configured and connected to Chillproxy for torrent scraping

---

## Step 1: Verify Prowlarr is Running

First, let's verify Prowlarr is accessible and get your API credentials.

**Check Prowlarr UI**:
- Open: http://localhost:9696/
- Login with your username/password
- You should see the dashboard

---

## Step 2: Verify Your Indexers Are Enabled

In Prowlarr UI:

1. **Settings** → **Indexers**
2. Verify these are enabled (checkmark visible):
   - ✅ YTS (Movies)
   - ✅ EZTV (TV)
   - ✅ RARBG (Movies/TV)
   - ✅ The Pirate Bay (Movies/TV)
   - ✅ TorrentGalaxy (Movies/TV)

3. **Disable all others** (to keep it fast and clean)

---

## Step 3: Get Your Prowlarr API Key

In Prowlarr UI:

1. **Settings** → **General**
2. Scroll down to "Security"
3. Look for **API Key** (long string of characters)
4. Copy it (you'll need this)

**Example**: `abc123def456ghi789jkl012mno345`

---

## Step 4: Get Your Torznab URL

In Prowlarr UI:

1. **Settings** → **Apps** (or **Integration**)
2. Look for **Torznab URL** or **Torznab Feed**
3. Copy the full URL

**Example**: `http://localhost:9696/api/v2.0/indexers/all/results/torznab`

---

## Step 5: Test Prowlarr Directly

Before connecting to Chillproxy, let's test Prowlarr works:

**In PowerShell**:

```powershell
# Set your API key
$apiKey = "YOUR_PROWLARR_API_KEY"

# Test a search
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=tvsearch&q=breaking+bad&season=1&ep=1&apikey=$apiKey"

# Make the request
$result = Invoke-WebRequest -Uri $url -UseBasicParsing
Write-Host "Status: $($result.StatusCode)"
Write-Host "Response Length: $($result.Content.Length) bytes"

# If successful, should be 200 with XML content
if ($result.StatusCode -eq 200) {
    Write-Host "✅ Prowlarr is working!"
    Write-Host "Response preview:"
    Write-Host $result.Content.Substring(0, 500)
} else {
    Write-Host "❌ Error: $($result.StatusCode)"
}
```

**What to expect**:
- Status: 200
- Response: XML with `<rss>` tags and torrent results

---

## Step 6: Create Chillproxy Configuration

Now let's create the configuration that Chillproxy will use.

**Create this JSON** (save to a file or use in PowerShell):

```json
{
  "stores": [
    {
      "c": "tb",
      "t": "",
      "auth": "3b94cb45-3f99-406e-9c40-ecce61a405cc"
    }
  ],
  "indexers": [
    {
      "url": "http://localhost:9696/api/v2.0/indexers/all/results/torznab",
      "apiKey": "YOUR_PROWLARR_API_KEY"
    }
  ]
}
```

**Replace**:
- `YOUR_PROWLARR_API_KEY` with your actual Prowlarr API key

---

## Step 7: Base64 Encode the Configuration

**In PowerShell**:

```powershell
# Create the config JSON
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

# Convert to JSON string
$configJson = $config | ConvertTo-Json -Compress
Write-Host "Config JSON:"
Write-Host $configJson
Write-Host ""

# Base64 encode it
$configBytes = [System.Text.Encoding]::UTF8.GetBytes($configJson)
$configBase64 = [Convert]::ToBase64String($configBytes)

Write-Host "Base64 Encoded Config:"
Write-Host $configBase64
Write-Host ""

# Save to file for easy reference
$configBase64 | Out-File -FilePath prowlarr_config.txt
Write-Host "✅ Saved to prowlarr_config.txt"
```

**Save the base64 string** - you'll use this for testing.

---

## Step 8: Test Chillproxy Manifest Endpoint

Make sure Chillproxy is running:

```powershell
# Read your base64 config
$configBase64 = Get-Content prowlarr_config.txt

# Test the manifest endpoint
$manifestUrl = "http://localhost:8080/stremio/torz/$configBase64/manifest.json"
Write-Host "Testing manifest:"
Write-Host "URL: $manifestUrl"
Write-Host ""

try {
    $response = Invoke-WebRequest -Uri $manifestUrl -UseBasicParsing
    Write-Host "✅ Status: $($response.StatusCode)"
    
    # Parse and display response
    $manifest = $response.Content | ConvertFrom-Json
    Write-Host "Manifest info:"
    Write-Host "  ID: $($manifest.id)"
    Write-Host "  Name: $($manifest.name)"
    Write-Host "  Resources: $($manifest.resources.Count)"
    Write-Host "  Catalogs: $($manifest.catalogs.Count)"
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)"
}
```

**Expected output**:
```
Status: 200
Manifest info:
  ID: com.stremthru.torz
  Name: StremThru Torz
  Resources: 1
  Catalogs: 0
```

---

## Step 9: Test Stream Search

Now test if Chillproxy can search via Prowlarr:

```powershell
# Read your base64 config
$configBase64 = Get-Content prowlarr_config.txt

# Test stream endpoint (Breaking Bad S01E01)
$streamUrl = "http://localhost:8080/stremio/torz/$configBase64/stream/series/tt0903747:1:1.json"
Write-Host "Testing stream search:"
Write-Host "URL: $streamUrl"
Write-Host ""

try {
    $response = Invoke-WebRequest -Uri $streamUrl -UseBasicParsing
    Write-Host "✅ Status: $($response.StatusCode)"
    
    # Parse response
    $streams = $response.Content | ConvertFrom-Json
    Write-Host "Results found: $($streams.streams.Count)"
    
    if ($streams.streams.Count -gt 0) {
        Write-Host ""
        Write-Host "First 3 streams:"
        $streams.streams[0..2] | ForEach-Object {
            Write-Host "  - $($_.title)"
            Write-Host "    URL: $(if ($_.url.Length -gt 80) { $_.url.Substring(0, 80) + '...' } else { $_.url })"
        }
    } else {
        Write-Host "⚠️  No streams found. Checking logs..."
    }
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)"
}
```

**Expected output**:
```
Status: 200
Results found: 15
First 3 streams:
  - Breaking Bad S01E01 1080p WEB-DL
    URL: magnet:?xt=urn:btih:ABC123...
  - Breaking Bad S01E01 720p HDTV
    URL: magnet:?xt=urn:btih:DEF456...
  ...
```

---

## Step 10: Verify Pool Key Assignment

Check that the pool key system is working:

```powershell
# Test pool key endpoint
$headers = @{
  'Authorization' = 'Bearer test_internal_key_phase3_2025'
  'Content-Type' = 'application/json'
}

$body = @{
  userId = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
  deviceId = "test-device-123"
  action = "stream-served"
  hash = "abc123"
} | ConvertTo-Json

Write-Host "Testing pool key assignment:"
$response = Invoke-WebRequest -Uri "http://localhost:3000/api/v1/internal/pool/get-key" `
  -Method POST `
  -Headers $headers `
  -Body $body `
  -UseBasicParsing

Write-Host "✅ Status: $($response.StatusCode)"

$result = $response.Content | ConvertFrom-Json
Write-Host "Response:"
Write-Host "  allowed: $($result.allowed)"
Write-Host "  poolKey: $(if ($result.poolKey) { $result.poolKey.Substring(0, 20) + '...' } else { 'N/A' })"
Write-Host "  deviceCount: $($result.deviceCount)"
```

**Expected output**:
```
Status: 200
Response:
  allowed: true
  poolKey: Njc0OGUzMTMtZmYyOS00...
  deviceCount: 1
```

---

## Troubleshooting

### Problem: Prowlarr returning 404

**Check**:
1. Prowlarr is running: `http://localhost:9696`
2. Indexers are enabled in Settings → Indexers
3. API key is correct
4. Torznab URL is correct

**Test directly**:
```powershell
$apiKey = "YOUR_KEY"
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=test&apikey=$apiKey"
Invoke-WebRequest -Uri $url -UseBasicParsing
```

---

### Problem: Chillproxy returning 0 streams

**Check**:
1. Prowlarr test above works
2. Config JSON is valid
3. Base64 encoding is correct
4. Chillproxy is running

**Debug**:
```powershell
# Check if Chillproxy can reach Prowlarr
Invoke-WebRequest -Uri "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=test&apikey=YOUR_KEY" -UseBasicParsing
```

---

### Problem: Pool key returns "not allowed"

**Check**:
1. User exists in Chillstreams database
2. Pool key exists and is active
3. User is assigned to pool key
4. Internal API key is correct

**Verify in database**:
```sql
-- Check user exists
SELECT * FROM users WHERE uuid = '3b94cb45-3f99-406e-9c40-ecce61a405cc';

-- Check pool key exists
SELECT * FROM torbox_pool LIMIT 1;

-- Check assignment
SELECT * FROM torbox_assignments WHERE user_id = '3b94cb45-3f99-406e-9c40-ecce61a405cc';
```

---

## Complete Configuration Script

Save this as `setup-prowlarr.ps1`:

```powershell
Write-Host "=== Prowlarr Setup Script ===" -ForegroundColor Green
Write-Host ""

# User input
$prowlarUrl = Read-Host "Prowlarr URL (default: http://localhost:9696)"
if ([string]::IsNullOrEmpty($prowlarUrl)) { $prowlarUrl = "http://localhost:9696" }

$apiKey = Read-Host "Prowlarr API Key"
if ([string]::IsNullOrEmpty($apiKey)) {
    Write-Host "❌ API Key required!" -ForegroundColor Red
    exit
}

$userUuid = Read-Host "Chillstreams User UUID (default: 3b94cb45-3f99-406e-9c40-ecce61a405cc)"
if ([string]::IsNullOrEmpty($userUuid)) { $userUuid = "3b94cb45-3f99-406e-9c40-ecce61a405cc" }

Write-Host ""
Write-Host "=== Step 1: Test Prowlarr ===" -ForegroundColor Cyan
$testUrl = "$prowlarUrl/api/v2.0/indexers/all/results/torznab?t=search&q=test&apikey=$apiKey"
try {
    $testResponse = Invoke-WebRequest -Uri $testUrl -UseBasicParsing -TimeoutSec 5
    if ($testResponse.StatusCode -eq 200) {
        Write-Host "✅ Prowlarr is responding" -ForegroundColor Green
    }
} catch {
    Write-Host "❌ Prowlarr not responding: $($_.Exception.Message)" -ForegroundColor Red
    exit
}

Write-Host ""
Write-Host "=== Step 2: Create Configuration ===" -ForegroundColor Cyan
$config = @{
  stores = @(
    @{
      c = "tb"
      t = ""
      auth = $userUuid
    }
  )
  indexers = @(
    @{
      url = "$prowlarUrl/api/v2.0/indexers/all/results/torznab"
      apiKey = $apiKey
    }
  )
}

$configJson = $config | ConvertTo-Json -Compress
$configBytes = [System.Text.Encoding]::UTF8.GetBytes($configJson)
$configBase64 = [Convert]::ToBase64String($configBytes)

Write-Host "✅ Configuration created" -ForegroundColor Green
Write-Host ""

Write-Host "=== Step 3: Save Configuration ===" -ForegroundColor Cyan
$configBase64 | Out-File -FilePath prowlarr_config.txt
Write-Host "✅ Saved to prowlarr_config.txt" -ForegroundColor Green
Write-Host ""

Write-Host "=== Step 4: Test Chillproxy Manifest ===" -ForegroundColor Cyan
$manifestUrl = "http://localhost:8080/stremio/torz/$configBase64/manifest.json"
try {
    $manifestResponse = Invoke-WebRequest -Uri $manifestUrl -UseBasicParsing -TimeoutSec 5
    if ($manifestResponse.StatusCode -eq 200) {
        Write-Host "✅ Manifest endpoint working" -ForegroundColor Green
        $manifest = $manifestResponse.Content | ConvertFrom-Json
        Write-Host "   ID: $($manifest.id)"
        Write-Host "   Name: $($manifest.name)"
    }
} catch {
    Write-Host "⚠️  Manifest endpoint error: $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Step 5: Test Stream Search ===" -ForegroundColor Cyan
$streamUrl = "http://localhost:8080/stremio/torz/$configBase64/stream/series/tt0903747:1:1.json"
try {
    $streamResponse = Invoke-WebRequest -Uri $streamUrl -UseBasicParsing -TimeoutSec 10
    if ($streamResponse.StatusCode -eq 200) {
        $streams = $streamResponse.Content | ConvertFrom-Json
        Write-Host "✅ Stream search working" -ForegroundColor Green
        Write-Host "   Found $($streams.streams.Count) results"
        
        if ($streams.streams.Count -gt 0) {
            Write-Host "   First result: $($streams.streams[0].title)"
        }
    }
} catch {
    Write-Host "⚠️  Stream search error: $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Configuration Ready ===" -ForegroundColor Green
Write-Host "Config file: prowlarr_config.txt"
Write-Host "Config (base64):"
Write-Host $configBase64
Write-Host ""
Write-Host "Use this in Stremio:"
Write-Host "http://localhost:8080/stremio/torz/$configBase64/manifest.json"
```

**Run it**:
```powershell
.\setup-prowlarr.ps1
```

---

## Next Steps

1. ✅ Run the setup script above
2. ✅ Verify all tests pass
3. ✅ Copy the base64 config
4. ✅ Add manifest URL to Stremio
5. ✅ Test searching for content in Stremio

---

**Status**: Ready to configure  
**Time Estimate**: 15 minutes  
**Success Indicator**: Stream search returns 5+ results

