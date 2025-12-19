# ⚠️ REMINDER: CHILLPROXY BUILD REQUIREMENTS

## 🚨 DO NOT FORGET

**CHILLPROXY BUILDS WITH DOCKER ONLY!**

---

## Why Docker is Required

1. **CGO Dependencies**: Chillproxy uses `github.com/jamespfennell/xz` which requires C libraries
2. **LZMA Compression**: The `lzma` package needs compiled C code
3. **Cross-Platform**: Docker ensures consistent builds on Windows, macOS, and Linux

---

## What NOT to Do

❌ `go build -o chillproxy.exe .`  
❌ `$env:CGO_ENABLED="1"; go build`  
❌ `choco install mingw`  
❌ Installing GCC or C compilers on Windows

**All of these will FAIL!**

---

## What TO Do

✅ `docker build -t chillproxy:latest .`  
✅ `docker-compose up -d --build`  
✅ `docker run -p 8080:8080 chillproxy:latest`

**Use Docker ALWAYS!**

---

## Quick Commands

```powershell
# Build
cd C:\chillproxy
docker build -t chillproxy:latest .

# Run
docker-compose up -d

# Logs
docker logs chillproxy -f

# Stop
docker-compose down
```

---

**See `BUILD_INSTRUCTIONS.md` for full details.**

**Last Updated**: December 19, 2025

