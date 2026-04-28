ALTER TABLE capacity_overrides ALTER COLUMN dept_id DROP NOT NULL;
ALTER TABLE daily_schedules ALTER COLUMN dept_id DROP NOT NULL;
ALTER TABLE triage_queues ALTER COLUMN dept_id DROP NOT NULL;
ALTER TABLE scheduler_checkpoints ALTER COLUMN dept_id DROP NOT NULL;
