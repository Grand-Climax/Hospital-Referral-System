package main

import (
	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/domain/entity"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// preFillNullColumns runs UPDATE statements to fill NULLs in columns that are about to
// receive a NOT NULL constraint. This is required for Neon (serverless Postgres) and any
// live DB that already has rows — GORM AutoMigrate will fail if existing rows have NULLs.
func preFillNullColumns(db *gorm.DB) {
	type fix struct {
		table  string
		column string
		value  string
	}

	fixes := []fix{
		// users: MiddleName was added as NOT NULL
		{"users", "middle_name", "''"},

		// attachments: new columns added as NOT NULL
		{"attachments", "file_name", "''"},
		{"attachments", "file_type", "''"},
		{"attachments", "storage_path", "''"},
		{"attachments", "category", "'GENERAL_CLINICAL'"},
	}

	for _, f := range fixes {
		if db.Migrator().HasTable(f.table) {
			if db.Migrator().HasColumn(&struct{ _ string }{}, f.column) || true {
				// Add column as nullable first (safe – IF NOT EXISTS is idempotent)
				db.Exec(fmt.Sprintf(
					"ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s TEXT",
					f.table, f.column,
				))
				// Fill any NULLs
				result := db.Exec(fmt.Sprintf(
					"UPDATE %s SET %s = %s WHERE %s IS NULL",
					f.table, f.column, f.value, f.column,
				))
				if result.Error != nil {
					log.Printf("Warning: pre-fill for %s.%s failed: %v", f.table, f.column, result.Error)
				} else {
					fmt.Printf("Pre-filled %d rows in %s.%s\n", result.RowsAffected, f.table, f.column)
				}
			}
		}
	}
}

func main() {
	cfg := config.LoadConfig()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TZ=Africa/Addis_Ababa",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("=== Hospital Referral Hub – DB Sync v2 ===")

	fmt.Println("Step 1: Pre-filling legacy NULL columns...")
	preFillNullColumns(db)

	fmt.Println("Step 2: Executing raw SQL schema update (Enums & Constraints)...")
	sqlFile := "scripts/db_migration/schema_update_v2.sql"
	content, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("Failed to read SQL migration file: %v", err)
	}

	if err := db.Exec(string(content)).Error; err != nil {
		log.Fatalf("Failed to execute SQL migration: %v", err)
	}
	fmt.Println("  ✓ SQL migration successful")

	fmt.Println("Step 3: Running AutoMigrate for all system entities...")
	entities := []interface{}{
		// Master Data
		&entity.Hospital{},
		&entity.Department{},
		&entity.HospitalDepartment{},
		&entity.ICDCode{},
		&entity.ReferralNetwork{},

		// Users & Auth
		&entity.User{},
		&entity.Session{},
		&entity.ReferralAccess{},

		// Patient & Referral Core
		&entity.Patient{},
		&entity.Referral{},
		&entity.ReferralForm{},
		&entity.ReferralDiagnosis{},
		&entity.Vital{},
		&entity.ReferralEmergencyDetail{},
		&entity.Attachment{},

		// Workflow & Lifecycle
		&entity.TriageQueue{},
		&entity.DailySchedule{},
		&entity.CapacityOverride{},
		&entity.ClinicalUpdate{},
		&entity.ReferralOutcome{},
		&entity.ReferralRedirection{},
		&entity.ReferralStatusHistory{},

		// Communication & System
		&entity.Notification{},
		&entity.MLPrediction{},
		&entity.SchedulerCheckpoint{},
		&entity.SystemConfig{},
		&entity.AuditLog{},
		&entity.Conversation{},
		&entity.ConversationParticipant{},
		&entity.ChatMessage{},
	}

	for _, e := range entities {
		name := fmt.Sprintf("%T", e)
		if err := db.AutoMigrate(e); err != nil {
			log.Fatalf("AutoMigrate failed for %s: %v", name, err)
		}
		fmt.Printf("  ✓ %s synced\n", name)
	}

	fmt.Println("\nStep 4: Seeding default system configurations...")
	defaults := []entity.SystemConfig{
		{Key: "buffer_days", Value: "2"},
		{Key: "aging_factor", Value: "1.0"},
		{Key: "max_horizon_days", Value: "14"},
		{Key: "overbook_limit_default", Value: "0"},
	}
	for _, cfg := range defaults {
		if err := db.FirstOrCreate(&entity.SystemConfig{}, entity.SystemConfig{Key: cfg.Key}).Error; err != nil {
			log.Printf("Warning: failed to seed config %s: %v", cfg.Key, err)
		} else {
			// Ensure value is set even if record existed (optional, but good for defaults)
			db.Model(&entity.SystemConfig{}).Where("key = ?", cfg.Key).Update("value", cfg.Value)
		}
	}
	fmt.Println("  ✓ System configurations seeded")

	fmt.Println("\n=== Database synchronisation complete ✓ ===")
}
