package main

import (
	"log"

	"Hospital-Referral-System/internal/delivery/http/routes"
	"Hospital-Referral-System/internal/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.New()

	router.Use(
		middleware.LoggingMiddleware(),
		middleware.RecoveryMiddleware(),
		middleware.ErrorHandlingMiddleware(),
		middleware.CORS(),
	)

	routes.Register(router)

	if err := router.Run(":8081"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
