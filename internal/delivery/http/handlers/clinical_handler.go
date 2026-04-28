package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ClinicalHandler struct {
	clinicalUC iusecase.ClinicalUseCase
}

func NewClinicalHandler(clinicalUC iusecase.ClinicalUseCase) *ClinicalHandler {
	return &ClinicalHandler{clinicalUC: clinicalUC}
}

// AddUpdate godoc
// @Summary      Add Clinical Update
// @Description  Add a clinical progress note to the referral.
// @Description  **Roles:** Requires clinical access (treating or consulted doctor).
// @Description  **Prerequisites:** Specialist must be assigned or have explicit access grant.
// @Description  **Side Effect:** If update_reason is CONDITION_CHANGE or MISSED_APPOINTMENT_RE_EVALUATION, the `requires_review` flag is set.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (no clinical access)
// @Tags         Clinical
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.AddClinicalUpdateRequest true "Update details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/clinical/updates [post]
func (h *ClinicalHandler) AddUpdate(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	var req dto.AddClinicalUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.clinicalUC.AddClinicalUpdate(c.Request.Context(), referralID, userID, req); err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Clinical update added successfully"})
}

// RecordOutcome godoc
// @Summary      Record Referral Outcome
// @Description  Record the final clinical outcome and close the referral episode.
// @Description  **Roles:** Requires clinical access (assigned specialist).
// @Description  **Prerequisites:** referral must be active.
// @Description  **State Transition:** → COMPLETED; if outcome = 'deceased', → DECEASED and is_archived=true.
// @Description  **Side Effect:** Pending appointments cancelled if deceased; clinical access revoked on completion.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (no clinical access)
// @Tags         Clinical
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.RecordOutcomeRequest true "Outcome details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/clinical/outcome [post]
func (h *ClinicalHandler) RecordOutcome(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	var req dto.RecordOutcomeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.clinicalUC.RecordOutcome(c.Request.Context(), referralID, userID, req); err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Referral outcome recorded successfully"})
}

// GetHistory godoc
// @Summary      Get Clinical History
// @Description  Returns the chronological history of clinical updates for a referral.
// @Description  **Roles:** DOCTOR, SPECIALIST, or RECEPTIONIST with access.
// @Description  **Prerequisites:** Must be the creator, assigned specialist, or hospital staff where referral is active.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (no clinical access)
// @Tags         Clinical
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/clinical/history [get]
func (h *ClinicalHandler) GetHistory(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	history, err := h.clinicalUC.GetClinicalHistory(c.Request.Context(), referralID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}
