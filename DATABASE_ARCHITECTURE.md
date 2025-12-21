# Chillproxy Database Architecture

## Overview

Chillproxy uses **PostgreSQL** as its primary database to manage:
- User authentication and subscriptions
- TorBox API key pooling and slot management
- Torrent metadata and streaming information
- Prowlarr indexer performance and search logs
- User activity and stream tracking
- Caching and distributed locks

The database consists of **17 main tables** plus **2 views**, organized into three logical domains:

1. **User & Authentication** - User accounts and subscriptions
2. **TorBox Pool Management** - API key pooling with slot-based user assignment
3. **Prowlarr Indexing** - Search performance tracking
4. **Logging & Monitoring** - Stream usage and system health

---

## Table Schema

### User & Authentication Domain

#### `users`
Stores user account information.

| Column | Type | Description |
|--------|------|-------------|
| `id` | bigint | Auto-increment ID |
| `uuid` | UUID | Unique user identifier |
| `email` | text | User email address |
| `password_hash` | text | Bcrypt hashed password |
| `config` | text | Encrypted user configuration |
| `config_salt` | text | Salt for config encryption |
| `created_at` | timestamp | Account creation timestamp |
| `updated_at` | timestamp | Last update timestamp |
| `accessed_at` | timestamp | Last access timestamp |

**Primary Key:** `uuid`

---

#### `user_subs`
Manages user subscriptions and trial periods.

| Column | Type | Description |
|--------|------|-------------|
| `id` | integer | Primary key |
| `uuid` | text | Reference to user UUID |
| `sub_type` | text | Subscription type (free/pro/premium) |
| `sub_status` | text | Status (active/cancelled/suspended) |
| `trial` | boolean | Whether account is in trial period |
| `trial_startdate` | timestamp | Trial start date |
| `trial_enddate` | timestamp | Trial end date |
| `stripe_customer_id` | text | Stripe customer ID for payments |
| `stripe_subscription_id` | text | Stripe subscription ID |
| `created_at` | timestamp | Record creation time |
| `updated_at` | timestamp | Last update time |

**Foreign Key:** `uuid` → `users.uuid`

---

### TorBox Pool Management Domain

#### `torbox_pool`
Represents a single TorBox API key and its capacity/health metrics.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `api_key` | text | Encrypted real TorBox API key |
| `is_active` | boolean | Whether pool is accepting new assignments |
| `priority` | integer | Assignment priority (higher = assigned first) |
| **User Slot Management** | | |
| `max_user_slots` | integer | Max users assignable (default: 100) |
| `current_user_slots` | integer | Currently assigned users |
| **Concurrency Tracking** | | |
| `max_concurrent_streams` | integer | Hard limit (TorBox API: 35) |
| `current_concurrent_streams` | integer | Active streams right now |
| **Health & Status** | | |
| `status` | varchar | 'healthy' / 'degraded' / 'failing' / 'disabled' |
| `status_reason` | text | Reason for non-healthy status |
| `last_success_at` | timestamp | Last successful API call |
| `last_failure_at` | timestamp | Last failed API call |
| `consecutive_failures` | integer | Count of consecutive failures |
| `total_failures_24h` | integer | Failures in last 24 hours |
| `total_requests_24h` | integer | Requests in last 24 hours |
| `failure_rate_24h` | real | Failure percentage (0.0-1.0) |
| `avg_response_time` | integer | Average API response time (ms) |
| **Timestamps** | | |
| `created_at` | timestamp | When pool was added |
| `updated_at` | timestamp | Last update time |

**Primary Key:** `id`

**Constraints:**
- `status` must be one of: 'healthy', 'degraded', 'failing', 'disabled'

**Important Distinction:**
- **max_user_slots / current_user_slots** = How many USERS can be assigned (capacity)
- **max_concurrent_streams / current_concurrent_streams** = How many STREAMS active NOW (TorBox API limit)

---

#### `torbox_slots`
Manages individual user slot assignments within a pool. Each row = one slot position.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `pool_key_id` | UUID | Reference to pool |
| `slot_number` | integer | Sequential slot number (1-100) |
| `user_id` | UUID | Assigned user (NULL = unassigned) |
| `assigned_at` | timestamp | When user assigned to slot |
| `last_activity_at` | timestamp | Last time user made request |
| `is_active` | boolean | Is slot currently assigned? |
| `total_streams` | integer | Total streams from this slot |
| `created_at` | timestamp | Slot creation time |
| `updated_at` | timestamp | Last update time |

**Primary Key:** `id`
**Unique Constraint:** `(pool_key_id, slot_number)` - One slot per number per pool
**Foreign Keys:**
- `pool_key_id` → `torbox_pool.id`
- `user_id` → `users.uuid`

**Slot Reclamation Rule:**
Slots with `last_activity_at > 3 days old` are reclaimed (cleared for reassignment)

---

#### `torbox_assignments`
Maps users to their assigned slots.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | User UUID |
| `device_id` | text | Device identifier (hash) |
| `slot_id` | UUID | Reference to assigned slot |
| `assigned_pool_key_id` | UUID | Backup: direct pool reference |
| `last_used_at` | timestamp | Last API request timestamp |
| `created_at` | timestamp | Assignment creation time |

**Primary Key:** `id`
**Unique Constraint:** `(user_id, device_id)` - One assignment per user-device
**Foreign Keys:**
- `user_id` → `users.uuid`
- `slot_id` → `torbox_slots.id`
- `assigned_pool_key_id` → `torbox_pool.id`

---

#### `torbox_pool_health`
Tracks health check history for pools.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `pool_key_id` | UUID | Pool being checked |
| `timestamp` | timestamp | When check was performed |
| `check_type` | varchar | Type of check (e.g., 'health_check') |
| `endpoint` | text | API endpoint tested |
| `http_status` | integer | HTTP response code |
| `response_time` | integer | Response time (ms) |
| `was_successful` | boolean | Did check pass? |
| `error_type` | varchar | Type of error (timeout, http_error, etc.) |
| `error_message` | text | Error details |
| `consecutive_failures` | integer | Failure count at time of check |

**Foreign Key:** `pool_key_id` → `torbox_pool.id`

---


#### `torbox_concurrency_log`
Tracks individual stream sessions for precise concurrency measurement.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `pool_key_id` | UUID | Which pool is being used |
| `user_id` | UUID | Which user is streaming |
| `slot_id` | UUID | Which slot this user occupies |
| `stream_started_at` | timestamp | When stream began |
| `stream_ended_at` | timestamp | When stream ended (NULL = active) |
| `duration_seconds` | integer | Total stream duration |
| **Stream Metadata** | | |
| `stream_type` | varchar | 'movie' or 'series' |
| `imdb_id` | varchar | IMDB identifier |
| `torrent_hash` | varchar | Torrent hash being streamed |

**Foreign Keys:**
- `pool_key_id` → `torbox_pool.id`
- `user_id` → `users.uuid`
- `slot_id` → `torbox_slots.id`

**Key Query Pattern:**
```sql
-- Get current concurrency for a pool
SELECT COUNT(*) 
FROM torbox_concurrency_log 
WHERE pool_key_id = 'pool-uuid' 
  AND stream_ended_at IS NULL
```

---

#### `torbox_usage_logs`
Detailed logs of every API request/stream access.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | User making request |
| `pool_key_id` | UUID | Pool key used |
| `device_id` | text | Device identifier |
| `stream_type` | varchar | 'movie' or 'series' |
| `title` | text | Content title |
| `imdb_id` | text | IMDB ID |
| `torrent_hash` | text | Torrent hash |
| `debrid_service` | varchar | 'torbox', 'realdebrid', etc. |
| `was_cached` | boolean | Was torrent cached? |
| `endpoint` | text | API endpoint accessed |
| `response_time_ms` | integer | Response time in milliseconds |
| `status_code` | integer | HTTP status code |
| `was_successful` | boolean | Did request succeed? |
| `error_type` | varchar | Type of error |
| `timestamp` | timestamp | Request timestamp |

**Foreign Keys:**
- `user_id` → `users.uuid`
- `pool_key_id` → `torbox_pool.id`

---

### Prowlarr Integration Domain

#### `prowlarr_indexers`
Metadata about Prowlarr indexers and their performance.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `indexer_name` | text | Indexer name (YTS, EZTV, TPB, etc.) |
| `indexer_id` | integer | Prowlarr internal ID |
| `prowlarr_id` | integer | Prowlarr system ID |
| `is_enabled` | boolean | Is indexer active? |
| `protocol` | varchar | 'torrent' or 'usenet' |
| `priority` | integer | Search priority |
| **Performance Metrics (24h)** | | |
| `total_requests_24h` | integer | Total queries in 24h |
| `successful_requests_24h` | integer | Successful queries |
| `failed_requests_24h` | integer | Failed queries |
| `timeout_count_24h` | integer | Timeout count |
| `queries_24h` | integer | Search queries (from Prowlarr) |
| `grabs_24h` | integer | Grabs (from Prowlarr) |
| **Historical Stats** | | |
| `queries_total` | integer | Total queries all-time |
| `grabs_total` | integer | Total grabs all-time |
| `rss_queries` | integer | RSS search count |
| `auth_queries` | integer | Auth-required queries |
| `failed_queries` | integer | Failed queries |
| `failed_grabs` | integer | Failed grabs |
| **Response Time** | | |
| `avg_response_time` | integer | Average (ms) |
| `median_response_time` | integer | Median (ms) |
| `p95_response_time` | integer | 95th percentile (ms) |
| `avg_grab_response_time` | integer | Average grab time (ms) |
| **Status** | | |
| `status` | varchar | 'operational' / 'degraded' / 'down' |
| `success_rate` | numeric | Success percentage (0-100) |
| `last_check_at` | timestamp | Last health check |
| `last_success_at` | timestamp | Last successful query |
| `last_failure_at` | timestamp | Last failed query |
| **Metadata** | | |
| `indexer_type` | text | Indexer category |
| `description` | text | Indexer description |
| `language` | varchar | Indexer language |
| `privacy` | varchar | Privacy level |
| `created_at` | timestamp | Record creation |
| `updated_at` | timestamp | Last update |

**Primary Key:** `id`

---

#### `prowlarr_search_logs`
Individual search query logs from Prowlarr integration.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `indexer_id` | UUID | Which indexer |
| `search_query` | text | Search query made |
| `search_type` | text | 'movie', 'series', 'search' |
| `imdb_id` | text | IMDB ID searched |
| `tmdb_id` | text | TMDB ID searched |
| `response_time` | integer | Response time (ms) |
| `http_status` | integer | HTTP status code |
| `results_count` | integer | Results returned |
| `was_successful` | boolean | Did search succeed? |
| `error_type` | varchar | Type of error |
| `error_message` | text | Error details |
| `user_uuid` | UUID | User who triggered search |
| `timestamp` | timestamp | When search occurred |

**Foreign Key:** `indexer_id` → `prowlarr_indexers.id`

---

#### `prowlarr_indexer_logs`
Tracks cumulative query counts from Prowlarr for distribution analysis.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `indexer_id` | UUID | Which indexer |
| `queries_count` | integer | Cumulative query count at time |
| `timestamp` | timestamp | When snapshot was taken |

**Foreign Key:** `indexer_id` → `prowlarr_indexers.id`

---

#### `prowlarr_indexer_metrics`
Hourly metrics breakdown per indexer.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `indexer_id` | UUID | Which indexer |
| `hour_bucket` | timestamp | Hour boundary (rounded down) |
| `total_requests` | integer | Requests in this hour |
| `successful_requests` | integer | Successful in hour |
| `failed_requests` | integer | Failed in hour |
| `timeout_requests` | integer | Timeouts in hour |
| `avg_response_time` | integer | Average time (ms) |
| `min_response_time` | integer | Minimum time (ms) |
| `max_response_time` | integer | Maximum time (ms) |
| `p50_response_time` | integer | Median time (ms) |
| `p95_response_time` | integer | 95th percentile (ms) |
| `total_results_returned` | integer | Total results in hour |
| `created_at` | timestamp | Record creation |

**Foreign Key:** `indexer_id` → `prowlarr_indexers.id`

---

#### `prowlarr_stats_history`
Historical snapshots of indexer performance for trend analysis.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `indexer_id` | UUID | Which indexer |
| `queries_total` | integer | Total queries at snapshot |
| `grabs_total` | integer | Total grabs at snapshot |
| `avg_response_time` | integer | Average response time (ms) |
| `success_rate` | numeric | Success percentage |
| `snapshot_at` | timestamp | When snapshot taken |

**Foreign Key:** `indexer_id` → `prowlarr_indexers.id`

---

#### `prowlarr_global_stats`
Aggregated statistics across all indexers.

| Column | Type | Description |
|--------|------|-------------|
| `id` | integer | Singleton key (always 1) |
| `total_searches_24h` | integer | All searches in 24h |
| `successful_searches_24h` | integer | Successful searches |
| `failed_searches_24h` | integer | Failed searches |
| `timeout_searches_24h` | integer | Timeout searches |
| `avg_response_time_24h` | integer | Average time (ms) |
| `p95_response_time_24h` | integer | 95th percentile (ms) |
| `slowest_indexer_id` | UUID | Slowest indexer |
| `slowest_indexer_avg_time` | integer | Its avg time (ms) |
| `most_reliable_indexer_id` | UUID | Most reliable indexer |
| `most_reliable_success_rate` | real | Its success rate |
| `last_updated_at` | timestamp | Last update time |

---

### Utility Tables

#### `cache`
In-memory caching layer for frequently accessed data.

| Column | Type | Description |
|--------|------|-------------|
| `key` | text | Cache key |
| `value` | text | Cached value (JSON) |
| `expires_at` | bigint | Unix timestamp for expiration |
| `created_at` | timestamp | When cached |
| `last_accessed` | timestamp | Last access time |

**Primary Key:** `key`

---

#### `distributed_locks`
Prevents concurrent execution of critical operations across multiple processes.

| Column | Type | Description |
|--------|------|-------------|
| `key` | text | Lock identifier |
| `owner` | text | Process that owns lock |
| `expires_at` | bigint | Lock expiration (Unix timestamp) |
| `result` | text | Lock result/status |

**Primary Key:** `key`

---

## Views

### `v_pool_status`
Shows current pool capacity and utilization with alert status.

| Column | Type | Description |
|--------|------|-------------|
| `pool_key_id` | UUID | Pool identifier |
| `current_user_slots` | integer | Currently assigned users |
| `max_user_slots` | integer | Max assignable users |
| `current_concurrent_streams` | integer | Active streams right now |
| `max_concurrent_streams` | integer | Hard limit (35) |
| `status` | varchar | Pool health status |
| `is_active` | boolean | Accepting assignments? |
| `slot_utilization_pct` | numeric | User capacity % |
| `concurrency_utilization_pct` | numeric | Stream capacity % |
| `slot_status` | text | 'OK', 'WARNING', 'CRITICAL' |
| `concurrency_status` | text | 'OK', 'WARNING', 'CRITICAL' |

---

### `v_user_slots`
Shows user slot assignments with activity tracking.

| Column | Type | Description |
|--------|------|-------------|
| `user_id` | UUID | User UUID |
| `email` | text | User email |
| `slot_id` | UUID | Assigned slot |
| `slot_number` | integer | Slot position (1-100) |
| `pool_key_id` | UUID | Pool assignment |
| `assigned_at` | timestamp | Assignment date |
| `last_activity_at` | timestamp | Last request time |
| `total_streams` | integer | Stream count |
| `is_active` | boolean | Slot currently assigned? |
| `pool_status` | varchar | Pool health |
| `days_since_activity` | numeric | Days without activity |

---

## Relationships Diagram

```
users (1) ─── (many) user_subs
      ↓
users (1) ─── (many) torbox_assignments
      ↓
torbox_assignments ─── torbox_slots
      ↓
torbox_slots (many) ─── (1) torbox_pool

torbox_pool (1) ─── (many) torbox_concurrency_log
torbox_slots (1) ─── (many) torbox_concurrency_log

prowlarr_indexers (1) ─── (many) prowlarr_search_logs
prowlarr_indexers (1) ─── (many) prowlarr_indexer_logs
prowlarr_indexers (1) ─── (many) prowlarr_indexer_metrics
prowlarr_indexers (1) ─── (many) prowlarr_stats_history

users (1) ─── (many) torbox_usage_logs ─── (many) torbox_pool
torbox_pool (1) ─── (many) torbox_pool_health
```

---

## Key Design Patterns

### 1. Slot-Based User Capacity
```sql
-- Find available slot in healthy pool
SELECT ts.id, ts.slot_number
FROM torbox_slots ts
JOIN torbox_pool tp ON ts.pool_key_id = tp.id
WHERE ts.is_active = false
  AND tp.is_active = true
  AND tp.current_user_slots < tp.max_user_slots
LIMIT 1;
```

### 2. Current Concurrency Query
```sql
-- Get real-time concurrent streams per pool
SELECT 
  pool_key_id,
  COUNT(*) AS active_streams
FROM torbox_concurrency_log
WHERE stream_ended_at IS NULL
GROUP BY pool_key_id;
```

### 3. Slot Inactivity Reclamation
```sql
-- Find slots inactive for 3+ days
SELECT id, user_id, pool_key_id
FROM torbox_slots
WHERE is_active = true
  AND last_activity_at < NOW() - INTERVAL '3 days';
```

### 4. Indexer Performance Analysis
```sql
-- Compare indexer performance
SELECT 
  indexer_name,
  avg_response_time,
  success_rate,
  total_requests_24h
FROM prowlarr_indexers
WHERE is_enabled = true
ORDER BY avg_response_time ASC;
```

---

## Capacity Thresholds

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| **Slot Utilization** | 80% | 95% | Add new pool key |
| **Concurrent Streams** | 30/35 | 33/35 | Very close to limit |
| **Indexer Response** | 500ms | 2000ms | Mark degraded |
| **Search Success** | <95% | <80% | Disable indexer |

---

## Migrations

Total migrations: **45**

Key migration milestones:
- **Init** - Create core tables (cache, locks, peer_token)
- **20250314** - User data tables
- **20250320** - Torrent info tracking
- **20250408** - Magnet cache → Torrent streams
- **20250425** - Torrent stream sync info
- **20250529** - Trakt list integration
- **20251029** - Job logging
- **20251109** - Prowlarr indexer tracking
- **20251205** - Stremio account integration
- **20251206** - Trakt account integration
- **20251208** - Cross-account sync linking
- **20251220** - Drop legacy torbox_pool_keys table (replaced by improved torbox_pool)

---

## Indexes for Performance

Key indexes created for fast queries:

```sql
-- Pool lookups
CREATE INDEX idx_torbox_pool_is_active ON torbox_pool(is_active);
CREATE INDEX idx_torbox_pool_priority ON torbox_pool(priority);

-- Slot lookups
CREATE INDEX idx_torbox_slots_pool_key ON torbox_slots(pool_key_id);
CREATE INDEX idx_torbox_slots_user_id ON torbox_slots(user_id);
CREATE INDEX idx_torbox_slots_active ON torbox_slots(is_active, last_activity_at);

-- Concurrency queries
CREATE INDEX idx_torbox_concurrency_pool_active ON torbox_concurrency_log(pool_key_id, stream_ended_at) WHERE stream_ended_at IS NULL;

-- Search logs
CREATE INDEX idx_prowlarr_search_indexer ON prowlarr_search_logs(indexer_id);
```

---

## Summary

Chillproxy's database architecture is designed to:

1. **Manage user capacity** via slot-based assignments (up to 100 per pool)
2. **Track concurrency** separately from capacity (hard limit: 35 streams per TorBox API)
3. **Monitor pool health** with detailed health checks and failure tracking
4. **Measure indexer performance** with hourly metrics and historical snapshots
5. **Log all activity** for auditing, usage tracking, and trend analysis
6. **Support efficient queries** with strategic indexing and views

The separation of slots (user capacity) from concurrency (active streams) allows Chillproxy to efficiently scale TorBox pools and provide fine-grained monitoring of system health.

