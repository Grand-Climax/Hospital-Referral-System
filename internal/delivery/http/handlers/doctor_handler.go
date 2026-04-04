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

type DoctorHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewDoctorHandler(referralUC iusecase.ReferralUseCase) *DoctorHandler {
	return &DoctorHandler{referralUC: referralUC}
}

// ListReferrals godoc
// @Summary      List Referrals for Doctor
// @Description  Get a paginated list of referrals created by the authenticated doctor.
// @Tags         Doctor Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals [get]
func (h *DoctorHandler) ListReferrals(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID, ok := userIdVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	statusFilter := c.Query("status")

	referrals, total, err := h.referralUC.ListForDoctor(c.Request.Context(), doctorID, limit, page, statusFilter)
	if err != nil {
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
		Data:     responseData,
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}

// GetReferral godoc
// @Summary      Get Referral Details for Doctor
// @Description  Get detailed information about a specific referral created by the doctor.
// @Tags         Doctor Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} entity.Referral
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id} [get]
func (h *DoctorHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	ref, err := h.referralUC.GetDetailsForDoctor(c.Request.Context(), id, doctorID)
	if err != nil {
		log.Printf("[DoctorHandler.GetReferral] error: %v", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ref)
}

// CreateOrSubmit godoc
// @Summary      Create or Submit Referral
// @Description  Create a new referral draft or submit it directly based on the status provided.
// @Tags         Doctor Referrals
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateReferralRequest true "Referral Details"
// @Success      201 {object} entity.Referral
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals [post]
func (h *DoctorHandler) CreateOrSubmit(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	var req dto.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ref, err := h.referralUC.CreateDraftOrSubmit(c.Request.Context(), doctorID, hospID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ref)
}

// UpdateAndResubmit godoc
// @Summary      Update and Resubmit Referral
// @Description  Update a previously returned/draft referral and optionally resubmit it.
// @Tags         Doctor Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.CreateReferralRequest true "Updated Referral Details"
// @Success      200 {object} entity.Referral
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id} [put]
func (h *DoctorHandler) UpdateAndResubmit(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	var req dto.CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ref, err := h.referralUC.UpdateAndResubmit(c.Request.Context(), id, doctorID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ref)
}

// Cancel godoc
// @Summary      Cancel Referral
// @Description  Cancel an active referral that has not yet been processed.
// @Tags         Doctor Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body map[string]string true "Cancellation Reason (key: reason)"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id}/cancel [post]
func (h *DoctorHandler) Cancel(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	var dto struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&dto)

	if err := h.referralUC.CancelReferral(c.Request.Context(), id, doctorID, dto.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "referral cancelled successfully"})
}
