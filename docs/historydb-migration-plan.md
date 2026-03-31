# History DB Migration Plan

This note documents a low-risk migration strategy for `internal/backend/storage/historydb` before we make any incompatible schema changes.

## Current State

- The store currently treats `schema.sql` as an idempotent bootstrap script.
- `Store.migrate()` executes the full schema blob on every open.
- There is no explicit schema version tracking yet.
- There is no migration registry or stepwise upgrade path.

That is acceptable while the schema only grows through `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, and additive triggers. It becomes risky as soon as we need data backfills, column rewrites, table rebuilds, or behavior that must run exactly once.

## Goals

- Track the on-disk schema version explicitly.
- Support forward-only, stepwise migrations.
- Keep first-install bootstrap simple.
- Make migration failures visible and safe to recover from.
- Avoid silent partial upgrades.

## Recommended Design

1. Add explicit version tracking.
   Use `PRAGMA user_version` as the canonical schema version for this local SQLite database.

2. Split bootstrap schema from incremental migrations.
   Keep a base schema for fresh installs, and add numbered migration steps for upgrades.

3. Make migrations transactional by default.
   Each version step should run inside a transaction unless SQLite requires otherwise.

4. Apply migrations one version at a time.
   If the database is at version `N`, run `N+1`, then `N+2`, and so on until `currentVersion`.

5. Fail closed.
   If any migration step fails, abort startup and surface a wrapped error that includes the source and target version.

## Suggested Layout

- `schema.sql`
  Keep this as the latest bootstrap schema for new databases, or rename it to `schema_base.sql` later if that becomes clearer.
- `migrations.go`
  Define the current schema version and the ordered migration list.
- `migration_steps/`
  Optional directory if SQL-based migrations grow large enough to justify separate files.

## Suggested Runtime Flow

1. Open SQLite connection.
2. Read `PRAGMA user_version`.
3. If version is `0` and the database is effectively empty:
   Execute the bootstrap schema and set `user_version = currentVersion`.
4. If version is between `1` and `currentVersion - 1`:
   Run each migration step in order.
5. If version is greater than `currentVersion`:
   Return an error because the binary is older than the database.

## Suggested Step Shape

Each migration step should capture:

- `fromVersion`
- `toVersion`
- `name`
- `run(ctx, tx)` function

That keeps schema evolution reviewable in code and easy to test.

## Testing Plan

Add migration-focused tests for:

1. Fresh install:
   Opening a new DB should create the schema and set the latest version.

2. Step upgrade:
   A DB at version `N` should upgrade to `N+1` and preserve existing data.

3. Multi-step upgrade:
   A DB several versions behind should apply every migration in order.

4. Unknown future version:
   Opening a DB with a newer `user_version` should fail with a clear error.

5. Failed migration:
   A broken migration should roll back its transaction and leave the DB usable for diagnosis.

## First Implementation Slice

The safest first slice is:

1. Introduce `PRAGMA user_version`.
2. Treat the current schema as version `1`.
3. Keep behavior unchanged for fresh installs.
4. Add tests proving that new DBs are stamped with version `1`.

That gives us version tracking now without forcing an immediate schema rewrite.

## Why This Order

- It keeps today’s stable schema behavior intact.
- It creates a clear seam for future incompatible changes.
- It avoids mixing migration infrastructure work with speculative schema edits.
