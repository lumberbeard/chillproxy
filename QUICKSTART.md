# Chillproxy - Quick Start Guide (Windows)

## Issue: Build Requires CGO

The chillproxy project requires CGO (C compiler) for the `xz` compression library, which is used for dataset downloads. On Windows, this requires additional setup.

## Solution Options

### Option 1: Use Docker (Recommended - Easiest)

**Prerequisites**: Docker Desktop must be running

#### 1. Start Docker Desktop

```pwsh
# Open Docker Desktop from Start Menu and wait for it to start
# You should see the Docker icon in system tray
```

#### 2. Build Docker Image

```pwsh
cd C:\chillproxy
docker build -t chillproxy:test .
```

#### 3. Run Container

```pwsh
# Basic run (no persistence)
docker run --rm -p 8080:8080 `
  -e STREMTHRU_DATABASE_URI=sqlite:///app/data/stremthru.db `
  chillproxy:test

# Or with volume for data persistence
docker run --rm -p 8080:8080 `
  -v ${PWD}/data:/app/data `
  -e STREMTHRU_DATABASE_URI=sqlite:///app/data/stremthru.db `
  -e STREMTHRU_LOG_LEVEL=INFO `
  chillproxy:test
```

#### 4. Test

```pwsh
# In another PowerShell window:
Invoke-WebRequest -Uri "http://localhost:8080/health"
```

---

### Option 2: Install GCC Compiler (More Complex)

#### 1. Install MSYS2 and GCC

```pwsh
# MSYS2 is already installed, now install GCC
C:\msys64\msys2.exe

# In the MSYS2 terminal that opens:
pacman -S mingw-w64-ucrt-x86_64-gcc
exit
```

#### 2. Add to PATH

```pwsh
$env:Path += ";C:\msys64\ucrt64\bin"

# Verify GCC is available
gcc --version
```

#### 3. Build with CGO

```pwsh
cd C:\chillproxy
$env:CGO_ENABLED="1"
go build --tags "fts5" -o chillproxy.exe .
```

#### 4. Run

```pwsh
.\chillproxy.exe
```

---

### Option 3: Use Pre-built Binary (If Available)

Check the GitHub releases page for pre-built Windows binaries:
```pwsh
# Download from: https://github.com/MunifTanjim/stremthru/releases
# Extract and run directly
```

---

### Option 4: Run on WSL2 (Alternative)

If you have WSL2 installed:

```pwsh
# Enter WSL
wsl

# Inside WSL:
cd /mnt/c/chillproxy
go build --tags "fts5" -o stremthru .
./stremthru
```

---

## Docker Compose (Recommended for Production-like Testing)

### 1. Check Docker Compose File

```pwsh
cd C:\chillproxy
Get-Content compose.example.yaml
```

### 2. Copy and Modify

```pwsh
Copy-Item compose.example.yaml compose.yaml
# Edit compose.yaml with your settings
```

### 3. Run with Docker Compose

```pwsh
docker compose up
```

---

## Testing Plan

Once running (via Docker or compiled binary):

### 1. Basic Health Check

```pwsh
Invoke-WebRequest -Uri "http://localhost:8080/health"
```

Expected: HTTP 200

### 2. Store Manifest

```pwsh
Invoke-WebRequest -Uri "http://localhost:8080/stremio/store/manifest.json"
```

Expected: JSON manifest

### 3. Test with TorBox Key (Optional)

```pwsh
# Set environment variable
$env:STREMTHRU_STORE_AUTH="testuser:torbox:<YOUR_TORBOX_KEY>"

# Restart server
# Then test authenticated endpoint
$auth = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes("testuser:password"))
Invoke-WebRequest -Uri "http://localhost:8080/v0/store/user" `
  -Headers @{"X-StremThru-Authorization" = "Basic $auth"}
```

---

## Current Status

- ✅ Go installed (v1.25.5)
- ✅ Dependencies downloaded
- ❌ Build fails due to missing C compiler
- ✅ Docker available (recommended path)
- ✅ `.env` file created

## Next Steps

**Recommended**: Use Docker option (easiest and most reliable)

1. Start Docker Desktop
2. Build image: `docker build -t chillproxy:test .`
3. Run container: `docker run --rm -p 8080:8080 chillproxy:test`
4. Test endpoints
5. Once verified working, proceed with Chillstreams integration

**Alternative**: If you prefer native Windows binary, complete the GCC setup in Option 2 above.

---

## Troubleshooting Docker

### Docker Desktop Not Running

```pwsh
# Start Docker Desktop from Start Menu
# Wait for "Docker Desktop is running" message in system tray
```

### Port 8080 Already in Use

```pwsh
# Find what's using port 8080
Get-NetTCPConnection -LocalPort 8080

# Kill the process
Stop-Process -Id <PID> -Force

# Or use different port
docker run --rm -p 8081:8080 chillproxy:test
```

### Build Fails - No Space

```pwsh
# Clean up Docker images
docker system prune -a
```

---

**Status**: Ready for Docker build  
**Recommendation**: Use Docker approach for quickest testing  
**Last Updated**: December 16, 2025

