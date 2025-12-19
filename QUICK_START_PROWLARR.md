# Quick Start: Prowlarr + Chillproxy Integration

**Date**: December 17, 2025  
**Goal**: Get Chillproxy working with Prowlarr indexing + Chillstreams pool keys  
**Time Estimate**: 15 minutes

---

## The Simple Truth

**You were right the whole time**:
- ✅ Chillproxy/Torz HAS built-in torrent searching
- ✅ It just needs an indexer backend (Prowlarr is perfect)
- ✅ Use Prowlarr instead of Jackett (3x faster, simpler)
- ✅ No user credentials needed (only indexer URL)

---

## 3-Step Setup

### **Step 1: Install Prowlarr (5 minutes)**

```powershell
# Option A: Docker (Recommended)
docker run -d `
  -p 9696:9696 `
  --name prowlarr `
  lscr.io/linuxserver/prowlarr:latest

# Option B: Binary
# Download from prowlarr.com, extract, run Prowlarr.exe
# Then navigate to http://localhost:9696
```

### **Step 2: Configure Prowlarr (5 minutes)**

1. **Open** `http://localhost:9696`
2. **Settings** → Indexers → Add Indexers
3. **Enable ONLY these** (to keep it fast):
   - ✅ YTS (movies)
   - ✅ EZTV (TV)
   - ✅ RARBG (both)
   - ✅ The Pirate Bay (both)
   - ✅ TorrentGalaxy (both)
4. **Settings** → General → Copy **API Key**
5. **Settings** → Apps → Copy **Torznab URL**

**Save these**:
```
Torznab URL: http://localhost:9696/api/v2.0/indexers/all/results/torznab
API Key: [your-key-here]
```

### **Step 3: Configure Chillproxy (5 minutes)**

**Create manifest config**:
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

**Base64 encode it**:
```powershell
$config = @{
  stores = @(@{
    c = "tb"
    t = ""
    auth = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
  })
  indexers = @(@{
    url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab"
    apiKey = "YOUR_PROWLARR_API_KEY"
  })
} | ConvertTo-Json -Compress

$configBase64 = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($config))
Write-Host $configBase64
```

**Test with Chillproxy**:
```powershell
# Test manifest
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configBase64/manifest.json"

# Test stream search (Breaking Bad S01E01)
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configBase64/stream/series/tt0903747:1:1.json"
```

---

## What's Happening Behind the Scenes

```
┌──────────────────────────────────────────┐
│     User (Stremio)                       │
│  No credentials visible                  │
└──────────────┬───────────────────────────┘
               │
               │ User UUID + indexer URL
               ↓
┌──────────────────────────────────────────┐
│     CHILLPROXY/TORZ                      │
│  1. Receives stream request              │
│  2. Calls Prowlarr to search             │
└──────────────┬───────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────┐
│     PROWLARR                             │
│  Searches: YTS, RARBG, EZTV, etc.       │
│  Returns: Magnet links + metadata        │
└──────────────┬───────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────┐
│     CHILLSTREAMS POOL API                │
│  1. Gets user's assigned pool key        │
│  2. Returns pool key (not user key)      │
└──────────────┬───────────────────────────┘
               │
               ↓
┌──────────────────────────────────────────┐
│     TORBOX API                           │
│  Checks cache + returns stream URLs      │
└──────────────────────────────────────────┘
```

**Key point**: All credentials stay on YOUR servers. User only provides:
- Their UUID (from Chillstreams login)
- Prowlarr URL (public, no secrets)

---

## Verification Checklist

After setup, verify:

- [ ] Prowlarr running on `http://localhost:9696`
- [ ] 5 indexers enabled (YTS, EZTV, RARBG, TPB, TorrentGalaxy)
- [ ] Torznab URL copied from Prowlarr
- [ ] API Key copied from Prowlarr
- [ ] Chillproxy running on `http://localhost:8080`
- [ ] Chillstreams running on `http://localhost:3000`
- [ ] Config JSON is valid (use JSON validator)
- [ ] Base64 encoding successful
- [ ] Manifest endpoint responds with status 200
- [ ] Stream endpoint returns results (not empty array)
- [ ] Results show magnet links
- [ ] Pool key from Chillstreams API is used
- [ ] No user credentials in manifest config

---

## Troubleshooting

### **Problem: Prowlarr returns 404**

**Cause**: Wrong Torznab URL  
**Solution**: Check `Settings → Apps` in Prowlarr UI, copy exact URL

### **Problem: Chillproxy returns no results**

**Cause**: Prowlarr not indexing properly, or API key wrong  
**Solution**:
1. Test Prowlarr directly: `http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=test`
2. Verify API key in settings
3. Check that indexers are enabled

### **Problem: Results but no magnet links**

**Cause**: Prowlarr returned results but missing magnet conversion  
**Solution**: Ensure TorBox store is configured with pool key

### **Problem: Pool key not assigned**

**Cause**: User UUID not found in Chillstreams  
**Solution**:
1. Create test user in Chillstreams
2. Assign pool key to user
3. Use that user UUID in config

---

## Performance Tips

**To speed up results**:

1. **Only keep top 5 indexers enabled**:
   - YTS (movies) - fastest
   - EZTV (TV) - fastest
   - RARBG (both) - good quality
   - TPB (both) - good coverage
   - TorrentGalaxy (both) - modern

2. **Disable slow indexers**:
   - Slow: KickassTorrents, 1337x, etc.
   - These have rate limits and slow API

3. **Enable parallel search** (Prowlarr default):
   - Searches all indexers simultaneously
   - Much faster than sequential

4. **Cache results** in Chillproxy:
   - Same movie searched twice = cached
   - Reduces Prowlarr calls

---

## Architecture Diagram

```
FINAL SETUP:

User's Stremio
    ↓
    │ (manifest: {auth: uuid, indexers: [prowlarr-url]})
    ↓
Chillproxy/Torz
    ├─ Built-in GetStreamsFromIndexers()
    ├─ Calls Prowlarr via Torznab API
    └─ Calls Chillstreams Pool API
    ├─ Calls TorBox API (with pool key)
    └─ Returns: Stream URLs to Stremio
    
Where:
- uuid = User's Chillstreams ID (no secrets)
- prowlarr-url = Public indexer URL (no secrets)
- pool-key = Shared secret (from Chillstreams only)
```

---

## Next Steps

1. ✅ Install Prowlarr
2. ✅ Enable 5 indexers
3. ✅ Get Torznab URL + API Key
4. ✅ Configure Chillproxy with Prowlarr
5. ✅ Test with Breaking Bad
6. ✅ Verify pool key being used
7. ✅ Check `torbox_assignments` table has user entry
8. ✅ Check `torbox_usage_logs` has usage records

---

## Summary

**What You Have Now**:
- ✅ Chillproxy with built-in `/stremio/torz/` searching
- ✅ Prowlarr as the indexer backend (5 top sites)
- ✅ Chillstreams Pool API for TorBox key management
- ✅ No user credentials exposed (only UUIDs and URLs)

**Time to Implementation**: 15 minutes  
**Complexity**: Simple (just 3 environment pieces)  
**Security**: High (credentials stay on servers)

---

**Ready to get started?** Follow the 3 steps above!

**Need help?** Check troubleshooting section or review the flow diagram.

**Status**: Implementation Ready  
**Next**: Execute 3-step setup

