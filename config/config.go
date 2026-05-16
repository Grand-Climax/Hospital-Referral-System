package config

import (
	"log"
	"os"

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

type Config struct {
	DB         DBConfig
	RedisURL   string
	JWTSecret  string
	Port       string
	Cloudinary CloudinaryConfig
	SMS        SMSConfig
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
	}
}