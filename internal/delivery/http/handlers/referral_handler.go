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
