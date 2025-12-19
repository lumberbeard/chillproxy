# ⚠️ CHILLPROXY BUILD INSTRUCTIONS - READ FIRST

## 🚨 CRITICAL: ALWAYS BUILD WITH DOCKER

**DO NOT attempt to build Chillproxy with `go build` on Windows or macOS!**

Chillproxy has **CGO dependencies** (C libraries) that require a Linux build environment.

---

## ✅ CORRECT WAY TO BUILD

### Using Docker (RECOMMENDED)

```powershell
# Navigate to chillproxy directory
cd C:\chillproxy

# Build Docker image
docker build -t chillproxy:latest .

# Start with docker-compose
docker-compose up -d

# OR start standalone container
docker run -d --name chillproxy -p 8080:8080 chillproxy:latest
```

### Using Docker Compose (BEST)

```powershell
cd C:\chillproxy
docker-compose up -d --build
```

---

## ❌ WHAT NOT TO DO

**These commands will FAIL**:

```powershell
# ❌ WRONG - Will fail with CGO errors
go build -o chillproxy.exe .

# ❌ WRONG - Still won't work
$env:CGO_ENABLED="1"
go build -o chillproxy.exe .

# ❌ WRONG - Don't install MinGW
choco install mingw
```

**Why they fail**:
- Chillproxy uses `github.com/jamespfennell/xz` package
- This package requires LZMA C library
- Windows Go toolchain cannot compile C code without complex setup
- Even with MinGW, cross-compilation issues arise

---

## 🐳 Docker Setup

### Prerequisites

1. **Docker Desktop** installed and running
2. **WSL 2** enabled (for Windows)

### Dockerfile

Chillproxy's Dockerfile uses multi-stage build:

```dockerfile
# Stage 1: Build (uses golang:alpine with CGO)
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY . .
RUN go build -o chillproxy .

# Stage 2: Runtime (minimal alpine image)
FROM alpine:latest
COPY --from=builder /app/chillproxy /chillproxy
ENTRYPOINT ["/chillproxy"]
```

This ensures:
- ✅ CGO dependencies compile correctly
- ✅ Minimal final image size
- ✅ Consistent builds across platforms

---

## 🔄 Development Workflow

### 1. Make Code Changes

Edit Go files in `C:\chillproxy\`

### 2. Rebuild Docker Image

```powershell
docker build -t chillproxy:latest .
```

### 3. Restart Container

```powershell
# If using docker-compose:
docker-compose down
docker-compose up -d

# If using standalone:
docker stop chillproxy
docker rm chillproxy
docker run -d --name chillproxy -p 8080:8080 chillproxy:latest
```

### 4. View Logs

```powershell
docker logs chillproxy -f
```

### 5. Test Changes

```powershell
# Health check
Invoke-WebRequest -Uri http://localhost:8080/health

# Test stream endpoint
Invoke-WebRequest -Uri http://localhost:8080/stremio/torz/{config}/manifest.json
```

---

## 🛠️ Troubleshooting

### "Cannot connect to Docker daemon"

**Problem**: Docker Desktop not running

**Solution**:
```powershell
# Start Docker Desktop
Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"

# Wait for it to start (30 seconds)
Start-Sleep -Seconds 30

# Verify
docker ps
```

### "Build failed: gcc not found"

**Problem**: Trying to use `go build` instead of Docker

**Solution**: **Use Docker!** See correct commands above.

### "Port 8080 already in use"

**Problem**: Old Chillproxy container still running

**Solution**:
```powershell
# Stop and remove old container
docker stop chillproxy
docker rm chillproxy

# Or use docker-compose
docker-compose down
```

### "Cannot find chillproxy image"

**Problem**: Image not built yet

**Solution**:
```powershell
cd C:\chillproxy
docker build -t chillproxy:latest .
```

---

## 📝 Environment Variables

Create `.env` file in `C:\chillproxy\`:

```bash
# Chillstreams Integration
CHILLSTREAMS_API_URL=http://host.docker.internal:3000
CHILLSTREAMS_API_KEY=your_internal_api_key_here

# Server
PORT=8080
HOST=0.0.0.0

# Database (optional)
DATABASE_URL=postgres://user:pass@host.docker.internal:5432/chillproxy

# Redis (optional)
REDIS_URI=redis://host.docker.internal:6379
```

**Note**: Use `host.docker.internal` to access services on host machine from container.

---

## 🎯 Quick Reference

| Task | Command |
|------|---------|
| **Build image** | `docker build -t chillproxy:latest .` |
| **Start (compose)** | `docker-compose up -d` |
| **Start (standalone)** | `docker run -d --name chillproxy -p 8080:8080 chillproxy:latest` |
| **Stop** | `docker-compose down` or `docker stop chillproxy` |
| **Logs** | `docker logs chillproxy -f` |
| **Rebuild & restart** | `docker-compose up -d --build` |
| **Clean up** | `docker system prune -a` |

---

## 🚀 Production Deployment

For production, use:

```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  chillproxy:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CHILLSTREAMS_API_URL=https://api.chillstreams.com
      - CHILLSTREAMS_API_KEY=${CHILLSTREAMS_API_KEY}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Deploy:
```powershell
docker-compose -f docker-compose.prod.yml up -d
```

---

## 📚 Additional Resources

- **Docker Documentation**: https://docs.docker.com/
- **Docker Compose**: https://docs.docker.com/compose/
- **CGO and Go**: https://go.dev/blog/cgo

---

**Last Updated**: December 19, 2025  
**Reminder**: ⚠️ **ALWAYS BUILD WITH DOCKER!** ⚠️

