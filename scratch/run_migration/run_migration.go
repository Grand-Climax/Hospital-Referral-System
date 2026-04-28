package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Connected to database successfully.")

	scripts := []string{
		"scripts/db_migration/schema_update_v3.sql",
	}

	for _, scriptPath := range scripts {
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			log.Fatalf("Failed to read script %s: %v", scriptPath, err)
		}

		fmt.Printf("Executing script: %s\n", scriptPath)

		err = db.Exec(string(content)).Error
		if err != nil {
			log.Fatalf("Failed to execute script %s: %v", scriptPath, err)
		}
		fmt.Printf("Script %s completed successfully.\n", scriptPath)
	}

	fmt.Println("All specified migrations completed successfully.")
}
