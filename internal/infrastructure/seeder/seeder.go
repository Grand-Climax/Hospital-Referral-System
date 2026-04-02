package seeder

import (
	"context"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

// SeedReferenceData inserts the initial reference data into the database.
func SeedReferenceData(ctx context.Context, db *gorm.DB) error {
	log.Println("Starting reference data seeding...")

	// Clean up legacy UUID mismatches for the new strict foreign key mappings
	db.Exec("ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_department;")
	db.Exec("UPDATE users SET department_id = NULL;")

	// Migrate schemas for these entities first to ensure tables exist
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
		&entity.Attachment{},
		&entity.AuditLog{},
	)
	if err != nil {
		return err
	}

	// 1. Seed Departments
	if err := seedDepartments(ctx, db); err != nil {
		return err
	}

	// 2. Seed Hospitals
	if err := seedHospitals(ctx, db); err != nil {
		return err
	}

	// 3. Seed Hospital Departments mapping
	if err := seedHospitalDepartments(ctx, db); err != nil {
		return err
	}

	// 4. Seed Users (all 7 roles mapped dynamically)
	if err := seedUsers(ctx, db); err != nil {
		return err
	}

	// 5. Seed ICD Codes
	if err := seedICDCodes(ctx, db); err != nil {
		return err
	}

	// 6. Seed Referral Networks (Routing pathways)
	if err := seedNetworks(ctx, db); err != nil {
		return err
	}

	// 7. Seed Patients
	if err := seedPatients(ctx, db); err != nil {
		return err
	}

	// 8. Seed Referrals (Requires all above dependencies)
	if err := seedReferrals(ctx, db); err != nil {
		return err
	}

	log.Println("Seeding completed successfully.")
	return nil
}

func seedDepartments(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding departments...")
	for _, raw := range initialDepartments {
		var existing entity.Department
		if err := db.WithContext(ctx).Where("name = ?", raw.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Insert new
				if err := db.WithContext(ctx).Create(&raw).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}

func seedHospitals(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding hospitals...")
	for _, raw := range initialHospitals {
		var existing entity.Hospital
		if err := db.WithContext(ctx).Where("name = ?", raw.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Insert new
				if err := db.WithContext(ctx).Create(&raw).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}

func seedHospitalDepartments(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding hospital departments...")

	// Just a simple mapping: Give all top tier hospitals all departments
	var allHospitals []entity.Hospital
	db.WithContext(ctx).Find(&allHospitals)

	var allDepartments []entity.Department
	db.WithContext(ctx).Find(&allDepartments)

	for _, h := range allHospitals {
		for _, d := range allDepartments {
			// Some logic to assign standardized limits based on hospital tier
			limit := 20
			if h.TierLevel == entity.TertiaryHosp || h.TierLevel == entity.SpecializedHosp {
				limit = 50
			}

			var existing entity.HospitalDepartment
			if err := db.WithContext(ctx).Where("hospital_id = ? AND department_id = ?", h.ID, d.ID).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					mapping := entity.HospitalDepartment{
						HospitalID:         h.ID,
						DepartmentID:       d.ID,
						StandardDailyLimit: limit,
					}
					if err := db.WithContext(ctx).Create(&mapping).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}
	}
	return nil
}

func seedUsers(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding test users for all roles...")

	// Default pre-hashed password: "password123" (bcrypt cost 12)
	defaultHash := "$2a$12$OV/iqbn3GrwIVdFdR60VIuKoydr0CWosGqgAvivL7H/vnUOQM0Tce"

	// 1. System/MoH Global Users (no hospital)
	for i := range systemTestUsers {
		systemTestUsers[i].PasswordHash = defaultHash
		var user entity.User
		if err := db.WithContext(ctx).Where("email = ?", systemTestUsers[i].Email).FirstOrCreate(&user, systemTestUsers[i]).Error; err != nil {
			log.Printf("Warning: failed to seed system user %s: %v", systemTestUsers[i].Email, err)
		}
	}

			// Cardiology is pinned to a stable UUID in data.go — use it directly
	cardiologyID := uuid.MustParse("dfc2b777-a5d5-424b-911a-976b2e8d8614")

	// helper: creates a user pinned to a hospital, optionally with a department
	seedHospUsers := func(hospName string, templates []hospitalUserTemplate) {
		var hosp entity.Hospital
		if err := db.WithContext(ctx).Where("name = ?", hospName).First(&hosp).Error; err != nil {
			log.Printf("Warning: hospital %q not found, skipping users: %v", hospName, err)
			return
		}

		for _, tmpl := range templates {
			user := entity.User{
				NationalID:   tmpl.NationalID,
				Email:        tmpl.Email,
				FirstName:    tmpl.FirstName,
				LastName:     tmpl.LastName,
				Role:         tmpl.Role,
				HospitalID:   &hosp.ID,
				DepartmentID: tmpl.DepartmentID, // directly from template (*uuid.UUID, nil if not set)
				PasswordHash: defaultHash,
			}
			if err := db.WithContext(ctx).Where("email = ?", user.Email).FirstOrCreate(&user).Error; err != nil {
				log.Printf("Warning: failed to seed user %s: %v", user.Email, err)
			}
		}
	}

	_ = cardiologyID // used via &cardiologyID in data.go templates

	// 2. Bishoftu Primary Hospital (= cd323204-bfb7-4583-88e9-bb5cbed68af0)
	seedHospUsers("Bishoftu Primary Hospital", primaryHospUsers)

	// 3. Adama General Hospital (= 0f74f069-d52d-4482-9ba5-41b007fdc1e5)
	seedHospUsers("Adama General Hospital", generalHospUsers)

	// 4. Jimma University Medical Center (specialized)
	seedHospUsers("Jimma University Medical Center", specializedHospUsers)

	return nil
}
