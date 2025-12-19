# Simple Prowlarr + Chillproxy Testing Script

$apiKey = "f963a60693dd49a08ff75188f9fc72d2"
$config = "eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0="

Write-Host "╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║          Prowlarr + Chillproxy Testing Script                ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Test 1: Prowlarr Health
Write-Host "[1] Testing Prowlarr..." -ForegroundColor Yellow
try {
    $r = Invoke-WebRequest -Uri "http://localhost:9696" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Host "    ✅ Prowlarr is running" -ForegroundColor Green
} catch {
    Write-Host "    ❌ Prowlarr is not responding" -ForegroundColor Red
    exit
}

# Test 2: Prowlarr Search
Write-Host "[2] Testing Prowlarr search..." -ForegroundColor Yellow
try {
    $url = "http://localhost:9696/api/v2.0/indexers/all/results/torznab?t=tvsearch&q=breaking+bad&season=1&ep=1&apikey=$apiKey"
    $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 10 -ErrorAction Stop

    if ($r.StatusCode -eq 200 -and $r.Content.Contains("<rss")) {
        Write-Host "    ✅ Prowlarr search is working" -ForegroundColor Green
    } else {
        Write-Host "    ⚠️  Prowlarr returned status $($r.StatusCode)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "    ❌ Prowlarr search failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 3: Chillstreams Health
Write-Host "[3] Testing Chillstreams..." -ForegroundColor Yellow
try {
    $r = Invoke-WebRequest -Uri "http://localhost:3000/api/v1/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Host "    ✅ Chillstreams is running" -ForegroundColor Green
} catch {
    Write-Host "    ❌ Chillstreams is not responding" -ForegroundColor Red
}

# Test 4: Chillproxy Health
Write-Host "[4] Testing Chillproxy..." -ForegroundColor Yellow
try {
    $r = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Host "    ✅ Chillproxy is running" -ForegroundColor Green
} catch {
    Write-Host "    ❌ Chillproxy is not responding" -ForegroundColor Red
    exit
}

# Test 5: Manifest
Write-Host "[5] Testing Chillproxy manifest..." -ForegroundColor Yellow
try {
    $r = Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$config/manifest.json" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop

    if ($r.StatusCode -eq 200) {
        $manifest = $r.Content | ConvertFrom-Json
        Write-Host "    ✅ Manifest loaded: $($manifest.name)" -ForegroundColor Green
    } else {
        Write-Host "    ❌ Manifest returned status $($r.StatusCode)" -ForegroundColor Red
    }
} catch {
    Write-Host "    ❌ Manifest failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 6: Stream Search
Write-Host "[6] Testing stream search (Breaking Bad S01E01)..." -ForegroundColor Yellow
try {
    $r = Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$config/stream/series/tt0903747:1:1.json" -UseBasicParsing -TimeoutSec 15 -ErrorAction Stop

    if ($r.StatusCode -eq 200) {
        $streams = $r.Content | ConvertFrom-Json

        if ($streams.streams -and $streams.streams.Count -gt 0) {
            Write-Host "    ✅ Found $($streams.streams.Count) results" -ForegroundColor Green
            Write-Host "    First result: $($streams.streams[0].title)" -ForegroundColor Cyan
        } else {
            Write-Host "    ⚠️  No results found (check Prowlarr indexers)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "    ❌ Stream search returned status $($r.StatusCode)" -ForegroundColor Red
    }
} catch {
    Write-Host "    ❌ Stream search failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 7: Pool Key
Write-Host "[7] Testing pool key assignment..." -ForegroundColor Yellow
try {
    $headers = @{
        'Authorization' = 'Bearer test_internal_key_phase3_2025'
        'Content-Type' = 'application/json'
    }

    $body = @{
        userId = "3b94cb45-3f99-406e-9c40-ecce61a405cc"
        deviceId = "test-device-123"
        action = "test"
        hash = "test-hash"
    } | ConvertTo-Json

    $r = Invoke-WebRequest -Uri "http://localhost:3000/api/v1/internal/pool/get-key" -Method POST -Headers $headers -Body $body -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop

    if ($r.StatusCode -eq 200) {
        $result = $r.Content | ConvertFrom-Json

        if ($result.allowed) {
            Write-Host "    ✅ Pool key assigned" -ForegroundColor Green
            Write-Host "    Pool Key ID: $($result.poolKeyId)" -ForegroundColor Cyan
        } else {
            Write-Host "    ❌ User not allowed: $($result.message)" -ForegroundColor Red
        }
    } else {
        Write-Host "    ⚠️  Pool endpoint returned status $($r.StatusCode)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "    ⚠️  Pool test failed: $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║                   Testing Complete                            ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

Write-Host "Your Manifest URL:" -ForegroundColor Green
Write-Host "http://localhost:8080/stremio/torz/$config/manifest.json" -ForegroundColor Yellow
Write-Host ""

Write-Host "Next: Add this URL to Stremio as an addon" -ForegroundColor Cyan

