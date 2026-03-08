package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "Hospital-Referral-System/docs"
	"Hospital-Referral-System/internal/delivery/http/routes"
	"Hospital-Referral-System/internal/infrastructure/middleware"
)

// @title           Hospital Referral System API
// @version         1.0
// @description     API for managing hospital referrals, users, hospitals, and departments.
// @host            localhost:8081
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format **Bearer &lt;token&gt;**

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env or .env.local file found. Using environment variables.")
		}
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=hospital_referral port=5432 sslmode=disable TimeZone=Africa/Addis_Ababa"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
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

	routes.Register(router, db, redisClient)

	if err := router.Run(":8081"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
