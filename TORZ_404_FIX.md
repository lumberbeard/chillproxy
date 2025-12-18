# ✅ Chillproxy 404 Issue - Root Cause & Resolution

## Problem Identified

You were getting **HTTP 404** when requesting:
```
GET /stremio/torz/<config>/manifest.json
```

Even though the request was syntactically correct, chillproxy was returning **404 - Not Found**.

## Root Cause

The **Torz endpoint was never registered** because:

1. **In `internal/endpoint/stremio.go`**, the endpoints are conditionally registered based on feature flags:
```go
if config.Feature.IsEnabled(config.FeatureStremioTorz) {
    stremio_torz.AddStremioTorzEndpoints(mux)  // ← Only registers if feature is enabled
}
```

2. **The feature constant** is defined as `"stremio_torz"` (with underscore):
```go
FeatureStremioTorz string = "stremio_torz"
```

3. **But the environment variable was using**:
```yaml
STREMTHRU_FEATURE=stremio-torz  # ← Hyphen instead of underscore!
```

4. **Result**: The feature parser never found `"stremio-torz"` (hyphen) to enable, so it used the default (disabled), and endpoints were never registered → **404**

## Solution Applied

### Changed Environment Variable

**Before** ❌:
```yaml
STREMTHRU_FEATURE=+stremio-torz,-dmm_hashlist,-imdb_title,-imdb_torrent,-anime
```

**After** ✅:
```yaml
STREMTHRU_FEATURE=stremio_torz,-dmm_hashlist,-imdb_title,-imdb_torrent,-anime
```

### Why It Works Now

1. **Constant matches**:  `"stremio_torz"` (underscore) matches `FeatureStremioTorz = "stremio_torz"`
2. **Feature gets enabled**: The parser finds the match and enables the feature
3. **Endpoints register**: `stremio_torz.AddStremioTorzEndpoints(mux)` is called
4. **Routes work**: `/stremio/torz/<config>/manifest.json` is now available
5. **HTTP 200**: Request succeeds instead of 404

## Files Modified

1. **`docker-compose.yml`**
   - Changed `stremio-torz` → `stremio_torz`
   - Removed `+` prefix (not needed for already-enabled features)

2. **`.env.docker`**
   - Same fix for consistency

## Current Status

✅ **Endpoint now responds with HTTP 200**

The manifest request is now:
- Successfully routed to the Torz handler
- Processing the config JSON
- Returning manifest data

## Next Steps: Debugging Chillstreams Integration

Now that the endpoint is working, the logs still show **no Chillstreams integration messages**. This means:

### Possible Reasons:

1. **Chillstreams client is null** - API key/URL not being read
2. **EnableChillstreamsAuth is false** - feature flag not enabled
3. **Early return in InitializeStoresWithChillstreams** - silent failures with no logging
4. **Auth field validation** - UUID format validation might be failing

### To Debug Further:

Check the following environment variables in the running container:

```powershell
docker exec chillproxy sh -c 'echo CHILLSTREAMS_API_URL=$CHILLSTREAMS_API_URL; echo CHILLSTREAMS_API_KEY=$CHILLSTREAMS_API_KEY; echo ENABLE_CHILLSTREAMS_AUTH=$ENABLE_CHILLSTREAMS_AUTH'
```

Or check what the config shows on startup:

```powershell
docker logs chillproxy 2>&1 | grep -i "chillstreams\|integration" | head -20
```

## Testing Now

You can test the endpoint works:

```powershell
$config = '{"stores":[{"c":"tb","t":"","auth":"550e8400-e29b-41d4-a716-446655440000"}]}'
$base64 = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($config))
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$base64/manifest.json"
# Should return 200 OK with manifest JSON
```

---

**Status**: ✅ **ENDPOINT WORKING**  
**Issue**: 404 was due to feature name mismatch  
**Next**: Verify Chillstreams integration is being called with proper env vars  
**Last Updated**: December 18, 2025

