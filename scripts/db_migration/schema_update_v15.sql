-- Schema Update v15: Hardening ML pipeline tables and referral tracking columns
ALTER TABLE referrals ADD COLUMN IF NOT EXISTS ml_run_started_at TIMESTAMP;
ALTER TABLE referrals ADD COLUMN IF NOT EXISTS ml_successful_rerun_count INT DEFAULT 0;
ALTER TABLE referrals ADD COLUMN IF NOT EXISTS ml_last_failed_at TIMESTAMP;

-- Alter ml_predictions
-- Drop old index if it exists (idx_ml_prediction_referral_active was on (referral_id, is_active))
DROP INDEX IF EXISTS idx_ml_prediction_referral_active;

-- Drop is_active column
ALTER TABLE ml_predictions DROP COLUMN IF EXISTS is_active;

-- Enforce strict unique index for referral_id on ml_predictions (to enforce 1-to-1)
CREATE UNIQUE INDEX IF NOT EXISTS idx_ml_predictions_referral_id_unique ON ml_predictions(referral_id);
