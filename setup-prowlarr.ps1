#!/usr/bin/env powershell
# Prowlarr + Chillproxy Setup Assistant
# This script will guide you through the entire setup process

param(
    [string]$ProwlarrUrl = "http://localhost:9696",
    [string]$ProwlarrApiKey = "",
    [string]$UserUuid = "3b94cb45-3f99-406e-9c40-ecce61a405cc",
    [string]$ChillproxUrl = "http://localhost:8080",
    [string]$ChillstreamsUrl = "http://localhost:3000"
)

function Write-Title {
    param([string]$Text)
    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║ $($Text.PadRight(64)) ║" -ForegroundColor Cyan
    Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Step {
    param([string]$Text, [int]$Number)
    Write-Host "[$Number] $Text" -ForegroundColor Yellow
}

function Write-Success {
    param([string]$Text)
    Write-Host "✅ $Text" -ForegroundColor Green
}

function Write-Error {
    param([string]$Text)
    Write-Host "❌ $Text" -ForegroundColor Red
}

function Write-Info {
    param([string]$Text)
    Write-Host "ℹ️  $Text" -ForegroundColor Blue
}

# Main Script

Write-Title "Prowlarr + Chillproxy Setup"

Write-Info "This script will help you:"
Write-Info "  1. Verify Prowlarr is running"
Write-Info "  2. Verify indexers are enabled"
Write-Info "  3. Get your API key"
Write-Info "  4. Create Chillproxy configuration"
Write-Info "  5. Test the integration"

Write-Host ""
Write-Host "Press Enter to continue..." -ForegroundColor Gray
Read-Host

# Step 1: Check Prowlarr
Write-Title "Step 1: Verify Prowlarr is Running"

Write-Step "Checking Prowlarr at $ProwlarrUrl" 1

try {
    $response = Invoke-WebRequest -Uri "$ProwlarrUrl" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Success "Prowlarr is running!"
} catch {
    Write-Error "Cannot reach Prowlarr at $ProwlarrUrl"
    Write-Host ""
    Write-Host "Troubleshooting:" -ForegroundColor Yellow
    Write-Host "  1. Make sure Prowlarr is installed and running"
    Write-Host "  2. Check if Prowlarr UI opens: $ProwlarrUrl"
    Write-Host "  3. If on different port, modify the URL above"
    Write-Host ""
    $continueAnyway = Read-Host "Continue anyway? (y/n)"
    if ($continueAnyway -ne 'y') { exit }
}

# Step 2: Get API Key
Write-Title "Step 2: Get Prowlarr API Key"

if ([string]::IsNullOrEmpty($ProwlarrApiKey)) {
    Write-Step "You need to get your API key from Prowlarr" 2
    Write-Host ""
    Write-Host "Instructions:" -ForegroundColor Cyan
    Write-Host "  1. Open: $ProwlarrUrl/settings/general"
    Write-Host "  2. Scroll down to 'Security' section"
    Write-Host "  3. Look for 'API Key' (long string)"
    Write-Host "  4. Copy it"
    Write-Host ""
    $ProwlarrApiKey = Read-Host "Paste your Prowlarr API Key"

    if ([string]::IsNullOrEmpty($ProwlarrApiKey)) {
        Write-Error "API Key is required!"
        exit
    }
}

Write-Success "API Key saved (first 10 chars): $($ProwlarrApiKey.Substring(0, [Math]::Min(10, $ProwlarrApiKey.Length)))..."

# Step 3: Test Prowlarr Search
Write-Title "Step 3: Test Prowlarr Search"

Write-Step "Testing Prowlarr search capability" 3

$torznabUrl = "$ProwlarrUrl/api/v2.0/indexers/all/results/torznab"
$testUrl = "$torznabUrl?t=tvsearch&q=breaking+bad&season=1&ep=1&apikey=$ProwlarrApiKey"

try {
    $response = Invoke-WebRequest -Uri $testUrl -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop

    if ($response.StatusCode -eq 200) {
        Write-Success "Prowlarr search is working!"
        Write-Info "Torznab URL: $torznabUrl"
    } else {
        Write-Error "Prowlarr returned status $($response.StatusCode)"
    }
} catch {
    Write-Error "Failed to query Prowlarr: $($_.Exception.Message)"
    Write-Host ""
    Write-Host "Possible issues:" -ForegroundColor Yellow
    Write-Host "  - API key is incorrect"
    Write-Host "  - Indexers are not enabled in Prowlarr"
    Write-Host "  - Prowlarr is not responding"
}

# Step 4: Create Configuration
Write-Title "Step 4: Create Chillproxy Configuration"

Write-Step "Creating configuration object" 4

$config = @{
  stores = @(
    @{
      c = "tb"
      t = ""
      auth = $UserUuid
    }
  )
  indexers = @(
    @{
      url = "$torznabUrl"
      apiKey = $ProwlarrApiKey
    }
  )
}

$configJson = $config | ConvertTo-Json -Compress
Write-Success "Configuration JSON created"
Write-Info "Size: $($configJson.Length) characters"

# Encode to Base64
Write-Step "Encoding configuration to Base64" 5

$configBytes = [System.Text.Encoding]::UTF8.GetBytes($configJson)
$configBase64 = [Convert]::ToBase64String($configBytes)

Write-Success "Configuration encoded"
Write-Info "Base64 size: $($configBase64.Length) characters"

# Save to file
$configFile = "prowlarr_config.txt"
$configBase64 | Out-File -FilePath $configFile -Force
Write-Success "Saved to: $configFile"

# Step 5: Test Chillproxy
Write-Title "Step 5: Test Chillproxy Integration"

Write-Step "Testing manifest endpoint" 6

$manifestUrl = "$ChillproxUrl/stremio/torz/$configBase64/manifest.json"

try {
    $response = Invoke-WebRequest -Uri $manifestUrl -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop

    if ($response.StatusCode -eq 200) {
        Write-Success "Manifest endpoint working!"

        $manifest = $response.Content | ConvertFrom-Json
        Write-Info "Addon ID: $($manifest.id)"
        Write-Info "Addon Name: $($manifest.name)"
        Write-Info "Resources: $($manifest.resources.Count)"
    } else {
        Write-Error "Manifest returned status $($response.StatusCode)"
    }
} catch {
    Write-Error "Failed to reach manifest endpoint: $($_.Exception.Message)"
    Write-Info "Make sure Chillproxy is running on $ChillproxUrl"
}

# Step 6: Test Stream Search
Write-Title "Step 6: Test Stream Search"

Write-Step "Searching for Breaking Bad S01E01" 7

$streamUrl = "$ChillproxUrl/stremio/torz/$configBase64/stream/series/tt0903747:1:1.json"

try {
    $response = Invoke-WebRequest -Uri $streamUrl -UseBasicParsing -TimeoutSec 15 -ErrorAction Stop

    if ($response.StatusCode -eq 200) {
        $streams = $response.Content | ConvertFrom-Json

        Write-Success "Stream search successful!"
        Write-Info "Found $($streams.streams.Count) results"

        if ($streams.streams.Count -gt 0) {
            Write-Host ""
            Write-Host "Sample results:" -ForegroundColor Cyan
            $streams.streams[0..2] | ForEach-Object {
                Write-Host "  • $($_.title)"
            }
        } else {
            Write-Error "No results found - check if indexers have content"
        }
    }
} catch {
    Write-Error "Failed to search streams: $($_.Exception.Message)"
}

# Step 7: Summary
Write-Title "Setup Complete!"

Write-Host "Your configuration is ready to use:" -ForegroundColor Green
Write-Host ""
Write-Host "Configuration File: $configFile" -ForegroundColor Cyan
Write-Host ""
Write-Host "Manifest URL:" -ForegroundColor Cyan
Write-Host "$manifestUrl" -ForegroundColor Yellow
Write-Host ""
Write-Host "Base64 Config (for reference):" -ForegroundColor Cyan
Write-Host $configBase64 -ForegroundColor Yellow
Write-Host ""

Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Copy the Manifest URL above"
Write-Host "  2. Add it to Stremio as an addon"
Write-Host "  3. Search for content in Stremio"
Write-Host ""

Write-Host "Troubleshooting:" -ForegroundColor Cyan
Write-Host "  • If no results, check Prowlarr indexers are enabled"
Write-Host "  • If 404 error, verify Chillproxy is running"
Write-Host "  • If pool key error, check Chillstreams is running"
Write-Host ""

Write-Host "Press Enter to exit..." -ForegroundColor Gray
Read-Host

