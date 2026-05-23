package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting database migration runner (v13 - User PhoneNumber support)...")
	
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection successful.")

	migrationFile := "scripts/db_migration/schema_update_v13.sql"
	log.Printf("Executing raw SQL migration from %s...", migrationFile)
	migrationContent, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file %s: %v", migrationFile, err)
	}

	if err := db.Exec(string(migrationContent)).Error; err != nil {
		log.Fatalf("Failed to execute SQL migration: %v", err)
	}

	log.Println("✓ Raw SQL migration executed successfully.")
	log.Println("✓ Database migration v13 completed successfully.")
}
