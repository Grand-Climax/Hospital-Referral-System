-- =====================================================================
-- Schema Update v16 - Schedule-on-Demand
-- =====================================================================
-- Replaces the rolling 30-day DailySchedule window with an on-demand
-- history log. Live capacity is now derived from TriageQueue counts plus
-- HospitalDepartment (StandardDailyLimit, OverbookLimit) and any active
-- CapacityOverride. DailySchedule rows are created on first booking and
-- only their BookedSlots column is updated thereafter (snapshot).
--
-- IMPORTANT: Run this migration ONCE on top of the existing schema. Do
-- NOT re-run the full migration or re-seed.
-- =====================================================================

-- 1. Add new HospitalDepartment columns. Default to 0 so existing rows
--    explicitly opt-in to overbooking only when the dept head updates
--    the value (e.g. via Hospital-Admin or DB UPDATE).
ALTER TABLE hospital_departments
    ADD COLUMN IF NOT EXISTS overbook_limit INT NOT NULL DEFAULT 0;

ALTER TABLE hospital_departments
    ADD COLUMN IF NOT EXISTS max_capacity_of_staff INT NOT NULL DEFAULT 0;

-- 2. Backfill OverbookLimit at the department level from any per-day
--    DailySchedule rows that previously held the value, so that the
--    department default reflects historical intent.
UPDATE hospital_departments hd
SET overbook_limit = sub.max_overbook
FROM (
    SELECT ds.hospital_id, ds.department_id,
           MAX(ds.overbook_limit) AS max_overbook
    FROM daily_schedules ds
    GROUP BY ds.hospital_id, ds.department_id
) sub
WHERE hd.hospital_id = sub.hospital_id
  AND hd.department_id = sub.department_id
  AND hd.overbook_limit = 0
  AND sub.max_overbook IS NOT NULL
  AND sub.max_overbook > 0;

-- 3. DailySchedule is now an immutable history log. Drop the optimistic
--    locking column (Version is no longer used by any code path) and
--    clear the table so it begins empty on the new model. Rows will be
--    re-created on the next booking for any given date.
ALTER TABLE daily_schedules DROP COLUMN IF EXISTS version;

TRUNCATE TABLE daily_schedules;

-- 4. Done. The new scheduling flow will:
--    - Count live bookings via TriageQueue (CountByDeptAndDate)
--    - Read MaxSlots from HospitalDepartment.StandardDailyLimit
--      (or CapacityOverride.NewLimit if an active override exists)
--    - Read OverbookLimit from HospitalDepartment.OverbookLimit
--    - Insert a DailySchedule snapshot on first booking, then update
--      BookedSlots on subsequent bookings for the same date.
