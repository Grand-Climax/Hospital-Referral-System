package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "Hospital-Referral-System/docs"
	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/delivery/http/routes"
	"Hospital-Referral-System/internal/infrastructure/middleware"
)

// @title           Hospital Referral System API
// @version         1.0
// @description     API for managing hospital referrals, users, hospitals, and departments.
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format **Bearer &lt;token&gt;**

func main() {
	cfg := config.LoadConfig()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback to individual fields if DATABASE_URL is missing
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Africa/Addis_Ababa",
			cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	redisAddr := cfg.RedisURL
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	var redisClient *redis.Client

	// Support Upstash/Cloud Redis URLs via ParseURL
	opt, parseErr := redis.ParseURL(redisAddr)
	if parseErr == nil {
		redisClient = redis.NewClient(opt)
	} else {
		// Fallback for simple localhost addresses
		redisClient = redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})
	}

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	router := gin.New()

	router.Use(
		middleware.LoggingMiddleware(),
		middleware.RecoveryMiddleware(),
		middleware.ErrorHandlingMiddleware(),
		middleware.CORS(),
	)

	routes.Register(router, db, redisClient, cfg)

	// Cloud Run injects the PORT dynamically. Fallback to 8081 for local dev.
	port := cfg.Port
	if port == "" {
		port = "8081"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
