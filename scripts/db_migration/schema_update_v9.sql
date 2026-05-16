-- Global configuration toggle
INSERT INTO system_configs (key, value) VALUES ('enable_cron_jobs', 'false')
ON CONFLICT (key) DO NOTHING;

-- Operational job state
CREATE TABLE IF NOT EXISTS job_checkpoints (
    job_name VARCHAR(100) PRIMARY KEY,
    last_run_at TIMESTAMP
);

INSERT INTO job_checkpoints (job_name, last_run_at) VALUES ('sms_processing', NULL), ('missed_appointments', NULL)
ON CONFLICT (job_name) DO NOTHING;
