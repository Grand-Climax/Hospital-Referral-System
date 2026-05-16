-- Phase 1: Regional Support & Schema Hardening

-- 1. Add region column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS region VARCHAR(50);

-- 2. Enforce unique constraints on patient hashes
-- We use UNIQUE INDEX to allow for conditional uniqueness (e.g. if we wanted to exclude deleted records)
-- but for Phase 1 we will stick to standard UNIQUE constraints for simplicity and strict enforcement.

-- First, ensure any existing non-unique indexes are replaced or augmented
-- National ID Hash uniqueness
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_patients_national_id_hash') THEN
        ALTER TABLE patients ADD CONSTRAINT uni_patients_national_id_hash UNIQUE (national_id_hash);
    END IF;
END $$;

-- Phone Hash uniqueness
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_patients_phone_hash') THEN
        ALTER TABLE patients ADD CONSTRAINT uni_patients_phone_hash UNIQUE (phone_hash);
    END IF;
END $$;
