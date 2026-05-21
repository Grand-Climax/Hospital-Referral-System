package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"Hospital-Referral-System/internal/infrastructure/ws"
)

type PushHandler struct {
	hub *ws.Hub
}

func NewPushHandler(hub *ws.Hub) *PushHandler {
	return &PushHandler{hub: hub}
}

type PushRequest struct {
	UserID string      `json:"user_id" binding:"required"`
	Type   string      `json:"type" binding:"required"`
	Data   interface{} `json:"data" binding:"required"`
}

// PushToUser delivers a real‑time message to a specific user via WebSocket.
// Protected by a shared secret (WS_PUSH_SECRET environment variable).
func (h *PushHandler) PushToUser(c *gin.Context) {
	secret := os.Getenv("WS_PUSH_SECRET")
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WS_PUSH_SECRET not configured"})
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "Bearer "+secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req PushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.SendToUser(req.UserID, req)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
