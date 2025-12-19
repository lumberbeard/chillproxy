# Prowlarr Addon Entry for wizard_pro.json

Add this to the `presets` array in your `wizard_pro.json` file:

```json
{
  "type": "custom",
  "instanceId": "prowlarr-torz",
  "enabled": true,
  "options": {
    "name": "Prowlarr Torz",
    "manifestUrl": "http://localhost:8080/stremio/torz/eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0=/manifest.json",
    "timeout": 20000,
    "resources": [],
    "mediaTypes": [],
    "libraryAddon": false,
    "formatPassthrough": false,
    "resultPassthrough": false,
    "forceToTop": false
  }
}
```

---

## How to Add It

Open `c:\chillstreams\resources\manifest\wizard_pro.json` and:

1. Find the `"presets"` array (around line 73)
2. Add the entry above to the array (after the existing `aiometadata` preset)
3. Save the file

**Before**:
```json
"presets": [
  {
    "type": "custom",
    "instanceId": "aiometadata",
    "enabled": true,
    ...
  }
],
```

**After**:
```json
"presets": [
  {
    "type": "custom",
    "instanceId": "aiometadata",
    "enabled": true,
    ...
  },
  {
    "type": "custom",
    "instanceId": "prowlarr-torz",
    "enabled": true,
    "options": {
      "name": "Prowlarr Torz",
      "manifestUrl": "http://localhost:8080/stremio/torz/...",
      ...
    }
  }
],
```

---

## Configuration Reference

| Field | Value | Notes |
|-------|-------|-------|
| `type` | `custom` | External addon |
| `instanceId` | `prowlarr-torz` | Unique identifier for this addon |
| `enabled` | `true` | Enable by default in wizard |
| `name` | `Prowlarr Torz` | Display name in Stremio |
| `manifestUrl` | See below | Complete manifest URL with base64 config |
| `timeout` | `20000` | 20 second timeout (torrent search can be slow) |
| `forceToTop` | `false` | Don't force to top (keep alphabetical) |

---

## Your Manifest URL

```
http://localhost:8080/stremio/torz/eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0=/manifest.json
```

This includes:
- ✅ Prowlarr indexer URL
- ✅ Prowlarr API key
- ✅ Chillstreams user UUID
- ✅ TorBox pool authentication

---

## When the Wizard Runs

Users going through the wizard will now see:
1. **AIOMetadata** (metadata addon)
2. **Prowlarr Torz** (torrent indexing + TorBox debrid)

Both will be included in the final manifest configuration.


