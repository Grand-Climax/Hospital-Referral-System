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
	godotenv.Load(".env.local")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=password dbname=referral_db port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Resetting database...")
	db.Exec("TRUNCATE users CASCADE")
	db.Exec("TRUNCATE hospitals CASCADE")
	db.Exec("TRUNCATE departments CASCADE")
	db.Exec("TRUNCATE patients CASCADE")
	db.Exec("TRUNCATE referrals CASCADE")
	db.Exec("TRUNCATE triage_queues CASCADE")
	db.Exec("TRUNCATE system_configs CASCADE")
	fmt.Println("Database reset successful.")
}
