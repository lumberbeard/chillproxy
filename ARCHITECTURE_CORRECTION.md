# Critical Rethink: Credential Exposure Problem

**Date**: December 17, 2025  
**Status**: Architecture Correction Required

---

## 🚨 The Problem You Identified

### **The Core Issue**

**TorBox Search (and TorrentGalaxy, etc.)** require the **actual TorBox API key** to be in the user-facing manifest/config.

```typescript
// THIS IS THE PROBLEM:
const config = {
  torBoxApiKey: "actual_torbox_key_here",  // ← EXPOSED TO USER!
  sources: ["torrent"],
  services: [...]
};

// User sees this in manifest URL (even base64 encoded)
const manifest = btoa(JSON.stringify(config));
// Browser/Stremio can extract the key
```

**Why this doesn't work**:
1. ❌ Key is visible in manifest URL
2. ❌ User can extract the key
3. ❌ User can share the key with others
4. ❌ Can't revoke individual users
5. ❌ Can't track/limit usage per user
6. ❌ Can't enforce device limits

---

## ✅ What YOU Actually Need

You're correct:

> **"It needs to only be from our server to torbox, not with the end user."**

This means:

1. ✅ **Chillstreams** (server-side) has the TorBox API key
2. ✅ **Chillproxy** (server-side) gets pool keys from Chillstreams
3. ✅ **Users never see any credentials**
4. ✅ **All TorBox calls are server-to-server**

---

## 🔄 The Correct Architecture

### **What Should Happen**

```
┌─────────────────────────────────────────────────────────┐
│                    USER (Browser/Stremio)               │
│                  (NO credentials here)                  │
└──────────────────────┬──────────────────────────────────┘
                       │
                       │ User UUID only
                       ↓
┌──────────────────────────────────────────────────────────┐
│              CHILLPROXY (Public Gateway)                 │
│  /stremio/torz/{base64_config}/stream/{id}              │
│                                                          │
│  config = {                                              │
│    stores: [{c: "tb", t: "", auth: "user-uuid"}]        │
│  }                                                       │
│                                                          │
│  1. Receives stream request                             │
│  2. Calls Chillstreams API (internal only)              │
└──────────────────┬───────────────────────────────────────┘
                   │
    Internal API call (server-to-server, not exposed)
                   │
                   ↓
┌──────────────────────────────────────────────────────────┐
│           CHILLSTREAMS (Internal Server)                 │
│      /api/v1/internal/pool/get-key (Private)            │
│                                                          │
│  - Has actual TorBox API key in env var                 │
│  - Looks up pool key for this user                      │
│  - Returns pool key to chillproxy                       │
└──────────────────┬───────────────────────────────────────┘
                   │
    Pool key only (shared, not user-specific)
                   │
                   ↓
┌──────────────────────────────────────────────────────────┐
│           CHILLPROXY (Continues Request)                │
│                                                          │
│  - Uses pool key to call TorBox API                     │
│  - TorBox never knows about user UUID                   │
│  - Returns stream URL to user                           │
└──────────────────────────────────────────────────────────┘
```

**Key Points**:
- ✅ User never sees TorBox key
- ✅ Pool key is shared (not sensitive)
- ✅ All actual TorBox API calls are server-side
- ✅ Chillproxy acts as proxy between user and TorBox

---

## ❌ What NOT To Do

### **TorBox Search / TorrentGalaxy in User Manifest**
```typescript
// DON'T DO THIS:
{
  "builtinAddons": {
    "torboxSearch": {
      "torBoxApiKey": "actual_key_here"  // ❌ EXPOSED
    }
  }
}
```

**Why**:
- The key is in the manifest
- Manifest is part of user config
- User can extract the key
- Defeats the entire pool system

---

## ✅ What YOU Should Do

### **Option 1: Chillproxy/Torz with Prowlarr (Recommended)**

```
Chillproxy Torz has BUILT-IN torrent searching:
├─ Prowlarr (indexer backend)  ← Server-side
│  └─ Searches: YTS, RARBG, EZTV, TPB, TorrentGalaxy
├─ Chillstreams Pool API       ← Server-side  
└─ TorBox API                  ← Server-side

User only sees:
└─ /stremio/torz/{config}/manifest.json
   where config = {stores: [{c: "tb", auth: "user-uuid"}],
                   indexers: [{url: "prowlarr-url", key: "..."}]}
```

**Flow**:
```
User request → Chillproxy/Torz (built-in search) 
            → Prowlarr (Torznab backend)
            → Torrent sites (YTS, RARBG, etc.)
            → Chillstreams Pool API
            → TorBox API
            → Stream
All server-side, user sees no credentials (only indexer URL)
```

### **Option 2: Chillstreams + Chillproxy (Both Server-Side)**

```
Chillstreams:
├─ External addons (Torrentio, Comet, etc.)
│  └─ These return magnet links
└─ TorBox Search (if we fix the credential issue)

Chillproxy:
├─ Jackett indexing
└─ TorBox streams (via pool keys)

Both use Chillstreams Pool API
```

**The Key Insight**: 
- External addons (Torrentio, Comet) return **magnet links**
- Magnet links have NO credentials (just hashes)
- Any debrid service can handle the magnet
- So Chillstreams can use external addons safely

**But TorBox Search**:
- Requires TorBox API key in the code
- Needs to be server-side only
- Can't be in user manifest

---

## 📋 Your Assessment Is Correct

### **Point 1: TorBox Search Requires Real Credentials**
✅ **You're right**. TorBox Search needs the actual API key, which means:
- ❌ Can't be in user manifest
- ✅ Must be called server-side only
- ✅ Chillstreams has the key, uses it internally
- ✅ Only returns results to users (not the key)

### **Point 2: TorrentGalaxy Probably Has Same Issue**
✅ **You're right**. Checking the code:
- Built-in indexers require credentials
- They use `services: [{id: "torbox", credentials: {...}}]`
- Same problem as TorBox Search
- Can't be in user manifest

### **Point 3: Only Chillproxy Should Handle Indexing**
✅ **You're right about Torz!** Here's the correction:

**Chillproxy HAS built-in torrent searching** via the `/stremio/torz/` endpoint:
- ✅ `GetStreamsFromIndexers()` in `stream.go` = built-in search function
- ❌ But it REQUIRES an indexer backend (Jackett or Prowlarr)
- ✅ Prowlarr is the faster/simpler option (vs Jackett)
- ✅ All credentials stay server-side (user only sees indexer URL)

**Flow**:
```
Chillproxy/Torz (built-in search) → Prowlarr (Torznab API) → Torrent sites
```

**Chillstreams can use**:
- ✅ External addons (Torrentio, Comet, etc.) - They return magnet links
- ❌ TorBox Search - Needs server-side API key (not in user manifest)
- ❌ TorrentGalaxy - Needs server-side credentials

**Chillproxy should use**:
- ✅ Prowlarr - Fastest, simplest, perfect for Stremio
- ✅ Jackett - Older, slower, more complex (skip unless you need 130 indexers)

---

## 🎯 The Right Solution

### **Forget TorBox Search in User Manifest**

TorBox Search should **only be called server-side by Chillstreams**, NOT exposed to users.

```typescript
// In Chillstreams (server-side):
class ChillstreamsInternalService {
  private torboxApiKey = process.env.TORBOX_API_KEY; // Server env var
  
  async getTorboxResults(imdbId: string) {
    // This is INTERNAL - never exposed to users
    const results = await fetch('https://search-api.torbox.app/torrents/imdb_id:' + imdbId, {
      headers: {
        Authorization: 'Bearer ' + this.torboxApiKey  // ← Hidden
      }
    });
    return results;
  }
}
```

But this should be **Chillstreams-only**, not in the user manifest.

### **What Users Get**

Users install two addons:

1. **Chillstreams Manifest**
   - Returns external addon results (Torrentio, Comet, etc.)
   - No credentials needed in user config
   - Uses Chillstreams pool API

2. **Chillproxy Manifest**
   - Searches via Jackett (server-side)
   - No credentials in user config
   - Uses Chillstreams pool API

---

## 📝 Corrected Architecture Document

The correct setup is:

### **Server-Side (Not Exposed)**
```
Chillstreams
├─ TorBox API key (env var)
├─ TorBox Search API (internal only)
└─ Pool key management

Chillproxy
├─ Jackett integration (local or remote)
└─ Uses Chillstreams pool API
```

### **User-Facing (No Credentials)**
```
User installs:
├─ Chillstreams manifest
│  └─ config: {addons: [...]}  ← No credentials
└─ Chillproxy manifest
   └─ config: {indexers: ["jackett-url"]}  ← Only Jackett URL
```

---

## 🚀 Recommended Path Forward

### **Phase 1: Verify Chillproxy + Jackett Works** ✅ (Current)
- Chillproxy with Jackett indexing
- Uses Chillstreams pool API
- Users need NO credentials in manifest

### **Phase 2: Keep Chillstreams Simple**
- External addons (Torrentio, Comet, etc.)
- They return magnet links (no credentials needed)
- Uses Chillstreams pool API

### **Phase 3: Optional - TorBox Search as Internal Feature**
- If you want, Chillstreams can use TorBox Search internally
- But ONLY for administrative features
- NOT exposed in user manifest
- Treat like any other server-side feature

---

## Summary of Corrections

| Question | Your Assessment | Correct Answer |
|----------|-----------------|-----------------|
| **TorBox Search requires real credentials?** | ✅ Yes | ✅ Correct - can't be in user manifest |
| **TorrentGalaxy has same issue?** | ✅ Probably | ✅ Correct - also needs credentials |
| **Only Chillproxy should handle indexing?** | ✅ Mostly | ✅ Correct - external addons are OK, built-in indexers are not |
| **Need server-to-server, not user-facing?** | ✅ Yes | ✅ Correct - this is the right approach |

---

## What This Means

### **Forget TorBox Search Integration into User Manifest**

Instead:

1. ✅ **Use Chillproxy with Jackett** for indexing
2. ✅ **Use Chillstreams with external addons** for aggregation
3. ✅ **Both use shared pool keys** via Chillstreams API
4. ✅ **No user credentials needed** anywhere

This is actually simpler than trying to expose TorBox Search!

---

**Status**: Rethinking Complete  
**Next**: Focus on Chillproxy + Jackett testing (original plan was correct)

