package database

import (
	"Hospital-Referral-System/config"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
func ConnectDB(cfg config.DBConfig) {
	var err error
	
	data_source_name := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
    cfg.Host, cfg.User, cfg.Password, cfg.DB_Name, cfg.Port, cfg.SSLMode)


	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  data_source_name,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

}