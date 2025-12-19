# Complete Prowlarr Installation Script
# Run this entire script in PowerShell

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Prowlarr Fresh Installation Script   " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Download
Write-Host "[1/6] Downloading Prowlarr..." -ForegroundColor Yellow
New-Item -ItemType Directory -Path "C:\Temp\Prowlarr" -Force | Out-Null
$url = "https://github.com/Prowlarr/Prowlarr/releases/download/v1.12.2.4211/Prowlarr.develop.1.12.2.4211.windows-core-x64.zip"
$output = "C:\Temp\Prowlarr\prowlarr.zip"

try {
    Invoke-WebRequest -Uri $url -OutFile $output -UseBasicParsing
    Write-Host "✅ Downloaded: $((Get-Item $output).Length / 1MB) MB" -ForegroundColor Green
} catch {
    Write-Host "❌ Download failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 2: Extract
Write-Host ""
Write-Host "[2/6] Extracting Prowlarr..." -ForegroundColor Yellow
try {
    Expand-Archive -Path $output -DestinationPath "C:\Prowlarr" -Force
    Write-Host "✅ Extracted to C:\Prowlarr" -ForegroundColor Green
    Write-Host "   Files: $(( Get-ChildItem 'C:\Prowlarr' | Measure-Object).Count)" -ForegroundColor Gray
} catch {
    Write-Host "❌ Extraction failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 3: Create Config
Write-Host ""
Write-Host "[3/6] Creating configuration..." -ForegroundColor Yellow
New-Item -ItemType Directory -Path "C:\ProgramData\Prowlarr" -Force | Out-Null

$configXml = @"
<Config>
  <BindAddress>*</BindAddress>
  <Port>9696</Port>
  <SslPort>9897</SslPort>
  <EnableSsl>False</EnableSsl>
  <LaunchBrowser>False</LaunchBrowser>
  <ApiKey>f963a60693dd49a08ff75188f9fc72d2</ApiKey>
  <AuthenticationMethod>None</AuthenticationMethod>
  <Branch>develop</Branch>
  <LogLevel>info</LogLevel>
  <UrlBase></UrlBase>
  <InstanceName>Prowlarr</InstanceName>
</Config>
"@

try {
    $configXml | Out-File -FilePath "C:\ProgramData\Prowlarr\config.xml" -Encoding utf8 -Force
    Write-Host "✅ Configuration created" -ForegroundColor Green
    Write-Host "   API Key: f963a60693dd49a08ff75188f9fc72d2" -ForegroundColor Gray
    Write-Host "   Port: 9696" -ForegroundColor Gray
    Write-Host "   Auth: None" -ForegroundColor Gray
} catch {
    Write-Host "❌ Config creation failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 4: Start Prowlarr
Write-Host ""
Write-Host "[4/6] Starting Prowlarr..." -ForegroundColor Yellow
try {
    Start-Process -FilePath "C:\Prowlarr\Prowlarr.exe" -WindowStyle Minimized
    Write-Host "✅ Process started" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to start: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 5: Wait and Test
Write-Host ""
Write-Host "[5/6] Waiting for Prowlarr to initialize (30 seconds)..." -ForegroundColor Yellow
for ($i = 30; $i -gt 0; $i--) {
    Write-Host "   $i seconds remaining..." -ForegroundColor Gray
    Start-Sleep -Seconds 1
}

Write-Host ""
Write-Host "[6/6] Testing API..." -ForegroundColor Yellow
$apiKey = 'f963a60693dd49a08ff75188f9fc72d2'
$headers = @{'X-Api-Key' = $apiKey}

# Test 1: Basic connectivity
Write-Host ""
Write-Host "Test 1: Basic HTTP connection" -ForegroundColor Cyan
try {
    $r = Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 5
    Write-Host "✅ Prowlarr UI is accessible (Status: $($r.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "❌ Cannot reach Prowlarr: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "   Check if process is running in Task Manager" -ForegroundColor Yellow
}

# Test 2: API endpoint
Write-Host ""
Write-Host "Test 2: API endpoint (/api/v1/indexer)" -ForegroundColor Cyan
try {
    $indexers = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/indexer" -Headers $headers -TimeoutSec 10
    Write-Host "✅ API is working!" -ForegroundColor Green
    Write-Host "   Indexers configured: $($indexers.Count)" -ForegroundColor Gray

    if ($indexers.Count -eq 0) {
        Write-Host "   (No indexers yet - this is normal for fresh install)" -ForegroundColor Yellow
    } else {
        $indexers | ForEach-Object {
            Write-Host "   - $($_.name) ($($_.protocol))" -ForegroundColor Gray
        }
    }
} catch {
    Write-Host "❌ API test failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "   This might mean Prowlarr is still initializing" -ForegroundColor Yellow
}

# Test 3: System status
Write-Host ""
Write-Host "Test 3: System status" -ForegroundColor Cyan
try {
    $status = Invoke-RestMethod -Uri "http://localhost:9696/api/v1/system/status" -Headers $headers -TimeoutSec 5
    Write-Host "✅ System status retrieved" -ForegroundColor Green
    Write-Host "   Version: $($status.version)" -ForegroundColor Gray
    Write-Host "   Branch: $($status.branch)" -ForegroundColor Gray
} catch {
    Write-Host "⚠️ Could not get system status: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Installation Summary                  " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "✅ Prowlarr installed to: C:\Prowlarr" -ForegroundColor Green
Write-Host "✅ Configuration: C:\ProgramData\Prowlarr\config.xml" -ForegroundColor Green
Write-Host "✅ API Key: f963a60693dd49a08ff75188f9fc72d2" -ForegroundColor Green
Write-Host "✅ Port: 9696" -ForegroundColor Green
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Open http://localhost:9696 in browser" -ForegroundColor White
Write-Host "2. Add indexers via Settings → Indexers" -ForegroundColor White
Write-Host "3. Test search: http://localhost:9696/api/v1/search?query=matrix&type=search" -ForegroundColor White
Write-Host ""
Write-Host "For Chillproxy Integration:" -ForegroundColor Yellow
Write-Host "  Endpoint: http://localhost:9696/api/v1/search?query={query}&type=search" -ForegroundColor White
Write-Host "  Header: X-Api-Key: f963a60693dd49a08ff75188f9fc72d2" -ForegroundColor White
Write-Host "  Response: JSON (not XML)" -ForegroundColor White
Write-Host ""

