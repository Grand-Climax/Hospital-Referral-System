package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register attaches all HTTP routes to the provided router.
func Register(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
