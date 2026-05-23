-- Migration v12: Seed and update 25 critical departments in the database

-- Obstetrics and Gynecology (renamed from Obstetrics & Gynecology)
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b7000000-0000-0000-0000-000000000007', 'Obstetrics and Gynecology', 'Women’s reproductive health and childbirth', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Emergency Department (ED)
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000012', 'Emergency Department (ED)', 'Immediate care for acute illnesses and injuries', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Cardiology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b1000000-0000-0000-0000-000000000001', 'Cardiology', 'Diagnosis and treatment of heart and vascular conditions', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Neurology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b2000000-0000-0000-0000-000000000002', 'Neurology', 'Care for brain, spinal cord, and nervous system disorders', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Pediatrics
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b5000000-0000-0000-0000-000000000005', 'Pediatrics', 'Medical care for infants, children, and adolescents', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Oncology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b8000000-0000-0000-0000-000000000008', 'Oncology', 'Diagnosis and treatment of cancer', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Orthopedics
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b3000000-0000-0000-0000-000000000003', 'Orthopedics', 'Treatment of musculoskeletal system issues', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Radiology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000013', 'Radiology', 'Imaging services for diagnosis and treatment', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Pathology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000014', 'Pathology', 'Laboratory analysis of body tissues and fluids', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- General Surgery
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b6000000-0000-0000-0000-000000000006', 'General Surgery', 'Surgical procedures for a wide range of conditions', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Urology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000015', 'Urology', 'Treatment of urinary and male reproductive systems', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Dermatology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000010', 'Dermatology', 'Care for skin, hair, and nail conditions', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Gastroenterology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000016', 'Gastroenterology', 'Treatment of digestive system disorders', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Nephrology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000017', 'Nephrology', 'Care for kidney-related conditions', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Pulmonology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000018', 'Pulmonology', 'Treatment of lung and respiratory tract disorders', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Psychiatry
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000019', 'Psychiatry', 'Mental health care and treatment', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Endocrinology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000020', 'Endocrinology', 'Treatment of hormonal and metabolic disorders', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Rheumatology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000021', 'Rheumatology', 'Care for autoimmune and inflammatory diseases', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Anesthesiology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000022', 'Anesthesiology', 'Pain management and anesthesia for surgeries', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Intensive Care Unit (ICU)
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000023', 'Intensive Care Unit (ICU)', 'Critical care for severely ill or injured patients', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Infectious Diseases
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000024', 'Infectious Diseases', 'Treatment of infections and contagious diseases', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Ophthalmology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b9000000-0000-0000-0000-000000000009', 'Ophthalmology', 'Eye care and vision services', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- ENT (Otorhinolaryngology)
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000025', 'ENT (Otorhinolaryngology)', 'Care for ear, nose, and throat conditions', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Hematology
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000026', 'Hematology', 'Treatment of blood disorders', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();

-- Physical Medicine and Rehab
INSERT INTO departments (id, name, description, created_at, updated_at)
VALUES ('b0000000-0000-0000-0000-000000000027', 'Physical Medicine and Rehab', 'Rehabilitation and physical therapy services', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();
