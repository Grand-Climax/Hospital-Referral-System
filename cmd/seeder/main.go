package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/seeds"
)

func main() {
	cfg := config.LoadConfig()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Provide a fallback for local testing if not set
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Africa/Addis_Ababa",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to internal database...")

	if err := seeds.SeedAll(db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Finished executing database seeders!")
}
