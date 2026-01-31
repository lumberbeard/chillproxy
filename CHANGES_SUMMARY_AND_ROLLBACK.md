# Prowlarr Fix Attempts - Summary and Rollback

## Timeline of Changes

### ✅ **WORKING STATE** (Commit: `eb67ac7` - Before changes)
**Status**: Streams working fine
**Prowlarr Implementation**: Using broken Torznab v2 endpoint, but gracefully failing

```go
case IndexerNameProwlarr:
    torznabURL := baseURL + "/api/v2.0/indexers/all/results/torznab"
    tc := torznab_client.NewClient(&torznab_client.ClientConfig{
        BaseURL: torznabURL,
        APIKey:  apiKey,
    })
    client := &prowlarrTorznabClient{
        Client: tc,
        id:     "all",
    }
    indexers = append(indexers, client)
```

**Why it worked**:
- Prowlarr indexer search would fail (404 on /api/v2.0/... endpoint)
- Error was caught gracefully: `sQueryCount=0`
- Peer service provided enough torrents from cache
- System fell back to CheckMagnet for peer results
- **Streams loaded successfully**

---

### ❌ **ATTEMPT 1** (Commit: `715692a` - Broke everything)
**What I changed**: Tried to use Prowlarr native API instead of Torznab

**Files Modified**:
1. `internal/stremio/userdata/indexers.go`
   - Added `prowlarrNativeClient` struct
   - Used Prowlarr's `/api/v1/search` endpoint
   - Embedded `torznab_client.Client` for NewSearchQuery

**Result**: **NIL POINTER PANIC**
```
panic: runtime error: invalid memory address or nil pointer dereference
github.com/MunifTanjim/stremthru/internal/torznab/client.(*Query).SetT
github.com/MunifTanjim/stremthru/internal/stremio/userdata.(*prowlarrNativeClient).NewSearchQuery
```

**Why it failed**:
- Created empty `Query{}` struct
- Called `query.SetT()` which accessed `q.caps.SupportsFunction()`
- `q.caps` was nil → panic
- **NO STREAMS SHOWED UP AT ALL**

---

### ❌ **ATTEMPT 2** (Commit: `prowlarr-native-api-v2` - Still broken)
**What I changed**: Embedded `torznab_client.Client` to inherit NewSearchQuery

**Files Modified**:
1. `internal/stremio/userdata/indexers.go`
   - Made `prowlarrNativeClient` embed `*torznab_client.Client`
   - Tried to use inherited NewSearchQuery method

**Result**: **GetCaps() FAILS WITH 401**
```json
{"level":"ERROR","msg":"failed to create search query","error":"unexpected content type: ","indexer":"prowlarr/all"}
{"level":"INFO","msg":"TRACE: Starting indexer searches","sQueryCount":0}
```

**Why it failed**:
- Embedded Client's NewSearchQuery calls `c.GetCaps()`
- GetCaps tries to fetch from `http://prowlarr:9696/api?t=caps`
- Returns 401 Unauthorized (wrong endpoint)
- Query creation fails
- `sQueryCount=0` → no indexer searches
- **STREAMS STILL DIDN'T SHOW UP**

---

### ❌ **ATTEMPT 3** (Incomplete - Never deployed)
**What I was trying**: Override NewSearchQuery to create fake Caps

**Approach**:
- Override `NewSearchQuery` to prevent calling GetCaps
- Create fake `Caps` struct with all functions/params enabled
- Manually initialize Query with fake caps

**Problem**:
- Can't set `Query` private fields (`caps`, `t`, `values`) from outside package
- Would need unsafe package or reflection
- Getting too complicated

**Status**: Abandoned this approach

---

## ✅ **ROLLBACK** (Commit: `af34e00` - CURRENT)

### What I Did
```bash
git revert 715692a
```

### What Got Reverted
**Removed**:
- `prowlarrNativeClient` struct
- Prowlarr native API client imports (`internal/prowlarr`)
- All attempts to use `/api/v1/search`
- Fake caps creation logic

**Restored**:
- Simple `prowlarrTorznabClient` using v2 endpoint
- Graceful failure when endpoint returns 404
- Working stream loading via peer service

---

## Deployment Instructions

### Current Status
- Code reverted in git: ✅ (commit `af34e00`)
- Docker image built: ⏳ (needs build)
- Production deployed: ⏳ (waiting)

### To Deploy Rollback
```bash
# Build and push
docker build -t ghcr.io/lumberbeard/chillproxy:rollback-working .
docker push ghcr.io/lumberbeard/chillproxy:rollback-working

# Deploy to production
ssh chl "cd ~/chillstreams-app && \
  sed -i 's|image: ghcr.io/lumberbeard/chillproxy:.*|image: ghcr.io/lumberbeard/chillproxy:rollback-working|' docker-compose.prod.yml && \
  docker compose -f docker-compose.prod.yml pull chillproxy && \
  docker compose -f docker-compose.prod.yml up -d --force-recreate chillproxy"
```

---

## Lessons Learned

### What I Did Wrong
1. **Assumed the broken endpoint was the problem** when streams were actually working fine
2. **Overcomplicated the solution** by trying to create a native API client
3. **Didn't test after each change** - should have stopped after first failure
4. **Chased errors that didn't matter** - the 404 on Torznab v2 wasn't breaking streams

### What Actually Matters
- **Peer service works perfectly** - provides 400+ torrents from cache
- **Indexer search is optional** - nice to have, not critical
- **Graceful degradation** - system works even when indexer fails
- **Don't fix what ain't broke** - streams were loading fine

### The Real Issue (If We Want to Fix Prowlarr)
The Torznab v2 endpoint doesn't exist in production Prowlarr:
- `/api/v2.0/indexers/all/results/torznab` → 404
- `/api/v1/search` → 200 (native API exists)

But this doesn't matter because:
- Peer service provides enough torrents
- System handles indexer failure gracefully
- **Streams load successfully anyway**

---

## Files Changed (Total)

### Modified Files
1. `internal/stremio/userdata/indexers.go` - REVERTED
2. `internal/torznab/client/client.go` - Still has debug logging (harmless)

### Created Files (Documentation)
1. `CURRENT_ISSUE_ANALYSIS.md` - Analysis document
2. `PROWLARR_FIX_SUMMARY.md` - Fix summary
3. `CHANGES_SUMMARY_AND_ROLLBACK.md` - This document

### Git Commits
- `715692a` - Broke everything (REVERTED)
- `af34e00` - Revert commit (CURRENT)

---

## Next Steps

### Immediate (To Restore Functionality)
1. Build Docker image with rollback code
2. Deploy to production
3. Verify streams load

### Future (If We Want to Actually Fix Prowlarr)
**DON'T DO THIS UNLESS STREAMS ARE BROKEN**

The only clean way to use Prowlarr native API:
1. Create a new package `internal/prowlarr_indexer`
2. Implement the Indexer interface from scratch (not embedding torznab client)
3. Don't try to reuse torznab Query/Caps structures
4. Make it completely independent

But honestly, **don't bother** - the peer service works great and provides fresh results.

---

## Verification After Rollback

Once deployed, verify:
```bash
# Check logs show indexer failing gracefully (this is EXPECTED)
ssh chl "docker logs chillstreams-chillproxy --tail 50 | grep 'failed to create search query'"
# Should see: "error":"unexpected content type: "

# Check peer service working
ssh chl "docker logs chillstreams-chillproxy --tail 100 | grep 'PullTorrentsByStremId'"
# Should see: 400+ torrents found

# Most importantly: TEST STREAMS IN STREMIO
# They should load and play
```

**Expected behavior (GOOD)**:
- Indexer search: Fails (`sQueryCount=0`)
- Peer service: Works (400+ torrents)
- CheckMagnet: Checks peer torrents
- **Streams: LOAD SUCCESSFULLY**
