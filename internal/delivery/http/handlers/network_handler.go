package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/usecase"
)

type NetworkHandler struct {
	networkUseCase usecase.NetworkUseCase
}

func NewNetworkHandler(uc usecase.NetworkUseCase) *NetworkHandler {
	return &NetworkHandler{networkUseCase: uc}
}

// Create godoc
// @Summary      Create Network Route
// @Description  Define a routing logic rule between two networked hospitals
// @Tags         Networks
// @Accept       json
// @Produce      json
// @Param        body body dto.CreateNetworkRouteRequest true "Network routing rule payload"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/network-routes [post]
func (h *NetworkHandler) Create(c *gin.Context) {
	var req dto.CreateNetworkRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	route, err := h.networkUseCase.CreateRoute(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create network route", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Network route created successfully",
		"data": dto.NetworkRouteResponse{
			ID:                    route.ID,
			SenderHospitalID:      route.SenderHospitalID,
			ReceiverHospitalID:    route.ReceiverHospitalID,
			ReferralType:          route.ReferralType,
			RequiresAdminApproval: route.RequiresAdminApproval,
		},
	})
}

// List godoc
// @Summary      List Network Routes
// @Description  Retrieve all routing rules, optionally filtered by sender hospital
// @Tags         Networks
// @Produce      json
// @Param        sender_hospital_id query string false "Sender Hospital ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/network-routes [get]
func (h *NetworkHandler) List(c *gin.Context) {
	var senderID *uuid.UUID
	if senderQuery := c.Query("sender_hospital_id"); senderQuery != "" {
		if parsed, err := uuid.Parse(senderQuery); err == nil {
			senderID = &parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sender_hospital_id format"})
			return
		}
	}

	routes, err := h.networkUseCase.ListRoutes(c.Request.Context(), senderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list network routes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": routes})
}

// Delete godoc
// @Summary      Delete Network Route
// @Description  Remove a network routing mapping
// @Tags         Networks
// @Produce      json
// @Param        id path string true "Route ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/network-routes/{id} [delete]
func (h *NetworkHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid route ID format"})
		return
	}

	if err := h.networkUseCase.DeleteRoute(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete route"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Network route deleted successfully"})
}
