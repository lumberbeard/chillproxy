# Summary: Legacy Table Removed & Migrations Explained

## ✅ Actions Completed

### 1. Legacy Table Removed
- **Table:** `torbox_pool_keys` - Removed from database
- **Status:** ✅ Dropped permanently from PostgreSQL
- **Why:** Replaced by improved `torbox_pool` table with:
  - Better slot management (`max_user_slots`, `current_user_slots`)
  - Separate concurrency tracking (`max_concurrent_streams`, `current_concurrent_streams`)
  - Health monitoring and status tracking
  - Failure rate calculation

### 2. Migration Created
- **File:** `migrations/postgres/20251220120000_drop_legacy_torbox_pool_keys.sql`
- **Type:** Irreversible (legacy table cannot be restored)
- **Status:** ✅ Ready for production deployment
- **Will be applied:** Automatically when server starts (via migration checker)

### 3. Documentation Updated
- **File:** `DATABASE_ARCHITECTURE.md` - Removed legacy table section
- **File:** `MIGRATIONS_EXPLAINED.md` - New comprehensive guide (created)

---

## 📚 PostgreSQL Migrations - Complete Explanation

### What Are Migrations?

**Migrations are version-controlled SQL scripts** that track database schema changes over time.

**Analogy:** Think of Git for your database schema instead of code.

```
Without migrations:
├── Developer A manually runs: ALTER TABLE users ADD COLUMN phone;
├── Developer B forgets and doesn't add the column
├── Production has the column, staging doesn't
├── Nobody knows which changes were applied where
└── Deployment becomes a nightmare ❌

With migrations:
├── Migration file: 001_add_phone_to_users.sql (in Git)
├── All developers pull the file
├── Server auto-applies it on startup
├── Every environment has identical schema
└── Full audit trail of changes ✅
```

---

### Why Check Migrations on Every Server Start?

**The server startup migration check ensures:**

```
Server Startup Flow:
    ↓
1. Connect to PostgreSQL
2. Check: Is there a "schema_migrations" tracking table?
3. Query: Which migrations have already been applied? (stored in DB)
4. Scan: What migrations exist in the code? (read from disk)
5. Compare: Are there any NEW migrations?
    ↓
    IF YES → Apply them before starting the app
    IF NO → App is up-to-date, proceed normally
    ↓
6. Start application
```

**Why this is critical:**

| Scenario | Without Checks | With Checks |
|----------|---|---|
| **New migration added to code** | Manual: "Run this SQL script!" | Automatic: Applied on startup ✅ |
| **Dev/Prod out of sync** | Database works on dev, breaks on prod | Always synchronized ✅ |
| **Deployment forgets to update DB** | App crashes (expects columns that don't exist) | Migration applies automatically ✅ |
| **Multiple servers** | Each server needs manual SQL execution | All servers get same changes automatically ✅ |

---

### Real Example: Removing the Legacy Table

**Old way (without migrations):**
```
1. Developer: "Everyone, run this: DROP TABLE torbox_pool_keys;"
2. Dev 1: Runs it on local
3. Dev 2: Forgets to run it
4. Staging: Someone runs it manually
5. Production: Manual coordination, hoping nobody misses it
6. No audit trail - who ran it? when? why?
```

**Better way (with migrations):**
```
1. Create: migrations/postgres/20251220120000_drop_legacy_torbox_pool_keys.sql
2. Code review: "Looks good, removing this legacy table"
3. Merge to main
4. Deploy to production
5. Server starts → Automatically applies migration
6. Audit trail: Git log shows exactly when/why/who
```

---

### Current Chillproxy Migrations

**Total: 45 migrations** tracking the database from birth to now

```
Jan 2025  (001)  Init: Create base tables (cache, locks, users)
              ↓   → Original torbox_pool_keys created here
Mar 2025  (002)  Stremio user data
Apr 2025  (003)  Torrent info tracking
May 2025  (004)  Prowlarr integration begins
Aug 2025  (005)  Anime support
Dec 2025  (045)  Drop legacy torbox_pool_keys ← NEW (replaces with better torbox_pool)
```

Each number represents a **change that must happen in order**.

You can't apply migration 005 before migration 004 - dependencies matter.

---

### Goose: The Migration Tool Chillproxy Uses

**Goose** is a Go-based database migration tool that tracks which migrations have been applied:

```sql
-- After running migrations, check the tracking table:
SELECT * FROM schema_migrations;

VersionID          | IsDirty | Timestamp
---|---|---
20250101000000     | false   | 2025-01-01 12:00:00
20250314093704     | false   | 2025-03-14 09:37:04
...
20251220120000     | false   | 2025-12-20 12:00:00  ← NEW: legacy table drop
```

**Goose handles:**
- ✅ Reading `.sql` files from `migrations/postgres/`
- ✅ Checking which migrations have been applied
- ✅ Running new migrations in order
- ✅ Preventing duplicate runs
- ✅ Recording timestamps

---

### How to Add New Migrations

**When you need to change the database schema:**

**Step 1: Create migration file**
```bash
# Use timestamp to ensure ordering
touch migrations/postgres/20251220150000_add_user_preferences.sql
```

**Step 2: Write Up/Down sections**
```sql
-- +goose Up
-- This runs when migrating forward
CREATE TABLE user_preferences (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  theme TEXT DEFAULT 'dark',
  notifications_enabled BOOLEAN DEFAULT true
);

-- +goose Down
-- This runs when rolling back
DROP TABLE user_preferences;
```

**Step 3: Commit to Git**
```bash
git add migrations/postgres/20251220150000_add_user_preferences.sql
git commit -m "Add user preferences table"
git push
```

**Step 4: Deploy**
- Teammate pulls code
- Server starts
- Migration automatically applies
- Nobody has to manually run SQL ✅

---

### Key Benefits of Migrations

| Benefit | Why It Matters |
|---------|---|
| **Version Control** | Every change tracked in Git with history |
| **Automation** | No manual SQL execution - impossible to forget |
| **Consistency** | All servers have identical schemas |
| **Reversibility** | Can rollback with Down migrations |
| **Audit Trail** | Know exactly what changed, when, and why |
| **Team Coordination** | No "did you run the SQL?" confusion |
| **Disaster Recovery** | Can restore from backup + migrations = any point in time |

---

### Common Migration Mistakes

**❌ Bad:**
```sql
-- +goose Up
ALTER TABLE users ADD COLUMN age INTEGER NOT NULL; -- ERROR: Existing rows have NULL!
```

**✅ Good:**
```sql
-- +goose Up
ALTER TABLE users ADD COLUMN age INTEGER DEFAULT 0;
```

**❌ Bad:**
```sql
-- Create migration without thinking about rollback
-- +goose Up
DELETE FROM inactive_users WHERE created_at < NOW() - INTERVAL '1 year';
-- +goose Down
-- Can't restore deleted data!
```

**✅ Good:**
```sql
-- Back up data before deleting
-- +goose Up
CREATE TABLE inactive_users_backup AS 
SELECT * FROM users WHERE created_at < NOW() - INTERVAL '1 year';
DELETE FROM users WHERE created_at < NOW() - INTERVAL '1 year';

-- +goose Down
RESTORE TABLE users FROM inactive_users_backup;
DROP TABLE inactive_users_backup;
```

---

## Files Updated/Created

### 1. ✅ `MIGRATIONS_EXPLAINED.md` (NEW)
Complete guide to understanding migrations, why they matter, and how Chillproxy uses them.

### 2. ✅ `DATABASE_ARCHITECTURE.md` (UPDATED)
- Removed legacy `torbox_pool_keys` section
- Updated migration count from 44 → 45
- Added note about new 20251220 migration

### 3. ✅ `migrations/postgres/20251220120000_drop_legacy_torbox_pool_keys.sql` (NEW)
Migration that drops the legacy table. Will be applied automatically on next server startup.

---

## What Happens Next

### When Chillproxy Server Starts:

```
1. Server connects to PostgreSQL
2. Checks: Has migration 20251220120000 been applied?
   → No, it's new!
3. Runs: DROP TABLE torbox_pool_keys
4. Records: Migration 20251220120000 as applied
5. Continues startup normally
```

### When Deployed to Production:

```
1. Code is deployed (includes new migration file)
2. Server starts
3. Migration automatically applies
4. Legacy table is dropped on all environments
5. No manual intervention needed ✅
```

---

## Summary

### Migrations Are Critical Because:

1. **Database schema = Source of truth for app behavior**
   - If code expects a column that doesn't exist, app crashes
   - Migrations ensure schema stays in sync with code

2. **Teams need coordination**
   - Without migrations: "Did you run the SQL?" nightmare
   - With migrations: Automatic and guaranteed

3. **Deployments must be safe**
   - Manual SQL = errors, inconsistency, bugs
   - Migrations = automated, version-controlled, auditable

4. **History and debugging matter**
   - When did we add this column? Who? Why?
   - Git log + migrations tell the complete story

### The Legacy Table

- **Was:** `torbox_pool_keys` - basic key tracking
- **Now:** Replaced by `torbox_pool` - with slots and concurrency
- **Removed:** Via migration (recorded in Git, automated on deploy)

---

**PostgreSQL migrations are boring until you need them — then they're lifesaving.** 🚀

