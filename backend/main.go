package main

import (
	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/infrastructure/database"
)

func main() {
	cfg := config.LoadConfig()
	database.ConnectDB(cfg)
}