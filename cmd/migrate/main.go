package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/domain/entity"
)

func main() {
	cfg := config.LoadConfig()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Africa/Addis_Ababa",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database for migration...")

	// 1. AutoMigrate all entities
	// This will add new columns and missing indexes automatically.
	err = db.AutoMigrate(
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
		log.Fatalf("AutoMigrate failed: %v", err)
	}
	log.Println("AutoMigrate completed.")

	// 2. Manual Column Renames / Data Fixes
	// If any specific renames were requested or identified, we handle them here.
	// For example, if 'middle_name' was previously 'mname', we would use:
	// db.Migrator().RenameColumn(&entity.User{}, "mname", "middle_name")

	// Ensure 'middle_name' has default empty string for existing records if not caught by AutoMigrate
	db.Model(&entity.User{}).Where("middle_name IS NULL").Update("middle_name", "")

	// Fix Referral statuses if they were renamed (e.g. DRAFT to something else)
	// Currently we use standard ENUM-like strings, so no logic needed unless they changed.

	log.Println("Dynamic column checks completed.")

	// 3. Ensure critical indexes for new filtering logic
	// AutoMigrate handles many GORM tags, but we can be explicit here for complex composite indexes.
	log.Println("Migration successful!")
}
