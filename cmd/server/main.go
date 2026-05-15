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
	"Hospital-Referral-System/internal/seeds"
)

// @title           Hospital Referral System API
// @version         1.0
// @description # Hospital Referral Hub API
// @description A national‑scale hospital referral management platform that digitises the entire patient‑transfer workflow, from initial doctor referral to final clinical outcome. All actions are governed by strict role‑based access controls and clinical governance rules.
// @description
// @description ---
// @description ## Referral Lifecycle
// @description DRAFT → SUBMITTED → (Liaison) UNDER_LIAISON_REVIEW → FORWARDED
// @description ↘ REJECTED_BY_LIAISON
// @description FORWARDED → (Specialist) UNDER_SPECIALIST_REVIEW → ACCEPTED / REJECTED_BY_SPECIALIST
// @description ACCEPTED → SCHEDULED → ASSIGNED → COMPLETED
// @description ↘ MISSED / RESCHEDULED / DECEASED
// @description
// @description ---
// @description ## Visibility Rules
// @description
// @description | Status | Visible To |
// @description |----------------------|-----------|
// @description | DRAFT / NEED_REVISION | Referring Doctor only |
// @description | SUBMITTED … FORWARDED | Liaison of the sender hospital |
// @description | FORWARDED … COMPLETED | Specialists of the target hospital |
// @description | ACCEPTED … SCHEDULED | Receptionists of the target hospital |
// @description | All statuses | System Admins (global); MoH Analysts (aggregated dashboards, no raw clinical data) |
// @description
// @description ---
// @description ## Critical Business Rules
// @description
// @description - **ML Triage Gate**: A referral cannot be accepted without a severity score (set manually via `POST /specialist/referrals/{id}/triage-severity`).
// @description - **Duplicate Prevention**: A patient may not have more than one active referral (status not COMPLETED, CANCELLED, REJECTED_*, DECEASED) to the same target department. The API returns 409 Conflict.
// @description - **Walk‑in Restriction**: Walk‑ins can only be registered for referrals in ACCEPTED or SCHEDULED status.
// @description - **Emergency Scheduling**: Bypasses buffer days and allows overbooking up to `overbook_limit`. Requires critical condition or explicit justification.
// @description - **Deceased Outcome**: Recording a deceased outcome sets the referral to DECEASED, soft‑archives it (`is_archived=true`), and cancels any pending appointments.
// @description - **Cancel After Send**: A doctor may cancel a referral after sending (REJECTED_AFTER_SEND) only if it has not been accepted yet.
// @description
// @description For detailed per‑endpoint rules, see the individual endpoint descriptions below.
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

	// Seeder Logic
	var count int64
	db.Table("system_configs").Count(&count)
	if count == 0 || os.Getenv("SEED_DB") == "true" {
		log.Println("Initializing database seed...")
		// Import the seeds package
		if err := seeds.SeedAll(db); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	}

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
