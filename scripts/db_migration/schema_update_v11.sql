-- ICD-10-CM High-Performance Search Indexes (v11)

-- Enable trgm extension for fast text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN index for fast substring search on description
CREATE INDEX IF NOT EXISTS idx_icd_codes_description_trgm 
    ON icd_codes USING gin (description gin_trgm_ops);

-- B-tree prefix index for fast left-anchored (prefix) matching on code
CREATE INDEX IF NOT EXISTS idx_icd_codes_code_prefix 
    ON icd_codes (code text_pattern_ops);

-- Index on category for category filter queries
CREATE INDEX IF NOT EXISTS idx_icd_codes_category 
    ON icd_codes (category);
