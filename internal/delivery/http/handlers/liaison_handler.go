package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/usecase"
)

type LiaisonHandler struct {
	liaisonUseCase iusecase.LiaisonUseCase
}

func NewLiaisonHandler(uc iusecase.LiaisonUseCase) *LiaisonHandler {
	return &LiaisonHandler{liaisonUseCase: uc}
}

// extractJWTContext pulls userID and hospID from Gin context set by RequireAuth middleware.
func extractJWTContext(c *gin.Context) (userID uuid.UUID, hospID uuid.UUID, ok bool) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return uuid.Nil, uuid.Nil, false
	}
	userID, valid := userIDVal.(uuid.UUID)
	if !valid {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
		return uuid.Nil, uuid.Nil, false
	}

	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "No hospital assigned to user"})
		return uuid.Nil, uuid.Nil, false
	}
	hospIDPtr, valid := hospIDVal.(*uuid.UUID)
	if !valid || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No hospital assigned to user"})
		return uuid.Nil, uuid.Nil, false
	}

	return userID, *hospIDPtr, true
}

func mapLiaisonError(err error) int {
	switch err {
	case usecase.ErrReferralNotFound:
		return http.StatusNotFound
	case usecase.ErrNotYourHospital:
		return http.StatusForbidden
	case usecase.ErrInvalidStatusForAction:
		return http.StatusConflict
	default:
		return http.StatusConflict
	}
}

// ListSubmitted godoc
// @Summary      List Submitted Referrals
// @Description  Fetch referrals in SUBMITTED status scoped to the liaison officer's hospital.
// @Tags         Liaison
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals [get]
func (h *LiaisonHandler) ListSubmitted(c *gin.Context) {
	_, hospID, ok := extractJWTContext(c)
	if !ok {
		return
	}

	referrals, err := h.liaisonUseCase.ListSubmittedReferrals(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submitted referrals", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": referrals})
}

// Approve godoc
// @Summary      Approve Referral (Liaison)
// @Description  Transition referral from SUBMITTED to UNDER_LIAISON_REVIEW. Claims the referral for the current liaison officer.
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.LiaisonActionResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/approve [post]
func (h *LiaisonHandler) Approve(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referral ID format"})
		return
	}

	userID, hospID, ok := extractJWTContext(c)
	if !ok {
		return
	}

	if err := h.liaisonUseCase.ApproveReferral(c.Request.Context(), referralID, userID, hospID); err != nil {
		c.JSON(mapLiaisonError(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.LiaisonActionResponse{
		Message:    "Referral approved and now under liaison review",
		ReferralID: referralID.String(),
		NewStatus:  string(entity.StatusUnderLiaisonReview),
	})
}

// Reject godoc
// @Summary      Reject Referral (Liaison)
// @Description  Transition referral from UNDER_LIAISON_REVIEW to NEEDS_REVISION. Requires a mandatory rejection reason.
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.LiaisonRejectRequest true "Rejection payload with mandatory reason"
// @Success      200 {object} dto.LiaisonActionResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/reject [post]
func (h *LiaisonHandler) Reject(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referral ID format"})
		return
	}

	var req dto.LiaisonRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	userID, hospID, ok := extractJWTContext(c)
	if !ok {
		return
	}

	if err := h.liaisonUseCase.RejectReferral(c.Request.Context(), referralID, userID, hospID, req.Reason); err != nil {
		c.JSON(mapLiaisonError(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.LiaisonActionResponse{
		Message:    "Referral sent back to doctor for revision",
		ReferralID: referralID.String(),
		NewStatus:  string(entity.StatusNeedsRevision),
	})
}

// Forward godoc
// @Summary      Forward Referral (Liaison)
// @Description  Transition referral from UNDER_LIAISON_REVIEW to FORWARDED, targeting a specific receiving hospital.
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.LiaisonForwardRequest true "Forward payload with target hospital"
// @Success      200 {object} dto.LiaisonActionResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/forward [post]
func (h *LiaisonHandler) Forward(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referral ID format"})
		return
	}

	var req dto.LiaisonForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	userID, hospID, ok := extractJWTContext(c)
	if !ok {
		return
	}

	if err := h.liaisonUseCase.ForwardReferral(c.Request.Context(), referralID, userID, hospID, req.TargetHospitalID, req.Comment); err != nil {
		c.JSON(mapLiaisonError(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.LiaisonActionResponse{
		Message:    "Referral forwarded to receiving hospital",
		ReferralID: referralID.String(),
		NewStatus:  string(entity.StatusForwarded),
	})
}
