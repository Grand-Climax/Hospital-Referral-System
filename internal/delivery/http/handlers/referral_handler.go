package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/usecase"
)

type ReferralHandler struct {
	referralUseCase usecase.ReferralUseCase
}

func NewReferralHandler(uc usecase.ReferralUseCase) *ReferralHandler {
	return &ReferralHandler{referralUseCase: uc}
}

// Create godoc
// @Summary      Create Referral
// @Description  Submit a new referral (Draft or Submitted status)
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        X-Doctor-ID header string true "Doctor ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        X-Hospital-ID header string true "Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        body body dto.CreateReferralRequest true "Referral payload"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals [post]
func (h *ReferralHandler) Create(c *gin.Context) {
	var req dto.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// For Sprint 4 testing, we extract ID logic headers since we are bypassing complex JWT extraction here
	doctorIDStr := c.GetHeader("X-Doctor-ID")
	hospitalIDStr := c.GetHeader("X-Hospital-ID")

	if doctorIDStr == "" || hospitalIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context headers"})
		return
	}

	doctorID, _ := uuid.Parse(doctorIDStr)
	hospitalID, _ := uuid.Parse(hospitalIDStr)

	referral, err := h.referralUseCase.CreateReferral(c.Request.Context(), doctorID, hospitalID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create referral transaction", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Referral created successfully",
		"data":    referral,
	})
}

// List godoc
// @Summary      List Referrals
// @Description  Get referrals scoped to the requesting hospital and role
// @Tags         Referrals
// @Produce      json
// @Param        X-Hospital-ID header string true "Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        X-User-Role header string true "User Role" default(HOSPITAL_ADMIN)
// @Param        status query string false "Filter by Status"
// @Param        date_from query string false "Start Date"
// @Param        date_to query string false "End Date"
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals [get]
func (h *ReferralHandler) List(c *gin.Context) {
	hospitalID, _ := uuid.Parse(c.GetHeader("X-Hospital-ID"))
	role := entity.UserRole(c.GetHeader("X-User-Role"))
	status := c.Query("status")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	referrals, err := h.referralUseCase.ListReferrals(c.Request.Context(), role, hospitalID, status, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referrals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": referrals,
	})
}

// GetByID godoc
// @Summary      Get Referral ID
// @Description  Retrieve full referral details by ID
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id} [get]
func (h *ReferralHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	referral, err := h.referralUseCase.GetReferral(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": referral})
}

// UpdateDraft godoc
// @Summary      Update Referral Draft
// @Description  Update an existing referral draft before submission
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.CreateReferralRequest true "Referral update payload"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id} [put]
func (h *ReferralHandler) UpdateDraft(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req dto.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	referral, err := h.referralUseCase.UpdateDraft(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update draft", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Draft updated successfully",
		"data":    referral,
	})
}

// DeleteDraft godoc
// @Summary      Delete Referral Draft
// @Description  Permanently delete a draft referral
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id} [delete]
func (h *ReferralHandler) DeleteDraft(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	if err := h.referralUseCase.DeleteDraft(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete draft", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Draft deleted successfully"})
}

// UpdateStatus godoc
// @Summary      Update Referral Status
// @Description  Transition a referral state forwards or backwards securely
// @Tags         Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        X-Doctor-ID header string true "Doctor ID deciding" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Param        body body object true "Status Payload"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/status [patch]
func (h *ReferralHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status payload", "details": err.Error()})
		return
	}

	// Assuming we extract the user ID deciding the change from the JWT/Headers
	userIDStr := c.GetHeader("X-Doctor-ID")
	userID, _ := uuid.Parse(userIDStr)

	if err := h.referralUseCase.UpdateReferralStatus(c.Request.Context(), id, entity.ReferralStatus(req.Status), userID, req.Reason); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Status transition failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Referral status updated to " + req.Status})
}
