package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/handlers"
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
			// Example: Only SYSTEM_ADMIN can hit this
			// protected.GET("/admin-only", middleware.RequireRole(entity.RoleSystemAdmin), someHandler)
			
			// Example: Both DOCTOR and SPECIALIST
			// protected.GET("/clinical-data", middleware.RequireRole(entity.RoleReferringDoctor, entity.RoleReceivingSpecialist), someHandler)
		}
	}
}
