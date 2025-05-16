package router

import (
	"Hospital-Referral-System/internal/delivery/http/handlers"

	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	RoleHandler *handlers.RoleHandler
}

func SetupRouter (config *RouterConfig) *gin.Engine{
	router := gin.Default()

	api := router.Group("/api")


	SetupRoleRouter(api, config.RoleHandler)
	return router
}