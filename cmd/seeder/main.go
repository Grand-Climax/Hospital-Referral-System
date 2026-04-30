package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/infrastructure/seeder_deprecated"
)

func main() {
	cfg := config.LoadConfig()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Provide a fallback for local testing if not set
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Africa/Addis_Ababa",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	ctx := context.Background()
	log.Println("Connected to internal database...")

	if err := seeder.SeedReferenceData(ctx, db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Finished executing database seeders!")
}
