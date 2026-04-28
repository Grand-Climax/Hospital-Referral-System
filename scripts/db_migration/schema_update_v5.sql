-- Phase 8.6: Sharded Automated Batch Scheduler with Auto-Notify

-- 1. Ensure scheduler_checkpoints table exists with correct constraints
CREATE TABLE IF NOT EXISTS scheduler_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id UUID NOT NULL REFERENCES hospitals(id) ON DELETE CASCADE,
    dept_id UUID NOT NULL REFERENCES hospital_departments(id) ON DELETE CASCADE,
    last_processed_at TIMESTAMP WITH TIME ZONE,
    lease_holder VARCHAR(255),
    lease_expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(hospital_id, dept_id)
);

CREATE INDEX IF NOT EXISTS idx_checkpoint_eligibility ON scheduler_checkpoints(last_processed_at) 
WHERE lease_expires_at IS NULL OR lease_expires_at < NOW();

-- 2. Seed auto_notify config
INSERT INTO system_configs (key, value) 
VALUES ('auto_notify', 'false')
ON CONFLICT (key) DO NOTHING;

-- 3. Pre-populate checkpoints for existing active departments
INSERT INTO scheduler_checkpoints (hospital_id, dept_id)
SELECT hd.hospital_id, hd.id
FROM hospital_departments hd
JOIN hospitals h ON hd.hospital_id = h.id
WHERE h.is_active = true 
  AND h.is_deleted = false
  AND hd.is_active = true
  AND NOT EXISTS (
      SELECT 1 FROM scheduler_checkpoints sc 
      WHERE sc.hospital_id = hd.hospital_id AND sc.dept_id = hd.id
  )
ON CONFLICT (hospital_id, dept_id) DO NOTHING;
