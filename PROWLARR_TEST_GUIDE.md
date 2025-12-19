# Prowlarr Testing Guide

**Date**: December 18, 2025  
**Status**: Ready to Test

---

## Current Status

❌ **Prowlarr is NOT running** on port 9696

You mentioned you installed Prowlarr but haven't started it yet. Here's how to get it running.

---

## Step 1: Find Your Prowlarr Installation

Prowlarr can be installed in several locations depending on your installation method:

**Check these locations**:
- `C:\Program Files\Prowlarr\`
- `C:\Program Files (x86)\Prowlarr\`
- `C:\ProgramData\Prowlarr\`
- `C:\Prowlarr\`

**In PowerShell**:
```powershell
# Find Prowlarr executable
Get-ChildItem -Path "C:\" -Recurse -Filter "Prowlarr.exe" -ErrorAction SilentlyContinue | Select-Object FullName
```

---

## Step 2: Start Prowlarr

### Option A: Installed as Windows Service

If you installed Prowlarr as a Windows service:

```powershell
# Check if service exists
Get-Service -Name Prowlarr -ErrorAction SilentlyContinue

# Start the service
Start-Service -Name Prowlarr

# Verify it started
Get-Service -Name Prowlarr

# You should see: Running
```

### Option B: Manual Binary Execution

If you have the binary:

```powershell
# Navigate to installation directory
cd "C:\Program Files\Prowlarr\"

# Run Prowlarr.exe
.\Prowlarr.exe

# Or run in background (new window)
Start-Process -FilePath ".\Prowlarr.exe"
```

### Option C: Docker (if using Docker)

```powershell
docker run -d `
  -p 9696:9696 `
  -e PUID=1000 `
  -e PGID=1000 `
  -v prowlarr_config:/config `
  --name prowlarr `
  lscr.io/linuxserver/prowlarr:latest
```

---

## Step 3: Verify Prowlarr is Running

```powershell
# Test if Prowlarr is responding
try {
    $r = Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 5
    Write-Host "✅ Prowlarr is running on port 9696"
    Write-Host "Status: $($r.StatusCode)"
} catch {
    Write-Host "❌ Prowlarr is not responding"
    Write-Host "Error: $($_.Exception.Message)"
}
```

---

## Step 4: Access Prowlarr UI

Once Prowlarr is running:

**Open in browser**: `http://localhost:9696`

You should see the Prowlarr dashboard.

---

## Step 5: Configure Prowlarr (if not already done)

### Verify Indexers are Enabled

1. Go to: `http://localhost:9696/settings/indexers`
2. Check that these are **enabled** (checkmark visible):
   - ✅ YTS
   - ✅ EZTV
   - ✅ RARBG
   - ✅ The Pirate Bay
   - ✅ TorrentGalaxy

3. If any are disabled, click them to enable

### Get Your API Key

1. Go to: `http://localhost:9696/settings/general`
2. Scroll down to "Security"
3. Copy your **API Key**

---

## Test Prowlarr API After Starting

Once Prowlarr is running, run this test:

```powershell
$apiKey = "YOUR_PROWLARR_API_KEY"  # Replace with actual key
$url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=search&q=matrix&apikey=$apiKey"

Write-Host "Testing Prowlarr search for Matrix..."
Write-Host "URL: $url"
Write-Host ""

try {
    $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 15
    Write-Host "✅ Status: $($r.StatusCode)"
    Write-Host ""
    
    # Count results
    if ($r.Content -match '<item>') {
        $items = [regex]::Matches($r.Content, '<item>')
        Write-Host "Found $($items.Count) torrents for 'Matrix'"
        Write-Host ""
        
        # Extract first torrent info
        if ($r.Content -match '<title>([^<]+)</title>') {
            Write-Host "First result: $($matches[1])"
        }
        if ($r.Content -match '<link>([^<]+)</link>') {
            Write-Host "First link: $($matches[1])"
        }
    } else {
        Write-Host "⚠️ No results found"
    }
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)"
}
```

---

## Testing the Full Integration

Once Prowlarr is running and tested:

### 1. Test Chillproxy with Prowlarr

```powershell
$config = "eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0="

Write-Host "Testing Chillproxy manifest..."
$manifestUrl = "http://localhost:8080/stremio/torz/$config/manifest.json"

try {
    $r = Invoke-WebRequest -Uri $manifestUrl -UseBasicParsing -TimeoutSec 10
    Write-Host "✅ Manifest Status: $($r.StatusCode)"
    
    $manifest = $r.Content | ConvertFrom-Json
    Write-Host "Addon: $($manifest.name)"
    Write-Host "Resources: $($manifest.resources.Count)"
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)"
}
```

### 2. Test Stream Search

```powershell
$config = "eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0="

Write-Host "Testing stream search for The Matrix (tt0133093)..."
$streamUrl = "http://localhost:8080/stremio/torz/$config/stream/movie/tt0133093.json"

try {
    $r = Invoke-WebRequest -Uri $streamUrl -UseBasicParsing -TimeoutSec 20
    Write-Host "✅ Stream Search Status: $($r.StatusCode)"
    
    $streams = $r.Content | ConvertFrom-Json
    Write-Host "Found $($streams.streams.Count) results"
    
    if ($streams.streams.Count -gt 0) {
        Write-Host ""
        Write-Host "First 3 results:"
        $streams.streams[0..2] | ForEach-Object {
            Write-Host "  - $($_.title)"
        }
    }
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)"
}
```

---

## Troubleshooting

### Prowlarr Won't Start

**Check logs**:
```powershell
# Windows service logs
Get-EventLog -LogName System -Source Prowlarr -Newest 10

# Or check Prowlarr's log files
Get-ChildItem "C:\ProgramData\Prowlarr\logs" -Recurse | Sort-Object LastWriteTime -Descending | Select-Object -First 5
```

### Port 9696 in Use

```powershell
# Find what's using port 9696
netstat -ano | Select-String "9696"

# Kill the process (if needed)
Stop-Process -Id <PID> -Force
```

### API Key Not Working

Make sure you're using the actual API key from Prowlarr settings, not the hardcoded one in the config.

---

## Next Steps

1. **Start Prowlarr** using one of the methods above
2. **Verify it's running** at `http://localhost:9696`
3. **Get your actual API key** from Prowlarr settings
4. **Update the config** with your real API key (the one starting with your actual key, not the example)
5. **Run the Prowlarr API test** above
6. **Run the Chillproxy integration test** above

Once you have Prowlarr running and the API test passes, we can test the full integration!

---

## Quick Start Commands

```powershell
# If using Windows service
Start-Service -Name Prowlarr

# Verify it's running
Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing

# Get your API key from browser
# Open: http://localhost:9696/settings/general
# Scroll to "Security" section
# Copy the API Key value
```

---

**Status**: Prowlarr needs to be started  
**Next**: Let me know when Prowlarr is running and I'll test the API!


