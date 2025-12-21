# PostgreSQL Migrations - Quick Reference Card

## What Is a Migration?

A **versioned SQL script** that changes database schema, tracked in Git.

```
Migration = "Version control for your database"
```

---

## Why Do We Have Them?

| Problem | Solution |
|---------|----------|
| Manual SQL errors | Migrations run automatically |
| Out-of-sync databases | All servers apply same changes |
| No audit trail | Git tracks every change |
| Team coordination nightmare | No more "did you run the SQL?" |
| Hard to rollback | Down migrations undo changes |

---

## Why Check on Every Server Start?

```
Server Start → Scan migrations → Run new ones → Continue
```

**If migrations aren't checked:**
- New code expects columns that don't exist → 💥 CRASH
- Different servers have different schemas → 😱 CHAOS
- Developers manually apply SQL → 🤦 ERRORS

**With automatic checks:**
- ✅ All servers identical
- ✅ Automatic and safe
- ✅ Never forgotten
- ✅ Auditable (Git history)

---

## Chillproxy Migration Timeline

```
January 2025  → Migration 001: Create base tables
              ↓  (includes legacy torbox_pool_keys)
...
              ↓
December 2025 → Migration 045: Drop legacy torbox_pool_keys ← NEW
              ↓  (replaced by improved torbox_pool)
              ↓
Server starts → Goose checks: "Has 045 been applied?"
              ↓  No → Apply it
              ↓  Yes → Continue
              ↓
Done! ✅
```

---

## How Migrations Work

### File Structure

```sql
-- migrations/postgres/20251220120000_drop_legacy_torbox_pool_keys.sql

-- +goose Up
-- Runs when migrating FORWARD
DROP TABLE torbox_pool_keys;

-- +goose Down
-- Runs when rolling BACK
-- (Optional - left empty if irreversible)
```

### Migration Tracking

```sql
-- Goose creates this table automatically:
SELECT * FROM schema_migrations;

-- Shows which migrations have been applied:
VersionID          | IsDirty | Timestamp
20250101000000     | false   | 2025-01-01 12:00:00
20251220120000     | false   | 2025-12-20 12:00:00  ← NEW
```

---

## Real-World Example: Adding a Table

### Without Migrations ❌
```
Developer: "Run this SQL: CREATE TABLE notifications..."
Dev 1: Runs it on local
Dev 2: Forgets
Staging: Someone runs it
Production: ???
Result: Different schemas everywhere 💥
```

### With Migrations ✅
```
1. Create: migrations/postgres/20251220120100_create_notifications.sql
2. Write the SQL (Up and Down)
3. Commit to Git
4. Deploy to production
5. Server starts
6. Migration auto-applies
7. Done! ✅
```

---

## Current Chillproxy Schema

### Before (Legacy)
```
torbox_pool_keys
└─ Basic tracking only
   ├── id
   ├── api_key
   ├── is_active
   └── current_assignments
```

### After (Improved)
```
torbox_pool
├── Slot capacity       ← NEW
│   ├── max_user_slots (100)
│   └── current_user_slots
│
├── Concurrency         ← NEW
│   ├── max_concurrent_streams (35)
│   └── current_concurrent_streams
│
├── Health tracking     ← NEW
│   ├── status
│   ├── last_success_at
│   └── failure_rate_24h
│
└── Better design overall
```

**Migration:** `20251220120000_drop_legacy_torbox_pool_keys.sql`

---

## Key Concepts

### Idempotent
Running a migration multiple times = safe (won't create table twice)

```sql
CREATE TABLE IF NOT EXISTS users (...)  ✅ Safe
CREATE TABLE users (...)                ❌ Unsafe (fails if exists)
```

### Ordered
Migrations run in sequence. Migration 045 depends on 044, 043, etc.

### Reversible
Down migrations should undo the Up changes (when possible)

```sql
-- +goose Up
CREATE TABLE users (id INT PRIMARY KEY);

-- +goose Down
DROP TABLE users;  ← Undoes the creation
```

### Irreversible
Some changes can't be undone (data deletion, column removal)

```sql
-- +goose Up
DELETE FROM inactive_users;  ← Data gone forever

-- +goose Down
-- Can't restore deleted data!
-- Mark as irreversible
```

---

## Common Commands

### Run migrations automatically (server startup)
```
Server starts → Goose checks → Auto-applies new migrations
(All handled by Chillproxy internally)
```

### Check migration status (manual)
```sql
SELECT * FROM schema_migrations ORDER BY version_id DESC;
```

### Create new migration
```bash
# Create file with timestamp to ensure ordering
touch migrations/postgres/20251220120200_my_change.sql

# Edit the file with Up/Down sections
# Commit to Git
# Deploy → Server auto-applies
```

---

## Why Chillproxy Uses Goose

| Feature | Goose | Others |
|---------|-------|--------|
| Language | Go (matches Chillproxy) | Various |
| Simple SQL files | ✅ Yes | Some use ORM syntax |
| Versioning | ✅ Timestamp-based | Various |
| Tracking | ✅ schema_migrations table | Similar |
| Easy to understand | ✅ Plain SQL | Not always |

---

## Legacy Table Removal in Action

### Before
```
Database
├── torbox_pool_keys (OLD - basic tracking)
├── torbox_pool (NEW - improved version)
└── All code uses torbox_pool now
    (torbox_pool_keys is unused)
```

### After (this migration)
```
Database
├── torbox_pool_keys (REMOVED via migration)
└── torbox_pool (NOW the only pool table)
```

### Migration File
```sql
-- +goose Up
DROP TABLE IF EXISTS torbox_pool_keys;

-- +goose Down
-- Irreversible - legacy table not restored
```

---

## Summary

| Concept | Explanation |
|---------|---|
| **What** | Version-controlled SQL scripts |
| **Why** | Sync database schema with code automatically |
| **When** | Checked every server start |
| **How** | Goose reads files, tracks progress in DB |
| **Result** | All servers have identical schemas |

**Bottom line:** Migrations = "Git for your database" = No manual SQL coordination = Fewer bugs = Faster deployments 🚀

---

## Quick Links

- **Full Guide:** `MIGRATIONS_EXPLAINED.md`
- **What We Did:** `LEGACY_TABLE_REMOVAL_SUMMARY.md`
- **Database Docs:** `DATABASE_ARCHITECTURE.md`
- **Migration File:** `migrations/postgres/20251220120000_drop_legacy_torbox_pool_keys.sql`

