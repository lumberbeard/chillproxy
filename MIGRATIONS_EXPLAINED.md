# PostgreSQL Migrations Explained

## What Are Database Migrations?

A **migration** is a versioned SQL script that makes changes to your database schema. Think of it as a **"version control for your database"** - similar to how Git tracks code changes.

### Real-World Analogy

Imagine you're building a house:
- **Initial migration** = Build the foundation and walls
- **Migration 2** = Add plumbing
- **Migration 3** = Add electrical wiring
- **Migration 4** = Paint the walls blue
- **Migration 5** = Change wall color to white

Each step is **irreversible without planning**, and each step depends on the previous one. You can't paint walls before they exist.

---

## Why Do We Need Migrations?

### Problem Without Migrations

**Without migrations**, you might have developers doing this:

```sql
-- Developer A's local changes (never documented)
ALTER TABLE users ADD COLUMN phone TEXT;

-- Developer B's local changes (conflicting!)
ALTER TABLE users ADD COLUMN phone_number TEXT;

-- Developer C manually runs SQL on production
UPDATE users SET email = 'admin@example.com' WHERE id = 1;

-- Now the database is a mess:
-- - Different schemas on dev/staging/production
-- - No way to know what changed or when
-- - Impossible to rollback a mistake
-- - New developers don't know what tables exist
```

### Solution With Migrations

```
migrations/postgres/
├── 001_init.sql                    # Create initial tables
├── 002_add_user_phone.sql          # Add phone column
├── 003_create_torbox_pool.sql      # Create pool management
├── 004_add_slots.sql               # Add slot system
└── 005_add_concurrency_tracking.sql # Add concurrency logs
```

Each migration is **numbered**, **documented**, and **version-controlled in Git**.

---

## How Chillproxy Uses Migrations

### Migration Files in Chillproxy

```
chillproxy/migrations/postgres/
├── 20250101000000_init.sql                           # Core tables
├── 20250314093704_create_table_stremio_userdata.sql  # User config
├── 20250320173921_create_table_torrent_info.sql      # Torrent metadata
├── 20250408141648_update_table_mcf_to_torrent_stream.sql  # Rename table
├── ...
└── 20251213120000_create_table_sync_stremio_stremio_link.sql  # Recent
```

**44 total migrations** tracking the entire database evolution since January 2025.

### Migration File Structure

Each migration uses **Goose** (a Go migration tool) with `Up` and `Down` sections:

```sql
-- +goose Up
-- This runs when migrating FORWARD
CREATE TABLE torbox_pool (
  id UUID PRIMARY KEY,
  api_key TEXT NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- This runs when migrating BACKWARD (rollback)
DROP TABLE torbox_pool;
```

---

## Why Check Migrations Every Server Start?

### The Migration Check Process

When Chillproxy starts:

```
Server Startup
    ↓
Check database for "migrations_applied" tracking table
    ↓
Read all migration files from disk (001, 002, 003... 044)
    ↓
Compare: Which migrations have been applied?
    ↓
Are there ANY new migrations NOT in the database?
    ↓
    If YES → Run them automatically
    If NO → Database is up-to-date, continue starting
    ↓
Server starts successfully
```

### Why This Is Critical

**Scenario 1: Development**
```
Developer A pushes new migration (045_add_new_feature.sql)
Developer B pulls the code
Developer B starts server
→ Server detects migration 045 hasn't run
→ Server automatically runs it
→ Both developers have identical databases ✅
```

**Scenario 2: Deployment**
```
Production deployment happens
Migration 45, 46, 47 are in the codebase
Server starts and checks: "Have migrations 45, 46, 47 been applied?"
→ No, so apply them before starting
→ Production database gets updated automatically ✅
```

**Scenario 3: Without Checks**
```
Production deployment happens
Database schema changes aren't applied (developer forgot to manually run them)
Server starts and crashes because new code expects columns that don't exist ❌
```

---

## Concrete Example: Chillproxy's Slot Architecture

### How the Slot System Was Added via Migrations

**Migration 005_slot_architecture.sql** (that we created) did this:

```sql
-- +goose Up
CREATE TABLE torbox_slots (
  id UUID PRIMARY KEY,
  pool_key_id UUID NOT NULL,
  slot_number INTEGER NOT NULL,
  user_id UUID,
  ...
);

CREATE TABLE torbox_concurrency_log (
  id UUID PRIMARY KEY,
  pool_key_id UUID NOT NULL,
  user_id UUID NOT NULL,
  stream_started_at TIMESTAMP,
  stream_ended_at TIMESTAMP,
  ...
);

-- Migrate existing data
ALTER TABLE torbox_assignments ADD COLUMN slot_id UUID;
-- ... populate slot_id from existing data ...
```

**Without the migration check:**
- New code expects `torbox_slots` table to exist
- If database doesn't have it yet, app crashes
- Manual coordination needed: "Run this SQL first, then deploy the code"
- Error-prone and slow

**With the migration check:**
- Code and database changes stay synchronized
- Server automatically runs missing migrations
- Deployment is automated and reliable

---

## Migration Tools: Goose vs Alternatives

Chillproxy uses **Goose**, a popular Go migration tool:

| Tool | Language | How It Works |
|------|----------|-------------|
| **Goose** | Go | Reads `.sql` files, tracks in DB table |
| **Flyway** | Java | Similar concept, tracks migration versions |
| **Knex/Alembic** | JS/Python | ORM-based migrations |
| **Rails Migrations** | Ruby | ORM syntax for defining schema |

Goose tracks which migrations have run in a special table:

```sql
SELECT * FROM schema_migrations;

-- Output:
VersionID | IsDirty | Timestamp
--------+--------+-------------------
20250101000000 | false  | 2025-01-01 12:00:00
20250314093704 | false  | 2025-03-14 09:37:04
20250320173921 | false  | 2025-03-20 17:39:21
... (44 total)
```

---

## Migration Workflow in Chillproxy

### For Developers (Adding a Feature)

**Step 1: Create migration file**
```bash
# Create new migration with timestamp
touch migrations/postgres/20250620120000_add_user_notifications.sql
```

**Step 2: Write the migration**
```sql
-- +goose Up
CREATE TABLE user_notifications (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  message TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE user_notifications;
```

**Step 3: Commit to Git**
```bash
git add migrations/postgres/20250620120000_add_user_notifications.sql
git commit -m "Add user notifications table"
git push
```

**Step 4: Deploy to production**
```
Team member pulls code → Server starts → Migration auto-runs → Done ✅
```

### No Manual SQL Execution Needed!

Without migrations:
```
1. Pull code
2. Connect to prod database
3. Manually run: CREATE TABLE user_notifications...
4. Hope you didn't typo it
5. Start the server
```

With migrations:
```
1. Pull code
2. Start server (auto-applies any new migrations)
3. Done ✅
```

---

## Current State: Chillproxy Has 44 Migrations

### Timeline of Evolution

```
January 2025    → 001: Initialize (cache, locks, user tables)
                      (torbox_pool created here - LEGACY version)

March 2025      → 002-004: Add torrent tracking, streaming

May 2025        → 005-009: Add Prowlarr integration, OAuth tokens

August 2025     → 010-014: Add anime support, metadata indexes

December 2025   → 015-044: Add Stremio accounts, Trakt sync, etc.
                           + 005_slot_architecture.sql (new slot system)
```

Each migration **builds on the previous one**, creating a complete audit trail of every schema change.

---

## What Happens When a Migration Fails?

### Scenario: Bad Migration

```sql
-- +goose Up
ALTER TABLE users ADD COLUMN age INTEGER NOT NULL; -- ERROR: existing rows have NULL

-- +goose Down
ALTER TABLE users DROP COLUMN age;
```

**When it fails:**
1. Migration 050 starts running
2. Error: Column age must have a default for existing rows
3. Migration stops (doesn't proceed)
4. Server doesn't start (safe fail)
5. Developer must fix the migration

**Correct version:**
```sql
-- +goose Up
ALTER TABLE users ADD COLUMN age INTEGER DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN age;
```

**Goose marks this as dirty** in `schema_migrations` to alert you something went wrong.

---

## Removing the Legacy Table: How Migrations Help

Instead of:
```
1. Developer manually connects to prod
2. Manually runs: DROP TABLE torbox_pool_keys;
3. Hope it works everywhere
4. No record of what happened
```

We can do:
```sql
-- migrations/postgres/20251220120000_drop_legacy_torbox_pool_keys.sql

-- +goose Up
DROP TABLE torbox_pool_keys;

-- +goose Down
-- Can't restore table without backup, so note this is irreversible
-- But Goose will track that we tried to remove it
```

**Benefits:**
- ✅ Change is versioned in Git
- ✅ Applied automatically on all environments
- ✅ If needed, we can see when it was removed
- ✅ Can be tracked in code review
- ✅ Safe: fails early if table has dependencies

---

## Summary

### Why Migrations?

| Problem | Solution |
|---------|----------|
| Database schema out of sync | Version control for DB schema |
| Manual SQL errors | Automated application |
| No audit trail | Complete history in Git |
| Coordination nightmare | Self-serve: deploy code → auto-update DB |
| Hard to rollback | Down migrations undo changes |
| New developers confused | Read migration files to understand evolution |

### Why Check at Startup?

1. **Automatic**: No manual "run these 5 SQL scripts" steps
2. **Safe**: Schema stays in sync with code
3. **Idempotent**: Running migrations multiple times is safe
4. **Auditable**: Every change tracked in Git
5. **Reversible**: Can rollback with `Down` migrations

### For Chillproxy Specifically

- **44 migrations** = 1 year of schema evolution fully tracked
- **5th migration** = New slot architecture we implemented
- **Next migration** = Removing legacy table (new migration file)
- Every deployment = Automatic schema updates

---

## Key Takeaway

**Migrations are like Git for your database:**
- Versions track changes
- Changes can be reviewed in pull requests
- Deployments apply changes automatically
- History is preserved forever
- Teams stay synchronized

Without migrations, databases become "snowflake servers" — each one is different, nobody knows why, and deployment becomes a nightmare.

With migrations, your database evolves safely alongside your code.

