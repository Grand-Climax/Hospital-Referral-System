package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ReceptionistHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewReceptionistHandler(referralUC iusecase.ReferralUseCase) *ReceptionistHandler {
	return &ReceptionistHandler{referralUC: referralUC}
}

// ListReferrals godoc
// @Summary      List Referrals for Receptionist
// @Description  Get a paginated list of referrals assigned to the receptionist's hospital.
// @Tags         Receptionist Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        offset query int false "Pagination offset" default(0)
// @Param        status query string false "Filter by status"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals [get]
func (h *ReceptionistHandler) ListReferrals(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user scopes"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	statusFilter := c.Query("status")

	referrals, total, err := h.referralUC.ListForReceptionist(c.Request.Context(), hospID, limit, offset, statusFilter)
	if err != nil {
		log.Printf("[ReceptionistHandler.List] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var responseData []dto.ListReferralResponse
	for _, r := range referrals {
		diag := ""
		icd := ""
		if len(r.Diagnoses) > 0 && r.Diagnoses[0].CodeInfo != nil {
			diag = r.Diagnoses[0].CodeInfo.Description
			icd = r.Diagnoses[0].ICDCode
		}
		responseData = append(responseData, dto.ListReferralResponse{
			ID:                  r.ID,
			PatientFirstName:    r.Patient.FirstName,
			PatientMiddleName:   r.Patient.MiddleName,
			PatientLastName:     r.Patient.LastName,
			Department:          r.TargetDeptID.String(),
			Date:                r.CreatedAt.Format("2006-01-02"),
			Status:              string(r.Status),
			ICDCode:             icd,
			Diagnosis:           diag,
			ConditionAtReferral: r.ReferralForm.ConditionAtReferral,
		})
	}

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		Data:     responseData,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

// GetReferral godoc
// @Summary      Get Referral Details for Receptionist
// @Description  Get detailed information about a specific referral.
// @Tags         Receptionist Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} entity.Referral
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id} [get]
func (h *ReceptionistHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	ref, err := h.referralUC.GetDetailsForReceptionist(c.Request.Context(), id, hospID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ref)
}

// ConfirmAttendance godoc
// @Summary      Confirm Referral Attendance
// @Description  Confirm that the patient has attended their referral appointment.
// @Tags         Receptionist Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body map[string]string true "Status (key: status)"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/confirm-attendance [post]
func (h *ReceptionistHandler) ConfirmAttendance(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	receptionistID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.ConfirmAttendance(c.Request.Context(), id, receptionistID, hospID, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "attendance status updated"})
}
