package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/seeds"
)

func main() {
	godotenv.Load(".env.local")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set in .env.local")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Running database seeder...")
	if err := seeds.SeedAll(db); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}
	fmt.Println("Database seeded successfully.")
}
