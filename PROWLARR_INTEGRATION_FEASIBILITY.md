# Prowlarr Integration with Chillproxy - Status Assessment

**Date**: December 18, 2025  
**Question**: Will Prowlarr work and be integrated with Chillproxy?  
**Answer**: ✅ **YES, but requires implementation work**

---

## Current State

### ✅ What's Already Done

1. **Chillstreams Client** (`internal/chillstreams/client.go`):
   - ✅ Implemented `GetPoolKey()` method
   - ✅ Implemented `LogUsage()` method
   - ✅ Ready for use

2. **Chillproxy Configuration**:
   - ✅ `.env` has `CHILLSTREAMS_API_URL` and `CHILLSTREAMS_API_KEY`
   - ✅ Feature flag `ENABLE_CHILLSTREAMS_AUTH=true`

3. **Prowlarr API**:
   - ✅ Running on `localhost:9696`
   - ✅ API working correctly (tested with 220+ Matrix results)
   - ✅ 4 indexers configured (EZTV, TPB, TorrentGalaxy, YTS)

### ❌ What's NOT Done Yet

The integration between **Prowlarr API** and **Chillproxy stream handling** is not implemented:

1. **No Prowlarr Client** in Chillproxy - `internal/` doesn't have a Prowlarr API client
2. **No Integration in Stream Handler** - `internal/stremio/torz/stream.go` doesn't call Prowlarr API
3. **No Configuration Parsing** - Config schema doesn't have Prowlarr settings
4. **No Pool Key Usage** - Even though Chillstreams client exists, it's not being used in stream flows

---

## What Needs to be Built

### Phase 1: Prowlarr Client (1-2 hours)

Create `internal/prowlarr/client.go`:

```go
package prowlarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type SearchResult struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	InfoHash    string `json:"infoHash"`
	Indexer     string `json:"indexer"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"downloadUrl"`
	PublishDate string `json:"publishDate"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Search(ctx context.Context, query string, searchType string) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", searchType)

	req, _ := http.NewRequestWithContext(ctx, "GET", 
		c.baseURL+"/api/v1/search?"+params.Encode(), nil)
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prowlarr search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr returned %d", resp.StatusCode)
	}

	var results []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	return results, nil
}
```

### Phase 2: Prowlarr Configuration (30 minutes)

Update `.env`:

```bash
# Prowlarr Indexer Integration
PROWLARR_URL=http://localhost:9696
PROWLARR_API_KEY=f963a60693dd49a08ff75188f9fc72d2
```

Load in `internal/config/config.go`:

```go
var Prowlarr = struct {
	URL    string
	APIKey string
}{
	URL:    os.Getenv("PROWLARR_URL"),
	APIKey: os.Getenv("PROWLARR_API_KEY"),
}
```

### Phase 3: Integration in Stream Handler (2-3 hours)

Update `internal/stremio/torz/stream.go` to:

1. Check if Prowlarr should be used
2. Call Prowlarr API to search for torrents
3. Extract infohashes and pass to TorBox via Chillstreams pool key
4. Return streams to Stremio

**Example logic** (pseudo-code):

```go
func GetStreamsFromIndexers(ctx *RequestContext, stremType, stremId string) {
    // ...existing code...
    
    // NEW: Try Prowlarr first if enabled
    if shouldUseProwlarr(ctx) {
        results, err := SearchProwlarr(ctx, searchQuery)
        if err == nil {
            // Convert Prowlarr results to TorBox streams
            for _, result := range results {
                // 1. Get pool key from Chillstreams
                poolKeyResp, err := chillstreamsClient.GetPoolKey(ctx, chillstreams.GetPoolKeyRequest{
                    UserID:   ctx.UserID,
                    DeviceID: ctx.DeviceID,
                    Action:   "check-cache",
                    Hash:     result.InfoHash,
                })
                if err != nil || !poolKeyResp.Allowed {
                    continue
                }
                
                // 2. Use pool key to check TorBox cache
                storeClient := torbox.NewClient(poolKeyResp.PoolKey)
                cached, err := storeClient.CheckCached(result.InfoHash)
                
                // 3. Return stream if cached or add torrent
                if cached || shouldAddUncached {
                    streamURL, err := storeClient.GetStream(result.InfoHash, fileID)
                    if err == nil {
                        streams = append(streams, &stremio.Stream{
                            Title: result.Title,
                            URL:   streamURL,
                        })
                    }
                }
                
                // 4. Log usage
                go chillstreamsClient.LogUsage(ctx, chillstreams.LogUsageRequest{
                    UserID:    ctx.UserID,
                    PoolKeyID: poolKeyResp.PoolKeyID,
                    Action:    "stream-served",
                    Hash:      result.InfoHash,
                    Cached:    cached,
                    Bytes:     result.Size,
                })
            }
        }
    }
    
    // ...existing code for other indexers...
}
```

### Phase 4: Device Tracking (30 minutes)

Already partially implemented (`internal/device/` exists), but needs to be:

1. Used in stream handler to get `deviceID`
2. Passed to Chillstreams in `GetPoolKey()` calls
3. Tracked for device limits (max 3 per user)

### Phase 5: Chillstreams Pool API Implementation (Chillstreams side - 2-3 hours)

In Chillstreams (`packages/server/src/routes/api/internal/pool.ts`):

```typescript
// GET /api/v1/internal/pool/get-key
// - Validate user exists
// - Check device count
// - Return assigned pool key
// - Update last_used timestamp

// POST /api/v1/internal/pool/log-usage
// - Log usage to database
// - Update user statistics
// - Check for abuse
```

---

## Implementation Timeline

| Phase | Component | Time | Status |
|-------|-----------|------|--------|
| **1** | Prowlarr Client | 1-2h | ⏳ TODO |
| **2** | Configuration | 30m | ⏳ TODO |
| **3** | Stream Integration | 2-3h | ⏳ TODO |
| **4** | Device Tracking | 30m | ⏳ TODO |
| **5** | Chillstreams API | 2-3h | ⏳ TODO |
| **6** | Testing | 1-2h | ⏳ TODO |

**Total**: ~8-12 hours of implementation

---

## Data Flow (Once Integrated)

```
User clicks stream in Stremio
        ↓
Stremio sends request to Chillproxy:
  GET /stremio/torz/{config}/stream/movie/tt0133093.json
        ↓
Chillproxy extracts auth (user UUID) from config
        ↓
Chillproxy calls Prowlarr:
  GET /api/v1/search?query=matrix&type=search
  Header: X-Api-Key: f963a60693dd49a08ff75188f9fc72d2
        ↓
Prowlarr returns 220 torrents with infohashes
        ↓
For each torrent infohash, Chillproxy:
  1. Calls Chillstreams: POST /api/v1/internal/pool/get-key
     {userId, deviceId, hash}
        ↓
  2. Gets pool key back: {poolKey, allowed, deviceCount}
        ↓
  3. Calls TorBox with pool key:
     POST /torrents/checkcached {hash}
        ↓
  4. If cached or can add torrent:
     GET stream URL from TorBox
        ↓
  5. Logs usage back to Chillstreams:
     POST /api/v1/internal/pool/log-usage
     {userId, poolKeyId, action, hash}
        ↓
Returns streams to Stremio:
  [{title: "Matrix 1080p", url: "https://torbox-cdn.com/..."}, ...]
        ↓
User clicks → Video plays via TorBox
```

---

## Success Criteria

Once implemented, you'll have:

✅ **Prowlarr Integration**:
- Search multiple torrent indexers (YTS, EZTV, TPB, TorrentGalaxy)
- Return 50+ torrent options per search
- Real-time indexing (no stale data)

✅ **Pool Key Authentication**:
- Zero exposure of TorBox API keys to users
- Device tracking (max 3 devices per user)
- Real-time user revocation

✅ **TorBox Caching**:
- Check if torrents are cached
- Auto-add uncached torrents
- Stream directly from TorBox CDN

✅ **Complete Streaming**:
- User searches for "Matrix" in Stremio
- Gets 50+ torrent options from Prowlarr
- Clicks one → Streams instantly from TorBox
- All tracked and logged

---

## Recommendation

### To Start Immediately ✅

I recommend starting with **Phase 1-3** (Prowlarr integration):

**Why**:
1. Prowlarr is already running and working (proven with 220 Matrix results)
2. Prowlarr client is simple (copy-paste from guide)
3. Integration in stream handler is straightforward
4. Quick win = real torrent search results in Stremio

### Timeline to Full Working System

- **Week 1**: Implement Prowlarr client + config + stream integration (Chillproxy side)
- **Week 2**: Implement Chillstreams pool API endpoints (Chillstreams side)
- **Week 3**: Testing, debugging, deployment

### Alternative: Use Direct Token (Fast Track)

If you want to test **Prowlarr + Chillproxy** integration **without** the Chillstreams pool key complexity:

1. Use direct TorBox API key (what you have now)
2. Integrate Prowlarr search
3. Return streams from TorBox
4. Add pool key layer later

This gets you **Prowlarr search + TorBox streaming** in 2-3 hours instead of 8-12.

---

## Current Blockers

❌ **Prowlarr search is not being called** from Chillproxy stream handler  
❌ **TorBox API is not being used** to check cache with Prowlarr results  
❌ **Chillstreams pool endpoints** are not implemented yet  

All are solvable with straightforward implementation.

---

## Bottom Line

**Will Prowlarr work with Chillproxy?**

✅ **YES, absolutely!**
- Prowlarr API is proven working (220 Matrix results)
- Chillproxy has infrastructure ready (Chillstreams client exists)
- Integration is straightforward (call Prowlarr API, use results with TorBox)
- Pool key system is already coded (just needs to be wired up)

**How long to full integration?**
- **Fast (2-3 hours)**: Prowlarr + TorBox (direct key)
- **Complete (8-12 hours)**: Prowlarr + TorBox + Chillstreams pool system

**What's blocking you right now?**
- **Nothing!** You can start today.

---

**Status**: ✅ **READY FOR IMPLEMENTATION**  
**Next**: Start Phase 1 (Build Prowlarr Client for Chillproxy)


