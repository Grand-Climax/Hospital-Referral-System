package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Host string
	User string
	Port string
	DB_Name string
	Password string
	SSLMode string
}

func LoadConfig() DBConfig{
	if err := godotenv.Load(); err != nil {
        log.Fatal("Failed to load .env file")
    }

	return DBConfig {
		Host: os.Getenv("DB_HOST"),
		User: os.Getenv("DB_USER"),
		Port: os.Getenv("DB_PORT"),
		DB_Name: os.Getenv("DB_NAME"),
		Password: os.Getenv("DB_PASSWORD"),
		SSLMode: os.Getenv("DB_SSLMODE"),
	}
}