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
