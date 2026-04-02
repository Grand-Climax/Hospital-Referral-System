package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type SpecialistHandler struct {
	specialistUseCase iusecase.SpecialistUseCase
}

func NewSpecialistHandler(uc iusecase.SpecialistUseCase) *SpecialistHandler {
	return &SpecialistHandler{specialistUseCase: uc}
}

// Accept godoc
// @Summary      Accept Referral (Specialist)
// @Description  Transition referral from FORWARDED/RECEIVED to SPECIALIST_ASSIGNED.
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.SpecialistActionResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/accept [post]
func (h *SpecialistHandler) Accept(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID context"})
		return
	}

	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "No hospital assigned to user"})
		return
	}
	hospIDPtr, valid := hospIDVal.(*uuid.UUID)
	if !valid || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No hospital assigned to user"})
		return
	}

	if err := h.specialistUseCase.AcceptReferral(c.Request.Context(), id, userID, *hospIDPtr); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to accept referral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SpecialistActionResponse{
		Message:    "Referral accepted successfully",
		ReferralID: id.String(),
		NewStatus:  string(entity.StatusSpecialistAssigned),
	})
}

// Reject godoc
// @Summary      Reject Referral (Specialist)
// @Description  Transition referral back to UNDER_LIAISON_REVIEW at origin hospital. Reason required.
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.SpecialistRejectRequest true "Reject Payload (reason required)"
// @Success      200 {object} dto.SpecialistActionResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/reject [post]
func (h *SpecialistHandler) Reject(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req dto.SpecialistRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed, reason is required", "details": err.Error()})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID context"})
		return
	}

	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "No hospital assigned to user"})
		return
	}
	hospIDPtr, valid := hospIDVal.(*uuid.UUID)
	if !valid || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No hospital assigned to user"})
		return
	}

	if err := h.specialistUseCase.RejectReferral(c.Request.Context(), id, userID, *hospIDPtr, req.Reason); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to reject referral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SpecialistActionResponse{
		Message:    "Referral rejected and returned to liaison review",
		ReferralID: id.String(),
		NewStatus:  string(entity.StatusUnderLiaisonReview),
	})
}
