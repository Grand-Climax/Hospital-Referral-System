package repositorytest

import (
	"Hospital-Referral-System/internal/domain/entity"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {

	var err error
	if os.Getenv("CI") != "true" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			log.Fatalf("Error loading .env file %v", err)
		}
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to test database: %v", err)
	}

	// Migrate schema
	err = testDB.AutoMigrate(&entity.Role{}, &entity.Department{})

	if err != nil {
		log.Fatalf("Failed to auto-migrate: %v", err)
	}

	code := m.Run()
	os.Exit(code)
}