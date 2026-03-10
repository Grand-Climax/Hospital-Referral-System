//go:build ignore

package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

// This script adds the `resource` and `resource_id` columns to the audit_logs table
// and re-migrates any other entities that have a schema delta vs the current DB.
func main() {
	_ = godotenv.Load(".env.local")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=6628 dbname=referral port=5432 sslmode=disable TimeZone=Africa/Addis_Ababa"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	log.Println("Running targeted AutoMigrate for audit_logs schema delta...")
	if err := db.AutoMigrate(&entity.AuditLog{}); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migration complete. `resource` and `resource_id` columns are now in audit_logs.")
}
