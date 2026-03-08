package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// Swagger docs
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ---- Dependency Injection ----

	// Cache / token stores
	tokenBlacklist := cache.NewRedisTokenBlacklist(redisClient)
	sessionStore := cache.NewRedisSessionStore(redisClient)

	// Repositories
	authRepo := repository.NewAuthRepository(db)
	userRepo := repository.NewUserRepository(db)
	hospitalRepo := repository.NewHospitalRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Use Cases
	authUseCase := usecase.NewAuthUseCase(authRepo, tokenBlacklist, sessionStore)
	userUseCase := usecase.NewUserUseCase(userRepo)
	hospitalUseCase := usecase.NewHospitalUseCase(hospitalRepo)
	departmentUseCase := usecase.NewDepartmentUseCase(departmentRepo, hospitalRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authUseCase)
	userHandler := handlers.NewUserHandler(userUseCase)
	hospitalHandler := handlers.NewHospitalHandler(hospitalUseCase)
	departmentHandler := handlers.NewDepartmentHandler(departmentUseCase)

	// ---- Routes ----
	v1 := router.Group("/api/v1")
	{
		// Auth (public)
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/login", middleware.RateLimiter(redisClient, 5, time.Minute), authHandler.Login)
			authRoutes.POST("/refresh", authHandler.Refresh)
			authRoutes.POST("/logout", authHandler.Logout)
		}

		// Protected routes (require authentication + audit logging)
		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth(tokenBlacklist))
		protected.Use(middleware.AuditLogger(auditLogRepo))
		{
			// ---- User Management ----
			// Profile – accessible by any authenticated user
			protected.GET("/users/me", userHandler.GetMyProfile)

			// Admin-only user CRUD
			adminUsers := protected.Group("/users")
			adminUsers.Use(middleware.RequireRole(entity.RoleSystemAdmin))
			{
				adminUsers.POST("", userHandler.CreateUser)
				adminUsers.GET("", userHandler.ListUsers)
				adminUsers.GET("/:id", userHandler.GetUser)
				adminUsers.PUT("/:id", userHandler.UpdateUser)
				adminUsers.DELETE("/:id", userHandler.DeleteUser)
				adminUsers.PATCH("/:id/role", userHandler.AssignRole)
			}

			// ---- Hospital Management ----
			// Read – accessible by any authenticated user
			protected.GET("/hospitals", hospitalHandler.ListHospitals)
			protected.GET("/hospitals/:id", hospitalHandler.GetHospital)

			// Admin-only hospital write operations
			adminHospitals := protected.Group("/hospitals")
			adminHospitals.Use(middleware.RequireRole(entity.RoleSystemAdmin))
			{
				adminHospitals.POST("", hospitalHandler.CreateHospital)
				adminHospitals.PUT("/:id", hospitalHandler.UpdateHospital)
				adminHospitals.DELETE("/:id", hospitalHandler.DeleteHospital)
			}

			// Hospital-Department linking (admin-only for write, read for all)
			protected.GET("/hospitals/:id/departments", departmentHandler.ListHospitalDepartments)

			adminHospDept := protected.Group("/hospitals")
			adminHospDept.Use(middleware.RequireRole(entity.RoleSystemAdmin))
			{
				adminHospDept.POST("/:id/departments", departmentHandler.LinkDepartmentToHospital)
				adminHospDept.DELETE("/:id/departments/:deptId", departmentHandler.UnlinkDepartmentFromHospital)
			}

			// ---- Department Management ----
			// Read – accessible by any authenticated user
			protected.GET("/departments", departmentHandler.ListDepartments)
			protected.GET("/departments/:id", departmentHandler.GetDepartment)

			// Admin-only department write operations
			adminDepts := protected.Group("/departments")
			adminDepts.Use(middleware.RequireRole(entity.RoleSystemAdmin))
			{
				adminDepts.POST("", departmentHandler.CreateDepartment)
				adminDepts.PUT("/:id", departmentHandler.UpdateDepartment)
				adminDepts.DELETE("/:id", departmentHandler.DeleteDepartment)
			}
		}
	}
}
