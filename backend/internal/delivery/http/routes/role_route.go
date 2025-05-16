package router

import (
	"Hospital-Referral-System/internal/delivery/http/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoleRouter(rg *gin.RouterGroup, Rolehandler *handlers.RoleHandler) {
	roleRoutes := rg.Group("/roles")
	{
		roleRoutes.POST("/", Rolehandler.CreateRole)
		roleRoutes.GET("/", Rolehandler.GetRole)
		roleRoutes.PUT("/", Rolehandler.UpdateRole)
		roleRoutes.DELETE("/:id", Rolehandler.DeleteRole)
	}
}
