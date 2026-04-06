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

	// Neon uses DATABASE_URL (postgres:// connection string)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // only show warnings/errors
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("=== Hospital Referral System – DB Sync ===")
	fmt.Println("Step 1: Pre-filling NULL columns to avoid NOT NULL migration errors...")
	preFillNullColumns(db)

	fmt.Println("Step 2: Running AutoMigrate for all modified entities...")
	entities := []interface{}{
		&entity.User{},
		&entity.Referral{},
		&entity.Attachment{},
		&entity.Patient{},
		&entity.Hospital{},
		&entity.Department{},
	}

	for _, e := range entities {
		name := fmt.Sprintf("%T", e)
		fmt.Printf("  Migrating %s ...\n", name)
		if err := db.AutoMigrate(e); err != nil {
			log.Fatalf("AutoMigrate failed for %s: %v", name, err)
		}
		fmt.Printf("  ✓ %s done\n", name)
	}

	fmt.Println("\n=== Database synchronisation complete ✓ ===")
}
