# Chillproxy Quick Reference

## ✅ Status: RUNNING & VERIFIED

---

## 🚀 Quick Start

### Access
- **Dashboard**: http://localhost:8080/
- **Container**: `chillproxy-test`
- **Version**: 0.94.3

### Credentials (Auto-Generated)
```
Username: st-r7szcnl
Password: LkTsmuDYNwA0x5TbidjKIlvJWGm
```

---

## 📡 Endpoints

### Main Addons
- **Store**: http://localhost:8080/stremio/store/manifest.json
- **Torz**: http://localhost:8080/stremio/torz/{base64_config}/manifest.json
- **Wrap**: http://localhost:8080/stremio/wrap/manifest.json
- **List**: http://localhost:8080/stremio/list/manifest.json

### Create Torz Config
```powershell
# Basic config (no TorBox key)
$config = @{stores=@(@{c="tb";t=""})} | ConvertTo-Json -Compress
$base64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($config))
Write-Host "http://localhost:8080/stremio/torz/$base64/manifest.json"

# With TorBox key
$config = @{stores=@(@{c="tb";t="YOUR_TORBOX_KEY"})} | ConvertTo-Json -Compress
$base64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($config))
Write-Host "http://localhost:8080/stremio/torz/$base64/manifest.json"
```

---

## 🐳 Docker Commands

### View Logs
```powershell
docker logs chillproxy-test           # All logs
docker logs -f chillproxy-test        # Follow mode
docker logs --tail 50 chillproxy-test # Last 50 lines
```

### Container Control
```powershell
docker stop chillproxy-test   # Stop
docker start chillproxy-test  # Start
docker restart chillproxy-test # Restart
docker rm chillproxy-test     # Remove (must stop first)
```

### Rebuild
```powershell
docker stop chillproxy-test; docker rm chillproxy-test
docker build -t chillproxy:test .
docker run -d --name chillproxy-test -p 8080:8080 chillproxy:test
```

---

## 🧪 Test Commands

### Quick Health Check
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/" -UseBasicParsing
```

### Full Test Suite
```powershell
# Saved in repo - run anytime:
# (see TEST_RESULTS.md for the command)
```

---

## 📋 Supported Stores

| Store | Code | Auth Type |
|-------|------|-----------|
| TorBox | `tb` | API Key |
| RealDebrid | `realdebrid` | API Token |
| AllDebrid | `alldebrid` | API Key |
| Premiumize | `premiumize` | API Key |
| Debrid-Link | `debridlink` | API Key |
| EasyDebrid | `easydebrid` | API Key |
| OffCloud | `offcloud` | Email:Password |
| PikPak | `pikpak` | Email:Password |

---

## 🔧 Troubleshooting

### Can't Connect
```powershell
# Check container is running
docker ps --filter name=chillproxy-test

# Check logs for errors
docker logs chillproxy-test --tail 20
```

### Port Conflict
```powershell
# Find what's using port 8080
Get-NetTCPConnection -LocalPort 8080

# Or use different port
docker run -d --name chillproxy-test -p 8081:8080 chillproxy:test
# Then access at http://localhost:8081/
```

### Clear Database
```powershell
docker stop chillproxy-test
docker rm chillproxy-test
# Recreate container (fresh DB)
docker run -d --name chillproxy-test -p 8080:8080 chillproxy:test
```

---

## 📚 Documentation

- **Full Test Results**: `TEST_RESULTS.md`
- **Setup Guide**: `TESTING_GUIDE.md`
- **Quick Start**: `QUICKSTART.md`
- **Integration Plan**: `docs/INTEGRATION_PLAN.md`
- **Architecture**: `docs/README.md`

---

## 🎯 Next Steps

### Option A: Test with TorBox
1. Get TorBox API key from https://torbox.app
2. Create manifest URL with key
3. Add to Stremio and test streams

### Option B: Begin Integration
1. Review `docs/INTEGRATION_PLAN.md`
2. Start Phase 1: Core Modifications
3. Add Chillstreams API client

---

## 💡 Key Learnings

### Current Authentication
```
Manifest URL contains base64 config:
{
  "stores": [
    {"c": "tb", "t": "user_api_key"}
           ↑            ↑
        Store code   User's key (EXPOSED!)
  ]
}
```

### Target Authentication (After Integration)
```
Manifest URL contains base64 config:
{
  "stores": [
    {"c": "tb", "auth": "user-uuid"}
           ↑              ↑
        Store code   Chillstreams user ID (SAFE!)
  ]
}

Chillproxy calls Chillstreams to get pool key → TorBox
```

---

**Status**: ✅ Baseline verified, ready for next phase  
**Last Updated**: December 16, 2025

