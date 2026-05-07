package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type AdminConfigHandler struct {
	adminConfigUC iusecase.AdminConfigUseCase
}

func NewAdminConfigHandler(adminConfigUC iusecase.AdminConfigUseCase) *AdminConfigHandler {
	return &AdminConfigHandler{adminConfigUC: adminConfigUC}
}

// GetConfig godoc
// @Summary      Get System Configuration
// @Description  Retrieve all system-wide runtime configuration keys (e.g., buffer_days, aging_factor, overbook_limit).
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Prerequisites:** Authenticated session.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 403 Forbidden (not super admin)
// @Tags         System Admin
// @Produce      json
// @Success      200 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/admin/config [get]
func (h *AdminConfigHandler) GetConfig(c *gin.Context) {
	config, err := h.adminConfigUC.GetConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// UpdateConfig godoc
// @Summary      Update System Configuration
// @Description  Update system-wide runtime configuration values.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **State Transition:** Immediately affects scheduling logic and triage weight calculations.
// @Description  **Common Keys:** buffer_days, aging_factor, max_horizon_days, overbook_limit_default.
// @Description  **Common Errors:**
// @Description  - 400 invalid key or value format
// @Description  - 403 forbidden (not super admin)
// @Tags         System Admin
// @Accept       json
// @Produce      json
// @Param        updates body map[string]string true "Key-value pairs to update"
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/config [put]
func (h *AdminConfigHandler) UpdateConfig(c *gin.Context) {
	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid request body"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.BaseResponse{Success: false, Message: "User ID not found in context"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	if err := h.adminConfigUC.UpdateConfig(c.Request.Context(), updates, userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Configuration updated successfully"})
}
