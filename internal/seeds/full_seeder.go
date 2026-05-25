package seeds

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/crypto"
)

func generateHash(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash)
}

var defaultHash = "" // Will be initialized in SeedAll

func SeedAll(db *gorm.DB) error {
	log.Println("Starting full database seeding...")
	ctx := context.Background()
	defaultHash = generateHash("password123")

	// 0. Clean up legacy UUID mismatches and AutoMigrate
	db.Exec("DROP TABLE IF EXISTS referral_outcomes CASCADE;")
	db.Exec("DROP TABLE IF EXISTS clinical_updates CASCADE;")
	db.Exec("DROP TABLE IF EXISTS scheduler_checkpoints CASCADE;")
	db.Exec("ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_department;")
	db.Exec("UPDATE users SET department_id = NULL;")

	err := db.AutoMigrate(
		&entity.Hospital{},
		&entity.Department{},
		&entity.HospitalDepartment{},
		&entity.User{},
		&entity.Session{},
		&entity.Patient{},
		&entity.ICDCode{},
		&entity.Referral{},
		&entity.ReferralForm{},
		&entity.ReferralDiagnosis{},
		&entity.ReferralNetwork{},
		&entity.Vital{},
		&entity.ReferralEmergencyDetail{},
		&entity.ReferralStatusHistory{},
		&entity.StaffReplacementLog{},
		&entity.Attachment{},
		&entity.AuditLog{},
		&entity.ReferralOutcome{},
		&entity.ClinicalUpdate{},
		&entity.TriageQueue{},
		&entity.DailySchedule{},
		&entity.MLPrediction{},
		&entity.Notification{},
		&entity.ReferralAccess{},
		&entity.CapacityOverride{},
		&entity.SystemConfig{},
		&entity.SchedulerCheckpoint{},
		&entity.JobCheckpoint{},
		&entity.ReferralRedirection{},
		&entity.Conversation{},
		&entity.ConversationParticipant{},
		&entity.ChatMessage{},
	)
	if err != nil {
		return err
	}

	// 1. Truncate all tables in reverse dependency order
	tables := []string{
		"chat_messages", "conversation_participants", "conversations",
		"audit_logs", "notifications", "referral_accesses", "referral_redirections", "referral_outcomes",
		"clinical_updates", "ml_predictions", "triage_queues", "daily_schedules", "capacity_overrides",
		"referral_diagnoses", "referral_status_histories", "vitals", "referral_emergency_details",
		"referral_forms", "attachments", "referrals", "patients", "scheduler_checkpoints", "job_checkpoints", "system_configs",
		"hospital_departments", "departments", "referral_networks", "sessions", "users", "hospitals",
		"icd_codes", "staff_replacement_logs", "in_app_notifications",
	}

	for _, table := range tables {
		if err := db.Exec("TRUNCATE TABLE " + table + " CASCADE").Error; err != nil {
			log.Printf("Warning: failed to truncate %s: %v", table, err)
		}
	}

	// 2. Seed SystemConfigs
	if err := seedSystemConfigs(ctx, db); err != nil {
		return err
	}

	// 3. Seed Hospitals
	if err := seedHospitals(ctx, db); err != nil {
		return err
	}

	// 4. Seed Departments
	if err := seedDepartments(ctx, db); err != nil {
		return err
	}

	// 5. Seed HospitalDepartments and DailySchedules
	if err := seedHospitalDepartmentsAndSchedules(ctx, db); err != nil {
		return err
	}

	// 6. Seed Users
	if err := seedUsers(ctx, db); err != nil {
		return err
	}

	// 7. Seed Patients
	if err := seedPatients(ctx, db); err != nil {
		return err
	}

	// 8. Seed ICD Codes
	if err := seedICDCodes(ctx, db); err != nil {
		return err
	}

	// 9. Seed Networks
	if err := seedNetworks(ctx, db); err != nil {
		return err
	}

	log.Println("Full database seeding completed successfully.")
	return nil
}

func seedSystemConfigs(ctx context.Context, db *gorm.DB) error {
	configs := []entity.SystemConfig{
		{Key: "buffer_days", Value: "2"},
		{Key: "aging_factor", Value: "1.0"},
		{Key: "max_horizon_days", Value: "14"},
		{Key: "overbook_limit_default", Value: "0"},
		{Key: "auto_notify", Value: "false"},
		{Key: "last_waiting_weight_update", Value: ""},
		{Key: "enable_cron_jobs", Value: "false"},
		{Key: "sms_otp_enabled", Value: "false"},
		{Key: "mfa_enabled", Value: "false"},
		{Key: "mfa_sms_fallback_email", Value: "true"},
		{Key: "mfa_otp_ttl_seconds", Value: "300"},
		{Key: "mfa_otp_max_attempts", Value: "5"},
		{Key: "mfa_otp_resend_cooldown_seconds", Value: "60"},
	}

	for _, cfg := range configs {
		if err := db.WithContext(ctx).Create(&cfg).Error; err != nil {
			return err
		}
	}
	
	// Seed job_checkpoints
	db.Exec("INSERT INTO job_checkpoints (job_name, last_run_at) VALUES ('sms_processing', NULL), ('missed_appointments', NULL) ON CONFLICT (job_name) DO NOTHING")
	
	return nil
}

func ptrStr(s string) *string { return &s }

func ptrRegion(r entity.EthiopianRegion) *entity.EthiopianRegion { return &r }

func seedHospitals(ctx context.Context, db *gorm.DB) error {
	hospitals := []entity.Hospital{
		{
			ID:           uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			Name:         "Tikur Anbessa Specialized Hospital",
			TierLevel:    entity.TertiaryHosp,
			Region:       entity.RegionAddisAbaba,
			Address:      ptrStr("Zambia St, Addis Ababa, Ethiopia"),
			ContactPhone: ptrStr("+251 11 551 1211"),
		},
		{
			ID:           uuid.MustParse("a2000000-0000-0000-0000-000000000002"),
			Name:         "St. Paul's Hospital Millennium Medical College",
			TierLevel:    entity.SpecializedHosp,
			Region:       entity.RegionAddisAbaba,
			Address:      ptrStr("Swaziland St, Addis Ababa, Ethiopia"),
			ContactPhone: ptrStr("+251 11 275 0122"),
		},
		{
			ID:           uuid.MustParse("a3000000-0000-0000-0000-000000000003"),
			Name:         "Black Lion Hospital",
			TierLevel:    entity.TertiaryHosp,
			Region:       entity.RegionAddisAbaba,
			Address:      ptrStr("Addis Ababa, Ethiopia"),
			ContactPhone: ptrStr("+251 11 111 1111"),
		},
		{ID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), Name: "Yekatit 12 Hospital", TierLevel: entity.SecondaryHosp, Region: entity.RegionAddisAbaba, Address: ptrStr("Addis Ababa, Ethiopia"), ContactPhone: ptrStr("+251 11 123 4567")},
		{ID: uuid.MustParse("a4000000-0000-0000-0000-000000000004"), Name: "Adama Primary Hospital", TierLevel: entity.PrimaryHosp, Region: entity.RegionOromia, Address: ptrStr("Adama, Ethiopia")},
		{ID: uuid.MustParse("a5000000-0000-0000-0000-000000000005"), Name: "Jimma Primary Clinic", TierLevel: entity.PrimaryHosp, Region: entity.RegionOromia, Address: ptrStr("Jimma, Ethiopia")},
		{ID: uuid.MustParse("a6000000-0000-0000-0000-000000000006"), Name: "Mekelle Health Center", TierLevel: entity.PrimaryHosp, Region: entity.RegionTigray, Address: ptrStr("Mekelle, Ethiopia")},
		{ID: uuid.MustParse("a7000000-0000-0000-0000-000000000007"), Name: "Hawassa Primary Hospital", TierLevel: entity.PrimaryHosp, Region: entity.RegionSidama, Address: ptrStr("Hawassa, Ethiopia")},
		{ID: uuid.MustParse("a8000000-0000-0000-0000-000000000008"), Name: "Dire Dawa Health Station", TierLevel: entity.PrimaryHosp, Region: entity.RegionDireDawa, Address: ptrStr("Dire Dawa, Ethiopia")},
	}

	for _, h := range hospitals {
		if err := db.WithContext(ctx).Create(&h).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDepartments(ctx context.Context, db *gorm.DB) error {
	departments := []entity.Department{
		{ID: uuid.MustParse("b1000000-0000-0000-0000-000000000001"), Name: "Cardiology", Description: ptrStr("Diagnosis and treatment of heart and vascular conditions")},
		{ID: uuid.MustParse("b2000000-0000-0000-0000-000000000002"), Name: "Neurology", Description: ptrStr("Care for brain, spinal cord, and nervous system disorders")},
		{ID: uuid.MustParse("b3000000-0000-0000-0000-000000000003"), Name: "Orthopedics", Description: ptrStr("Treatment of musculoskeletal system issues")},
		{ID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), Name: "Internal Medicine", Description: ptrStr("General internal medicine")},
		{ID: uuid.MustParse("b5000000-0000-0000-0000-000000000005"), Name: "Pediatrics", Description: ptrStr("Medical care for infants, children, and adolescents")},
		{ID: uuid.MustParse("b6000000-0000-0000-0000-000000000006"), Name: "General Surgery", Description: ptrStr("Surgical procedures for a wide range of conditions")},
		{ID: uuid.MustParse("b7000000-0000-0000-0000-000000000007"), Name: "Obstetrics and Gynecology", Description: ptrStr("Women’s reproductive health and childbirth")},
		{ID: uuid.MustParse("b8000000-0000-0000-0000-000000000008"), Name: "Oncology", Description: ptrStr("Diagnosis and treatment of cancer")},
		{ID: uuid.MustParse("b9000000-0000-0000-0000-000000000009"), Name: "Ophthalmology", Description: ptrStr("Eye care and vision services")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000010"), Name: "Dermatology", Description: ptrStr("Care for skin, hair, and nail conditions")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000011"), Name: "Emergency Medicine", Description: ptrStr("Acute care for trauma and illnesses")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000012"), Name: "Emergency Department (ED)", Description: ptrStr("Immediate care for acute illnesses and injuries")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000013"), Name: "Radiology", Description: ptrStr("Imaging services for diagnosis and treatment")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000014"), Name: "Pathology", Description: ptrStr("Laboratory analysis of body tissues and fluids")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000015"), Name: "Urology", Description: ptrStr("Treatment of urinary and male reproductive systems")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000016"), Name: "Gastroenterology", Description: ptrStr("Treatment of digestive system disorders")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000017"), Name: "Nephrology", Description: ptrStr("Care for kidney-related conditions")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000018"), Name: "Pulmonology", Description: ptrStr("Treatment of lung and respiratory tract disorders")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000019"), Name: "Psychiatry", Description: ptrStr("Mental health care and treatment")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000020"), Name: "Endocrinology", Description: ptrStr("Treatment of hormonal and metabolic disorders")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000021"), Name: "Rheumatology", Description: ptrStr("Care for autoimmune and inflammatory diseases")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000022"), Name: "Anesthesiology", Description: ptrStr("Pain management and anesthesia for surgeries")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000023"), Name: "Intensive Care Unit (ICU)", Description: ptrStr("Critical care for severely ill or injured patients")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000024"), Name: "Infectious Diseases", Description: ptrStr("Treatment of infections and contagious diseases")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000025"), Name: "ENT (Otorhinolaryngology)", Description: ptrStr("Care for ear, nose, and throat conditions")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000026"), Name: "Hematology", Description: ptrStr("Treatment of blood disorders")},
		{ID: uuid.MustParse("b0000000-0000-0000-0000-000000000027"), Name: "Physical Medicine and Rehab", Description: ptrStr("Rehabilitation and physical therapy services")},
	}

	for _, d := range departments {
		if err := db.WithContext(ctx).Save(&d).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedHospitalDepartmentsAndSchedules(ctx context.Context, db *gorm.DB) error {
	// Under the Schedule-on-Demand model we DO NOT pre-create DailySchedule
	// rows. Each row is inserted lazily on the first booking for a given
	// (hospital, department, date). We only seed the HospitalDepartment
	// rows (with defaults for the new OverbookLimit and MaxCapacityOfStaff
	// columns) and the SchedulerCheckpoint rows.
	defaultOverbook := 2
	defaultStaffCap := 10

	mappings := []entity.HospitalDepartment{
		// TA (Tertiary) - Cardiology, Neurology, General Surgery, Oncology
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b1000000-0000-0000-0000-000000000001"), StandardDailyLimit: 20, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b2000000-0000-0000-0000-000000000002"), StandardDailyLimit: 20, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000003"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b6000000-0000-0000-0000-000000000006"), StandardDailyLimit: 15, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000004"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b8000000-0000-0000-0000-000000000008"), StandardDailyLimit: 10, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000005"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b3000000-0000-0000-0000-000000000003"), StandardDailyLimit: 15, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		// St. Paul's (Specialized) - Orthopedics, Internal Med, OB/GYN
		{ID: uuid.MustParse("c2000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), DepartmentID: uuid.MustParse("b3000000-0000-0000-0000-000000000003"), StandardDailyLimit: 20, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c2000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 22, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c2000000-0000-0000-0000-000000000003"), HospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), DepartmentID: uuid.MustParse("b7000000-0000-0000-0000-000000000007"), StandardDailyLimit: 18, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		// Black Lion (Tertiary) - Pediatrics, Cardiology, Oncology
		{ID: uuid.MustParse("c3000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), DepartmentID: uuid.MustParse("b5000000-0000-0000-0000-000000000005"), StandardDailyLimit: 20, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c3000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), DepartmentID: uuid.MustParse("b1000000-0000-0000-0000-000000000001"), StandardDailyLimit: 18, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c3000000-0000-0000-0000-000000000003"), HospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), DepartmentID: uuid.MustParse("b8000000-0000-0000-0000-000000000008"), StandardDailyLimit: 12, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		// Yekatit 12 (Secondary) - Internal Med, General Surgery, Emergency
		{ID: uuid.MustParse("c9000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 30, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c9000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), DepartmentID: uuid.MustParse("b6000000-0000-0000-0000-000000000006"), StandardDailyLimit: 20, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c9000000-0000-0000-0000-000000000003"), HospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), DepartmentID: uuid.MustParse("b0000000-0000-0000-0000-000000000011"), StandardDailyLimit: 25, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		// Primary hospitals - Internal Medicine only
		{ID: uuid.MustParse("c4000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a4000000-0000-0000-0000-000000000004"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 10, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c5000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a5000000-0000-0000-0000-000000000005"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 10, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c6000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a6000000-0000-0000-0000-000000000006"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 10, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c7000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a7000000-0000-0000-0000-000000000007"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 10, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
		{ID: uuid.MustParse("c8000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a8000000-0000-0000-0000-000000000008"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 10, OverbookLimit: defaultOverbook, MaxCapacityOfStaff: defaultStaffCap},
	}

	for _, m := range mappings {
		if err := db.WithContext(ctx).Create(&m).Error; err != nil {
			return err
		}
	}

	var hdList []entity.HospitalDepartment
	db.Find(&hdList)
	for _, hd := range hdList {
		db.FirstOrCreate(&entity.SchedulerCheckpoint{
			HospitalID:   hd.HospitalID,
			DepartmentID: hd.DepartmentID,
		}, entity.SchedulerCheckpoint{HospitalID: hd.HospitalID, DepartmentID: hd.DepartmentID})
	}
	return nil
}

func seedUsers(ctx context.Context, db *gorm.DB) error {
	hosp1 := uuid.MustParse("a1000000-0000-0000-0000-000000000001") // Tikur Anbessa
	hosp2 := uuid.MustParse("a2000000-0000-0000-0000-000000000002") // St. Paul's
	hosp3 := uuid.MustParse("a3000000-0000-0000-0000-000000000003") // Black Lion
	hosp9 := uuid.MustParse("a9000000-0000-0000-0000-000000000009") // Yekatit 12

	deptCardio := uuid.MustParse("b1000000-0000-0000-0000-000000000001")
	deptOrtho := uuid.MustParse("b3000000-0000-0000-0000-000000000003")
	deptPeds := uuid.MustParse("b5000000-0000-0000-0000-000000000005")
	deptInternal := uuid.MustParse("b4000000-0000-0000-0000-000000000004")

	users := []entity.User{
		{ID: uuid.MustParse("d0000000-0000-0000-0000-000000000001"), NationalID: "SYS-001", Email: "superadmin@moh.gov.et", FirstName: "System", LastName: "Super Admin", Role: entity.RoleSystemSuperAdmin, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d0000000-0000-0000-0000-000000000002"), NationalID: "MOH-001", Email: "analyst@moh.gov.et", FirstName: "MoH", LastName: "Analyst", Role: entity.RoleMohAnalyst, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},

		// Tikur Anbessa
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000001"), NationalID: "DOC-TA-001", Email: "doctor.ta@hospital.et", FirstName: "Alemayehu", LastName: "Doctor", Role: entity.RoleReferringDoctor, HospitalID: &hosp1, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000002"), NationalID: "LIA-TA-001", Email: "liaison.ta@hospital.et", FirstName: "Sara", LastName: "Liaison", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp1, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000003"), NationalID: "SPEC-TA-001", Email: "specialist.ta@hospital.et", FirstName: "Yohannes", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000004"), NationalID: "REC-TA-001", Email: "reception.ta@hospital.et", FirstName: "Aster", LastName: "Receptionist", Role: entity.RoleReceptionist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000005"), NationalID: "HEAD-TA-001", Email: "depthead.ta@hospital.et", FirstName: "Genet", LastName: "Dept Head", Role: entity.RoleDeptHead, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		
		// README Accounts mapped to TA
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000006"), NationalID: "DOC-READ-001", Email: "doc.primary@hospital.et", FirstName: "Primary", LastName: "Doc", Role: entity.RoleReferringDoctor, HospitalID: &hosp1, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000007"), NationalID: "SPEC-READ-001", Email: "specialist.cardio@hospital.et", FirstName: "Cardio", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000008"), NationalID: "REC-READ-001", Email: "reception.primary@hospital.et", FirstName: "Primary", LastName: "Reception", Role: entity.RoleReceptionist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000009"), NationalID: "LIA-READ-001", Email: "liaison@moh.gov.et", FirstName: "MoH", LastName: "Liaison", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp1, PasswordHash: defaultHash},

		// St. Paul's
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000001"), NationalID: "DOC-SP-001", Email: "doctor.sp@hospital.et", FirstName: "Tesfaye", LastName: "Doctor", Role: entity.RoleReferringDoctor, HospitalID: &hosp2, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000002"), NationalID: "LIA-SP-001", Email: "liaison.sp@hospital.et", FirstName: "Liaison", LastName: "SP", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp2, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000003"), NationalID: "SPEC-SP-001", Email: "specialist.sp@hospital.et", FirstName: "Kidist", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp2, DepartmentID: &deptOrtho, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000004"), NationalID: "REC-SP-001", Email: "reception.sp@hospital.et", FirstName: "Etagegn", LastName: "Receptionist", Role: entity.RoleReceptionist, HospitalID: &hosp2, DepartmentID: &deptOrtho, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000005"), NationalID: "HEAD-SP-001", Email: "depthead.sp@hospital.et", FirstName: "Henok", LastName: "Dept Head", Role: entity.RoleDeptHead, HospitalID: &hosp2, DepartmentID: &deptOrtho, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000006"), NationalID: "ADMIN-SP-001", Email: "admin.specialized@hospital.et", FirstName: "Hospital", LastName: "Admin", Role: entity.RoleHospitalAdmin, HospitalID: &hosp2, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},

		// Black Lion
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000001"), NationalID: "DOC-BL-001", Email: "doctor.bl@hospital.et", FirstName: "Doctor", LastName: "BL", Role: entity.RoleReferringDoctor, HospitalID: &hosp3, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000002"), NationalID: "LIA-BL-001", Email: "liaison.bl@hospital.et", FirstName: "Liaison", LastName: "BL", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp3, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000003"), NationalID: "SPEC-BL-001", Email: "specialist.bl@hospital.et", FirstName: "Martha", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp3, DepartmentID: &deptPeds, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000004"), NationalID: "REC-BL-001", Email: "reception.bl@hospital.et", FirstName: "Frehiwot", LastName: "Receptionist", Role: entity.RoleReceptionist, HospitalID: &hosp3, DepartmentID: &deptPeds, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000005"), NationalID: "HEAD-BL-001", Email: "depthead.bl@hospital.et", FirstName: "Head", LastName: "BL", Role: entity.RoleDeptHead, HospitalID: &hosp3, DepartmentID: &deptPeds, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000006"), NationalID: "ADMIN-BL-001", Email: "admin.bl@hospital.et", FirstName: "Hospital", LastName: "AdminBL", Role: entity.RoleHospitalAdmin, HospitalID: &hosp3, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},

		// TA Hospital Admin
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000010"), NationalID: "ADMIN-TA-001", Email: "admin.ta@hospital.et", FirstName: "Hospital", LastName: "AdminTA", Role: entity.RoleHospitalAdmin, HospitalID: &hosp1, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},

		// Yekatit 12 (Secondary)
		{ID: uuid.MustParse("d9000000-0000-0000-0000-000000000001"), NationalID: "ADMIN-Y12-001", Email: "admin.secondary@hospital.et", FirstName: "Hospital", LastName: "AdminY12", Role: entity.RoleHospitalAdmin, HospitalID: &hosp9, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d9000000-0000-0000-0000-000000000002"), NationalID: "LIA-Y12-001", Email: "liaison.y12@hospital.et", FirstName: "Liaison", LastName: "Y12", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp9, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("d9000000-0000-0000-0000-000000000003"), NationalID: "SPEC-Y12-001", Email: "specialist.y12@hospital.et", FirstName: "Specialist", LastName: "Y12", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp9, DepartmentID: &deptInternal, PasswordHash: defaultHash, Region: ptrRegion(entity.RegionAddisAbaba)},
	}

	for _, u := range users {
		if err := db.WithContext(ctx).Create(&u).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedPatients(ctx context.Context, db *gorm.DB) error {
	phone := "+251911234567"
	dob1, _ := time.Parse("2006-01-02", "1985-01-01")

	// Initialize Crypto Service
	aesKey := os.Getenv("PATIENT_AES_KEY")
	hmacKey := os.Getenv("PATIENT_HMAC_KEY")

	if aesKey == "" {
		aesKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	}
	if hmacKey == "" {
		hmacKey = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
	}

	cryptoSvc, err := crypto.NewPatientCryptoService(aesKey, hmacKey)
	if err != nil {
		return err
	}

	// Helper to encrypt
	encryptStr := func(s string) string {
		enc, _ := cryptoSvc.Encrypt([]byte(s))
		return enc
	}
	encryptPhone := func(s string) *string {
		norm, _ := crypto.NormalizePhone(s)
		enc, _ := cryptoSvc.Encrypt([]byte(norm))
		return &enc
	}
	hashPhone := func(s string) *string {
		norm, _ := crypto.NormalizePhone(s)
		h := cryptoSvc.GenerateHMAC(norm)
		return &h
	}
	encryptStrPtr := func(s string) *string {
		enc, _ := cryptoSvc.Encrypt([]byte(s))
		return &enc
	}
	hashStr := func(s string) *string {
		h := cryptoSvc.GenerateHMAC(s)
		return &h
	}

	patients := []entity.Patient{
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000001"), FirstNameEnc: encryptStr("Abebe"), MiddleNameEnc: encryptStr("Kebede"), LastNameEnc: encryptStr("Balcha"), Sex: "male", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-001"), NationalIDHash: hashStr("NAT-SEED-001"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000002"), FirstNameEnc: encryptStr("Meseret"), MiddleNameEnc: encryptStr("Tesfaye"), LastNameEnc: encryptStr("Gebre"), Sex: "female", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-002"), NationalIDHash: hashStr("NAT-SEED-002"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000003"), FirstNameEnc: encryptStr("Dawit"), MiddleNameEnc: encryptStr("Haile"), LastNameEnc: encryptStr("Mengistu"), Sex: "male", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-003"), NationalIDHash: hashStr("NAT-SEED-003"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000004"), FirstNameEnc: encryptStr("Tigist"), MiddleNameEnc: encryptStr("Belay"), LastNameEnc: encryptStr("Negash"), Sex: "female", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-004"), NationalIDHash: hashStr("NAT-SEED-004"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionOromia)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000005"), FirstNameEnc: encryptStr("Bereket"), MiddleNameEnc: encryptStr("Alemayehu"), LastNameEnc: encryptStr("Tekle"), Sex: "male", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-005"), NationalIDHash: hashStr("NAT-SEED-005"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionOromia)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000006"), FirstNameEnc: encryptStr("Hiwot"), MiddleNameEnc: encryptStr("Mekonnen"), LastNameEnc: encryptStr("Asrat"), Sex: "female", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-006"), NationalIDHash: hashStr("NAT-SEED-006"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionTigray)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000007"), FirstNameEnc: encryptStr("Yonas"), MiddleNameEnc: encryptStr("Girma"), LastNameEnc: encryptStr("Tadesse"), Sex: "male", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-007"), NationalIDHash: hashStr("NAT-SEED-007"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionSidama)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000008"), FirstNameEnc: encryptStr("Meron"), MiddleNameEnc: encryptStr("Dereje"), LastNameEnc: encryptStr("Worku"), Sex: "female", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-008"), NationalIDHash: hashStr("NAT-SEED-008"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionDireDawa)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000009"), FirstNameEnc: encryptStr("Henok"), MiddleNameEnc: encryptStr("Teshome"), LastNameEnc: encryptStr("Abate"), Sex: "male", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-009"), NationalIDHash: hashStr("NAT-SEED-009"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionAddisAbaba)},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000010"), FirstNameEnc: encryptStr("Selam"), MiddleNameEnc: encryptStr("Yohannes"), LastNameEnc: encryptStr("Fikre"), Sex: "female", DateOfBirth: &dob1, PhoneNumberEnc: encryptPhone(phone), PhoneHash: hashPhone(phone), NationalIDEnc: encryptStrPtr("NAT-SEED-010"), NationalIDHash: hashStr("NAT-SEED-010"), AllowSMS: true, HomeRegion: ptrRegion(entity.RegionAddisAbaba)},
	}

	for _, p := range patients {
		if err := db.WithContext(ctx).Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedICDCodes(ctx context.Context, db *gorm.DB) error {
	var icdCount int64
	db.WithContext(ctx).Model(&entity.ICDCode{}).Count(&icdCount)
	if icdCount == 0 {
		log.Println("WARNING: ICD-10 codes not loaded. Run 'go run scripts/load_icd10.go' to populate the table.")
	}
	return nil
}

func seedNetworks(ctx context.Context, db *gorm.DB) error {
	networks := []entity.ReferralNetwork{
		// PRIMARY (a4-a8) → SECONDARY (a9) only
		{SenderHospitalID: uuid.MustParse("a4000000-0000-0000-0000-000000000004"), ReceiverHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a5000000-0000-0000-0000-000000000005"), ReceiverHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a6000000-0000-0000-0000-000000000006"), ReceiverHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a7000000-0000-0000-0000-000000000007"), ReceiverHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a8000000-0000-0000-0000-000000000008"), ReceiverHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReferralType: "routine"},
		// SECONDARY (a9) → SPECIALIZED/TERTIARY
		{SenderHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReceiverHospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReceiverHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a9000000-0000-0000-0000-000000000009"), ReceiverHospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), ReferralType: "routine"},
		// SPECIALIZED (a2) → TERTIARY (a1, a3)
		{SenderHospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), ReceiverHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), ReceiverHospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), ReferralType: "routine"},
		// TERTIARY ↔ TERTIARY
		{SenderHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReceiverHospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), ReceiverHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReferralType: "routine"},
		{SenderHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReceiverHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReferralType: "routine"},
		// TERTIARY → SPECIALIZED
		{SenderHospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), ReceiverHospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), ReferralType: "routine"},
	}

	for _, n := range networks {
		if err := db.WithContext(ctx).Create(&n).Error; err != nil {
			return err
		}
	}
	return nil
}
