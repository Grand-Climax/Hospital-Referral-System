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

type SpecialistHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewSpecialistHandler(referralUC iusecase.ReferralUseCase) *SpecialistHandler {
	return &SpecialistHandler{referralUC: referralUC}
}

// ListReferrals godoc
// @Summary      List Referrals for Specialist
// @Description  Get a paginated list of referrals forwarded to the specialist's department.
// @Tags         Specialist Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals [get]
func (h *SpecialistHandler) ListReferrals(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user scopes"})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID, _ := userIdVal.(uuid.UUID)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	statusFilter := c.Query("status")

	referrals, total, err := h.referralUC.ListForSpecialist(c.Request.Context(), hospID, specialistID, limit, page, statusFilter)
	if err != nil {
		log.Printf("[SpecialistHandler.List] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No referrals found in your department",
			},
			Data:     []dto.ListReferralResponse{},
			Total:    0,
			Page:     page,
			PageSize: limit,
		})
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
		patientNameFirst := ""
		patientNameMiddle := ""
		patientNameLast := ""
		if r.Patient != nil {
			patientNameFirst = r.Patient.FirstName
			patientNameMiddle = r.Patient.MiddleName
			patientNameLast = r.Patient.LastName
		}

		condition := ""
		if r.ReferralForm != nil {
			condition = r.ReferralForm.ConditionAtReferral
		}

		responseData = append(responseData, dto.ListReferralResponse{
			ID:                  r.ID,
			PatientFirstName:    patientNameFirst,
			PatientMiddleName:   patientNameMiddle,
			PatientLastName:     patientNameLast,
			Department:          r.TargetDeptID.String(),
			Date:                r.CreatedAt.Format("2006-01-02"),
			Status:              string(r.Status),
			ICDCode:             icd,
			Diagnosis:           diag,
			ConditionAtReferral: condition,
		})
	}

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referrals retrieved successfully",
		},
		Data:         responseData,
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

// GetReferral godoc
// @Summary      Get Referral Details for Specialist
// @Description  Get detailed information about a specific referral.
// @Tags         Specialist Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} entity.Referral
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id} [get]
func (h *SpecialistHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	ref, err := h.referralUC.GetDetailsForSpecialist(c.Request.Context(), id, hospID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.SuccessPayload(ref, "Referral details retrieved successfully"))
}

// Read godoc
// @Summary      Mark Referral as Read
// @Description  Acknowledge receipt and claim the referral for review by the specialist.
// @Tags         Specialist Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/read [post]
func (h *SpecialistHandler) Read(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistRead(c.Request.Context(), id, specialistID, hospID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Referral marked as read and claimed",
	})
}

// Accept godoc
// @Summary      Accept Referral
// @Description  Accept an incoming referral and assign a severity score.
// @Tags         Specialist Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body map[string]float64 false "Severity Score (key: severity_score)"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/accept [post]
func (h *SpecialistHandler) Accept(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	var req struct {
		SeverityScore *float64 `json:"severity_score,omitempty"`
	}
	_ = c.ShouldBindJSON(&req)

	userIdVal, _ := c.Get("userID")
	specialistID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistAccept(c.Request.Context(), id, specialistID, hospID, req.SeverityScore); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Referral accepted and severity score assigned",
	})
}

// Reject godoc
// @Summary      Reject Referral
// @Description  Reject an incoming referral.
// @Tags         Specialist Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Rejection Reason"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/reject [post]
func (h *SpecialistHandler) Reject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	var req dto.RejectDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistReject(c.Request.Context(), id, specialistID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Referral rejected",
	})
}

// RerunML godoc
// @Summary      Rerun ML Prediction
// @Description  Rerun the machine learning prediction for a specific referral.
// @Tags         Specialist Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/rerun-ml [post]
func (h *SpecialistHandler) RerunML(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistRerunML(c.Request.Context(), id, specialistID, hospID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ML prediction rerun successfully",
	})
}
