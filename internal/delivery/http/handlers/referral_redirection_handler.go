package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type RedirectionHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewRedirectionHandler(referralUC iusecase.ReferralUseCase) *RedirectionHandler {
	return &RedirectionHandler{referralUC: referralUC}
}

// GetRedirectionHistory godoc
// @Summary      Get Redirection History
// @Description  Get a chronological list of all redirections for a specific referral.
// @Description  **Access:** Referring Doctor, Sender Liaison, Current Hospital Specialist, System Admin.
// @Tags         Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/redirections [get]
func (h *RedirectionHandler) GetRedirectionHistory(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	roleVal, _ := c.Get("role")
	role := ""
	if r, ok := roleVal.(entity.UserRole); ok {
		role = string(r)
	} else if s, ok := roleVal.(string); ok {
		role = s
	}

	redirections, err := h.referralUC.GetRedirectionHistory(c.Request.Context(), referralID, userID, role, hospID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	// Map to DTO
	resp := make([]dto.RedirectionResponse, 0, len(redirections))
	for _, r := range redirections {
		resp = append(resp, dto.RedirectionResponse{
			ID:                       r.ID,
			ReferralID:               r.ReferralID,
			RedirectedFromHospitalID: r.RedirectedFromHospitalID,
			RedirectedFromHospitalName: r.RedirectedFromHospital.Name,
			RedirectedToHospitalID:   r.RedirectedToHospitalID,
			RedirectedToHospitalName:   r.RedirectedToHospital.Name,
			RedirectedBySpecialistID: r.RedirectedBySpecialistID,
			RedirectionReason:        *r.RedirectionReason,
			CreatedAt:                r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}
