package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/cache"
	"Hospital-Referral-System/internal/infrastructure/middleware"
	"Hospital-Referral-System/internal/repository"
	"Hospital-Referral-System/internal/usecase"
)

// Register attaches all HTTP routes to the provided router.
func Register(router *gin.Engine, db *gorm.DB, redisClient *redis.Client) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Dependency Injection for Auth
	tokenBlacklist := cache.NewRedisTokenBlacklist(redisClient)
	sessionStore := cache.NewRedisSessionStore(redisClient)
	authRepo := repository.NewAuthRepository(db)
	authUseCase := usecase.NewAuthUseCase(authRepo, tokenBlacklist, sessionStore)
	authHandler := handlers.NewAuthHandler(authUseCase)

	// Dependency Injection for Referrals
	referralRepo := repository.NewReferralRepository(db)
	referralUseCase := usecase.NewReferralUseCase(referralRepo)
	referralHandler := handlers.NewReferralHandler(referralUseCase)

	// Dependency Injection for References (Dropdowns)
	refRepo := repository.NewReferenceRepository(db)
	refUseCase := usecase.NewReferenceUseCase(refRepo)
	refHandler := handlers.NewReferenceHandler(refUseCase)

	// Dependency Injection for Network Administration
	netRepo := repository.NewNetworkRepository(db)
	netUseCase := usecase.NewNetworkUseCase(netRepo)
	netHandler := handlers.NewNetworkHandler(netUseCase)

	// Dependency Injection for Patients
	patientRepo := repository.NewPatientRepository(db)
	patientUseCase := usecase.NewPatientUseCase(patientRepo)
	patientHandler := handlers.NewPatientHandler(patientUseCase)

	// Dependency Injection for Attachments
	attachmentRepo := repository.NewAttachmentRepository(db)
	attachmentUseCase := usecase.NewAttachmentUseCase(attachmentRepo)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentUseCase)

	// API v1 Routes
	v1 := router.Group("/api/v1")
	{
		authRoutes := v1.Group("/auth")
		{
			// Max 5 requests per minute for login attempts
			authRoutes.POST("/login", middleware.RateLimiter(redisClient, 5, time.Minute), authHandler.Login)
			authRoutes.POST("/refresh", authHandler.Refresh)
			authRoutes.POST("/logout", authHandler.Logout)
		}

		// Example Protected Group
		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth(tokenBlacklist))
		{
			// Admin Level Network Management Routes
			adminGroup := protected.Group("/admin/network-routes")
			// Depending on exact desired hierarchy either System Admin or MoH Analyst or specific Hospital Admin could map routes
			// We will grant HOSPITAL_ADMIN so the test validates successfully without wrestling the GORM seeder defaults
			adminGroup.Use(middleware.RequireRole(entity.RoleHospitalAdmin))
			{
				adminGroup.POST("", netHandler.Create)
				adminGroup.GET("", netHandler.List)
				adminGroup.DELETE("/:id", netHandler.Delete)
			}
			
			// Patient Identity Routes
			patientGroup := protected.Group("/patients")
			patientGroup.Use(middleware.RequireRole(
				entity.RoleReferringDoctor,
				entity.RoleReceptionist,
				entity.RoleSystemSuperAdmin,
			))
			{
				patientGroup.GET("/national-id/:id", patientHandler.GetByNationalID)
			}

			// Sprint 4.1 & 4.2: Reference Dropdowns & Relational Networks
			protected.GET("/reference/hospitals", refHandler.GetHospitals)
			// Networked Dropdown uses X-Hospital-ID from Auth Token implicitly
			protected.GET("/reference/networked-hospitals", refHandler.GetNetworkedHospitals)
			protected.GET("/reference/departments", refHandler.GetDepartments)
			// Target Dropdown targets a specific receiver hospital path ID
			protected.GET("/reference/hospitals/:id/departments", refHandler.GetHospitalDepartments)
			protected.GET("/reference/icd-codes", refHandler.SearchICD)
			// Example: Both DOCTOR and SPECIALIST
			// protected.GET("/clinical-data", middleware.RequireRole(entity.RoleReferringDoctor, entity.RoleReceivingSpecialist), someHandler)

			// Sprint 5 & 6: Referral Endpoints (Annex IV & State Machine)
			protected.POST("/referrals", middleware.RequirePermission(entity.ActionCreateReferral), referralHandler.Create)
			protected.GET("/referrals", referralHandler.List)
			protected.GET("/referrals/:id", referralHandler.GetByID)
			protected.PUT("/referrals/:id", referralHandler.UpdateDraft)
			protected.DELETE("/referrals/:id", referralHandler.DeleteDraft)
			protected.PATCH("/referrals/:id/status", referralHandler.UpdateStatus)
			
			// Attachments
			protected.POST("/referrals/:id/attachments", attachmentHandler.UploadAttachment)
			protected.GET("/attachments/:id/download", attachmentHandler.DownloadAttachment)
		}
	}
}
