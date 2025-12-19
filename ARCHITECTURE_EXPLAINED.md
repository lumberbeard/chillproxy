# Understanding Chillproxy (StremThru) Architecture
## Deep Dive: Indexers, Stores, and Stream Flow

**Date**: December 18, 2025  
**Scope**: How Chillproxy/StremThru works with external addons and debrid services

---

## Executive Summary

**What You're Confused About**: You thought Chillstreams was calling external addons like Torrentio/Comet for indexing, but now you're realizing **Chillproxy (StremThru) has its OWN built-in indexing capabilities** that we need to understand before integrating TorBox.

**The Reality**: 
- ✅ **Chillstreams**: Wrapper that calls external addons (Torrentio, Comet, etc.) for streams
- ✅ **Chillproxy**: Has **TWO modes** - "Torz" (indexer mode) and "Store" (debrid catalog mode)
- ✅ TorBox is involved in **BOTH** modes but differently

---

## The Three Pieces You Need to Understand

### 1️⃣ **Chillstreams** (TypeScript - Your Main App)

**What it does**:
- **Wrapper/Aggregator** for external Stremio addons
- Calls Torrentio, Comet, MediaFusion, etc. to get streams
- Applies filters, sorts, and formats the results
- Returns aggregated streams to Stremio

**How it works**:
```
User → Stremio → Chillstreams Manifest
                      ↓
              [Chillstreams calls EXTERNAL addons]
                      ↓
    ┌─────────────────┼─────────────────┐
    ↓                 ↓                 ↓
Torrentio          Comet          MediaFusion
(external)      (external)        (external)
    ↓                 ↓                 ↓
[Returns streams with magnet links/hashes]
    ↓                 ↓                 ↓
    └─────────────────┴─────────────────┘
                      ↓
        Chillstreams aggregates & formats
                      ↓
        Returns to Stremio with stream URLs
```

**Important**: Chillstreams itself does NOT index torrents. It just wraps other addons that do.

---

### 2️⃣ **Chillproxy "Torz" Mode** (Go - Indexer + Debrid Proxy)

**What it does**:
- **Built-in torrent indexer** (searches Jackett, Prowlarr, Torznab indexers)
- **Debrid service integration** (TorBox, RealDebrid, etc.)
- Returns debrid-cached streams directly to Stremio

**How it works**:
```
User → Stremio → Chillproxy/Torz Manifest
                      ↓
        [Chillproxy SEARCHES torrent indexers]
                      ↓
    ┌─────────────────┼─────────────────┐
    ↓                 ↓                 ↓
Jackett          Prowlarr          YTS/EZTV
(Torznab API)   (Torznab API)    (RSS feeds)
    ↓                 ↓                 ↓
[Returns .torrent files and magnet links]
    ↓                 ↓                 ↓
    └─────────────────┴─────────────────┘
                      ↓
    [Chillproxy checks if cached in TorBox]
                      ↓
        ┌─────────────────────────┐
        │  TorBox API             │
        │  - Check cache status   │
        │  - Add torrent if needed│
        │  - Get stream URL       │
        └─────────────────────────┘
                      ↓
        Returns stream URL to Stremio
```

**Key Point**: Chillproxy/Torz **REPLACES** Torrentio/Comet. It does the indexing itself.

---

### 3️⃣ **Chillproxy "Store" Mode** (Go - Debrid Catalog Browser)

**What it does**:
- **Browse your existing debrid library** (already downloaded files)
- Acts as a Stremio catalog for files in your TorBox/RealDebrid account
- No searching/indexing - just lists what you have

**How it works**:
```
User → Stremio → Chillproxy/Store Manifest
                      ↓
    [Chillproxy fetches YOUR TorBox library]
                      ↓
        ┌─────────────────────────┐
        │  TorBox API             │
        │  - List downloads       │
        │  - List cached files    │
        │  - Get file metadata    │
        └─────────────────────────┘
                      ↓
    Shows as Stremio catalog/library
                      ↓
    User picks a file → plays directly
```

**Key Point**: This is for **already cached/downloaded** content, not searching.

---

## Architecture Comparison

### **Before Chillproxy Integration** (Current Chillstreams Setup)

```
┌─────────────────────────────────────────────────────────────────┐
│                        USER'S STREMIO APP                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ↓
┌────────────────────────────────────────────────────────────────┐
│                      CHILLSTREAMS (TS)                          │
│  - Aggregates external addons                                   │
│  - Filters & sorts streams                                      │
│  - Formats titles                                               │
└─────────┬───────────────┬──────────────┬───────────────────────┘
          │               │              │
          ↓               ↓              ↓
    ┌──────────┐    ┌──────────┐   ┌──────────┐
    │Torrentio │    │  Comet   │   │MediaFusion│ ← External addons
    │(external)│    │(external)│   │(external)│
    └─────┬────┘    └─────┬────┘   └─────┬────┘
          │               │              │
          └───────────────┴──────────────┘
                         │
          Returns streams with magnet links
                         │
                         ↓
          User's debrid service adds torrent
                         │
                         ↓
          User plays from their own account
```

**Problem**: No centralized debrid management, users need their own keys.

---

### **After Chillproxy Integration** (Two Possible Setups)

#### **Option A: Use Chillstreams + Chillproxy/Torz** (What we're building)

```
┌─────────────────────────────────────────────────────────────────┐
│                        USER'S STREMIO APP                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ↓                             ↓
    ┌─────────────────┐           ┌─────────────────┐
    │ CHILLSTREAMS    │           │ CHILLPROXY/TORZ │
    │ (External addon │           │ (Built-in       │
    │  aggregator)    │           │  indexer)       │
    └────┬───────┬────┘           └────────┬────────┘
         │       │                         │
         │       ↓                         ↓
         │  [Torrentio, Comet, etc.]  [Jackett, Prowlarr]
         │       │                         │
         │       │                         │
         │       └─────────────┬───────────┘
         │                     │
         │          Magnet links / hashes
         │                     │
         │                     ↓
         │         ┌───────────────────────┐
         └────────→│  CHILLSTREAMS API     │
                   │  (Pool Key Manager)   │
                   └──────────┬────────────┘
                              │
                   Returns TorBox pool key
                              │
                              ↓
                   ┌───────────────────────┐
                   │    TORBOX API         │
                   │  - Check cache        │
                   │  - Add torrent        │
                   │  - Get stream URL     │
                   └───────────────────────┘
```

**How this works**:
1. User installs **both** Chillstreams AND Chillproxy manifests in Stremio
2. **Chillstreams**: Aggregates Torrentio/Comet/etc. (existing behavior)
3. **Chillproxy/Torz**: Does its own indexing via Jackett/Prowlarr
4. **Both** use Chillstreams API to get TorBox pool keys
5. **Both** proxy streams through shared pool

**Benefit**: More stream sources (external addons + built-in indexers)

---

#### **Option B: Use Only Chillproxy/Torz** (Replace Chillstreams)

```
┌─────────────────────────────────────────────────────────────────┐
│                        USER'S STREMIO APP                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ↓
                   ┌─────────────────┐
                   │ CHILLPROXY/TORZ │
                   │ (Built-in       │
                   │  indexer)       │
                   └────────┬────────┘
                            │
                            ↓
              [Searches Jackett, Prowlarr, YTS, etc.]
                            │
                            ↓
         ┌────────────────────────────────────┐
         │  CHILLSTREAMS API (Pool Manager)   │
         │  - Assigns TorBox pool key         │
         │  - Tracks devices                  │
         │  - Logs usage                      │
         └─────────────────┬──────────────────┘
                           │
                Returns pool key
                           │
                           ↓
                 ┌───────────────────────┐
                 │    TORBOX API         │
                 │  - Check cache        │
                 │  - Add torrent        │
                 │  - Get stream URL     │
                 └───────────────────────┘
```

**How this works**:
1. User installs **only** Chillproxy manifest
2. Chillproxy does all indexing internally
3. Uses Chillstreams API only for pool key management
4. Streams from TorBox pool

**Benefit**: Simpler setup, one addon only

---

## What Each Component Actually Does

### **Chillstreams** (What you already have)
| Feature | Description |
|---------|-------------|
| **Type** | External addon wrapper/aggregator |
| **Indexing** | ❌ No - calls external addons |
| **External Addons** | ✅ Torrentio, Comet, MediaFusion, etc. |
| **Debrid Integration** | ✅ Via external addons (user keys) |
| **Built-in Search** | ❌ No |
| **Use Case** | Aggregate multiple addon sources |

### **Chillproxy/Torz** (What we're integrating)
| Feature | Description |
|---------|-------------|
| **Type** | Built-in torrent indexer + debrid proxy |
| **Indexing** | ✅ Yes - Jackett, Prowlarr, Torznab |
| **External Addons** | ❌ No - indexes directly |
| **Debrid Integration** | ✅ TorBox, RealDebrid, etc. (pool keys) |
| **Built-in Search** | ✅ Yes |
| **Use Case** | Self-contained indexing + debrid |

### **Chillproxy/Store** (Different mode)
| Feature | Description |
|---------|-------------|
| **Type** | Debrid library browser |
| **Indexing** | ❌ No - shows existing files only |
| **External Addons** | ❌ No |
| **Debrid Integration** | ✅ Shows your cached/downloaded files |
| **Built-in Search** | ❌ No (browse only) |
| **Use Case** | Browse debrid library like Netflix |

---

## How Torznab/Jackett/Prowlarr Fit In

### **What is Torznab?**
- **Torznab** = Torrent + Newznab (Usenet indexer protocol)
- It's an **API standard** for querying torrent indexers
- Like a universal language for torrent search

### **What is Jackett?**
- **Jackett** = Torrent indexer proxy/aggregator
- Converts various torrent site APIs → Torznab API
- You run Jackett locally, it queries 100+ torrent sites
- Chillproxy connects to Jackett via Torznab API

### **What is Prowlarr?**
- **Prowlarr** = Similar to Jackett (newer, better)
- Also provides Torznab API
- Better integration with *arr apps (Sonarr, Radarr)

### **Flow**:
```
Chillproxy → Torznab API → Jackett/Prowlarr → Torrent Sites
                                (proxy)        (100+ sites)
                                  ↓
                        [Returns .torrent files]
                                  ↓
                        Chillproxy checks TorBox cache
                                  ↓
                        Returns stream if cached
```

---

## TorBox's Role in Each Mode

### **In Chillproxy/Torz Mode** (Indexer)
```
1. User searches for "Breaking Bad S01E01"
2. Chillproxy queries Jackett/Prowlarr
3. Gets 50 torrent results
4. For EACH result:
   ├─ Extracts info hash
   ├─ Calls TorBox: "Do you have this cached?"
   ├─ If YES: Returns instant stream URL
   └─ If NO: Adds torrent to TorBox, waits, then streams
5. User sees all cached + newly added streams
```

**TorBox API Calls**:
- `POST /torrents/checkcached` - Check if hash is already in TorBox
- `POST /torrents/createtorrent` - Add new torrent to TorBox
- `GET /torrents/mylist` - List user's torrents
- `GET /torrents/requestdl` - Get direct download URL

### **In Chillproxy/Store Mode** (Library)
```
1. User opens Stremio catalog
2. Chillproxy calls TorBox: "What files do I have?"
3. TorBox returns list of cached/downloaded content
4. Chillproxy formats as Stremio catalog items
5. User browses library like Netflix
6. Click to play → Direct stream URL
```

**TorBox API Calls**:
- `GET /torrents/mylist` - List all torrents in account
- `GET /torrents/info` - Get torrent details
- `GET /torrents/requestdl` - Get stream URL

---

## Configuration Examples

### **Chillproxy/Torz Config** (Indexer Mode)
```json
{
  "stores": [
    {
      "c": "tb",      // TorBox
      "t": "",        // Empty (uses pool key)
      "auth": "user-uuid-here"  // Chillstreams user ID
    }
  ],
  "indexers": [
    {
      "url": "http://localhost:9117/api/v2.0/indexers/all/results/torznab",
      "apiKey": "your_jackett_api_key"
    }
  ]
}
```

**What this does**:
- Searches Jackett for torrents
- Uses Chillstreams API to get TorBox pool key
- Checks TorBox cache for each result
- Returns streams

### **Chillproxy/Store Config** (Library Mode)
```json
{
  "stores": [
    {
      "c": "tb",
      "t": "",
      "auth": "user-uuid-here"
    }
  ]
}
```

**What this does**:
- Lists files in TorBox account
- Displays as Stremio catalog
- Plays existing files

---

## Answer to Your Specific Questions

### **Q: Will I be using the StremThru store?**
**A**: You have **two options**:

**Option 1**: Use **Chillproxy/Torz** (indexer mode) for scraping
- This is what you want for **searching/indexing** torrents
- Jackett/Prowlarr → Chillproxy → TorBox pool → Streams

**Option 2**: Use **Chillproxy/Store** (library mode) for browsing
- This is for browsing **already cached** content
- TorBox library → Chillproxy → Stremio catalog

**Recommendation**: Start with **Torz** mode since you want scraping.

### **Q: Is there a built-in Torz addon within Chillproxy?**
**A**: Yes! **Torz IS the built-in indexer**. It's not disabled, but you need to:
1. Enable it with `STREMTHRU_FEATURE=+stremio-torz`
2. Configure indexers (Jackett/Prowlarr URLs)
3. Configure TorBox auth (via Chillstreams pool)

### **Q: How does scraping work before TorBox?**
**A**: 
```
Step 1: Configure Indexers
  ├─ Install Jackett locally
  ├─ Add indexers in Jackett (TPB, RARBG, etc.)
  └─ Get Jackett Torznab URL

Step 2: Configure Chillproxy/Torz
  ├─ Add Jackett URL to config
  ├─ Set auth to Chillstreams user ID
  └─ Enable Torz feature

Step 3: Test Scraping (WITHOUT TorBox first)
  ├─ Make stream request to Chillproxy
  ├─ Chillproxy queries Jackett
  ├─ Returns torrent results (magnet links)
  └─ Verify you see results

Step 4: Add TorBox Integration
  ├─ Chillproxy checks TorBox cache for each result
  ├─ Returns instant streams for cached torrents
  └─ Adds uncached torrents to TorBox queue
```

---

## Recommended Testing Path

### **Phase 3.1: Test Chillproxy Indexing (No TorBox)**
```pwsh
# 1. Install Jackett
docker run -d --name jackett -p 9117:9117 linuxserver/jackett

# 2. Configure Jackett
# - Open http://localhost:9117
# - Add indexers (YTS, EZTV, etc.)
# - Get Torznab API URL

# 3. Start Chillproxy with indexer only
cd C:\chillproxy
$env:STREMTHRU_FEATURE="+stremio-torz"
go run main.go

# 4. Test manifest
$config = @{
  stores = @()
  indexers = @(@{
    url = "http://localhost:9117/api/v2.0/indexers/all/results/torznab"
    apiKey = "YOUR_JACKETT_KEY"
  })
} | ConvertTo-Json

$configB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($config))
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configB64/manifest.json"

# 5. Test stream search (should return magnet links)
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configB64/stream/movie/tt0133093.json"
```

**Expected Result**: JSON with magnet links from Jackett

### **Phase 3.2: Add TorBox Integration**
```pwsh
# Modify config to include TorBox auth
$config = @{
  stores = @(@{
    c = "tb"
    t = ""
    auth = "3b94cb45-3f99-406e-9c40-ecce61a405cc"  # Your user UUID
  })
  indexers = @(@{
    url = "http://localhost:9117/api/v2.0/indexers/all/results/torznab"
    apiKey = "YOUR_JACKETT_KEY"
  })
} | ConvertTo-Json

# Test again - now should check TorBox cache
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$configB64/stream/movie/tt0133093.json"
```

**Expected Result**: Streams with TorBox URLs (cached + newly added)

---

## Summary Diagram: Complete Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                      YOUR COMPLETE SETUP                          │
└──────────────────────────────────────────────────────────────────┘

User opens Stremio
       │
       ├─── Installs Chillstreams manifest
       │         │
       │         └─→ [Chillstreams aggregates Torrentio, Comet, etc.]
       │                   │
       │                   └─→ Returns external addon streams
       │
       └─── Installs Chillproxy/Torz manifest
                 │
                 └─→ [Chillproxy/Torz searches Jackett/Prowlarr]
                           │
                           ├─ Gets 50 torrent results
                           │
                           ├─ For each torrent:
                           │    ├─ Extract info hash
                           │    ├─ Call Chillstreams API
                           │    │    └─→ Get TorBox pool key
                           │    │
                           │    ├─ Call TorBox API
                           │    │    ├─ Check if cached
                           │    │    ├─ Add if not cached
                           │    │    └─ Get stream URL
                           │    │
                           │    └─ Return stream to Stremio
                           │
                           └─→ User sees aggregated results from:
                                 ├─ Chillstreams (external addons)
                                 └─ Chillproxy (Jackett indexers)
```

---

## Next Steps

1. **Install Jackett** locally for testing
2. **Test Chillproxy/Torz** indexing WITHOUT TorBox first
3. **Verify** you get magnet links back
4. **Then** add TorBox integration via Chillstreams pool
5. **Test** that cached torrents return instant streams

---

**Key Takeaway**: 
- **Chillstreams** = External addon aggregator (what you have)
- **Chillproxy/Torz** = Built-in indexer (what you're adding)
- **Both can coexist** and give users more stream sources

You want to get **Chillproxy/Torz scraping working first** (without TorBox), then add TorBox integration second.

**Status**: Understanding Phase Complete  
**Next**: Test Chillproxy/Torz indexing with Jackett

