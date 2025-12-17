# GitHub Copilot Instructions for Chillproxy

## Project Overview

**Chillproxy** is a forked and customized version of [StremThru](https://github.com/MunifTanjim/stremthru) that serves as a **debrid service proxy** for the Chillstreams ecosystem. It intercepts torrent/usenet requests from Stremio and handles debrid service authentication using a **shared pool key system** managed by Chillstreams.

**Key Difference from StremThru**: Instead of users providing their own debrid keys in manifest URLs, Chillproxy authenticates users via **Chillstreams user IDs** and uses a **pool of shared TorBox keys** managed server-side by Chillstreams.

**Project Relationship**:
- **Chillstreams** (TypeScript): Main addon aggregator, user management, pool key assignment
- **Chillproxy** (Go): Debrid service proxy that validates users and fetches streams using pool keys

## Architecture

This is a **Go monorepo** with an embedded **React TypeScript dashboard**.

```
chillproxy/
├── main.go                    # Entry point, HTTP server setup
├── schema.go                  # GraphQL schema definitions
├── core/                      # Core utilities (base64, errors, IP detection)
├── internal/                  # Internal packages (NOT importable outside)
│   ├── server/                # HTTP server, routing, middleware
│   ├── endpoint/              # HTTP handlers (/health, /stremio, /api)
│   ├── stremio/               # Stremio addon implementations (list, store, torz, wrap)
│   ├── buddy/                 # Stream caching/proxy orchestration layer
│   ├── cache/                 # Redis/LRU caching
│   ├── db/                    # PostgreSQL/SQLite database
│   ├── sync/                  # Background jobs (metadata sync)
│   └── worker/                # Task queue workers
├── store/                     # Debrid service clients (exported)
│   ├── torbox/                # TorBox API client
│   ├── realdebrid/            # RealDebrid API client
│   ├── alldebrid/             # AllDebrid API client
│   └── ...                    # Other debrid services
├── stremio/                   # Stremio protocol types (manifest, streams)
├── apps/dash/                 # React dashboard (Vite + TypeScript)
└── migrations/                # Database migrations (postgres, sqlite)
```

## Tech Stack

- **Runtime**: Go 1.21+
- **Database**: PostgreSQL (primary), SQLite (dev/testing)
- **Caching**: Redis (optional), LRU in-memory
- **Frontend**: React 18, TypeScript, Vite, TailwindCSS
- **GraphQL**: gqlgen for Go GraphQL server
- **Build**: Go modules, pnpm for frontend

## Key Concepts

### 1. Debrid Service Proxy
Chillproxy acts as a **middleware** between Stremio and debrid services:
```
User → Stremio → Addon (Torrentio/Comet) → Chillproxy → TorBox API
```

**Flow**:
1. Addon returns stream with chillproxy URL
2. User clicks stream in Stremio
3. Chillproxy intercepts, validates user via Chillstreams API
4. Chillproxy fetches stream from debrid service using pool key
5. Returns cached/direct stream URL to Stremio

### 2. Store System
**Stores** are debrid service implementations in `store/`:
- **TorBox** (`store/torbox/`)
- **RealDebrid** (`store/realdebrid/`)
- **AllDebrid** (`store/alldebrid/`)
- **Premiumize** (`store/premiumize/`)
- **Debrid-Link** (`store/debridlink/`)
- **OffCloud** (`store/offcloud/`)
- **PikPak** (`store/pikpak/`)
- **EasyDebrid** (`store/easydebrid/`)

Each store implements the `Store` interface with methods like:
- `CheckCached(hash string)` - Check if torrent is cached
- `AddTorrent(magnetLink string)` - Add torrent to debrid service
- `GetStream(torrentID, fileID string)` - Get direct stream URL

### 3. Buddy Layer
**Buddy** (`internal/buddy/`) is the caching/proxy orchestration layer:
- Checks if torrent is cached in debrid service
- Adds uncached torrents to debrid service
- Selects best file from torrent
- Returns stream URLs with expiry/refresh logic

### 4. Stremio Addons
Chillproxy implements multiple Stremio addon types in `internal/stremio/`:
- **Store**: Direct debrid service integration
- **Torz**: Torrent indexer/search addon
- **Wrap**: Wraps external addons with debrid service
- **List**: Curated content lists
- **Sidekick**: Companion addon features

### 5. Current Authentication (Original StremThru)
**Before our changes**, StremThru expects debrid keys in the manifest URL:
```
GET /stremio/torz/{config}/manifest.json

config = base64({
  "stores": [
    {"c": "tb", "t": "user_torbox_api_key_here"}
  ]
})
```

**Problem**: User's API key is **visible** in the manifest URL.

### 6. Planned Authentication (Chillstreams Integration)
**After our changes**, Chillproxy will use Chillstreams user IDs:
```
GET /stremio/torz/{config}/manifest.json

config = base64({
  "stores": [
    {"c": "tb", "t": "", "auth": "chillstreams-user-uuid"}
  ]
})
```

**When user requests stream**:
1. Extract `auth` (user UUID) from config
2. Call Chillstreams API: `POST /api/v1/internal/pool/get-key`
3. Chillstreams returns assigned pool key for user
4. Use pool key to call TorBox API
5. Log usage back to Chillstreams

## Code Style & Conventions

### Go Style
- **Package names**: lowercase, no underscores (e.g., `package endpoint`)
- **Files**: snake_case (e.g., `request_ip.go`, `store_cache.go`)
- **Types**: PascalCase for exported, camelCase for private (e.g., `type TorrentInfo struct`)
- **Functions**: PascalCase for exported, camelCase for private
- **Constants**: PascalCase or UPPER_SNAKE (e.g., `const MaxRetries = 3`)
- **Interfaces**: PascalCase (e.g., `type Store interface`)

### Import Style
```go
// Group imports: stdlib, external, internal
import (
    "context"
    "encoding/json"
    "net/http"
    
    "github.com/MunifTanjim/stremthru/core"
    "github.com/MunifTanjim/stremthru/store"
    
    "github.com/MunifTanjim/stremthru/internal/buddy"
    "github.com/MunifTanjim/stremthru/internal/cache"
)
```

### Naming Conventions
- **Exported functions**: Start with uppercase (e.g., `CheckCached`, `GetUser`)
- **Private functions**: Start with lowercase (e.g., `parseConfig`, `handleError`)
- **Receivers**: Short (1-2 letters), consistent (e.g., `c *Client`, `s *Store`)
- **Error variables**: Prefix with `Err` (e.g., `ErrNotFound`, `ErrUnauthorized`)

### Code Organization
- Use `internal/` for packages not meant to be imported externally
- Keep store implementations in top-level `store/` package (exported)
- Group related functionality in subdirectories
- Keep HTTP handlers in `internal/endpoint/`
- Keep business logic in `internal/buddy/`, `internal/stremio/`

## Development Patterns

### Error Handling
```go
// Return errors, don't panic
func doSomething() error {
    if err := validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}

// Use custom error types for specific cases
var ErrNotCached = errors.New("torrent not cached")

if errors.Is(err, store.ErrNotCached) {
    // Handle uncached torrent
}
```

### Logging
```go
// Use structured logging
import "github.com/MunifTanjim/stremthru/internal/logger"

log := logger.Get("module-name")
log.Info().Str("hash", hash).Msg("checking cache")
log.Error().Err(err).Msg("failed to fetch")
log.Warn().Int("count", count).Msg("high usage detected")
```

### Context Usage
```go
// Always pass context for cancellation/timeouts
func FetchStream(ctx context.Context, hash string) error {
    // Use ctx for HTTP requests, DB queries
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := client.Do(req)
    // ...
}
```

### Caching
```go
// Use cache package for performance
import "github.com/MunifTanjim/stremthru/internal/cache"

c := cache.NewCache[string](&cache.CacheConfig{
    Name:     "my-cache",
    Lifetime: 10 * time.Minute,
})

// Set value
c.Add("key", "value")

// Get value
var value string
if c.Get("key", &value) {
    // Cache hit
}
```

### HTTP Client Configuration
```go
// Use appropriate HTTP client for tunneling
transport := config.DefaultHTTPTransport.Clone()
transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_NONE)

client := &http.Client{
    Transport: transport,
    Timeout:   60 * time.Second,
}
```

## Common Tasks

### Adding a New Debrid Service

1. Create package in `store/newservice/`
```go
package newservice

type Store struct {
    client *APIClient
}

func NewStoreClient(config *StoreClientConfig) *Store {
    // Initialize client
}

func (s *Store) CheckCached(hash string) (bool, error) {
    // Implement cache check
}

func (s *Store) AddTorrent(magnet string) (string, error) {
    // Implement add torrent
}
```

2. Register in `store/store.go`
```go
case "ns": // New service code
    return newservice.NewStoreClient(config), nil
```

3. Add to supported stores in `main.go`

### Adding a New Endpoint

1. Create handler in `internal/endpoint/`
```go
package endpoint

func HandleNewRoute(w http.ResponseWriter, r *http.Request) {
    // Implement logic
    json.NewEncoder(w).Encode(response)
}
```

2. Register route in appropriate file (e.g., `internal/endpoint/stremio.go`)
```go
mux.HandleFunc("/stremio/new-route", HandleNewRoute)
```

### Database Migrations

```pwsh
# Create new migration
cd migrations/postgres
New-Item 006_add_chillstreams_auth.sql

# Write migration
# migrations/postgres/006_add_chillstreams_auth.sql
```

```sql
-- +goose Up
CREATE TABLE chillstreams_auth (
    user_id VARCHAR(36) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL,
    last_used TIMESTAMP
);

-- +goose Down
DROP TABLE chillstreams_auth;
```

## Building & Running

### Local Development

```pwsh
# Install Go dependencies
go mod download

# Build Go binary
go build -o chillproxy.exe .

# Run server
$env:DATABASE_URL="postgres://..."
$env:REDIS_URI="redis://..."
.\chillproxy.exe

# Build dashboard (optional)
cd apps/dash
pnpm install
pnpm build

# Run dashboard dev server
pnpm dev
```

### Docker

```pwsh
# Build image
docker build -t chillproxy:latest .

# Run container
docker run -p 8080:8080 `
  -e DATABASE_URL="postgres://..." `
  -e REDIS_URI="redis://..." `
  chillproxy:latest
```

### Testing

```pwsh
# Run all tests
go test ./...

# Test specific package
go test ./store/torbox

# Run with coverage
go test -cover ./...

# Verbose output
go test -v ./internal/buddy
```

###PowerShell Commands
Important: When recommending terminal commands during development, do not use curl or bash. This is a powershell terminal.

Use ; to chain commands instead of &&
Use backquotes ` for line continuation
Avoid bash-specific syntax and curl commands
Avoid head, tail, grep, and other Unix tools - use PowerShell equivalents
Examples:
# Correct: Use semicolon for chaining
pnpm build; pnpm start

# Correct: Use Test-Path for file checks
if (Test-Path 'path/to/file') { ... }

# Incorrect: Don't use curl
curl http://localhost:3000  # ❌

# Correct: Use Invoke-WebRequest for PowerShell
Invoke-WebRequest -Uri http://localhost:3000  # ✅

# Incorrect: Don't use head/tail/grep/pipes
pnpm build 2>&1 | head -50  # ❌

# Correct: Let full output show or use Select-Object
pnpm build  # ✅
pnpm build | Select-Object -First 50  # ✅


## Environment Variables

Key environment variables:

```bash
# Server
PORT=8080                          # HTTP server port
HOST=0.0.0.0                       # Bind address
STREMTHRU_BASE_URL=http://...      # Public base URL

# Database
DATABASE_URL=postgres://...         # PostgreSQL connection
STREMTHRU_DATA_DIR=./data          # Data directory for SQLite

# Cache
REDIS_URI=redis://localhost:6379   # Redis (optional)

# Integration (NEW - for Chillstreams)
CHILLSTREAMS_API_URL=http://localhost:3000  # Chillstreams API
CHILLSTREAMS_API_KEY=internal_secret        # Internal API auth

# Debrid Services (for testing)
TORBOX_API_KEY=...                 # Test TorBox key
REALDEBRID_API_KEY=...             # Test RealDebrid key

# Features
STREMTHRU_FEATURE_TORZ=true        # Enable Torz addon
STREMTHRU_FEATURE_STORE=true       # Enable Store addon
STREMTHRU_FEATURE_WRAP=true        # Enable Wrap addon
```

## Important Files

- **`main.go`**: Server entry point, config loading, route registration
- **`internal/server/router.go`**: HTTP routing setup (if exists)
- **`internal/endpoint/stremio.go`**: Stremio addon endpoint registration
- **`internal/buddy/buddy.go`**: Core stream caching logic
- **`store/store.go`**: Debrid service factory
- **`store/torbox/store.go`**: TorBox client implementation
- **`internal/db/db.go`**: Database connection management
- **`schema.go`**: GraphQL schema for dashboard API

## Modification Plan for Chillstreams Integration

See `docs/INTEGRATION_PLAN.md` for detailed transformation steps.

**High-level changes**:
1. Modify config schema to support `auth` field (user UUID)
2. Add Chillstreams API client (`internal/chillstreams/`)
3. Update store initialization to fetch pool keys dynamically
4. Add device tracking and usage logging
5. Implement revocation checks before serving streams

## Common Debugging Tasks

### Test TorBox Integration
```pwsh
# Check if server is running
Invoke-WebRequest -Uri "http://localhost:8080/health"

# Test torrent cache check (manual)
# Requires valid config with TorBox key
$config = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes('{"stores":[{"c":"tb","t":"YOUR_KEY"}]}'))
Invoke-WebRequest -Uri "http://localhost:8080/stremio/torz/$config/manifest.json"
```

### View Logs
```pwsh
# Run with debug logging
$env:STREMTHRU_LOG_LEVEL="DEBUG"
.\chillproxy.exe > app.log 2>&1

# Filter for errors
Get-Content app.log | Select-String "ERROR"
```

### Database Inspection
```pwsh
# Connect to PostgreSQL
psql $env:DATABASE_URL

# View tables
\dt

# Query data
SELECT * FROM torrents WHERE hash = 'XXX';
```

## PowerShell Commands
**Important**: When recommending terminal commands during development, do not use curl or bash. This is a powershell terminal.
- Use `;` to chain commands instead of `&&`
- Use backquotes `` ` `` for line continuation
- Avoid bash-specific syntax and curl commands
- Avoid head, tail, grep, and other Unix tools - use PowerShell equivalents
- Examples:
  ```pwsh
  # Correct: Use semicolon for chaining
  pnpm build; pnpm start

  # Correct: Use Test-Path for file checks
  if (Test-Path 'path/to/file') { ... }

  # Incorrect: Don't use curl
  curl http://localhost:3000  # ❌

  # Correct: Use Invoke-WebRequest for PowerShell
  Invoke-WebRequest -Uri http://localhost:3000  # ✅

  # Incorrect: Don't use head/tail/grep/pipes
  pnpm build 2>&1 | head -50  # ❌

  # Correct: Let full output show or use Select-Object
  go build  # ✅
  go build | Select-Object -First 50  # ✅
  ```

## Related Documentation

- **`docs/INTEGRATION_PLAN.md`**: Detailed transformation plan for Chillstreams integration
- **`README.md`**: Original StremThru documentation (upstream)
- **Upstream**: [StremThru GitHub](https://github.com/MunifTanjim/stremthru)
- **Chillstreams**: [Integration documentation](../chillstreams/SOLUTION_ANALYSIS_FINAL.md)

## Contributing

When modifying chillproxy:
1. ✅ Keep changes focused on Chillstreams integration
2. ✅ Maintain compatibility with existing stores (RealDebrid, etc.)
3. ✅ Add tests for new functionality
4. ✅ Document API changes in `docs/`
5. ❌ Don't break existing StremThru functionality (when possible)
6. ✅ Follow Go best practices and idioms
7. ✅ Use structured logging for debugging

## Notes

- This is a **fork**, not upstream contribution - we own the code
- Focus on **TorBox integration** first, other stores later
- Prioritize **security** (pool keys must never leak to users)
- Design for **scale** (handle 100+ concurrent users per pool key)
- Test thoroughly before production deployment
- Consider backward compatibility with legacy token auth

## Testing Strategy

### Unit Tests
- Test Chillstreams API client (mock responses)
- Test device ID generation (consistent hashing)
- Test store initialization with auth

### Integration Tests
- End-to-end flow: user ID → pool key → stream
- Device limit enforcement (max 3 devices)
- Pool key caching and expiration
- Usage logging accuracy

### Load Tests
- 100+ concurrent users requesting streams
- Pool key distribution across users
- Response times under load
- Error handling when Chillstreams is down

