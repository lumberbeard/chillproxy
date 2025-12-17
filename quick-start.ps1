# Quick Start Script for Chillproxy Mock Testing
# This script sets up and starts the test environment

param(
    [switch]$BuildOnly,
    [switch]$StartMockOnly,
    [switch]$StartProxyOnly,
    [string]$TorBoxKey = $env:TORBOX_API_KEY
)

Write-Host "`n🚀 Chillproxy Mock Server Testing" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

# Check if TorBox API key is set
if (-not $TorBoxKey) {
    Write-Host "`n⚠️  WARNING: TorBox API key not set!" -ForegroundColor Yellow
    Write-Host "   Set it with: `$env:TORBOX_API_KEY = 'your_key'" -ForegroundColor Yellow
    Write-Host "   Or pass it: .\quick-start.ps1 -TorBoxKey 'your_key'" -ForegroundColor Yellow
    Write-Host "`n   Continuing without real TorBox testing...`n" -ForegroundColor Yellow
}

# Step 1: Check dependencies
Write-Host "`n📦 Checking dependencies..." -ForegroundColor Blue

# Check Node.js
try {
    $nodeVersion = node --version
    Write-Host "   ✅ Node.js: $nodeVersion" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Node.js not found! Install from https://nodejs.org" -ForegroundColor Red
    exit 1
}

# Check Go
try {
    $goVersion = go version
    Write-Host "   ✅ Go: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Go not found! Install from https://go.dev" -ForegroundColor Red
    exit 1
}

# Step 2: Install npm dependencies
Write-Host "`n📦 Installing npm dependencies..." -ForegroundColor Blue
if (-not (Test-Path "node_modules\express")) {
    npm install express --silent
    Write-Host "   ✅ Express installed" -ForegroundColor Green
} else {
    Write-Host "   ✅ Dependencies already installed" -ForegroundColor Green
}

# Step 3: Configure environment
Write-Host "`n⚙️  Configuring environment..." -ForegroundColor Blue
Copy-Item .env.local .env -Force
Write-Host "   ✅ .env configured for local testing" -ForegroundColor Green

# Step 4: Build chillproxy
if (-not $StartMockOnly) {
    Write-Host "`n🔨 Building chillproxy..." -ForegroundColor Blue
    $buildOutput = go build -o chillproxy.exe . 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "   ✅ Build successful" -ForegroundColor Green
    } else {
        Write-Host "   ❌ Build failed!" -ForegroundColor Red
        Write-Host $buildOutput
        exit 1
    }
}

if ($BuildOnly) {
    Write-Host "`n✅ Build complete! Ready to test." -ForegroundColor Green
    exit 0
}

# Step 5: Start services
Write-Host "`n🚀 Starting services..." -ForegroundColor Blue
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

if (-not $StartProxyOnly) {
    Write-Host "`n📡 Starting Mock Chillstreams API on port 3000..." -ForegroundColor Blue
    Write-Host "   Open in new terminal: node mock-chillstreams-api.js" -ForegroundColor Yellow

    # Set TorBox key for mock server
    if ($TorBoxKey) {
        $env:TORBOX_API_KEY = $TorBoxKey
    }

    # Start mock server in background
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PWD'; `$env:TORBOX_API_KEY='$TorBoxKey'; node mock-chillstreams-api.js"

    Write-Host "   ⏳ Waiting 3 seconds for mock server to start..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
}

if (-not $StartMockOnly) {
    Write-Host "`n🔧 Starting Chillproxy on port 8080..." -ForegroundColor Blue
    Write-Host "   Open in new terminal: .\chillproxy.exe" -ForegroundColor Yellow

    # Start chillproxy in background
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PWD'; .\chillproxy.exe"

    Write-Host "   ⏳ Waiting 3 seconds for chillproxy to start..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
}

# Step 6: Test endpoints
Write-Host "`n🧪 Testing endpoints..." -ForegroundColor Blue

# Test mock API
try {
    $mockHealth = Invoke-WebRequest -Uri http://localhost:3000/health -UseBasicParsing -TimeoutSec 5
    Write-Host "   ✅ Mock API: $($mockHealth.StatusCode) OK" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Mock API: Not responding" -ForegroundColor Red
}

# Test chillproxy
try {
    $proxyHealth = Invoke-WebRequest -Uri http://localhost:8080/health -UseBasicParsing -TimeoutSec 5
    Write-Host "   ✅ Chillproxy: $($proxyHealth.StatusCode) OK" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Chillproxy: Not responding" -ForegroundColor Red
}

# Step 7: Show test commands
Write-Host "`n✅ Setup complete! Services running." -ForegroundColor Green
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

Write-Host "`n📋 Test Commands:" -ForegroundColor Blue
Write-Host "   # Create test config" -ForegroundColor White
Write-Host "   `$config = @{stores=@(@{c='tb';t='';auth='test-user-12345'})} | ConvertTo-Json -Compress"
Write-Host "   `$base64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(`$config))"
Write-Host ""
Write-Host "   # Test manifest" -ForegroundColor White
Write-Host "   Invoke-WebRequest `"http://localhost:8080/stremio/torz/`$base64/manifest.json`""
Write-Host ""
Write-Host "   # Test stream (The Matrix)" -ForegroundColor White
Write-Host "   Invoke-WebRequest `"http://localhost:8080/stremio/torz/`$base64/stream/movie/tt0133093.json`""
Write-Host ""
Write-Host "   # View stats" -ForegroundColor White
Write-Host "   Invoke-WebRequest http://localhost:3000/api/v1/internal/pool/stats | ConvertFrom-Json"
Write-Host ""

Write-Host "📖 Full guide: TESTING_MOCK_SERVER.md" -ForegroundColor Cyan
Write-Host ""

