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

	// Migrate schemas for these entities first to ensure tables exist
	err := db.AutoMigrate(&entity.Hospital{}, &entity.Department{}, &entity.HospitalDepartment{}, &entity.User{}, &entity.Session{}, &entity.Referral{}, &entity.ReferralStatusHistory{}, &entity.AuditLog{})
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

	// 1. System/MoH Global Users
	for _, u := range systemTestUsers {
		u.PasswordHash = defaultHash
		db.WithContext(ctx).Where("email = ?", u.Email).FirstOrCreate(&u)
	}

	// 2. Map Primary and Specialized Templates
	var primaryHospitals, specializedHospitals []entity.Hospital
	db.WithContext(ctx).Where("tier_level = ?", entity.PrimaryHosp).Find(&primaryHospitals)
	db.WithContext(ctx).Where("tier_level = ?", entity.SpecializedHosp).Find(&specializedHospitals)

	var cardiologyDept entity.Department
	db.WithContext(ctx).Where("name = ?", "Cardiology").First(&cardiologyDept)

	// Seed primary users
	if len(primaryHospitals) > 0 {
		priHosp := primaryHospitals[0]
		for _, tmpl := range primaryHospUsers {
			user := entity.User{
				NationalID:   tmpl.NationalID,
				Email:        tmpl.Email,
				FirstName:    tmpl.FirstName,
				LastName:     tmpl.LastName,
				Role:         tmpl.Role,
				HospitalID:   &priHosp.ID,
				PasswordHash: defaultHash,
			}
			db.WithContext(ctx).Where("email = ?", user.Email).FirstOrCreate(&user)
		}
	}

	// Seed specialized users
	if len(specializedHospitals) > 0 {
		specHosp := specializedHospitals[0]

		var specCardioDept entity.HospitalDepartment
		db.WithContext(ctx).Where("hospital_id = ? AND department_id = ?", specHosp.ID, cardiologyDept.ID).First(&specCardioDept)

		for _, tmpl := range specializedHospUsers {
			user := entity.User{
				NationalID:   tmpl.NationalID,
				Email:        tmpl.Email,
				FirstName:    tmpl.FirstName,
				LastName:     tmpl.LastName,
				Role:         tmpl.Role,
				HospitalID:   &specHosp.ID,
				PasswordHash: defaultHash,
			}
			
			if tmpl.DeptName == "Cardiology" && specCardioDept.ID != uuid.Nil {
				user.DepartmentID = &specCardioDept.ID
			}

			db.WithContext(ctx).Where("email = ?", user.Email).FirstOrCreate(&user)
		}
	}

	return nil
}
