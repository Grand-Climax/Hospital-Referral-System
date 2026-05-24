# Database Migrations

This folder holds raw SQL deltas that an operator applies on top of the
running database. Migrations are versioned by file name (`schema_update_v16.sql`,
`schema_update_v17.sql`, ...).

## Applying `schema_update_v16.sql` (Schedule-on-Demand)

`v16` introduces the Schedule-on-Demand model. It:

1. Adds `overbook_limit` and `max_capacity_of_staff` columns to
   `hospital_departments`.
2. Backfills `overbook_limit` from the historical per-day values that lived
   on `daily_schedules`.
3. Drops the `version` column from `daily_schedules`.
4. `TRUNCATE`s `daily_schedules` so the table becomes an append-on-demand
   history log going forward.

Step 4 is destructive for booking history. Apply once, with eyes open.

### Option A - one-shot Go runner (recommended)

```
go run scripts/run_db_migration_v16.go --confirm
```

The runner:

- Refuses to run without `--confirm`.
- Reads DB connection from the standard `DB_HOST` / `DB_USER` / ... env
  vars (or a single `DATABASE_URL` override if set).
- Checks `system_configs.schema_version` first. If it already equals
  `v16`, prints "already applied" and exits 0.
- Wraps the SQL plus the upsert of `system_configs.schema_version = 'v16'`
  in a single transaction.
- Does NOT self-delete. Re-running is safe (it no-ops on the guard row).

### Option B - psql

```
psql "$DATABASE_URL" -f scripts/db_migration/schema_update_v16.sql
```

If you go this route, also insert the guard row manually:

```sql
INSERT INTO system_configs (key, value, updated_at)
VALUES ('schema_version', 'v16', NOW())
ON CONFLICT (key) DO UPDATE
    SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;
```
