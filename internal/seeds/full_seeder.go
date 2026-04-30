package seeds

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"

	"Hospital-Referral-System/internal/domain/entity"
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
		&entity.ReferralRedirection{},
	)
	if err != nil {
		return err
	}

	// 1. Truncate all tables in reverse dependency order
	tables := []string{
		"audit_logs", "notifications", "referral_accesses", "referral_redirections", "referral_outcomes",
		"clinical_updates", "ml_predictions", "triage_queues", "daily_schedules", "capacity_overrides",
		"referral_diagnoses", "referral_status_histories", "vitals", "referral_emergency_details",
		"referral_forms", "attachments", "referrals", "patients", "scheduler_checkpoints", "system_configs",
		"hospital_departments", "departments", "referral_networks", "sessions", "users", "hospitals",
		"icd_codes", "staff_replacement_logs",
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
	}

	for _, cfg := range configs {
		if err := db.WithContext(ctx).Create(&cfg).Error; err != nil {
			return err
		}
	}
	return nil
}

func ptrStr(s string) *string { return &s }

func seedHospitals(ctx context.Context, db *gorm.DB) error {
	hospitals := []entity.Hospital{
		{
			ID:           uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			Name:         "Tikur Anbessa Specialized Hospital",
			TierLevel:    entity.TertiaryHosp,
			Region:       "Addis Ababa",
			Address:      ptrStr("Zambia St, Addis Ababa, Ethiopia"),
			ContactPhone: ptrStr("+251 11 551 1211"),
		},
		{
			ID:           uuid.MustParse("a2000000-0000-0000-0000-000000000002"),
			Name:         "St. Paul's Hospital Millennium Medical College",
			TierLevel:    entity.SpecializedHosp,
			Region:       "Addis Ababa",
			Address:      ptrStr("Swaziland St, Addis Ababa, Ethiopia"),
			ContactPhone: ptrStr("+251 11 275 0122"),
		},
		{
			ID:           uuid.MustParse("a3000000-0000-0000-0000-000000000003"),
			Name:         "Black Lion Hospital",
			TierLevel:    entity.TertiaryHosp,
			Region:       "Addis Ababa",
			Address:      ptrStr("Addis Ababa, Ethiopia"),
			ContactPhone: ptrStr("+251 11 111 1111"),
		},
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
		{ID: uuid.MustParse("b1000000-0000-0000-0000-000000000001"), Name: "Cardiology", Description: ptrStr("Heart and blood vessel disorders")},
		{ID: uuid.MustParse("b2000000-0000-0000-0000-000000000002"), Name: "Neurology", Description: ptrStr("Disorders of the nervous system")},
		{ID: uuid.MustParse("b3000000-0000-0000-0000-000000000003"), Name: "Orthopedics", Description: ptrStr("Conditions involving the musculoskeletal system")},
		{ID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), Name: "Internal Medicine", Description: ptrStr("General internal medicine")},
		{ID: uuid.MustParse("b5000000-0000-0000-0000-000000000005"), Name: "Pediatrics", Description: ptrStr("Care of infants, children, and adolescents")},
	}

	for _, d := range departments {
		if err := db.WithContext(ctx).Create(&d).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedHospitalDepartmentsAndSchedules(ctx context.Context, db *gorm.DB) error {
	mappings := []entity.HospitalDepartment{
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b1000000-0000-0000-0000-000000000001"), StandardDailyLimit: 20}, // TA - Cardio
		{ID: uuid.MustParse("c1000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a1000000-0000-0000-0000-000000000001"), DepartmentID: uuid.MustParse("b2000000-0000-0000-0000-000000000002"), StandardDailyLimit: 20}, // TA - Neuro
		{ID: uuid.MustParse("c2000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), DepartmentID: uuid.MustParse("b3000000-0000-0000-0000-000000000003"), StandardDailyLimit: 20}, // SP - Ortho
		{ID: uuid.MustParse("c2000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a2000000-0000-0000-0000-000000000002"), DepartmentID: uuid.MustParse("b4000000-0000-0000-0000-000000000004"), StandardDailyLimit: 22}, // SP - Internal Med
		{ID: uuid.MustParse("c3000000-0000-0000-0000-000000000001"), HospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), DepartmentID: uuid.MustParse("b5000000-0000-0000-0000-000000000005"), StandardDailyLimit: 20}, // BL - Peds
		{ID: uuid.MustParse("c3000000-0000-0000-0000-000000000002"), HospitalID: uuid.MustParse("a3000000-0000-0000-0000-000000000003"), DepartmentID: uuid.MustParse("b1000000-0000-0000-0000-000000000001"), StandardDailyLimit: 18}, // BL - Cardio
	}

	for _, m := range mappings {
		if err := db.WithContext(ctx).Create(&m).Error; err != nil {
			return err
		}

		// Generate 30 days of schedules
		today := time.Now().Truncate(24 * time.Hour)
		for i := 0; i <= 30; i++ {
			scheduleDate := today.AddDate(0, 0, i)
			sched := entity.DailySchedule{
				HospitalID:    m.HospitalID,
				DepartmentID:  m.DepartmentID,
				DeptID:        m.ID, // Link to hospital_departments
				ScheduleDate:  scheduleDate,
				MaxSlots:      m.StandardDailyLimit,
				OverbookLimit: 2,
				BookedSlots:   0,
				Version:       1,
			}
			if err := db.WithContext(ctx).Create(&sched).Error; err != nil {
				return err
			}
		}
	}

	var hdList []entity.HospitalDepartment
	db.Find(&hdList)
	for _, hd := range hdList {
		db.FirstOrCreate(&entity.SchedulerCheckpoint{
			HospitalID: hd.HospitalID,
			DeptID:     hd.DepartmentID, // may be zero if column nullable
		}, entity.SchedulerCheckpoint{HospitalID: hd.HospitalID, DeptID: hd.DepartmentID})
	}
	return nil
}

func seedUsers(ctx context.Context, db *gorm.DB) error {
	hosp1 := uuid.MustParse("a1000000-0000-0000-0000-000000000001") // Tikur Anbessa
	hosp2 := uuid.MustParse("a2000000-0000-0000-0000-000000000002") // St. Paul's
	hosp3 := uuid.MustParse("a3000000-0000-0000-0000-000000000003") // Black Lion

	deptCardio := uuid.MustParse("b1000000-0000-0000-0000-000000000001")
	deptOrtho := uuid.MustParse("b3000000-0000-0000-0000-000000000003")
	deptPeds := uuid.MustParse("b5000000-0000-0000-0000-000000000005")

	users := []entity.User{
		{ID: uuid.MustParse("d0000000-0000-0000-0000-000000000001"), NationalID: "SYS-001", Email: "superadmin@moh.gov.et", FirstName: "System", LastName: "Super Admin", Role: entity.RoleSystemSuperAdmin, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d0000000-0000-0000-0000-000000000002"), NationalID: "MOH-001", Email: "analyst@moh.gov.et", FirstName: "MoH", LastName: "Analyst", Role: entity.RoleMohAnalyst, PasswordHash: defaultHash},

		// Tikur Anbessa
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000001"), NationalID: "DOC-TA-001", Email: "doctor.ta@hospital.et", FirstName: "Alemayehu", LastName: "Doctor", Role: entity.RoleReferringDoctor, HospitalID: &hosp1, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000002"), NationalID: "LIA-TA-001", Email: "liaison.ta@hospital.et", FirstName: "Sara", LastName: "Liaison", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp1, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000003"), NationalID: "SPEC-TA-001", Email: "specialist.ta@hospital.et", FirstName: "Yohannes", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000004"), NationalID: "REC-TA-001", Email: "reception.ta@hospital.et", FirstName: "Aster", LastName: "Receptionist", Role: entity.RoleReceptionist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000005"), NationalID: "HEAD-TA-001", Email: "depthead.ta@hospital.et", FirstName: "Genet", LastName: "Dept Head", Role: entity.RoleDeptHead, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		
		// README Accounts mapped to TA
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000006"), NationalID: "DOC-READ-001", Email: "doc.primary@hospital.et", FirstName: "Primary", LastName: "Doc", Role: entity.RoleReferringDoctor, HospitalID: &hosp1, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000007"), NationalID: "SPEC-READ-001", Email: "specialist.cardio@hospital.et", FirstName: "Cardio", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000008"), NationalID: "REC-READ-001", Email: "reception.primary@hospital.et", FirstName: "Primary", LastName: "Reception", Role: entity.RoleReceptionist, HospitalID: &hosp1, DepartmentID: &deptCardio, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d1000000-0000-0000-0000-000000000009"), NationalID: "LIA-READ-001", Email: "liaison@moh.gov.et", FirstName: "MoH", LastName: "Liaison", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp1, PasswordHash: defaultHash},

		// St. Paul's
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000001"), NationalID: "DOC-SP-001", Email: "doctor.sp@hospital.et", FirstName: "Tesfaye", LastName: "Doctor", Role: entity.RoleReferringDoctor, HospitalID: &hosp2, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000002"), NationalID: "LIA-SP-001", Email: "liaison.sp@hospital.et", FirstName: "Liaison", LastName: "SP", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp2, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000003"), NationalID: "SPEC-SP-001", Email: "specialist.sp@hospital.et", FirstName: "Kidist", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp2, DepartmentID: &deptOrtho, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000004"), NationalID: "REC-SP-001", Email: "reception.sp@hospital.et", FirstName: "Etagegn", LastName: "Receptionist", Role: entity.RoleReceptionist, HospitalID: &hosp2, DepartmentID: &deptOrtho, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000005"), NationalID: "HEAD-SP-001", Email: "depthead.sp@hospital.et", FirstName: "Henok", LastName: "Dept Head", Role: entity.RoleDeptHead, HospitalID: &hosp2, DepartmentID: &deptOrtho, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d2000000-0000-0000-0000-000000000006"), NationalID: "ADMIN-SP-001", Email: "admin.specialized@hospital.et", FirstName: "Hospital", LastName: "Admin", Role: entity.RoleHospitalAdmin, HospitalID: &hosp2, PasswordHash: defaultHash},

		// Black Lion
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000001"), NationalID: "DOC-BL-001", Email: "doctor.bl@hospital.et", FirstName: "Doctor", LastName: "BL", Role: entity.RoleReferringDoctor, HospitalID: &hosp3, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000002"), NationalID: "LIA-BL-001", Email: "liaison.bl@hospital.et", FirstName: "Liaison", LastName: "BL", Role: entity.RoleLiaisonOfficer, HospitalID: &hosp3, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000003"), NationalID: "SPEC-BL-001", Email: "specialist.bl@hospital.et", FirstName: "Martha", LastName: "Specialist", Role: entity.RoleReceivingSpecialist, HospitalID: &hosp3, DepartmentID: &deptPeds, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000004"), NationalID: "REC-BL-001", Email: "reception.bl@hospital.et", FirstName: "Frehiwot", LastName: "Receptionist", Role: entity.RoleReceptionist, HospitalID: &hosp3, DepartmentID: &deptPeds, PasswordHash: defaultHash},
		{ID: uuid.MustParse("d3000000-0000-0000-0000-000000000005"), NationalID: "HEAD-BL-001", Email: "depthead.bl@hospital.et", FirstName: "Head", LastName: "BL", Role: entity.RoleDeptHead, HospitalID: &hosp3, DepartmentID: &deptPeds, PasswordHash: defaultHash},
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

	patients := []entity.Patient{
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000001"), FirstName: "Abebe", MiddleName: "Kebede", LastName: "Balcha", Sex: "male", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000002"), FirstName: "Meseret", MiddleName: "Tesfaye", LastName: "Gebre", Sex: "female", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000003"), FirstName: "Dawit", MiddleName: "Haile", LastName: "Mengistu", Sex: "male", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000004"), FirstName: "Tigist", MiddleName: "Belay", LastName: "Negash", Sex: "female", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000005"), FirstName: "Bereket", MiddleName: "Alemayehu", LastName: "Tekle", Sex: "male", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000006"), FirstName: "Hiwot", MiddleName: "Mekonnen", LastName: "Asrat", Sex: "female", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000007"), FirstName: "Yonas", MiddleName: "Girma", LastName: "Tadesse", Sex: "male", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000008"), FirstName: "Meron", MiddleName: "Dereje", LastName: "Worku", Sex: "female", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000009"), FirstName: "Henok", MiddleName: "Teshome", LastName: "Abate", Sex: "male", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
		{ID: uuid.MustParse("e0000000-0000-0000-0000-000000000010"), FirstName: "Selam", MiddleName: "Yohannes", LastName: "Fikre", Sex: "female", DateOfBirth: &dob1, PhoneNumber: &phone, AllowSMS: true},
	}

	for _, p := range patients {
		if err := db.WithContext(ctx).Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedICDCodes(ctx context.Context, db *gorm.DB) error {
	icds := []entity.ICDCode{
		{Code: "I21.9", Description: "Acute myocardial infarction, unspecified", Category: "Diseases of the circulatory system"},
		{Code: "S06.9X9A", Description: "Unspecified intracranial injury with loss of consciousness of unspecified duration, initial encounter", Category: "Injury, poisoning and certain other consequences of external causes"},
		{Code: "J18.9", Description: "Pneumonia, unspecified organism", Category: "Diseases of the respiratory system"},
	}

	for _, icd := range icds {
		if err := db.WithContext(ctx).Create(&icd).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedNetworks(ctx context.Context, db *gorm.DB) error {
	networks := []entity.ReferralNetwork{
		// Tikur Anbessa <-> Tikur Anbessa (Self)
		{
			SenderHospitalID:      uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			ReceiverHospitalID:    uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		},
		// St. Paul's <-> St. Paul's (Self)
		{
			SenderHospitalID:      uuid.MustParse("a2000000-0000-0000-0000-000000000002"),
			ReceiverHospitalID:    uuid.MustParse("a2000000-0000-0000-0000-000000000002"),
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		},
		// Black Lion <-> Black Lion (Self)
		{
			SenderHospitalID:      uuid.MustParse("a3000000-0000-0000-0000-000000000003"),
			ReceiverHospitalID:    uuid.MustParse("a3000000-0000-0000-0000-000000000003"),
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		},
		// Tikur Anbessa <-> Black Lion
		{
			SenderHospitalID:      uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			ReceiverHospitalID:    uuid.MustParse("a3000000-0000-0000-0000-000000000003"),
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		},
		{
			SenderHospitalID:      uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			ReceiverHospitalID:    uuid.MustParse("a2000000-0000-0000-0000-000000000002"),
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		},
		{
			SenderHospitalID:      uuid.MustParse("a2000000-0000-0000-0000-000000000002"),
			ReceiverHospitalID:    uuid.MustParse("a1000000-0000-0000-0000-000000000001"),
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		},
	}

	for _, n := range networks {
		if err := db.WithContext(ctx).Create(&n).Error; err != nil {
			return err
		}
	}
	return nil
}
