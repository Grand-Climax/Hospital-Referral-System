package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/infrastructure/seeder"
)

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env or .env.local file found. Using environment variables.")
		}
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Provide a fallback for local testing if not set
		dsn = "host=localhost user=postgres password=postgres dbname=hospital_referral port=5432 sslmode=disable TimeZone=Africa/Addis_Ababa"
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
