package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type NetworkHandler struct {
	networkUseCase iusecase.NetworkUseCase
}

func NewNetworkHandler(uc iusecase.NetworkUseCase) *NetworkHandler {
	return &NetworkHandler{networkUseCase: uc}
}

// Create godoc
// @Summary      Create Network Route
// @Description  Define a routing rule linking two hospitals. Only SYSTEM_SUPER_ADMIN can create global network routes.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 500 Internal Server Error
// @Tags         Network Routes (Admin)
// @Accept       json
// @Produce      json
// @Param        body body dto.CreateNetworkRouteRequest true "Network routing rule payload"
// @Success      201 {object} dto.NetworkRouteResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/network-routes [post]
func (h *NetworkHandler) Create(c *gin.Context) {
	var req dto.CreateNetworkRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload: " + err.Error(),
		})
		return
	}

	route, err := h.networkUseCase.CreateRoute(c.Request.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "cannot create") || strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to create network route: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.NetworkRouteResponse{
		ID:                    route.ID,
		SenderHospitalID:      route.SenderHospitalID,
		ReceiverHospitalID:    route.ReceiverHospitalID,
		ReferralType:          route.ReferralType,
		RequiresAdminApproval: route.RequiresAdminApproval,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Network route created successfully",
		},
	})
}

// List godoc
// @Summary      List Network Routes
// @Description  Retrieve all routing rules, optionally filtered by sender hospital. Only HOSPITAL_ADMIN can view admin routes.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid filter format
// @Description  - 500 Internal Server Error
// @Tags         Network Routes (Admin)
// @Produce      json
// @Param        sender_hospital_id query string false "Filter by Sender Hospital ID"
// @Success      200 {object} dto.NetworkRouteListResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/network-routes [get]
func (h *NetworkHandler) List(c *gin.Context) {
	var senderID *uuid.UUID
	if senderQuery := c.Query("sender_hospital_id"); senderQuery != "" {
		if parsed, err := uuid.Parse(senderQuery); err == nil {
			senderID = &parsed
		} else {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Error:   "Invalid sender_hospital_id format",
			})
			return
		}
	}

	routes, err := h.networkUseCase.ListRoutes(c.Request.Context(), senderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to list network routes",
		})
		return
	}

	var responseData []dto.NetworkRouteResponse
	for _, r := range routes {
		responseData = append(responseData, dto.NetworkRouteResponse{
			ID:                    r.ID,
			SenderHospitalID:      r.SenderHospitalID,
			ReceiverHospitalID:    r.ReceiverHospitalID,
			ReferralType:          r.ReferralType,
			RequiresAdminApproval: r.RequiresAdminApproval,
			BaseResponse: dto.BaseResponse{
				Success: true,
			},
		})
	}

	c.JSON(http.StatusOK, dto.NetworkRouteListResponse{
		Data: responseData,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Network routes retrieved successfully",
		},
	})
}

// Delete godoc
// @Summary      Delete Network Route
// @Description  Remove a referral network routing rule. Only SYSTEM_SUPER_ADMIN can delete routes.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid ID format
// @Description  - 500 Internal Server Error
// @Tags         Network Routes (Admin)
// @Produce      json
// @Param        id path string true "Route ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/network-routes/{id} [delete]
func (h *NetworkHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid route ID format",
		})
		return
	}

	if err := h.networkUseCase.DeleteRoute(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to delete route",
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Network route deleted successfully",
	})
}
