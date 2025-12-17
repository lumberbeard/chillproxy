# Chillproxy Local Testing Guide

**Goal**: Get chillproxy (StremThru fork) running locally to verify it works before making modifications.

---

## Prerequisites

### 1. Install Go

**Download Go 1.21 or higher**:
- Visit: https://go.dev/dl/
- Download: `go1.23.x.windows-amd64.msi` (or latest)
- Run installer (default settings are fine)

**Verify Installation**:
```pwsh
# Close and reopen PowerShell, then test:
go version
# Should show: go version go1.23.x windows/amd64
```

### 2. Install Make (Optional)

**Using Chocolatey** (if installed):
```pwsh
choco install make
```

**Or Download**:
- Visit: https://gnuwin32.sourceforge.net/packages/make.htm
- Add to PATH after installation

---

## Quick Start

### Step 1: Create Configuration

Already created `.env` file with basic config:
```dotenv
STREMTHRU_PORT=8080
STREMTHRU_BASE_URL=http://localhost:8080
STREMTHRU_DATABASE_URI=sqlite://./data/stremthru.db
STREMTHRU_FEATURE=+stremio-store,+stremio-torz,+stremio-wrap
STREMTHRU_LOG_LEVEL=INFO
STREMTHRU_LOG_FORMAT=text
STREMTHRU_DATA_DIR=./data
```

**Optional**: Add TorBox credentials for full testing:
```pwsh
# Edit .env and uncomment these lines:
# STREMTHRU_PROXY_AUTH=testuser:testpass
# STREMTHRU_STORE_AUTH=testuser:torbox:<YOUR_TORBOX_API_KEY>
```

### Step 2: Download Dependencies

```pwsh
cd C:\chillproxy
go mod download
```

This will download all required Go packages.

### Step 3: Build the Binary

```pwsh
go build -o chillproxy.exe .
```

This creates `chillproxy.exe` in the current directory.

### Step 4: Run the Server

```pwsh
.\chillproxy.exe
```

**Expected Output**:
```
stremthru listening on localhost:8080
```

---

## Testing Endpoints

### 1. Health Check

```pwsh
Invoke-WebRequest -Uri "http://localhost:8080/health"
```

**Expected**: HTTP 200 with health status

### 2. Root Endpoint

```pwsh
Invoke-WebRequest -Uri "http://localhost:8080/"
```

**Expected**: Welcome page or API info

### 3. Store Manifest (Basic)

```pwsh
# Basic store addon manifest (no auth required)
Invoke-WebRequest -Uri "http://localhost:8080/stremio/store/manifest.json"
```

**Expected**: JSON manifest with addon info

### 4. Torz Addon (Requires Config)

To test the Torz addon, you need a base64 config:

```pwsh
# Create config JSON
$config = @{
    stores = @(
        @{
            c = "tb"  # TorBox
            t = ""    # Empty for now (we'll add real key later)
        }
    )
} | ConvertTo-Json -Compress

# Encode to base64
$bytes = [System.Text.Encoding]::UTF8.GetBytes($config)
$base64 = [Convert]::ToBase64String($bytes)

# Test manifest
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$base64/manifest.json"
```

**Expected**: JSON manifest for Torz addon

---

## Testing with TorBox Integration

### Prerequisites

1. **TorBox Account**: Sign up at https://torbox.app
2. **Get API Key**: 
   - Login → Settings → API
   - Copy your API key

### Configure Store Auth

**Option 1: Environment Variable** (Recommended)
```pwsh
$env:STREMTHRU_STORE_AUTH="testuser:torbox:<YOUR_API_KEY>"
.\chillproxy.exe
```

**Option 2: .env File**
```dotenv
# Edit .env
STREMTHRU_PROXY_AUTH=testuser:testpass
STREMTHRU_STORE_AUTH=testuser:torbox:<YOUR_API_KEY>
```

### Test TorBox Integration

**1. Check User Info**:
```pwsh
# Create base64 auth header
$auth = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes("testuser:testpass"))

# Test authenticated endpoint
Invoke-WebRequest -Uri "http://localhost:8080/v0/store/user" `
  -Headers @{"X-StremThru-Authorization" = "Basic $auth"}
```

**Expected**: JSON with your TorBox user info

**2. Check Torrent Cache**:
```pwsh
# Example: Check if Big Buck Bunny is cached
$hash = "dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c"

Invoke-WebRequest -Uri "http://localhost:8080/v0/store/magnet/check" `
  -Method POST `
  -Headers @{
      "X-StremThru-Authorization" = "Basic $auth"
      "Content-Type" = "application/json"
  } `
  -Body (@{hash = $hash} | ConvertTo-Json)
```

**Expected**: JSON indicating if torrent is cached

**3. Generate Manifest with TorBox**:
```pwsh
# Create config with your TorBox key
$config = @{
    stores = @(
        @{
            c = "tb"
            t = "<YOUR_TORBOX_API_KEY>"
        }
    )
} | ConvertTo-Json -Compress

$bytes = [System.Text.Encoding]::UTF8.GetBytes($config)
$base64 = [Convert]::ToBase64String($bytes)

# Get manifest
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$base64/manifest.json"
```

**Expected**: JSON manifest with Torz addon configured for TorBox

---

## Testing with Stremio

### 1. Generate Manifest URL

```pwsh
# With your TorBox API key
$config = @{
    stores = @(
        @{
            c = "tb"
            t = "<YOUR_TORBOX_API_KEY>"
        }
    )
} | ConvertTo-Json -Compress

$bytes = [System.Text.Encoding]::UTF8.GetBytes($config)
$base64 = [Convert]::ToBase64String($bytes)

$manifestUrl = "http://localhost:8080/stremio/torz/$base64/manifest.json"
Write-Host "Manifest URL: $manifestUrl"
```

### 2. Add to Stremio

1. Open Stremio Desktop
2. Click **Addons** (puzzle piece icon)
3. Paste the manifest URL in the search box
4. Click **Install**

### 3. Test Streaming

1. Search for a movie/show in Stremio
2. Click on it
3. Look for streams from "Torz" addon
4. Click a stream to play

**Expected**: Stream should load and play

---

## Troubleshooting

### Port Already in Use

```pwsh
# Check what's using port 8080
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue

# Kill process if needed
Stop-Process -Id <PID> -Force

# Or change port in .env
STREMTHRU_PORT=8081
```

### Database Errors

```pwsh
# Clear database and restart
Remove-Item -Recurse -Force .\data -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path .\data
.\chillproxy.exe
```

### TorBox API Errors

**Common errors**:
- `401 Unauthorized`: Check API key is correct
- `429 Too Many Requests`: TorBox rate limit, wait a minute
- `500 Internal Server Error`: TorBox service issue

**Debug**:
```pwsh
# Test TorBox API directly
$key = "<YOUR_API_KEY>"
Invoke-WebRequest -Uri "https://api.torbox.app/v1/api/user/me" `
  -Headers @{"Authorization" = "Bearer $key"}
```

### Build Errors

```pwsh
# Clear Go cache
go clean -cache -modcache

# Re-download dependencies
go mod download

# Rebuild
go build -o chillproxy.exe .
```

---

## Verification Checklist

Before modifying chillproxy, verify:

- [ ] Go installed (`go version` works)
- [ ] Dependencies downloaded (`go mod download` successful)
- [ ] Binary built (`chillproxy.exe` exists)
- [ ] Server starts (`.\chillproxy.exe` runs without errors)
- [ ] Health endpoint works (`/health` returns 200)
- [ ] Basic manifests load (`/stremio/store/manifest.json` works)
- [ ] TorBox auth works (if configured)
- [ ] Can add to Stremio (manifest URL installs)
- [ ] Streams play in Stremio (end-to-end test)

---

## Next Steps

Once verified working:
1. ✅ Baseline functionality confirmed
2. ✅ Understand how config/auth works
3. ✅ Ready to implement Chillstreams integration
4. 📋 Proceed with Phase 1 from `docs/INTEGRATION_PLAN.md`

---

## Common Commands Reference

```pwsh
# Build
go build -o chillproxy.exe .

# Run
.\chillproxy.exe

# Run with debug logging
$env:STREMTHRU_LOG_LEVEL="DEBUG"
.\chillproxy.exe

# Test health
Invoke-WebRequest -Uri "http://localhost:8080/health"

# Stop server
# Press Ctrl+C in the terminal running chillproxy

# View logs
# Logs output to console, redirect to file:
.\chillproxy.exe > app.log 2>&1
```

---

## Environment Variable Quick Reference

| Variable | Purpose | Example |
|----------|---------|---------|
| `STREMTHRU_PORT` | Server port | `8080` |
| `STREMTHRU_BASE_URL` | Public URL | `http://localhost:8080` |
| `STREMTHRU_DATABASE_URI` | Database | `sqlite://./data/stremthru.db` |
| `STREMTHRU_PROXY_AUTH` | Basic auth | `username:password` |
| `STREMTHRU_STORE_AUTH` | Store creds | `username:torbox:api_key` |
| `STREMTHRU_LOG_LEVEL` | Log level | `INFO`, `DEBUG` |
| `STREMTHRU_FEATURE` | Features | `+stremio-torz,+stremio-store` |

---

**Status**: Ready for testing  
**Last Updated**: December 16, 2025

