package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Host     string
	User     string
	Port     string
	DB_Name  string
	Password string
	SSLMode  string
}

type CloudinaryConfig struct {
	CloudName    string
	APIKey       string
	APISecret    string
	UploadPreset string
}

type SMSConfig struct {
	From   string
	Sender string
}

// MLConfig holds defaults for the local ML service (http://localhost:8000).
type MLConfig struct {
	BaseURL    string
	TimeoutSec int
	Enabled    bool
	MaxRetries int
}

type Config struct {
	DB         DBConfig
	RedisURL   string
	JWTSecret  string
	Port       string
	Cloudinary CloudinaryConfig
	SMS        SMSConfig
	ML         MLConfig
}

func LoadConfig() Config {
	if err := godotenv.Load(".env.local"); err != nil {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env or .env.local file found. Using environment variables.")
		}
	}

	return Config{
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			User:     os.Getenv("DB_USER"),
			Port:     os.Getenv("DB_PORT"),
			DB_Name:  os.Getenv("DB_NAME"),
			Password: os.Getenv("DB_PASSWORD"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		RedisURL:  os.Getenv("REDIS_URL"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		Port:      os.Getenv("PORT"),
		Cloudinary: CloudinaryConfig{
			CloudName:    os.Getenv("CLOUDINARY_CLOUD_NAME"),
			APIKey:       os.Getenv("CLOUDINARY_API_KEY"),
			APISecret:    os.Getenv("CLOUDINARY_API_SECRET"),
			UploadPreset: os.Getenv("CLOUDINARY_UPLOAD_PRESET"),
		},
		SMS: SMSConfig{
			From:   os.Getenv("AFROMESSAGE_IDENTIFIER_ID"),
			Sender: os.Getenv("AFROMESSAGE_SENDER_NAME"),
		},
		ML: loadMLConfig(),
	}
}

func loadMLConfig() MLConfig {
	baseURL := strings.TrimSpace(os.Getenv("ML_SERVICE_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	timeoutSec := 120
	if v := os.Getenv("ML_SERVICE_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}
	maxRetries := 3
	if v := os.Getenv("ML_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxRetries = n
		}
	}
	enabled := true
	if v := os.Getenv("ML_ENABLED"); v == "false" || v == "0" {
		enabled = false
	}
	return MLConfig{
		BaseURL:    baseURL,
		TimeoutSec: timeoutSec,
		Enabled:    enabled,
		MaxRetries: maxRetries,
	}
}