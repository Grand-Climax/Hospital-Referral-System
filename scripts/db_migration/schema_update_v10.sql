-- ML prediction external service linkage (v10)

ALTER TABLE ml_predictions
    ADD COLUMN IF NOT EXISTS external_prediction_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS severity_tier VARCHAR(30),
    ADD COLUMN IF NOT EXISTS processing_time_ms NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS feedback_sent_at TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS idx_ml_predictions_external_prediction_id
    ON ml_predictions (external_prediction_id)
    WHERE external_prediction_id IS NOT NULL;
