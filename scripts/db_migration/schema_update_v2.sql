-- Hospital Referral Hub - Database Schema Upgrade (v2)
-- This script adds enums, alters existing tables, and creates new tables for the final design.

-- Part 1: Create Missing Enums
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'diagnosiscertainty') THEN
        CREATE TYPE DiagnosisCertainty AS ENUM ('CONFIRMED', 'SUSPECTED', 'SYMPTOM_ONLY');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'arrivalstatus') THEN
        CREATE TYPE ArrivalStatus AS ENUM ('EXPECTED', 'ARRIVED', 'ADMITTED', 'MISSED');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'deliverystatus') THEN
        CREATE TYPE DeliveryStatus AS ENUM ('QUEUED', 'SENT', 'DELIVERED', 'FAILED');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'accesstype') THEN
        CREATE TYPE AccessType AS ENUM ('TREATING_DOCTOR', 'CONSULTED_DOCTOR');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'missreason') THEN
        CREATE TYPE MissReason AS ENUM ('PATIENT_NO_SHOW', 'PATIENT_CONTACTED_RESCHEDULE', 'HOSPITAL_CANCELLED', 'HOSPITAL_CAPACITY_ISSUE');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'triagestatus') THEN
        CREATE TYPE TriageStatus AS ENUM ('AUTO_SCORED', 'REVIEWED', 'OVERRIDDEN');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'queuestatus') THEN
        CREATE TYPE QueueStatus AS ENUM ('WAITING', 'SCHEDULED', 'ARRIVED', 'MISSED', 'CANCELLED');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notificationtype') THEN
        CREATE TYPE NotificationType AS ENUM ('ACCEPTANCE', 'SCHEDULING', 'REMINDER', 'RESCHEDULE');
    END IF;
END $$;

-- Update or Create ReferralStatus enum
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'referralstatus') THEN
        CREATE TYPE ReferralStatus AS ENUM (
            'DRAFT', 'SUBMITTED', 'UNDER_LIAISON_REVIEW', 'FORWARDED', 
            'UNDER_SPECIALIST_REVIEW', 'ACCEPTED', 'SCHEDULED', 'ASSIGNED', 
            'COMPLETED', 'NEED_REVISION', 'CANCELLED', 'REJECTED_BY_LIAISON', 
            'REJECTED_BY_SPECIALIST', 'MISSED', 'RESCHEDULED', 'REDIRECTED', 
            'ADMITTED', 'REJECTED_AFTER_SEND', 'DECEASED'
        );
    ELSE
        ALTER TYPE ReferralStatus ADD VALUE IF NOT EXISTS 'REDIRECTED';
        ALTER TYPE ReferralStatus ADD VALUE IF NOT EXISTS 'ADMITTED';
        ALTER TYPE ReferralStatus ADD VALUE IF NOT EXISTS 'REJECTED_AFTER_SEND';
        ALTER TYPE ReferralStatus ADD VALUE IF NOT EXISTS 'DECEASED';
    END IF;
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Update or Create ActionType enum
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'actiontype') THEN
        CREATE TYPE ActionType AS ENUM (
            'CREATE_REFERRAL', 'APPROVE_REFERRAL', 'REJECT_REFERRAL', 
            'ACCEPT_REFERRAL', 'REDIRECT_REFERRAL', 'OVERRIDE_ML_SCORE', 
            'VIEW_PATIENT_DATA', 'LOGIN', 'LOGOUT', 'EXPORT_DATA', 
            'MANAGE_USERS', 'ASSIGN_ROLES', 'MANAGE_CAPACITY', 
            'VIEW_AUDIT_LOG', 'RESET_MFA', 'MANAGE_HOSPITALS', 
            'MANAGE_DEPTS', 'API_CALL', 'OVERRIDE_QUEUE', 
            'GENERATE_REPORTS', 'UPDATE_PATIENT_STATUS', 'CANCEL_REFERRAL'
        );
    ELSE
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'MANAGE_CAPACITY';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'OVERRIDE_QUEUE';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'CANCEL_REFERRAL';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'UPDATE_PATIENT_STATUS';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'MANAGE_HOSPITALS';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'MANAGE_DEPTS';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'VIEW_AUDIT_LOG';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'RESET_MFA';
        ALTER TYPE ActionType ADD VALUE IF NOT EXISTS 'ASSIGN_ROLES';
    END IF;
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Part 2: Alter Existing Tables
-- Referral
ALTER TABLE referrals DROP COLUMN IF EXISTS triage_status;
ALTER TABLE referrals ADD COLUMN triage_status TriageStatus DEFAULT 'AUTO_SCORED';

-- Patient
ALTER TABLE patients ADD COLUMN IF NOT EXISTS allow_sms BOOLEAN DEFAULT TRUE;

-- Part 3: Create Missing Tables

-- 1. CapacityOverride
CREATE TABLE IF NOT EXISTS capacity_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dept_id UUID NOT NULL REFERENCES hospital_departments(id) ON DELETE CASCADE,
    target_date DATE NOT NULL,
    new_limit INT NOT NULL CHECK (new_limit >= 0),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(dept_id, target_date)
);
CREATE INDEX IF NOT EXISTS idx_capacity_override_dept_date ON capacity_overrides(dept_id, target_date);

-- 2. TriageQueue
CREATE TABLE IF NOT EXISTS triage_queues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    dept_id UUID NOT NULL REFERENCES hospital_departments(id),
    appointment_date DATE,
    composite_score DECIMAL(5,2) NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    arrival_status ArrivalStatus NOT NULL DEFAULT 'EXPECTED',
    arrived_at TIMESTAMP WITH TIME ZONE,
    marked_by UUID REFERENCES users(id),
    miss_reason MissReason,
    assigned_doctor_id UUID REFERENCES users(id),
    doctor_assigned_at TIMESTAMP WITH TIME ZONE,
    queue_status QueueStatus DEFAULT 'WAITING',
    arrival_boost INT DEFAULT 0,
    reschedule_reason VARCHAR(50),
    UNIQUE(referral_id, appointment_date)
);
CREATE INDEX IF NOT EXISTS idx_triage_queue_dept_score ON triage_queues(dept_id, appointment_date, composite_score DESC);
CREATE INDEX IF NOT EXISTS idx_triage_queue_date_status ON triage_queues(appointment_date, arrival_status);
CREATE INDEX IF NOT EXISTS idx_triage_queue_doctor ON triage_queues(assigned_doctor_id);

-- 3. DailySchedule
CREATE TABLE IF NOT EXISTS daily_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dept_id UUID NOT NULL REFERENCES hospital_departments(id),
    schedule_date DATE NOT NULL,
    booked_slots INT DEFAULT 0 CHECK (booked_slots >= 0),
    max_slots INT NOT NULL CHECK (max_slots > 0),
    overbook_limit INT DEFAULT 0,
    version INT DEFAULT 1,
    UNIQUE(dept_id, schedule_date)
);
CREATE INDEX IF NOT EXISTS idx_daily_schedule_availability ON daily_schedules(dept_id, schedule_date, booked_slots);

-- 4. Notification
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    delivery_status DeliveryStatus NOT NULL DEFAULT 'QUEUED',
    failure_reason TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    notification_type NotificationType
);
CREATE INDEX IF NOT EXISTS idx_notification_referral ON notifications(referral_id);
CREATE INDEX IF NOT EXISTS idx_notification_status ON notifications(delivery_status);

-- 5. MLPrediction
CREATE TABLE IF NOT EXISTS ml_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    trigger_reason VARCHAR(30) NOT NULL DEFAULT 'INITIAL',
    input_features JSONB NOT NULL,
    output_score DECIMAL(5,2) NOT NULL CHECK (output_score BETWEEN 0 AND 100),
    confidence_level DECIMAL(5,2) CHECK (confidence_level BETWEEN 0 AND 100),
    explanation JSONB,
    model_version VARCHAR(50) NOT NULL,
    predicted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_overridden BOOLEAN DEFAULT FALSE,
    overridden_score DECIMAL(5,2) CHECK (overridden_score BETWEEN 0 AND 100),
    overridden_by UUID REFERENCES users(id),
    override_justification TEXT,
    overridden_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_ml_prediction_referral_active ON ml_predictions(referral_id, is_active);
CREATE INDEX IF NOT EXISTS idx_ml_prediction_model_override ON ml_predictions(model_version, is_overridden);

-- 6. ReferralAccess
CREATE TABLE IF NOT EXISTS referral_accesses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    access_type AccessType NOT NULL,
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoke_reason TEXT,
    UNIQUE(referral_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_referral_access_referral ON referral_accesses(referral_id);
CREATE INDEX IF NOT EXISTS idx_referral_access_user ON referral_accesses(user_id);
CREATE INDEX IF NOT EXISTS idx_referral_access_user_active ON referral_accesses(user_id) WHERE revoked_at IS NULL;

-- 7. ClinicalUpdate
CREATE TABLE IF NOT EXISTS clinical_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    updated_by UUID NOT NULL REFERENCES users(id),
    update_reason VARCHAR(100) NOT NULL, -- 'MISSED_APPOINTMENT_RE_EVALUATION','CONDITION_CHANGE','SPECIALIST_NOTE'
    clinical_notes TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    requires_review BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_clinical_update_referral ON clinical_updates(referral_id, created_at);

-- 8. ReferralOutcome
CREATE TABLE IF NOT EXISTS referral_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL UNIQUE REFERENCES referrals(id) ON DELETE CASCADE,
    outcome VARCHAR(50) NOT NULL, -- 'improved','deteriorated','deceased','transferred','discharged'
    length_of_stay_days INT,
    was_referral_appropriate BOOLEAN,
    outcome_notes TEXT,
    recorded_by UUID NOT NULL REFERENCES users(id),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 9. ReferralRedirection
CREATE TABLE IF NOT EXISTS referral_redirections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL UNIQUE REFERENCES referrals(id) ON DELETE CASCADE,
    redirected_to_hospital_id UUID REFERENCES hospitals(id),
    redirection_reason TEXT
);

-- 10. SchedulerCheckpoint
CREATE TABLE IF NOT EXISTS scheduler_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id UUID NOT NULL REFERENCES hospitals(id),
    dept_id UUID NOT NULL REFERENCES hospital_departments(id),
    last_processed_at TIMESTAMP WITH TIME ZONE,
    lease_holder VARCHAR(255),
    lease_expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(hospital_id, dept_id)
);

-- 11. SystemConfig
CREATE TABLE IF NOT EXISTS system_configs (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- Insert default SystemConfig values
INSERT INTO system_configs (key, value) VALUES 
('buffer_days', '2'),
('aging_factor', '1.0'),
('max_horizon_days', '14'),
('overbook_limit_default', '0')
ON CONFLICT (key) DO NOTHING;
