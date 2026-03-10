package main

import (
	"log"

	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=6628 dbname=referral port=5432 sslmode=disable TimeZone=Africa/Addis_Ababa"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to local DB: %v", err)
	}

	log.Println("Dropping specific tables to force clean seeder execution...")
	
	tables := []string{
		"audit_logs",
		"attachments",
		"referral_status_histories",
		"referral_emergency_details",
		"vitals",
		"referral_diagnoses",
		"referral_forms",
		"referrals",
		"referral_networks",
		"hospital_departments",
		"users",
		"patients",
	}

	for _, table := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE;").Error; err != nil {
			log.Printf("Warning: failed to drop table %s: %v", table, err)
		}
	}

	log.Println("AutoMigrating User and Attachment again to ensure schema...")
	db.AutoMigrate(&entity.User{}, &entity.Attachment{})

	log.Println("Database reset successful.")
}
