package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/domain/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func main() {
	csvPath := flag.String("csv", "docs/ICDCODE10.csv", "Path to the ICD-10-CM CSV file")
	flag.Parse()

	log.Println("Starting ICD-10-CM loader script...")
	startTime := time.Now()

	// 1. Load configuration and connect to database
	cfg := config.LoadConfig()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TZ=Africa/Addis_Ababa",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection successful.")

	// Apply GIN and prefix indexes migration (schema_update_v11.sql)
	log.Println("Applying database indexes migration (schema_update_v11.sql)...")
	migrationContent, err := os.ReadFile("scripts/db_migration/schema_update_v11.sql")
	if err != nil {
		log.Fatalf("Failed to read schema_update_v11.sql migration: %v", err)
	}
	if err := db.Exec(string(migrationContent)).Error; err != nil {
		log.Fatalf("Failed to execute migration schema_update_v11.sql: %v", err)
	}
	log.Println("Database indexes migrated successfully.")

	// 2. Open and parse CSV file
	file, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("Failed to open CSV file at %s: %v", *csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Read header
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Failed to read CSV header: %v", err)
	}
	log.Printf("CSV Header read: %v", header)

	// Validate columns
	if len(header) < 3 || header[0] != "CODE" || header[1] != "DESCRIPTION" || header[2] != "CATEGORY" {
		log.Fatalf("Invalid CSV structure. Expected columns: CODE, DESCRIPTION, CATEGORY")
	}

	var records []entity.ICDCode
	rowCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failed to read CSV row: %v", err)
		}
		rowCount++

		records = append(records, entity.ICDCode{
			Code:        record[0],
			Description: record[1],
			Category:    record[2],
		})
	}
	log.Printf("Parsed %d rows from CSV file in %v.", rowCount, time.Since(startTime))

	// 3. Perform super-fast GORM batch upserts
	log.Println("Uploading ICD-10 codes to database in batches...")
	batchStartTime := time.Now()

	err = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "category"}),
	}).CreateInBatches(records, 1000).Error

	if err != nil {
		log.Fatalf("Failed to perform batch upsert into database: %v", err)
	}

	log.Printf("Successfully loaded/upserted %d ICD-10 codes in %v.", len(records), time.Since(batchStartTime))
	log.Printf("Total execution time: %v.", time.Since(startTime))
}
