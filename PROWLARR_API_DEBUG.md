# 🔍 PROWLARR API DEBUGGING - REAL ERRORS FOUND

**Date**: December 18, 2025  
**Status**: ❌ **PROWLARR API RETURNING 404 ERRORS**

---

## ❌ The Real Problem

When testing with correct header authentication:

```powershell
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
$url = 'http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=matrix'
$r = Invoke-WebRequest -Uri $url -Headers $headers -UseBasicParsing -TimeoutSec 20
```

**Result**: ❌ **404 Not Found**

This means:
1. Prowlarr is running ✅ (responds to requests)
2. API endpoint doesn't exist at that path ❌
3. OR Prowlarr requires additional authentication ❌
4. OR Prowlarr settings need configuration ❌

---

## 🔍 Investigation Results

### What Works
- ✅ `http://localhost:9696` - Prowlarr UI responds
- ✅ `http://localhost:9696/` - Main page loads

### What Fails (404)
- ❌ `/api/v2.0/indexers/all/results/torznab` - Returns 404
- ❌ Even with `X-Api-Key` header - Still 404

---

## 🎯 Possible Root Causes

### 1. **Prowlarr API Not Enabled**
Prowlarr may have API disabled in settings

**Fix**:
1. Open `http://localhost:9696`
2. Go to Settings → General
3. Enable "API Enabled" checkbox
4. Restart Prowlarr

### 2. **Wrong API Path Version**
API endpoint path might be different

**Possible paths to try**:
- `/api/indexers/all/results/torznab` (no v2.0)
- `/api/v1.0/indexers/all/results/torznab`
- `/indexers/all/results/torznab`

### 3. **Authentication Required Before API Access**
Prowlarr may require authentication token first

**Steps**:
1. Get JWT token from `/api/authentication`
2. Use token in `Authorization: Bearer {token}` header
3. Then call search endpoint

### 4. **Indexers Not Configured**
API path `/indexers/all` requires at least one indexer enabled

**Fix**:
1. Go to Settings → Indexers
2. Add/enable indexers (YTS, EZTV, RARBG, etc.)
3. Test API again

---

## 📋 Prowlarr Configuration Checklist

**Required Settings to Check**:

- [ ] **General Settings**:
  - [ ] API Enabled: **ON**
  - [ ] API Key set: **YES** (should be `f963a60693dd49a08ff75188f9fc72d2`)
  - [ ] API Port: **9696**
  - [ ] Branch: **Stable** (or matching your version)

- [ ] **Indexers**:
  - [ ] At least 1 indexer enabled
  - [ ] Recommended: YTS, EZTV, RARBG, TPB, TorrentGalaxy

- [ ] **Advanced Settings** (if applicable):
  - [ ] Allow requests from localhost: **ON**
  - [ ] Authentication required: Check if API exempt

---

## 🧪 Proper Prowlarr API Testing Steps

### Step 1: Get API Info (No Auth Needed)
```powershell
# This should work without header
$r = Invoke-WebRequest -Uri "http://localhost:9696/api" -UseBasicParsing -TimeoutSec 5
# If 404 = API disabled
# If 200 = API enabled
```

### Step 2: Get Indexers List
```powershell
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
$r = Invoke-WebRequest -Uri "http://localhost:9696/api/indexers" -Headers $headers -UseBasicParsing -TimeoutSec 5
# Lists all configured indexers
```

### Step 3: Search via Torznab
```powershell
# TRY DIFFERENT PATHS:
$paths = @(
    "/api/indexers/1/results/torznab?t=search&q=matrix",
    "/api/indexers/all/results/torznab?t=search&q=matrix",
    "/torznab/1/api?t=search&q=matrix",
    "/api/v2.0/search?query=matrix"
)

foreach ($path in $paths) {
    Write-Host "Testing: $path"
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:9696$path" -Headers $headers -UseBasicParsing -TimeoutSec 5
        Write-Host "✅ Found: $($r.StatusCode)"
        break
    } catch {
        Write-Host "❌ Failed: $($_.Exception.Message)"
    }
}
```

---

## 🔧 How to Fix

### Option 1: Enable API in Prowlarr UI

1. Open `http://localhost:9696` in browser
2. Click **Settings** (gear icon)
3. Go to **General** tab
4. Find **API** section
5. Verify **API Enabled** is checked
6. Verify your API Key is shown
7. Click **Save**
8. Restart Prowlarr

### Option 2: Check Prowlarr Config File

**Windows Location**:
```
C:\ProgramData\Prowlarr\config.xml
```

**Look for**:
```xml
<ApiKey>f963a60693dd49a08ff75188f9fc72d2</ApiKey>
<ApiEnabled>true</ApiEnabled>
<Port>9696</Port>
```

If not there or false, update it manually.

### Option 3: Reinstall Prowlarr

If all else fails:
```powershell
# Stop service
Stop-Service Prowlarr

# Backup config
Copy-Item "C:\ProgramData\Prowlarr" "C:\ProgramData\Prowlarr.backup"

# Uninstall and reinstall
```

---

## ✅ What We Need to Do

1. **Check Prowlarr Settings** - Ensure API is enabled
2. **Find correct API path** - Test different endpoint paths
3. **Verify indexers configured** - Need at least one
4. **Test with correct path** - Once we find working endpoint
5. **Update Chillproxy integration** - Use correct endpoint

---

## 📝 Test Commands (Run These)

Copy and paste these one by one:

```powershell
# Test 1: Check if Prowlarr is running
Write-Host "Test 1: Basic Prowlarr connection"
$r = Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 3 -ErrorAction SilentlyContinue
if ($r) { Write-Host "✅ Prowlarr is running" } else { Write-Host "❌ Prowlarr not responding" }
```

```powershell
# Test 2: Check if API root exists
Write-Host "Test 2: API root endpoint"
$r = Invoke-WebRequest -Uri "http://localhost:9696/api" -UseBasicParsing -TimeoutSec 3 -ErrorAction SilentlyContinue
if ($r.StatusCode -eq 200) { Write-Host "✅ API is enabled" } else { Write-Host "❌ API returned $($r.StatusCode)" }
```

```powershell
# Test 3: Check indexers list
Write-Host "Test 3: List indexers"
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
$r = Invoke-WebRequest -Uri "http://localhost:9696/api/indexers" -Headers $headers -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
if ($r.StatusCode -eq 200) { Write-Host "✅ Can list indexers"; $r.Content | ConvertFrom-Json | ForEach-Object { Write-Host "  - $($_.name)" } } else { Write-Host "❌ Failed: $($r.StatusCode)" }
```

```powershell
# Test 4: Try different search paths
Write-Host "Test 4: Finding correct search endpoint"
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}

$paths = @(
    "/api/indexers/1/results/torznab?t=search&q=matrix",
    "/api/v1/search?query=matrix",
    "/api/search?query=matrix",
    "/api/indexers/all/results/torznab?t=search&q=matrix"
)

foreach ($path in $paths) {
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:9696$path" -Headers $headers -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        Write-Host "✅ FOUND WORKING PATH: $path (Status: $($r.StatusCode))"
        break
    } catch {
        Write-Host "❌ $path - Error: $($_.Exception.Response.StatusCode)"
    }
}
```

---

## 🎯 Next Steps

1. **Run the test commands above** - Find which endpoint works
2. **Report back** - Tell me which status codes you get
3. **Check Prowlarr settings** - Enable API if disabled
4. **Once working** - We'll update Chillproxy with correct endpoint

---

**Status**: ❌ **API ENDPOINT NOT FOUND - NEEDS INVESTIGATION**  
**Blocker**: Prowlarr API path is not responding  
**Action Required**: Check Prowlarr settings and API configuration


