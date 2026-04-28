package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type SpecialistHandler struct {
	referralUC iusecase.ReferralUseCase
	schedUC    iusecase.SchedulingUseCase
	triageUC   iusecase.TriageUseCase
}

func NewSpecialistHandler(referralUC iusecase.ReferralUseCase, schedUC iusecase.SchedulingUseCase, triageUC iusecase.TriageUseCase) *SpecialistHandler {
	return &SpecialistHandler{
		referralUC: referralUC,
		schedUC:    schedUC,
		triageUC:   triageUC,
	}
}

// ListReferrals godoc
// @Summary      List Referrals for Specialist
// @Description  Get a paginated list of referrals forwarded to the specialist's hospital.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Visibility:** All referrals forwarded to the specialist's specific department.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name (any order)"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals [get]
func (h *SpecialistHandler) ListReferrals(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user scopes",
		})
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

	filter := irepository.ReferralFilter{
		Status:      c.Query("status"),
		Region:      c.Query("region"),
		PatientName: c.Query("patient_name"),
		Sort:        c.Query("sort"),
		Limit:       limit,
		Page:        page,
	}

	if filter.Status != "" && !h.referralUC.IsValidStatus(filter.Status) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "forbidden: unknown or invalid referral status",
		})
		return
	}

	referrals, total, err := h.referralUC.ListForSpecialist(c.Request.Context(), hospID, specialistID, filter)
	if err != nil {
		log.Printf("[SpecialistHandler.List] error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No referrals found in your hospital",
			},
			Data:     []dto.ListReferralResponse{},
			Total:    0,
			Page:     page,
			PageSize: limit,
		})
		return
	}

	responseData := toListReferralResponseSlice(referrals)

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
// @Description  Get detailed information about a forwarded referral.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** Status must be FORWARDED or later.
// @Description  **Common Errors:**
// @Description  - 400 Invalid format
// @Description  - 403 Forbidden (wrong hospital)
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id} [get]
func (h *SpecialistHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	ref, err := h.referralUC.GetDetailsForSpecialist(c.Request.Context(), id, hospID)
	if err != nil {
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

// Read godoc
// @Summary      Mark Referral as Read
// @Description  Acknowledge receipt and claim the referral for review.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** referral.status = FORWARDED.
// @Description  **State Transition:** → UNDER_SPECIALIST_REVIEW, sets specialist_id.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 unauthorized hospital access
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/read [post]
func (h *SpecialistHandler) Read(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistRead(c.Request.Context(), id, specialistID, hospID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral marked as read and claimed",
	})
}

// Accept godoc
// @Summary      Accept Referral
// @Description  Accept the referral and place it in the triage queue.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** status = UNDER_SPECIALIST_REVIEW; severity score must be set (manual or ML).
// @Description  **State Transition:** → ACCEPTED; lands in triage queue.
// @Description  **Gatekeepers:** Severity gate (manual or ML score must exist).
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 422 (no severity score)
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body map[string]float64 false "Severity Score (key: severity_score)"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/accept [post]
func (h *SpecialistHandler) Accept(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req struct {
		SeverityScore *float64 `json:"severity_score,omitempty"`
	}
	_ = c.ShouldBindJSON(&req)

	userIdVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistAccept(c.Request.Context(), id, specialistID, hospID, req.SeverityScore); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Trigger Triage Queue Placement
	if err := h.triageUC.LandInQueue(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Referral accepted but failed to land in triage queue: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral accepted and severity score assigned",
	})
}

// Reject godoc
// @Summary      Reject Referral
// @Description  Reject the referral back to the referring hospital.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** status = UNDER_SPECIALIST_REVIEW.
// @Description  **State Transition:** → REJECTED_BY_SPECIALIST.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 unauthorized hospital access
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Rejection Reason"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/reject [post]
func (h *SpecialistHandler) Reject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req dto.RejectDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistReject(c.Request.Context(), id, specialistID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral rejected",
	})
}

// RerunML godoc
// @Summary      Rerun ML Prediction
// @Description  Rerun the machine learning prediction for a specific referral.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** Status must be FORWARDED or UNDER_SPECIALIST_REVIEW.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 unauthorized hospital access
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/rerun-ml [post]
func (h *SpecialistHandler) RerunML(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistRerunML(c.Request.Context(), id, specialistID, hospID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "ML prediction rerun successfully",
	})
}

// Release godoc
// @Summary      Release Referral (Unassign Self)
// @Description  Unassign self and return the referral to the hospital pool.
// @Description  **Roles:** RECEIVING_SPECIALIST (Assigned)
// @Description  **Prerequisites:** status = UNDER_SPECIALIST_REVIEW.
// @Description  **State Transition:** → FORWARDED.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 unauthorized access
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Reason for Release (Reason field reused)"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/release [post]
func (h *SpecialistHandler) Release(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req dto.RejectDTO // Reuse RejectDTO for the reason
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.SpecialistRelease(c.Request.Context(), id, specialistID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral successfully released back to the hospital pool",
	})
}

// ManualEmergencySchedule godoc
// @Summary      Manual Emergency Scheduling
// @Description  Schedule an emergency appointment bypassing buffer days and allowing overbooking.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** referral must be accepted; condition must be `critical` OR justification provided.
// @Description  **State Transition:** Sets appointment_date, bypasses buffer, allows overbooking.
// @Description  **Gatekeepers:** Allows overbooking up to `overbook_limit`.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 500 internal error
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.ManualEmergencyScheduleRequest true "Scheduling details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/emergency-schedule [post]
func (h *SpecialistHandler) ManualEmergencySchedule(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	var req dto.ManualEmergencyScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.AppointmentDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid date format, use YYYY-MM-DD"})
		return
	}

	if err := h.schedUC.ManualEmergencySchedule(c.Request.Context(), referralID, date, req.Justification, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Emergency appointment scheduled successfully"})
}

// SetManualSeverity godoc
// @Summary      Set Manual Severity Score
// @Description  Manually set the severity score for a referral. Overrides ML score and updates triage queue.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** referral must exist and be under the specialist's purview.
// @Description  **Side Effect:** Overrides ML score, updates triage queue composite score.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 unauthorized hospital access
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.SetManualSeverityRequest true "Severity details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/triage-severity [post]
func (h *SpecialistHandler) SetManualSeverity(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	var req dto.SetManualSeverityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.triageUC.SetManualSeverity(c.Request.Context(), referralID, userID, req.Score, req.Justification); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Severity score overridden successfully"})
}

// GetTriageQueue godoc
// @Summary      Get Triage Queue
// @Description  Get the current prioritized triage queue for the specialist's hospital.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** Authenticated session in a hospital.
// @Description  **Gatekeepers:** Sorting based on severity score and waiting time.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        limit query int false "Pagination limit" default(50)
// @Param        page query int false "Page number" default(1)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/triage-queue [get]
func (h *SpecialistHandler) GetTriageQueue(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	} else if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	offset := (page - 1) * limit

	queues, total, err := h.triageUC.ListForTriage(c.Request.Context(), hospID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    queues,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetCapacity godoc
// @Summary      Get Department Capacity Status
// @Description  View capacity slots for the next N days.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** Authenticated session in a hospital/department.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        days query int false "Number of days to view" default(14)
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/capacity [get]
func (h *SpecialistHandler) GetCapacity(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))

	status, err := h.schedUC.GetCapacityStatus(c.Request.Context(), hospID, deptID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// Schedule godoc
// @Summary      Schedule Appointment
// @Description  Manually assigns an appointment date to a referral. Use this for routine scheduling after acceptance.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** status = ACCEPTED.
// @Description  **State Transition:** → SCHEDULED.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 500 internal error
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.SchedulingRequest true "Scheduling details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/schedule [post]
func (h *SpecialistHandler) Schedule(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid referral ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	var req dto.SchedulingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	if err := h.schedUC.ScheduleAppointment(c.Request.Context(), referralID, userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Appointment scheduled successfully"})
}
