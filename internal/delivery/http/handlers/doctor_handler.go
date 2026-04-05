package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
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
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals [get]
func (h *DoctorHandler) ListReferrals(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID, ok := userIdVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user",
		})
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No referrals found matching your criteria",
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
// @Summary      Get Referral Details for Doctor
// @Description  Get detailed information about a specific referral created by the doctor.
// @Tags         Doctor Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id} [get]
func (h *DoctorHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid id format",
		})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	ref, err := h.referralUC.GetDetailsForDoctor(c.Request.Context(), id, doctorID)
	if err != nil {
		log.Printf("[DoctorHandler.GetReferral] error: %v", err)
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ReferralDetailResponse{
		Referral: *ref,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referral details retrieved successfully",
		},
	})
}

// GetStats godoc
// @Summary      Get Doctor Dashboard Stats
// @Description  Get a summary of referral counts (Total, Pending, Accepted, Critical) for the doctor's dashboard.
// @Tags         Doctor Dashboard
// @Produce      json
// @Success      200 {object} dto.DoctorDashboardStatsResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/stats [get]
func (h *DoctorHandler) GetStats(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	stats, err := h.referralUC.GetDoctorDashboardStats(c.Request.Context(), doctorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.DoctorDashboardStatsResponse{
		DoctorDashboardStats: *stats,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Dashboard statistics retrieved successfully",
		},
	})
}

// GetLatestPending godoc
// @Summary      Get Latest Pending Referrals
// @Description  Get a list of the most recent pending referrals for the doctor's dashboard.
// @Tags         Doctor Dashboard
// @Produce      json
// @Param        limit query int false "Number of records to fetch" default(5)
// @Success      200 {object} dto.LatestPendingReferralsResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/latest-pending [get]
func (h *DoctorHandler) GetLatestPending(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID, ok := userIdVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user session",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		// If negative or non-numeric, default to 5 as per user request to "handle" it
		limit = 5
	}

	referrals, err := h.referralUC.GetLatestPendingReferrals(c.Request.Context(), doctorID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if len(referrals) == 0 {
		c.JSON(http.StatusOK, dto.BaseResponse{
			Success: false,
			Message: "No pending referrals found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.LatestPendingReferralsResponse{
		Data: referrals,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Latest pending referrals retrieved successfully",
		},
	})
}

// CreateOrSubmit godoc
// @Summary      Create or Submit Referral
// @Description  Create a new referral draft or submit it directly based on the status provided.
// @Tags         Doctor Referrals
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateReferralRequest true "Referral Details"
// @Success      201 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Default to SUBMITTED if status is missing
	if req.Status == "" {
		req.Status = string(entity.StatusSubmitted)
	}

	// Validate status is only DRAFT or SUBMITTED
	if req.Status != string(entity.StatusDraft) && req.Status != string(entity.StatusSubmitted) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid status: only DRAFT or SUBMITTED are allowed during creation",
		})
		return
	}

	ref, err := h.referralUC.CreateDraftOrSubmit(c.Request.Context(), doctorID, hospID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	msg := "New referral draft created (DRAFT)"
	if req.Status == string(entity.StatusSubmitted) {
		msg = "New referral submitted for review (SUBMITTED)"
	}
	c.JSON(http.StatusCreated, dto.ReferralDetailResponse{
		Referral: *ref,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: msg,
		},
	})
}

// UpdateAndResubmit godoc
// @Summary      Update (Draft) or Submit Referral
// @Description  Allows updating a referral. Saving without the /submit suffix persists changes as a DRAFT. Adding /submit finalizes the referral and moves it to SUBMITTED status for liaison processing.
// @Tags         Doctor Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.UpdateReferralRequest true "Updated Referral Details"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id} [put]
// @Router       /api/v1/doctor/referrals/{id}/submit [put]
func (h *DoctorHandler) UpdateAndResubmit(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	var rawBody map[string]interface{}
	if err := c.ShouldBindJSON(&rawBody); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if _, exists := rawBody["status"]; exists {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "status field is not allowed in update; use the specific route to submit",
		})
		return
	}

	jsonData, _ := json.Marshal(rawBody)
	var req dto.UpdateReferralRequest
	_ = json.Unmarshal(jsonData, &req)

	submit := false
	fullPath := c.FullPath()
	if strings.HasSuffix(fullPath, "/submit") {
		submit = true
	}

	ref, err := h.referralUC.UpdateAndResubmit(c.Request.Context(), id, doctorID, req, submit)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	msg := "Referral draft changes saved (DRAFT)"
	if submit {
		msg = "Referral submitted for review (SUBMITTED)"
	}
	c.JSON(http.StatusOK, dto.ReferralDetailResponse{
		Referral: *ref,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: msg,
		},
	})
}

// Cancel godoc
// @Summary      Cancel Referral
// @Description  Cancel an active referral that has not yet been processed (must be in DRAFT or NEED_REVISION status).
// @Tags         Doctor Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.CancelReferralRequest true "Cancellation Reason"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id}/cancel [post]
func (h *DoctorHandler) Cancel(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID, _ := userIdVal.(uuid.UUID)

	var req dto.CancelReferralRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.referralUC.CancelReferral(c.Request.Context(), id, doctorID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral cancelled successfully",
	})
}
