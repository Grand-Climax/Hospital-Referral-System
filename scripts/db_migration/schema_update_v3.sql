-- Schema Update V3: Add hospital_id and department_id to scheduling tables

-- 1. DailySchedule
ALTER TABLE daily_schedules
  ADD COLUMN IF NOT EXISTS hospital_id UUID REFERENCES hospitals(id),
  ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id);

-- Populate from HospitalDepartment mapping
UPDATE daily_schedules ds
SET hospital_id = hd.hospital_id,
    department_id = hd.department_id
FROM hospital_departments hd
WHERE ds.dept_id = hd.id;

-- Make NOT NULL after population
ALTER TABLE daily_schedules ALTER COLUMN hospital_id SET NOT NULL;
ALTER TABLE daily_schedules ALTER COLUMN department_id SET NOT NULL;

-- 2. CapacityOverride
ALTER TABLE capacity_overrides
  ADD COLUMN IF NOT EXISTS hospital_id UUID REFERENCES hospitals(id),
  ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id);

UPDATE capacity_overrides co
SET hospital_id = hd.hospital_id,
    department_id = hd.department_id
FROM hospital_departments hd
WHERE co.dept_id = hd.id;

ALTER TABLE capacity_overrides ALTER COLUMN hospital_id SET NOT NULL;
ALTER TABLE capacity_overrides ALTER COLUMN department_id SET NOT NULL;

-- 3. TriageQueue
ALTER TABLE triage_queues
  ADD COLUMN IF NOT EXISTS hospital_id UUID REFERENCES hospitals(id),
  ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id);

UPDATE triage_queues tq
SET hospital_id = hd.hospital_id,
    department_id = hd.department_id
FROM hospital_departments hd
WHERE tq.dept_id = hd.id;

ALTER TABLE triage_queues ALTER COLUMN hospital_id SET NOT NULL;
ALTER TABLE triage_queues ALTER COLUMN department_id SET NOT NULL;

-- 4. Indexes
-- Drop old indexes if they were strict on dept_id
-- We'll add new composite indexes
CREATE INDEX IF NOT EXISTS idx_schedule_hosp_dept_date ON daily_schedules(hospital_id, department_id, schedule_date);
CREATE INDEX IF NOT EXISTS idx_override_hosp_dept_date ON capacity_overrides(hospital_id, department_id, target_date);
CREATE INDEX IF NOT EXISTS idx_triage_hosp_dept_status ON triage_queues(hospital_id, department_id, queue_status);
