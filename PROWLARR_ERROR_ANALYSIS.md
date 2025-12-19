# 🔴 PROWLARR INTEGRATION ERROR - ROOT CAUSE IDENTIFIED

**Date**: December 19, 2025  
**Error**: `indexers[0].url: unsupported indexer: ""`  
**Status**: ❌ **CONFIG FORMAT MISMATCH**

---

## 🔍 Problem Identified

### What's Happening

The error occurs when Chillstreams tries to call Chillproxy with Prowlarr indexer configuration:

```
❌ Error: "unsupported indexer: "
   Location: indexers[0].url validation
   Config sent: {"url":"http://localhost:9696/api/v2.0/indexers/all/results/torznab","apiKey":"f963a60693dd49a08ff75188f9fc72d2"}
```

### Root Cause

**MISMATCH IN CONFIG SCHEMA**

**Chillstreams is sending** (wrong format):
```json
{
  "indexers": [
    {
      "url": "http://localhost:9696/api/v2.0/indexers/all/results/torznab",
      "apiKey": "f963a60693dd49a08ff75188f9fc72d2"
    }
  ],
  "stores": [...]
}
```

**Chillproxy expects** (correct format):
```json
{
  "indexers": [
    {
      "n": "prowlarr",                      // ❌ MISSING - indexer name
      "u": "http://localhost:9696",         // ❌ WRONG URL - should be base, not torznab endpoint
      "ak": "f963a60693dd49a08ff75188f9fc72d2"  // ✅ OK - but wrong key name
    }
  ],
  "stores": [...]
}
```

---

## 📋 Chillproxy Schema Requirements

From `internal/stremio/userdata/indexers.go`:

```go
type Indexer struct {
    Name   IndexerName `json:"n"`      // "prowlarr", "jackett", "generic"
    URL    string      `json:"u"`      // Base URL only (e.g., "http://localhost:9696")
    APIKey string      `json:"ak,omitempty"`
}

const (
    IndexerNameGeneric   IndexerName = "generic"
    IndexerNameJackett   IndexerName = "jackett"
    IndexerNameProwlarr  IndexerName = "prowlarr"  // ← We added this
)
```

**Validation Logic**:
```go
func (i Indexer) Validate() (string, error) {
    if i.Name == "" {
        return "name", fmt.Errorf("indexer name is required")  // ← ERROR HERE
    }
    if i.URL == "" {
        return "url", fmt.Errorf("indexer url is required")
    }
    // ...
}
```

**The error `"unsupported indexer: ""` is because `Name` field is empty!**

---

## 🔧 How to Fix

### Option 1: Update Chillstreams Manifest Generation (RECOMMENDED)

Chillstreams needs to format Prowlarr addon as a proper StremThru/Chillproxy indexer.

**Where to Fix**: The code that generates the manifest URL with the config

**Current** (wrong):
```typescript
// Somewhere in Chillstreams codebase
const config = {
  indexers: [{
    url: "http://localhost:9696/api/v2.0/indexers/all/results/torznab",
    apiKey: prowlarrApiKey
  }],
  stores: [...]
}
```

**Should be** (correct):
```typescript
const config = {
  indexers: [{
    n: "prowlarr",                    // IndexerName
    u: "http://localhost:9696",       // Base URL (NOT the torznab endpoint)
    ak: prowlarrApiKey                // API key
  }],
  stores: [...]
}
```

### Option 2: Add Fallback Parser in Chillproxy

Modify chillproxy to accept both formats (backward compatible):

**File**: `internal/stremio/userdata/indexers.go`

```go
// Add a new unmarshal method to handle both formats
func (i *Indexer) UnmarshalJSON(data []byte) error {
    // Try standard format first
    type alias Indexer
    var standard alias
    if err := json.Unmarshal(data, &standard); err == nil && standard.Name != "" {
        *i = Indexer(standard)
        return nil
    }

    // Try alternate format (url, apiKey)
    var alt struct {
        URL    string `json:"url"`
        APIKey string `json:"apiKey"`
    }
    if err := json.Unmarshal(data, &alt); err != nil {
        return err
    }

    // Convert to standard format
    if strings.Contains(alt.URL, "9696") {
        // Assume Prowlarr if port 9696
        i.Name = IndexerNameProwlarr
        i.URL = extractBaseURL(alt.URL)  // Extract "http://localhost:9696"
        i.APIKey = alt.APIKey
    } else {
        // Default to generic
        i.Name = IndexerNameGeneric
        i.URL = alt.URL
        i.APIKey = alt.APIKey
    }
    
    return nil
}

func extractBaseURL(fullURL string) string {
    u, err := url.Parse(fullURL)
    if err != nil {
        return fullURL
    }
    return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}
```

---

## 🎯 Recommended Fix (Chillstreams Side)

Find where the Prowlarr addon configuration is being serialized and fix it:

### Step 1: Find the Code

Search for where the manifest config is built:
```typescript
// Likely in packages/core/src/builtins/prowlarr/ or packages/server/
```

### Step 2: Update Config Generation

**Before**:
```typescript
const prowlarrConfig = {
  indexers: [{
    url: prowlarrUrl,        // ❌ Wrong
    apiKey: prowlarrApiKey   // ❌ Wrong key name
  }]
}
```

**After**:
```typescript
const prowlarrConfig = {
  indexers: [{
    n: "prowlarr",           // ✅ Correct - indexer name
    u: prowlarrUrl,          // ✅ Correct - base URL
    ak: prowlarrApiKey       // ✅ Correct - API key
  }]
}
```

### Step 3: Update URL Format

Ensure the URL is the **base URL only**, not the Torznab endpoint:

**Wrong**:
```
http://localhost:9696/api/v2.0/indexers/all/results/torznab
```

**Correct**:
```
http://localhost:9696
```

Chillproxy will internally construct the correct endpoint based on the indexer name.

---

## 📝 Test Cases

After fixing, test with this config:

```json
{
  "indexers": [
    {
      "n": "prowlarr",
      "u": "http://localhost:9696",
      "ak": "f963a60693dd49a08ff75188f9fc72d2"
    }
  ],
  "stores": [
    {
      "c": "tb",
      "t": "",
      "auth": "3b94cb45-3f99-406e-9c40-ecce61a405cc"
    }
  ]
}
```

**Base64 encode it**:
```powershell
$config = @'
{"indexers":[{"n":"prowlarr","u":"http://localhost:9696","ak":"f963a60693dd49a08ff75188f9fc72d2"}],"stores":[{"c":"tb","t":"","auth":"3b94cb45-3f99-406e-9c40-ecce61a405cc"}]}
'@
[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($config))
```

**Test URL**:
```
http://localhost:8080/stremio/torz/{base64_config}/stream/movie/tt9362736.json
```

---

## 🔍 Where to Look in Chillstreams

Search these locations:

1. **Wizard Pro JSON**:
   - `resources/manifest/wizard_pro.json`
   - Look for Prowlarr addon configuration

2. **Builtin Prowlarr**:
   - `packages/core/src/builtins/prowlarr/addon.ts`
   - Check how config is serialized

3. **Manifest Generation**:
   - `packages/server/src/routes/stremio/manifest.ts`
   - Check how addons are added to manifest

4. **User Data Schema**:
   - `packages/core/src/db/schemas.ts`
   - Check addon configuration schema

---

## 🚀 Quick Fix Command

If you can't find the source, add this to wizard_pro.json:

```json
{
  "type": "custom",
  "instanceId": "prowlarr-torz",
  "enabled": true,
  "options": {
    "name": "Prowlarr Torz",
    "manifestUrl": "http://localhost:8080/stremio/torz/eyJpbmRleGVycyI6W3sibiI6InByb3dsYXJyIiwidSI6Imh0dHA6Ly9sb2NhbGhvc3Q6OTY5NiIsImFrIjoiZjk2M2E2MDY5M2RkNDlhMDhmZjc1MTg4ZjlmYzcyZDIifV0sInN0b3JlcyI6W3siYyI6InRiIiwidCI6IiIsImF1dGgiOiJ7e3VzZXJJZH19In1dfQ==/manifest.json",
    "timeout": 50000,
    "resources": ["stream"],
    "mediaTypes": ["movie", "series"]
  }
}
```

---

## ✅ Summary

**Problem**: Config format mismatch between Chillstreams and Chillproxy

**Solution**: Update Chillstreams to send:
```
{n: "prowlarr", u: "http://localhost:9696", ak: "..."}
```

Instead of:
```
{url: "http://localhost:9696/...", apiKey: "..."}
```

**Next Step**: Find where Chillstreams generates the Prowlarr config and fix the format

---

**Status**: ❌ **BLOCKED - CONFIG FORMAT ERROR**  
**Action Required**: Update Chillstreams manifest generation


