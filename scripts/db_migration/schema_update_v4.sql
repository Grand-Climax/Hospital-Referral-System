-- Schema Update V4: Refine Notification subsystem

-- 1. Extend DeliveryStatus enum
-- Note: In Postgres, you can't add enum values inside a transaction easily in some versions,
-- but we use the IF NOT EXISTS pattern or just plain ALTER TYPE.
ALTER TYPE deliverystatus ADD VALUE IF NOT EXISTS 'RESEND';
ALTER TYPE deliverystatus ADD VALUE IF NOT EXISTS 'CANCELLED';

-- 2. Add retry_count to notifications
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;

-- 3. Add provider_message_id if missing (added in Phase 7 implementation but let's be sure)
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS provider_message_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_notification_provider_msg ON notifications(provider_message_id);
