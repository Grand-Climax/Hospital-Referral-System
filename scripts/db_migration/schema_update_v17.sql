-- Schema Update v17: Extend notificationtype enum with MISSED + MISSED_RESCHEDULE
--
-- Postgres won't allow adding enum values inside a transaction in older
-- versions, but `ALTER TYPE ... ADD VALUE IF NOT EXISTS` is safe on Neon
-- (Postgres 14+) and idempotent. Without these labels the application's
-- INSERT INTO notifications (notification_type = 'MISSED' / 'MISSED_RESCHEDULE')
-- silently fails inside GORM (rows:0), which is why patients never got
-- the missed-appointment or missed-then-rescheduled SMS.

ALTER TYPE notificationtype ADD VALUE IF NOT EXISTS 'MISSED';
ALTER TYPE notificationtype ADD VALUE IF NOT EXISTS 'MISSED_RESCHEDULE';
