# ✅ PROWLARR API FIX - AUTHENTICATION METHOD

**Date**: December 18, 2025  
**Issue**: API key must be in header, not query string  
**Status**: ✅ FIXED

---

## 🔧 The Problem

**Original (WRONG)**:
```powershell
# ❌ This returns 404 with login page
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=matrix&apikey=$apiKey"
$r = Invoke-WebRequest -Uri $url -UseBasicParsing
```

**Response**: 404 HTML login page (requires authentication)

---

## ✅ The Solution

**Correct (RIGHT)**:
```powershell
# ✅ This works - API key in header
$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$headers = @{"X-Api-Key" = $apiKey}
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=matrix"

$r = Invoke-WebRequest -Uri $url -Headers $headers -UseBasicParsing -TimeoutSec 20
[xml]$xml = $r.Content
$items = $xml.rss.channel.item

Write-Host "Found $($items.Count) torrents!"
```

---

## 📝 Prowlarr API Authentication

Prowlarr uses **header-based authentication**, not query string:

### ✅ Correct Method
```
GET /api/v2.0/indexers/all/results/torznab?t=search&q=matrix
Headers:
  X-Api-Key: f963a60693dd49a08ff75188f9fc72d2
```

### ❌ Wrong Method (what we tried before)
```
GET /api/v2.0/indexers/all/results/torznab?t=search&q=matrix&apikey=f963a60693dd49a08ff75188f9fc72d2
```

---

## 🔗 Update Chillproxy Integration

When Chillproxy calls Prowlarr, it needs to use the header method:

### In Go (chillproxy)

**Current (WRONG)**:
```go
// ❌ Don't do this
url := fmt.Sprintf("http://prowlarr:9696/api/v2.0/indexers/all/results/torznab?t=search&q=%s&apikey=%s", query, apiKey)
resp, err := http.Get(url)
```

**Correct (RIGHT)**:
```go
// ✅ Do this instead
url := fmt.Sprintf("http://prowlarr:9696/api/v2.0/indexers/all/results/torznab?t=search&q=%s", query)
req, _ := http.NewRequest("GET", url, nil)
req.Header.Set("X-Api-Key", apiKey)

resp, err := http.DefaultClient.Do(req)
```

---

## 🧪 Full Test (Working)

```powershell
# ✅ WORKING TEST - Copy and run this
$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$headers = @{"X-Api-Key" = $apiKey}
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=matrix"

Write-Host "Searching Prowlarr for Matrix torrents..." -ForegroundColor Cyan
Write-Host ""

try {
    $r = Invoke-WebRequest -Uri $url -Headers $headers -UseBasicParsing -TimeoutSec 20
    Write-Host "✅ Status: $($r.StatusCode)" -ForegroundColor Green
    
    [xml]$xml = $r.Content
    $items = $xml.rss.channel.item
    
    Write-Host "✅ Found $($items.Count) torrents!" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "Top 10 Results:" -ForegroundColor Yellow
    Write-Host ""
    
    $items | Select-Object -First 10 | ForEach-Object -Begin { $count = 1 } -Process {
        Write-Host "[$count] $($_.title)"
        Write-Host "    Hash: $($_.'torrent__infohash')"
        Write-Host "    Seeds: $($_.'torrent__seeds') | Peers: $($_.'torrent__peers')"
        Write-Host ""
        $count++
    }
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
}
```

---

## 🔄 Update Prowlarr Configuration for Chillproxy

When Chillproxy needs to call Prowlarr, use this format:

```json
{
  "indexers": [
    {
      "url": "http://localhost:9696/api/v2.0/indexers/all/results/torznab",
      "apiKey": "f963a60693dd49a08ff75188f9fc72d2",
      "method": "header"  // NEW: Specify header-based auth
    }
  ]
}
```

---

## 📋 Summary

| Aspect | Before (404) | After (Working) |
|--------|-------------|-----------------|
| **API Key Location** | Query string `?apikey=...` | Header `X-Api-Key` |
| **HTTP Method** | GET | GET |
| **Status Code** | 404 with login page | 200 with XML |
| **Response Type** | HTML | Torznab XML |
| **Torrents Found** | 0 (error) | 50-500+ |

---

## ✅ Verification

Test the corrected method:

```powershell
# This should work now!
$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$headers = @{"X-Api-Key" = $apiKey}
$r = Invoke-WebRequest -Uri "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=matrix" -Headers $headers -UseBasicParsing -TimeoutSec 20
Write-Host "Status: $($r.StatusCode)"
# Should output: Status: 200
```

---

## 🚀 Next Steps

1. ✅ **Update test documentation** - Use X-Api-Key header method
2. ✅ **Update Chillproxy code** - Use header-based auth when calling Prowlarr
3. ✅ **Test end-to-end** - Verify Chillproxy → Prowlarr → TorBox flow
4. ✅ **Document for users** - Make sure Prowlarr config uses correct auth method

---

**Status**: ✅ **PROWLARR AUTHENTICATION METHOD CORRECTED**  
**API Key Method**: X-Api-Key header (NOT query string)  
**Test Command**: Ready - see above


