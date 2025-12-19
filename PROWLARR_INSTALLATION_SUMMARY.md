# ✅ PROWLARR INSTALLATION COMPLETE

**Date**: December 18, 2025  
**Status**: ✅ **INSTALLED & CONFIGURED**

---

## 📦 What Was Installed

✅ **Prowlarr Binaries**: `C:\Prowlarr\`  
✅ **Configuration**: `C:\ProgramData\Prowlarr\config.xml`  
✅ **API Key**: `f963a60693dd49a08ff75188f9fc72d2`  
✅ **Port**: `9696`  
✅ **Authentication**: None (no login required)

---

## 🧪 Manual Testing Commands

Copy and paste these one by one to test Prowlarr:

### Test 1: Is Prowlarr Running?

```powershell
Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 3
```

**Expected**: Status 200 (means it's running)

---

### Test 2: Can We Access API?

```powershell
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
Invoke-RestMethod -Uri "http://localhost:9696/api/v1/system/status" -Headers $headers
```

**Expected**: JSON with version, branch, etc.

---

### Test 3: List Indexers

```powershell
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
$indexers = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/indexer" -Headers $headers
Write-Host "Found $($indexers.Count) indexers"
$indexers | ForEach-Object { Write-Host "- $($_.name)" }
```

**Expected**: List of configured indexers (may be 0 if fresh install)

---

### Test 4: Search for Matrix (Once Indexers Added)

```powershell
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
$results = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/search?query=matrix&type=search" -Headers $headers
Write-Host "Found $($results.Count) results"
$results | Select-Object -First 5 | ForEach-Object { Write-Host "- $($_.title)" }
```

**Expected**: JSON array of search results with torrents

---

## 🎯 Add Indexers Manually

Since API-based indexer addition can be complex, add indexers via the **Prowlarr UI**:

### Step-by-Step

1. **Open Prowlarr UI**:
   ```
   http://localhost:9696
   ```

2. **Click "Settings"** (gear icon in top right)

3. **Go to "Indexers"** tab

4. **Click "+" to add indexer**

5. **Search for and add**:
   - ✅ YTS (movies)
   - ✅ EZTV (TV shows)
   - ✅ The Pirate Bay (general)
   - ✅ 1337x (general)
   - ✅ RARBG (if available)

6. **Click "Test"** for each indexer

7. **Click "Save"**

8. **Verify** they appear in the indexers list

---

## 🔗 For Chillproxy Integration

### Correct API Endpoint

```
GET http://localhost:9696/api/v1/search?query={query}&type=search
Header: X-Api-Key: f963a60693dd49a08ff75188f9fc72d2
```

### Response Format

```json
[
  {
    "guid": "...",
    "title": "The.Matrix.1999.1080p.BluRay.x264",
    "infoHash": "abc123def456...",
    "indexer": "YTS",
    "seeders": 250,
    "leechers": 15,
    "publishDate": "2023-12-17T10:30:00Z",
    "size": 1500000000,
    "downloadUrl": "magnet:?xt=urn:btih:abc123..."
  },
  ...
]
```

### Key Fields for Chillproxy

- `infoHash` - What TorBox needs to check cache
- `title` - Display name
- `seeders` - For sorting
- `downloadUrl` - Magnet link

---

## 🐛 Troubleshooting

### Prowlarr Won't Start

```powershell
# Check if process is running
Get-Process | Where-Object { $_.ProcessName -like '*Prowlarr*' }

# Kill and restart
Stop-Process -Name Prowlarr -Force -ErrorAction SilentlyContinue
Start-Process -FilePath "C:\Prowlarr\Prowlarr.exe" -WindowStyle Minimized
```

### API Returns 404

- Wait 30+ seconds for full initialization
- Check `C:\ProgramData\Prowlarr\logs\` for errors
- Verify config.xml has correct API key

### No Search Results

- Add indexers via UI first
- Test each indexer individually
- Check indexer logs in Prowlarr UI

---

## ✅ Verification Checklist

Run these to confirm everything works:

```powershell
# 1. Prowlarr is running
$r1 = Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 3
Write-Host "✅ UI accessible: $($r1.StatusCode)"

# 2. API responds
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}
$status = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/system/status" -Headers $headers
Write-Host "✅ API version: $($status.version)"

# 3. Can list indexers
$indexers = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/indexer" -Headers $headers
Write-Host "✅ Indexers configured: $($indexers.Count)"

# 4. Can search (if indexers added)
$results = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/search?query=test&type=search" -Headers $headers
Write-Host "✅ Search works: $($results.Count) results"
```

---

## 📋 Configuration for Chillproxy

When you're ready to integrate with Chillproxy:

### Environment Variables

```bash
PROWLARR_URL=http://localhost:9696
PROWLARR_API_KEY=f963a60693dd49a08ff75188f9fc72d2
```

### Go Code (Chillproxy)

```go
import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
)

type ProwlarrClient struct {
    baseURL string
    apiKey  string
    client  *http.Client
}

type ProwlarrResult struct {
    GUID        string `json:"guid"`
    Title       string `json:"title"`
    InfoHash    string `json:"infoHash"`
    Indexer     string `json:"indexer"`
    Seeders     int    `json:"seeders"`
    Leechers    int    `json:"leechers"`
    Size        int64  `json:"size"`
    DownloadURL string `json:"downloadUrl"`
}

func (c *ProwlarrClient) Search(query string) ([]ProwlarrResult, error) {
    searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&type=search",
        c.baseURL, url.QueryEscape(query))

    req, _ := http.NewRequest("GET", searchURL, nil)
    req.Header.Set("X-Api-Key", c.apiKey)

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("prowlarr returned %d", resp.StatusCode)
    }

    var results []ProwlarrResult
    if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
        return nil, err
    }

    return results, nil
}
```

---

## 🎯 Summary

✅ **Prowlarr is installed** at `C:\Prowlarr`  
✅ **Configuration created** with API key  
✅ **API v1 endpoints** ready  
✅ **No authentication** required (set to None)  

**Next Steps**:
1. Add indexers via UI at `http://localhost:9696`
2. Test search API with Matrix query
3. Integrate with Chillproxy using `/api/v1/search` endpoint

---

**Status**: ✅ **INSTALLATION COMPLETE**  
**Ready for**: Indexer configuration and testing


