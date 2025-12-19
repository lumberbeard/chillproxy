# Prowlarr Fresh Installation & Setup Guide

**Date**: December 18, 2025  
**Environment**: Windows (Development) → Ubuntu (Production)  
**Status**: Fresh Install Required

---

## 🎯 Installation Steps for Windows

### Step 1: Download Prowlarr

```powershell
# Create temp directory
New-Item -ItemType Directory -Path "C:\Temp\Prowlarr" -Force

# Download latest Prowlarr for Windows
$url = "https://github.com/Prowlarr/Prowlarr/releases/download/v1.12.2.4211/Prowlarr.develop.1.12.2.4211.windows-core-x64.zip"
$output = "C:\Temp\Prowlarr\prowlarr.zip"

Write-Host "Downloading Prowlarr..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $url -OutFile $output

Write-Host "✅ Download complete" -ForegroundColor Green
```

### Step 2: Extract and Install

```powershell
# Extract
Write-Host "Extracting Prowlarr..." -ForegroundColor Cyan
Expand-Archive -Path "C:\Temp\Prowlarr\prowlarr.zip" -DestinationPath "C:\Prowlarr" -Force

Write-Host "✅ Extracted to C:\Prowlarr" -ForegroundColor Green
```

### Step 3: Initial Configuration

Create config file **before** first run:

```powershell
# Create config directory
New-Item -ItemType Directory -Path "C:\ProgramData\Prowlarr" -Force

# Create initial config.xml
$configXml = @"
<Config>
  <BindAddress>*</BindAddress>
  <Port>9696</Port>
  <SslPort>9897</SslPort>
  <EnableSsl>False</EnableSsl>
  <LaunchBrowser>True</LaunchBrowser>
  <ApiKey>f963a60693dd49a08ff75188f9fc72d2</ApiKey>
  <AuthenticationMethod>None</AuthenticationMethod>
  <Branch>develop</Branch>
  <LogLevel>info</LogLevel>
  <UrlBase></UrlBase>
  <InstanceName>Prowlarr</InstanceName>
</Config>
"@

$configXml | Out-File -FilePath "C:\ProgramData\Prowlarr\config.xml" -Encoding utf8 -Force

Write-Host "✅ Configuration file created" -ForegroundColor Green
Write-Host "   API Key: f963a60693dd49a08ff75188f9fc72d2" -ForegroundColor Yellow
Write-Host "   Auth: None (no login required)" -ForegroundColor Yellow
```

### Step 4: Start Prowlarr

```powershell
# Start Prowlarr
Write-Host "Starting Prowlarr..." -ForegroundColor Cyan
Start-Process -FilePath "C:\Prowlarr\Prowlarr.exe" -WindowStyle Minimized

# Wait for startup
Write-Host "Waiting 10 seconds for Prowlarr to start..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Check if running
try {
    $r = Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 3
    Write-Host "✅ Prowlarr is running!" -ForegroundColor Green
    Write-Host "   URL: http://localhost:9696" -ForegroundColor Cyan
} catch {
    Write-Host "⚠️ Prowlarr may still be starting..." -ForegroundColor Yellow
    Write-Host "   Check http://localhost:9696 in browser" -ForegroundColor Yellow
}
```

### Step 5: Add Indexers via API

```powershell
# API configuration
$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$headers = @{
    "X-Api-Key" = $apiKey
    "Content-Type" = "application/json"
}
$baseUrl = "http://localhost:9696/api/v1"

Write-Host "Adding indexers..." -ForegroundColor Cyan

# Add YTS indexer
$ytsConfig = @{
    enableRss = $true
    enableAutomaticSearch = $true
    enableInteractiveSearch = $true
    supportsRss = $true
    supportsSearch = $true
    protocol = "torrent"
    priority = 25
    downloadClientId = 0
    name = "YTS"
    fields = @(
        @{
            name = "baseUrl"
            value = "https://yts.mx"
        }
        @{
            name = "apiPath"
            value = "/api/v2/list_movies.json"
        }
    )
    implementationName = "YTS"
    implementation = "YTS"
    configContract = "YTSSettings"
    tags = @()
} | ConvertTo-Json -Depth 10

try {
    $r = Invoke-RestMethod -Uri "$baseUrl/indexer" -Method Post -Headers $headers -Body $ytsConfig
    Write-Host "✅ Added YTS indexer" -ForegroundColor Green
} catch {
    Write-Host "⚠️ YTS: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Add EZTV indexer
$eztvConfig = @{
    enableRss = $true
    enableAutomaticSearch = $true
    enableInteractiveSearch = $true
    supportsRss = $true
    supportsSearch = $true
    protocol = "torrent"
    priority = 25
    downloadClientId = 0
    name = "EZTV"
    fields = @(
        @{
            name = "baseUrl"
            value = "https://eztv.re"
        }
    )
    implementationName = "EZTV"
    implementation = "EZTV"
    configContract = "EZTVSettings"
    tags = @()
} | ConvertTo-Json -Depth 10

try {
    $r = Invoke-RestMethod -Uri "$baseUrl/indexer" -Method Post -Headers $headers -Body $eztvConfig
    Write-Host "✅ Added EZTV indexer" -ForegroundColor Green
} catch {
    Write-Host "⚠️ EZTV: $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✅ Prowlarr setup complete!" -ForegroundColor Green
```

---

## 🧪 Test the Setup

```powershell
# Test API access
$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$headers = @{"X-Api-Key" = $apiKey}

Write-Host "Testing Prowlarr API..." -ForegroundColor Cyan
Write-Host ""

# Test 1: List indexers
Write-Host "Test 1: List indexers" -ForegroundColor Yellow
try {
    $indexers = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/indexer" -Headers $headers
    Write-Host "✅ Found $($indexers.Count) indexers" -ForegroundColor Green
    $indexers | ForEach-Object { Write-Host "   - $($_.name)" }
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test 2: Search via API
Write-Host "Test 2: Search for 'matrix'" -ForegroundColor Yellow
try {
    $search = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/search?query=matrix&type=search" -Headers $headers
    Write-Host "✅ Found $($search.Count) results" -ForegroundColor Green
    $search | Select-Object -First 3 | ForEach-Object {
        Write-Host "   - $($_.title)"
    }
} catch {
    Write-Host "❌ Failed: $($_.Exception.Message)" -ForegroundColor Red
}
```

---

## 📋 Prowlarr API Endpoints

### Correct API Paths (v1, not v2.0!)

**List Indexers**:
```
GET http://localhost:9696/api/v1/indexer
Header: X-Api-Key: f963a60693dd49a08ff75188f9fc72d2
```

**Search All Indexers**:
```
GET http://localhost:9696/api/v1/search?query=matrix&type=search
Header: X-Api-Key: f963a60693dd49a08ff75188f9fc72d2
```

**Get Specific Indexer Torznab Feed** (for external tools):
```
GET http://localhost:9696/1/api?t=search&q=matrix&apikey=f963a60693dd49a08ff75188f9fc72d2
```

Note: The path is `/{indexerId}/api` NOT `/api/v2.0/indexers/all/results/torznab`

---

## 🔧 For Chillproxy Integration

When calling Prowlarr from Chillproxy, use this endpoint:

```go
// CORRECT
url := fmt.Sprintf("http://prowlarr:9696/api/v1/search?query=%s&type=search", 
    url.QueryEscape(query))

req, _ := http.NewRequest("GET", url, nil)
req.Header.Set("X-Api-Key", apiKey)
req.Header.Set("Accept", "application/json")

// Response is JSON, not XML
var results []ProwlarrSearchResult
json.NewDecoder(resp.Body).Decode(&results)
```

**NOT** the Torznab XML endpoint (that's for external indexer managers like Sonarr/Radarr).

---

## 🐧 Ubuntu Production Setup

For your production environment on Ubuntu:

```bash
# Install Prowlarr
cd /opt
wget https://github.com/Prowlarr/Prowlarr/releases/download/v1.12.2.4211/Prowlarr.develop.1.12.2.4211.linux-core-x64.tar.gz
tar -xvzf Prowlarr.develop.1.12.2.4211.linux-core-x64.tar.gz

# Create user
useradd -r -s /bin/false prowlarr

# Set permissions
chown -R prowlarr:prowlarr /opt/Prowlarr

# Create config directory
mkdir -p /var/lib/prowlarr
chown prowlarr:prowlarr /var/lib/prowlarr

# Create systemd service
cat > /etc/systemd/system/prowlarr.service << 'EOF'
[Unit]
Description=Prowlarr Daemon
After=network.target

[Service]
User=prowlarr
Group=prowlarr
Type=simple
ExecStart=/opt/Prowlarr/Prowlarr -nobrowser -data=/var/lib/prowlarr
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Create initial config
cat > /var/lib/prowlarr/config.xml << 'EOF'
<Config>
  <BindAddress>*</BindAddress>
  <Port>9696</Port>
  <ApiKey>f963a60693dd49a08ff75188f9fc72d2</ApiKey>
  <AuthenticationMethod>None</AuthenticationMethod>
  <Branch>develop</Branch>
  <LogLevel>info</LogLevel>
</Config>
EOF

chown prowlarr:prowlarr /var/lib/prowlarr/config.xml

# Start service
systemctl daemon-reload
systemctl enable prowlarr
systemctl start prowlarr

# Check status
systemctl status prowlarr
```

---

## 📝 Summary

**Key Differences from Before**:
1. ✅ Using **API v1** endpoints (not v2.0)
2. ✅ Using **/api/v1/search** (not Torznab XML)
3. ✅ Response is **JSON** (not XML)
4. ✅ No authentication required (set to None)
5. ✅ API key pre-configured before first run

**For Chillproxy**:
- Use `http://prowlarr:9696/api/v1/search?query={query}&type=search`
- Header: `X-Api-Key: {apiKey}`
- Parse JSON response (not XML)
- Each result has: `title`, `guid`, `infoHash`, `seeders`, `peers`

---

**Status**: Ready to install  
**Next**: Run the PowerShell commands above to install fresh


