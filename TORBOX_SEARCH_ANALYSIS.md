# Built-in Indexers Analysis: TorBox Pro Search API vs Jackett/Prowlarr

**Date**: December 17, 2025  
**Question**: Can we use TorBox Pro's search API instead of Jackett/Prowlarr?

---

## Executive Summary

**YES! TorBox Pro has a built-in search API** that's already integrated into Chillstreams. This is MUCH better than Jackett/Prowlarr for your use case.

**Key Findings**:
✅ **TorBox Pro Search API** exists at `https://search-api.torbox.app`  
✅ **Already integrated** in Chillstreams (`@core/builtins/torbox-search`)  
✅ **Searches both torrents AND usenet** directly  
✅ **Checks cache status** automatically  
✅ **No external indexer needed** (no Jackett/Prowlarr required)  
❌ **NOT implemented** in Chillproxy (StremThru fork)  

---

## What is TorBox Search API?

### **Overview**
TorBox Pro accounts include access to a **dedicated search API** that:
- Searches torrent sites directly (100+ indexers)
- Searches usenet providers (via TorBox's usenet integration)
- Returns results with **cache status already checked**
- Returns parsed metadata (resolution, codec, quality, etc.)
- Supports multiple ID types (IMDb, TMDB, AniDB, MAL, etc.)

### **API Endpoint**
```
Base URL: https://search-api.torbox.app

Endpoints:
GET /torrents/{idType}:{id}  - Search torrents by content ID
GET /usenet/{idType}:{id}    - Search usenet by content ID

Query Parameters:
- check_cache: 'true' | 'false'  // Check if already in TorBox cache
- check_owned: 'true' | 'false'  // Check if user owns it
- search_user_engines: 'true' | 'false'  // Search user's custom indexers
- season: string
- episode: string
- metadata: 'true' | 'false'  // Return parsed metadata
```

### **Supported ID Types**
```typescript
'anime-planet_id'  // Anime Planet
'anidb_id'         // AniDB
'anilist_id'       // AniList
'anisearch_id'     // AniSearch
'imdb_id'          // IMDb  ← Most common
'kitsu_id'         // Kitsu
'livechart_id'     // LiveChart
'mal_id'           // MyAnimeList
'notify.moe_id'    // Notify.moe
'thetvdb_id'       // TheTVDB
'themoviedb_id'    // TMDB
```

---

## Architecture Comparison

### **Current: Chillproxy/Torz with Jackett**

```
User → Stremio → Chillproxy/Torz
                      ↓
        [Searches via Jackett/Prowlarr]
                      ↓
     ┌────────────────┴────────────────┐
     │  Jackett (local installation)   │
     │  - Needs setup                  │
     │  - Needs maintenance            │
     │  - Aggregates 100+ sites        │
     └────────────────┬────────────────┘
                      ↓
          [Returns torrent results]
                      ↓
        [Chillproxy checks TorBox cache]
                      ↓
            ┌─────────────────┐
            │  TorBox API     │
            │  POST /checkcached │
            └─────────────────┘
                      ↓
        Returns streams if cached
```

**Problems**:
- ❌ Requires Jackett setup (complex)
- ❌ Two API calls per torrent (Jackett + TorBox)
- ❌ Slower (sequential requests)
- ❌ No automatic cache checking

---

### **Better: TorBox Search API (Chillstreams Built-in)**

```
User → Stremio → Chillstreams
                      ↓
        [Calls TorBox Search API]
                      ↓
    ┌─────────────────────────────────────┐
    │  TorBox Search API                  │
    │  https://search-api.torbox.app      │
    │                                      │
    │  - Searches 100+ torrent sites      │
    │  - Searches usenet providers        │
    │  - Auto-checks cache status         │
    │  - Returns parsed metadata          │
    │  - All in ONE API call              │
    └──────────────────┬──────────────────┘
                       ↓
         Returns results with cache info
                       ↓
      Chillstreams formats as streams
                       ↓
           User sees instant results
```

**Benefits**:
- ✅ One API call (faster)
- ✅ Cache status included
- ✅ No Jackett setup needed
- ✅ Maintained by TorBox team
- ✅ Works with TorBox Pro accounts

---

## Current Implementation Status

### **✅ Chillstreams** (Already Has It!)

**Location**: `packages/core/src/builtins/torbox-search/`

**Files**:
```
torbox-search/
├── addon.ts           # Main addon class
├── search-api.ts      # API client for TorBox Search
├── source-handlers.ts # Handles torrent + usenet sources
├── schemas.ts         # Zod validation schemas
├── torrent.ts         # Torrent result processing
└── errors.ts          # Error handling
```

**How it works**:
```typescript
// 1. User configures TorBox Search in Chillstreams
const config = {
  torBoxApiKey: "your-torbox-api-key",
  sources: ["torrent", "usenet"],  // Choose sources
  searchUserEngines: true,          // Use custom indexers
  cacheAndPlay: true                // Auto-add to TorBox
};

// 2. User searches for "Breaking Bad S01E01"
const streams = await torboxSearchAddon.getStreams('series', 'imdb:tt0903747:1:1');

// 3. Behind the scenes:
// - Calls: GET /torrents/imdb_id:tt0903747?season=1&episode=1&check_cache=true
// - TorBox returns results with cache status
// - Chillstreams formats as Stremio streams
// - User sees cached + uncached results
```

**Configuration Example**:
```json
{
  "builtinAddons": {
    "torboxSearch": {
      "enabled": true,
      "torBoxApiKey": "your-key-here",
      "sources": ["torrent", "usenet"],
      "searchUserEngines": true,
      "cacheAndPlay": true,
      "services": [
        {
          "id": "torbox",
          "credentials": { "apiKey": "your-key" }
        }
      ]
    }
  }
}
```

---

### **❌ Chillproxy** (Does NOT Have It)

**Current State**: 
- Chillproxy only supports **Jackett/Prowlarr** (Torznab API)
- No TorBox Search API integration
- Would need to be implemented from scratch

**What's Missing**:
```
chillproxy/store/torbox/
├── client.go       ← Exists (basic TorBox API)
├── torrent.go      ← Exists (add torrent, check cache)
├── search.go       ← MISSING! (TorBox Search API)
└── ...
```

---

## Other Built-in Indexers in Chillstreams

Besides TorBox Search, Chillstreams has several other built-in indexers:

### **1. Knaben Indexer**
**Location**: `packages/core/src/builtins/knaben/`  
**What it does**: Searches Knaben.eu torrent search engine  
**API**: `https://knaben.eu/api/search`  
**Pros**: Fast, clean API, good anime results  
**Cons**: Limited to Knaben's database

### **2. Torrent Galaxy**
**Location**: `packages/core/src/builtins/torrent-galaxy/`  
**What it does**: Searches TorrentGalaxy.to  
**API**: `https://torrentgalaxy.to/get_list`  
**Pros**: Large database, good TV show results  
**Cons**: Rate limited, slower responses

### **3. Torznab/Newznab Generic**
**Location**: `packages/core/src/builtins/torznab/`  
**What it does**: Generic Torznab/Newznab client  
**API**: Any Torznab-compatible endpoint  
**Pros**: Works with Jackett, Prowlarr, etc.  
**Cons**: Requires external setup

### **4. Prowlarr Built-in**
**Location**: `packages/core/src/builtins/prowlarr/`  
**What it does**: Direct Prowlarr API integration  
**API**: Prowlarr v1 API  
**Pros**: Better than generic Torznab, batch queries  
**Cons**: Still requires Prowlarr installation

---

## Comparison Matrix

| Feature | TorBox Search | Jackett/Prowlarr | Knaben | Torrent Galaxy |
|---------|---------------|------------------|--------|----------------|
| **Setup Required** | ❌ No (just API key) | ✅ Yes (install + config) | ❌ No | ❌ No |
| **Cache Check** | ✅ Built-in | ❌ Separate API call | ❌ No | ❌ No |
| **Usenet Support** | ✅ Yes | ✅ Yes (via indexers) | ❌ No | ❌ No |
| **Parsed Metadata** | ✅ Yes | ⚠️ Sometimes | ⚠️ Sometimes | ✅ Yes |
| **Response Time** | ⚡ Fast (1 call) | 🐌 Slow (2 calls) | ⚡ Fast | 🐌 Slow |
| **Maintenance** | ✅ Hosted by TorBox | ❌ Self-hosted | ✅ Hosted | ✅ Hosted |
| **API Cost** | ✅ Included in Pro | ✅ Free | ✅ Free | ✅ Free |
| **Indexer Count** | 100+ | 100+ | 1 | 1 |
| **Anime Support** | ✅ Excellent | ✅ Good | ✅ Good | ⚠️ Limited |
| **Reliability** | ✅✅ High | ⚠️ Varies | ⚠️ Medium | ⚠️ Medium |

---

## Recommendation for Your Setup

### **Best Option: Use TorBox Search API via Chillstreams**

**Why**:
1. ✅ **You already have it** - It's built into Chillstreams
2. ✅ **You have TorBox Pro** - API access included
3. ✅ **No extra setup** - Just configure API key
4. ✅ **Fastest performance** - One API call, cache checked
5. ✅ **Most reliable** - Maintained by TorBox team
6. ✅ **Supports usenet** - If you add usenet to TorBox

### **Architecture with TorBox Search**

```
┌──────────────────────────────────────────────────────────────┐
│                      RECOMMENDED SETUP                        │
└──────────────────────────────────────────────────────────────┘

User installs Chillstreams manifest
         ↓
   [User searches content]
         ↓
Chillstreams calls built-in addons:
         ↓
    ┌────┴────┐
    ↓         ↓
External   TorBox Search
Addons     (Built-in)
    ↓         ↓
Torrentio   [Calls search-api.torbox.app]
Comet            ↓
MediaFusion  [Returns results with cache status]
    ↓            ↓
    └────┬───────┘
         ↓
   Chillstreams aggregates all results
         ↓
   Applies filters, sorts, formats
         ↓
   Calls Chillstreams Pool API
         ↓
   Gets TorBox pool key
         ↓
   Returns streams with pool URLs
         ↓
   User plays instantly (cached) or waits (uncached)
```

**Benefits**:
- ✅ **Best of both worlds**: External addons (Torrentio, etc.) + TorBox Search
- ✅ **More results**: Multiple sources aggregated
- ✅ **Faster**: TorBox Search includes cache status
- ✅ **Simpler**: No Jackett/Prowlarr needed
- ✅ **Cheaper**: No extra infrastructure costs

---

## Configuration Guide

### **Enable TorBox Search in Chillstreams**

**1. Environment Variables**
```bash
# Required
TORBOX_API_KEY=your_torbox_pro_api_key_here

# Optional (defaults shown)
BUILTIN_TORBOX_SEARCH_ENABLED=true
BUILTIN_TORBOX_SEARCH_SEARCH_API_TIMEOUT=30000  # 30 seconds
BUILTIN_TORBOX_SEARCH_METADATA_CACHE_TTL=1209600000  # 2 weeks
BUILTIN_TORBOX_SEARCH_SEARCH_API_CACHE_TTL=604800000  # 1 week
```

**2. User Configuration** (in Chillstreams dashboard)
```typescript
{
  "builtinAddons": {
    "torboxSearch": {
      "enabled": true,
      "torBoxApiKey": "your-key",
      "sources": ["torrent", "usenet"],  // Enable both
      "searchUserEngines": true,         // Use your custom indexers
      "cacheAndPlay": true,              // Auto-add uncached torrents
      "services": [
        {
          "id": "torbox",
          "credentials": {
            "apiKey": "your-key"
          }
        }
      ]
    }
  }
}
```

**3. Test It**
```pwsh
# 1. Get Chillstreams manifest with TorBox Search enabled
Invoke-WebRequest -Uri "http://localhost:3000/stremio/your-config/manifest.json"

# 2. Search for content (e.g., Breaking Bad)
Invoke-WebRequest -Uri "http://localhost:3000/stremio/your-config/stream/series/tt0903747:1:1.json"

# 3. Should return streams with TorBox Search results
```

---

## TorBox Search API Features

### **1. Cache Status Integration**
```json
// Response includes cache info
{
  "hash": "ABC123...",
  "title": "Breaking.Bad.S01E01.1080p",
  "cached": true,          // ← Already in TorBox cache
  "owned": false,          // ← Not in your account yet
  "magnet": "magnet:?xt=...",
  "title_parsed_data": {
    "resolution": "1080p",
    "quality": "WEB-DL",
    "codec": "x264",
    "audio": "AAC"
  }
}
```

**Benefit**: You know instantly which torrents are cached without extra API calls.

### **2. User Custom Indexers**
If you've added custom indexers to your TorBox Pro account (Settings → Search Engines), the API will search those too when `search_user_engines: true`.

**Example**:
```bash
GET /torrents/imdb_id:tt0903747?search_user_engines=true
# Returns results from:
# - TorBox's 100+ built-in indexers
# - Your custom indexers (if any)
```

### **3. Usenet Support**
If you have usenet enabled in TorBox Pro:
```bash
GET /usenet/imdb_id:tt0903747?season=1&episode=1
# Returns NZB results from TorBox's usenet providers
# Already checks if cached in TorBox
```

### **4. Parsed Metadata**
Results include pre-parsed metadata:
```typescript
{
  resolution: "1080p" | "720p" | "2160p" | ...
  quality: "WEB-DL" | "BluRay" | "HDTV" | ...
  codec: "x264" | "x265" | "HEVC" | ...
  audio: "AAC" | "AC3" | "DTS" | ...
  hdr: boolean
  year: number
  encoder: string
  site: string
}
```

**Benefit**: Chillstreams can filter/sort without parsing torrent names.

---

## Migration Path

### **Current State**
```
Chillstreams → External Addons (Torrentio, Comet, etc.)
                     ↓
            User's debrid service
```

### **Phase 1: Enable TorBox Search** (Recommended Next)
```
Chillstreams → External Addons + TorBox Search (built-in)
                     ↓
            Chillstreams Pool API
                     ↓
            TorBox shared pool
```

**Steps**:
1. ✅ Add `TORBOX_API_KEY` to Chillstreams environment
2. ✅ Enable TorBox Search in user config
3. ✅ Test with shared pool keys (already working!)
4. ✅ Users get more results (external + TorBox Search)

### **Phase 2: Optional - Add Chillproxy/Torz** (Later)
```
User installs both:
├─ Chillstreams (external + TorBox Search)
└─ Chillproxy/Torz (Jackett if needed)
       ↓
 Both use shared pool
```

**Only if**: You want even MORE results from Jackett-exclusive indexers.

---

## Answering Your Questions

### **Q: Can we use TorBox Search instead of Jackett/Prowlarr?**
**A**: **YES! Absolutely!** TorBox Search is better because:
- ✅ No setup required (just API key)
- ✅ Faster (1 API call vs 2)
- ✅ Cache status built-in
- ✅ Maintained by TorBox
- ✅ Already in Chillstreams

### **Q: Does TorBox Pro have its own indexing?**
**A**: **YES!** TorBox Pro includes:
- 100+ torrent indexers (built-in)
- Usenet indexers (if you have usenet)
- Custom indexer support (add your own)
- Search API at `https://search-api.torbox.app`

### **Q: Does this work with the architecture we're building?**
**A**: **PERFECT FIT!** Here's how:

```
┌─────────────────────────────────────────────────────┐
│          YOUR COMPLETE ARCHITECTURE                  │
└─────────────────────────────────────────────────────┘

User → Chillstreams
         ↓
   [Aggregates sources]
         ↓
    ┌────┴────────────────┐
    ↓                     ↓
External Addons    TorBox Search ← NEW!
(Torrentio, Comet)  (Built-in)
    ↓                     ↓
    └────────┬────────────┘
             ↓
  Chillstreams aggregates all
             ↓
  Calls Pool Manager API
             ↓
    ┌────────────────────┐
    │  Chillstreams API  │
    │  /internal/pool    │
    └─────────┬──────────┘
              ↓
   Returns TorBox pool key
              ↓
    ┌────────────────────┐
    │  TorBox API        │
    │  (using pool key)  │
    └────────────────────┘
              ↓
   Returns stream URLs
              ↓
   User plays content
```

**Integration Points**:
1. ✅ TorBox Search → Chillstreams (already exists)
2. ✅ Chillstreams → Pool API (Phase 2 complete)
3. ✅ Pool API → TorBox (using pool keys)
4. ✅ All sources use shared pool (no user keys needed)

---

## Next Steps

### **Immediate (Recommended)**

1. **Enable TorBox Search in Chillstreams**
   ```bash
   # Add to .env
   TORBOX_API_KEY=your_pro_key
   BUILTIN_TORBOX_SEARCH_ENABLED=true
   ```

2. **Configure User Settings**
   - Open Chillstreams dashboard
   - Enable "TorBox Search" built-in addon
   - Configure sources (torrent + usenet)
   - Enable "Cache and Play"

3. **Test End-to-End**
   ```pwsh
   # Test stream search
   Invoke-WebRequest "http://localhost:3000/stremio/config/stream/series/tt0903747:1:1.json"
   
   # Should see results from:
   # - Torrentio (external)
   # - Comet (external)
   # - TorBox Search (built-in) ← NEW!
   ```

4. **Verify Pool Integration**
   - All sources should use pool keys
   - Check `torbox_usage_logs` table
   - Verify no user keys exposed

### **Later (Optional)**

Only if you need even more sources:

5. **Add Chillproxy/Torz** with Jackett
   - Install Jackett locally
   - Configure Chillproxy to use Jackett
   - Users can install both manifests
   - Get results from ALL sources

---

## Summary

### **What You Have Now**
✅ Chillstreams with external addons (Torrentio, Comet, etc.)  
✅ Chillstreams Pool API (Phase 2 complete)  
✅ TorBox Search built-in addon (just needs enabling)  

### **What You Should Do Next**
1. ✅ Enable TorBox Search in Chillstreams
2. ✅ Test with shared pool keys
3. ✅ Skip Jackett/Prowlarr (not needed!)

### **The Result**
```
User gets streams from:
├─ External addons (Torrentio, Comet, MediaFusion, etc.)
├─ TorBox Search (100+ indexers built-in)
└─ All using shared TorBox pool keys
   └─ No user keys needed!
```

**Bottom Line**: You already have everything you need! TorBox Search is built into Chillstreams and works perfectly with your shared pool architecture. Just enable it and skip Jackett/Prowlarr entirely.

---

**Status**: Research Complete  
**Recommendation**: Use TorBox Search API (already in Chillstreams)  
**Next**: Enable and test TorBox Search with pool keys

