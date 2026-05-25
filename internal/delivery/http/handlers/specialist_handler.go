package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type SpecialistHandler struct {
	referralUC iusecase.ReferralUseCase
	schedUC    iusecase.SchedulingUseCase
	triageUC   iusecase.TriageUseCase
	patientUC  iusecase.PatientUseCase
	mlUC       iusecase.MLUseCase
	arrivalUC  iusecase.ArrivalUseCase
}

func NewSpecialistHandler(referralUC iusecase.ReferralUseCase, schedUC iusecase.SchedulingUseCase, triageUC iusecase.TriageUseCase, patientUC iusecase.PatientUseCase, mlUC iusecase.MLUseCase, arrivalUC iusecase.ArrivalUseCase) *SpecialistHandler {
	return &SpecialistHandler{
		referralUC: referralUC,
		schedUC:    schedUC,
		triageUC:   triageUC,
		patientUC:  patientUC,
		mlUC:       mlUC,
		arrivalUC:  arrivalUC,
	}
}

// ListReferrals godoc
// @Summary      List Referrals for Specialist
// @Description  Get a paginated list of referrals forwarded to the specialist's hospital.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Visibility:** All referrals forwarded to the specialist's department, including REDIRECTED and REJECTED_AFTER_SEND.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_id query string false "Filter by patient ID"
// @Param        national_id query string false "Filter by patient national ID"
// @Param        sort_by query string false "Sort by field (created_at, updated_at)" default(created_at)
// @Param        sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals [get]
func (h *SpecialistHandler) ListReferrals(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
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
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}

	filter := irepository.ReferralFilter{
		Status:    c.Query("status"),
		Region:    c.Query("region"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Limit:     limit,
		Page:      page,
	}

	if pID := c.Query("patient_id"); pID != "" {
		parsedID, err := uuid.Parse(pID)
		if err == nil {
			filter.PatientID = &parsedID
		}
	}

	if nID := c.Query("national_id"); nID != "" && filter.PatientID == nil {
		pID, err := h.patientUC.LookupByNationalID(c.Request.Context(), nID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "failed to lookup patient"})
			return
		}
		if pID == nil {
			c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
				BaseResponse: dto.BaseResponse{Success: false, Message: "Patient not found"},
				Data:         []dto.ListReferralResponse{},
				Total:        0,
				Page:         page,
				PageSize:     limit,
			})
			return
		}
		filter.PatientID = pID
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

// ListApprovedReferrals godoc
// @Summary      List Approved Referrals for Specialist
// @Description  Returns referrals forwarded to the specialist's hospital with status ACCEPTED, SCHEDULED, or COMPLETED.
// @Description  **Roles:** SPECIALIST
// @Description  **Statuses:** ACCEPTED, SCHEDULED, COMPLETED (pre-applied filter).
// @Tags         Specialist
// @Produce      json
// @Param        limit       query int    false "Pagination limit" default(20)
// @Param        page        query int    false "Page number" default(1)
// @Param        patient_id  query string false "Filter by patient ID"
// @Param        national_id query string false "Filter by patient's national ID"
// @Param        region      query string false "Filter by patient region"
// @Param        sort_by     query string false "Sort by field (created_at, updated_at)" default(created_at)
// @Param        sort_order  query string false "Sort order (asc, desc)" default(desc)
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/approved [get]
func (h *SpecialistHandler) ListApprovedReferrals(c *gin.Context) {
	h.listFilteredReferrals(c, []entity.ReferralStatus{
		entity.StatusAccepted,
		entity.StatusScheduled,
		entity.StatusCompleted,
	})
}

// ListRejectedReferrals godoc
// @Summary      List Rejected Referrals for Specialist
// @Description  Returns referrals forwarded to the specialist's hospital with status REJECTED_BY_SPECIALIST, or REJECTED_AFTER_SEND.
// @Description  **Roles:** SPECIALIST
// @Description  **Statuses:** REJECTED_BY_SPECIALIST, REJECTED_AFTER_SEND (pre-applied filter).
// @Tags         Specialist
// @Produce      json
// @Param        limit       query int    false "Pagination limit" default(20)
// @Param        page        query int    false "Page number" default(1)
// @Param        patient_id  query string false "Filter by patient ID"
// @Param        national_id query string false "Filter by patient's national ID"
// @Param        region      query string false "Filter by patient region"
// @Param        sort_by     query string false "Sort by field (created_at, updated_at)" default(created_at)
// @Param        sort_order  query string false "Sort order (asc, desc)" default(desc)
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/rejected [get]
func (h *SpecialistHandler) ListRejectedReferrals(c *gin.Context) {
	h.listFilteredReferrals(c, []entity.ReferralStatus{
		entity.StatusRejectedBySpecialist,
		entity.StatusRejectedAfterSend,
	})
}

func (h *SpecialistHandler) listFilteredReferrals(c *gin.Context, statuses []entity.ReferralStatus) {
	if c.Query("status") != "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "status filter is not allowed on this endpoint; the statuses are pre-defined",
		})
		return
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	userIdVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	if hospID == uuid.Nil || specialistID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user scopes",
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

	filter := irepository.ReferralFilter{
		Statuses:  statuses,
		Region:    c.Query("region"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Limit:     limit,
		Page:      page,
	}

	if pID := c.Query("patient_id"); pID != "" {
		parsedID, err := uuid.Parse(pID)
		if err == nil {
			filter.PatientID = &parsedID
		}
	}

	if nID := c.Query("national_id"); nID != "" && filter.PatientID == nil {
		pID, err := h.patientUC.LookupByNationalID(c.Request.Context(), nID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "failed to lookup patient"})
			return
		}
		if pID == nil {
			c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
				BaseResponse: dto.BaseResponse{Success: false, Message: "Patient not found"},
				Data:         []dto.ListReferralResponse{},
				Total:        0,
				Page:         page,
				PageSize:     limit,
			})
			return
		}
		filter.PatientID = pID
	}

	referrals, total, err := h.referralUC.ListForSpecialist(c.Request.Context(), hospID, specialistID, filter)
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
// @Description  Get detailed information about a forwarded referral, including ML triage (`ml_severity_score`, `ml_severity_tier`, `ml_explanations`, `ml_model_version`, `ml_status`).
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** Referral status must be one of: FORWARDED, UNDER_SPECIALIST_REVIEW, ACCEPTED, SCHEDULED, COMPLETED, REJECTED_BY_SPECIALIST, REDIRECTED, REJECTED_AFTER_SEND, CANCELLED, or DECEASED.
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
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
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
		Redirections: toRedirectionResponseSlice(ref.Redirections),
	})
}

// Read godoc
// @Summary      Mark Referral as Read
// @Description  Acknowledge receipt and claim the referral for review.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** referral.status = FORWARDED or REDIRECTED.
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
// @Description  **Prerequisites:** Status must be UNDER_SPECIALIST_REVIEW (unless ML state is failed or stuck pending).
// @Description  **Ownership Check:** The specialist must be the one claimed/assigned to the referral.
// @Description  **State Transition:** Triggers a forced rerun check and queues the prediction update.
// @Description  **Common Errors:**
// @Description  - 400 Invalid format
// @Description  - 403 Forbidden (wrong hospital or not assigned)
// @Description  - 422 Unprocessable Entity (ML service not configured or daily limit exceeded)
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
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

// GetMLPrediction godoc
// @Summary      Get ML Prediction Details
// @Description  Get the machine learning prediction details for a specific referral, containing inputs and outputs.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Visibility:** The specialist must belong to the target hospital of the referral.
// @Description  **Prerequisites:** Referral must exist at the specialist's target hospital.
// @Description  **Common Errors:**
// @Description  - 400 Invalid format
// @Description  - 403 Forbidden (unauthorized hospital access)
// @Description  - 404 ML prediction not found
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.MLPredictionResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/ml-prediction [get]
func (h *SpecialistHandler) GetMLPrediction(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	pred, err := h.referralUC.GetMLPredictionForSpecialist(c.Request.Context(), id, hospID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Error:   "ML prediction not found for this referral",
			})
			return
		}
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.MLPredictionResponse{
		Success: true,
		Message: "ML prediction retrieved successfully",
		Data:    pred,
	})
}


// Release godoc
// @Summary      Release Referral (Unassign Self)
// @Description  Unassign self and return the referral to the hospital pool.
// @Description  **Roles:** RECEIVING_SPECIALIST (Assigned)
// @Description  **Prerequisites:** status = UNDER_SPECIALIST_REVIEW.
// @Description  **State Transition:** → FORWARDED (or → REDIRECTED if it was previously redirected).
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
// @Summary      Manual emergency schedule (Schedule-on-Demand, overbook allowed)
// @Description  Books a referral as an emergency for a specific date. This is the ONLY scheduling path allowed to consume overbook capacity.
// @Description
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description
// @Description  **Prerequisites:**
// @Description  - Referral.Status must be ACCEPTED or SCHEDULED.
// @Description  - TriageQueue.ArrivalStatus must be EXPECTED or MISSED (already-ARRIVED / ADMITTED patients are blocked).
// @Description  - Appointment date must be today or in the future.
// @Description  - Either Referral.ReferralForm.condition_at_referral = "critical" OR a non-empty justification must be provided.
// @Description
// @Description  **Capacity Rule (Schedule-on-Demand):**
// @Description  - booked count is read live from TriageQueue (Expected/Arrived/Admitted) via CountByDeptAndDate.
// @Description  - maxSlots = active CapacityOverride.NewLimit OR HospitalDepartment.StandardDailyLimit.
// @Description  - overbookLimit = HospitalDepartment.OverbookLimit.
// @Description  - Rejected when booked >= maxSlots + overbookLimit.
// @Description
// @Description  **State Transitions:**
// @Description  - Referral.Status: ACCEPTED/SCHEDULED -> SCHEDULED.
// @Description  - TriageQueue.ArrivalStatus: MISSED -> EXPECTED (rescheduled_from_missed=true) or unchanged.
// @Description  - TriageQueue.AppointmentDate set to the target date.
// @Description
// @Description  **Side Effects:**
// @Description  - Creates or updates the DailySchedule snapshot row for the (hospital, department, date).
// @Description  - Writes an audit log row with action EMERGENCY_SCHEDULE.
// @Description  - Queues SMS via NotifyScheduling / NotifyMissedReschedule.
// @Description  - Dispatches in-app event APPOINTMENT_SCHEDULED or MISSED_APPOINTMENT_RESCHEDULED.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid referral ID / past date / not critical and no justification
// @Description  - 401 Unauthorized
// @Description  - 500 capacity full even with overbook / DB error
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID (UUID)"
// @Param        body body dto.ManualEmergencyScheduleRequest true "Emergency scheduling details"
// @Success      200 {object} dto.SchedulingResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
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

	wasMissed, err := h.schedUC.ManualEmergencySchedule(c.Request.Context(), referralID, date, req.Justification, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SchedulingResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Emergency appointment scheduled successfully"},
		RescheduledFromMissed: wasMissed,
	})
}

// MLSeverityOverride godoc
// @Summary      Manual ML Severity Override
// @Description  Allows a specialist to manually override the machine learning model's severity score for a patient referral.
// @Description  This action sets the referral's machine learning pipeline status (`ml_status`) to MANUAL, updates the referral's severity score,
// @Description  deletes any existing automated ML predictions for this referral to ensure data integrity, and recalculates the priority composite score in the triage queue.
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Prerequisites:** The referral must belong to the specialist's target hospital and department.
// @Description  **Side Effects:** Changes `ml_status` to MANUAL, sets `triage_status` to OVERRIDDEN, clears active ML prediction association, and updates the triage composite score.
// @Description  **Common Errors:**
// @Description  - 400 Bad Request: Invalid referral ID format or malformed request payload
// @Description  - 401 Unauthorized: Invalid or missing authorization token
// @Description  - 403 Forbidden: Specialist does not belong to the target hospital/department of the referral
// @Description  - 500 Internal Server Error: Database transaction failures
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID (UUID)"
// @Param        body body dto.MLSeverityOverrideRequest true "Manual override parameters including new score and clinical justification"
// @Success      200 {object} dto.BaseResponse "Severity score manually overridden and triage queue successfully updated"
// @Failure      400 {object} dto.ErrorResponse "Invalid inputs / bad request format"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized access"
// @Failure      403 {object} dto.ErrorResponse "Forbidden operation / mismatching hospital scopes"
// @Failure      500 {object} dto.ErrorResponse "Internal server / database transaction error"
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/ml-severity-override [post]
func (h *SpecialistHandler) MLSeverityOverride(c *gin.Context) {
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

	var req dto.MLSeverityOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.mlUC.MLSeverityOverride(c.Request.Context(), referralID, userID, req.Score, req.Justification); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Severity score overridden successfully"})
}

// GetTriageQueue godoc
// @Summary      Get Triage Queue (filterable)
// @Description  Returns the prioritized triage queue scoped to the specialist's hospital. Supports filtering by department, arrival/referral status, doctor assignment, patient identifiers, and configurable sort. Terminal referrals (COMPLETED / DECEASED / CANCELLED / REJECTED_* / REDIRECTED) are excluded by default; pass include_terminal=true for audit views.
// @Description
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Scope:** Hospital-wide (caller's hospital from JWT).
// @Description  **Default sort:** composite_score DESC (highest priority first).
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        limit query int false "Pagination limit (1-100)" default(20)
// @Param        page query int false "Page number (1-based)" default(1)
// @Param        department_id query string false "Filter by HospitalDepartment ID"
// @Param        arrival_status query string false "Comma-separated arrival statuses: EXPECTED,ARRIVED,ADMITTED,MISSED"
// @Param        referral_status query string false "Comma-separated referral statuses: ACCEPTED,SCHEDULED"
// @Param        has_doctor_assigned query bool false "Filter by treating-doctor assignment"
// @Param        patient_id query string false "Filter by patient UUID"
// @Param        national_id query string false "Filter by patient national ID (hashed server-side)"
// @Param        sort_by query string false "Sort field: composite_score|appointment_date|created_at" default(composite_score)
// @Param        sort_order query string false "asc|desc" default(desc)
// @Param        include_terminal query bool false "Include terminal-status referrals" default(false)
// @Success      200 {object} dto.TriageListEnvelope
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/triage-queue [get]
func (h *SpecialistHandler) GetTriageQueue(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "missing hospital scope"})
		return
	}

	filter := parseTriageListFilter(c)
	filter.HospitalID = hospID

	items, total, err := h.triageUC.ListTriageFiltered(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TriageListEnvelope{
		Success: true,
		Data:    items,
		Total:   total,
		Page:    pageFromOffset(filter.Limit, filter.Offset),
		Limit:   filter.Limit,
		HasMore: hasMorePage(filter.Offset, filter.Limit, total),
	})
}

// GetTriageDetail godoc
// @Summary      Get Triage Detail (specialist view)
// @Description  Returns the rich clinical detail for a referral on the triage queue: full PII, vitals, ICD diagnoses, clinical summary, ML severity, treating + consulting doctors with grant timestamps, arrival_history timeline derived from audit_logs, and the role-aware available_actions bitmap.
// @Description
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description  **Path param:** {id} = REFERRAL UUID (not the queue UUID).
// @Description  **Available actions** are computed from the same guards the action endpoints enforce, so a button enabled here will not 400 when invoked.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 404 Referral not in triage queue
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral UUID"
// @Success      200 {object} dto.TriageDetailSpecialistResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/triage-detail [get]
func (h *SpecialistHandler) GetTriageDetail(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id"})
		return
	}
	userIDVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else if uid, ok := userIDVal.(*uuid.UUID); ok && uid != nil {
		userID = *uid
	}

	resp, err := h.triageUC.GetTriageDetailForSpecialist(c.Request.Context(), referralID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "referral not in triage queue"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
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
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(uuid.UUID); ok {
		deptID = dID
	} else if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
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
// @Summary      Schedule appointment (routine, Schedule-on-Demand)
// @Description  Books or reschedules a referral for a specific date under the routine capacity rule (no overbooking).
// @Description
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description
// @Description  **Prerequisites:**
// @Description  - Referral.Status must be ACCEPTED or SCHEDULED.
// @Description  - TriageQueue.ArrivalStatus must be EXPECTED or MISSED.
// @Description  - Appointment date must be today or in the future.
// @Description
// @Description  **Capacity Rule (Schedule-on-Demand):**
// @Description  - booked = live TriageQueue count (Expected/Arrived/Admitted) for that (hospital, department, date).
// @Description  - maxSlots = active CapacityOverride.NewLimit OR HospitalDepartment.StandardDailyLimit.
// @Description  - Rejected when booked >= maxSlots. Overbook capacity is NEVER used here - call /emergency-schedule for that.
// @Description
// @Description  **State Transitions:**
// @Description  - Referral.Status: ACCEPTED/SCHEDULED -> SCHEDULED.
// @Description  - TriageQueue.ArrivalStatus: MISSED -> EXPECTED (rescheduled_from_missed=true).
// @Description  - TriageQueue.AppointmentDate set.
// @Description
// @Description  **Side Effects:**
// @Description  - Creates / updates DailySchedule snapshot row.
// @Description  - Writes audit log.
// @Description  - Queues SMS and in-app notification (APPOINTMENT_SCHEDULED or MISSED_APPOINTMENT_RESCHEDULED).
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid referral ID / past date / patient already arrived or admitted
// @Description  - 401 Unauthorized
// @Description  - 500 capacity reached - emergency override required / DB error
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID (UUID)"
// @Param        body body dto.SchedulingRequest true "Scheduling details"
// @Success      200 {object} dto.SchedulingResponse
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
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	var req dto.SchedulingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	wasMissed, err := h.schedUC.ScheduleAppointment(c.Request.Context(), referralID, userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SchedulingResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Appointment scheduled successfully"},
		RescheduledFromMissed: wasMissed,
	})
}

// RedirectReferral godoc
// @Summary      Redirect Referral to another hospital
// @Description  Redirect an active referral (ACCEPTED/UNDER_SPECIALIST_REVIEW) to another hospital in the network.
// @Description  **The target hospital must have the same department as the referral’s current target_dept_id. If the desired hospital lacks that department, call PUT /specialist/referrals/{id}/department first.**
// @Description  **Roles:** RECEIVING_SPECIALIST (non-primary hospital)
// @Description  **Constraints:** No loops allowed, target must be in outgoing network.
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.RedirectReferralRequest true "Redirection details (optional department_id to update target department)"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/redirect [post]
func (h *SpecialistHandler) RedirectReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req dto.RedirectReferralRequest
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

	if err := h.referralUC.RedirectReferral(c.Request.Context(), id, specialistID, hospID, req.TargetHospitalID, req.Reason, req.DepartmentID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Referral redirected successfully"})
}

// ListRedirectionOptions godoc
// @Summary      List possible hospitals for redirection
// @Description  Get a list of hospitals in the network that haven't handled this referral yet.
// @Description  **If the returned list is empty, the specialist should call PUT /specialist/referrals/{id}/department to select a different target department before retrying.**
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        department_id query string false "Optional: filter hospitals by this department ID instead of the referral's current department"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/redirect-options [get]
func (h *SpecialistHandler) ListRedirectionOptions(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var filterDeptID *uuid.UUID
	if depIDStr := c.Query("department_id"); depIDStr != "" {
		if dID, err := uuid.Parse(depIDStr); err == nil {
			filterDeptID = &dID
		}
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

	hospitals, err := h.referralUC.ListRedirectionOptions(c.Request.Context(), id, specialistID, hospID, filterDeptID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    hospitals,
	})
}

// ChangeDepartment godoc
// @Summary      Change Referral Target Department
// @Description  Updates the target department of a referral as long as it is ACCEPTED or UNDER_SPECIALIST_REVIEW.
// @Description  The new department must exist at the current hospital. This unlocks new redirect options.
// @Description  **After updating the department, call GET /specialist/referrals/{id}/redirect-options again to see the expanded list of eligible hospitals.**
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Tags         Specialist
// @Accept       json
// @Produce      json
// @Param        id   path     string                              true  "Referral ID"
// @Param        body body     dto.ChangeDepartmentRequest         true  "Department change payload"
// @Success      200  {object} dto.BaseResponse
// @Failure      400  {object} dto.ErrorResponse
// @Failure      401  {object} dto.ErrorResponse
// @Failure      422  {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/department [put]
func (h *SpecialistHandler) ChangeDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral ID"})
		return
	}

	specialistIDVal, _ := c.Get("userID")
	specialistID := uuid.Nil
	if uID, ok := specialistIDVal.(uuid.UUID); ok {
		specialistID = uID
	} else if uID, ok := specialistIDVal.(*uuid.UUID); ok && uID != nil {
		specialistID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	var req dto.ChangeDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid request body: " + err.Error()})
		return
	}

	if err := h.referralUC.ChangeDepartment(c.Request.Context(), id, specialistID, hospID, req.DepartmentID); err != nil {
		status := http.StatusUnprocessableEntity
		if err.Error() == "unauthorized: referral is not at your hospital" {
			status = http.StatusForbidden
		}
		c.JSON(status, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Department updated successfully"})
}

// ReturnToTriage godoc
// @Summary      Return missed patient to triage
// @Description  Resets a MISSED patient back to the waiting pool so they can be re-scheduled. Under Schedule-on-Demand this also frees the slot for the original date automatically (capacity is read live from TriageQueue, and Missed rows are excluded from the count).
// @Description
// @Description  **Roles:** RECEIVING_SPECIALIST
// @Description
// @Description  **Prerequisites:** Valid TriageQueue ID; record must currently be MISSED.
// @Description
// @Description  **State Transitions:**
// @Description  - TriageQueue.ArrivalStatus: MISSED -> EXPECTED.
// @Description  - TriageQueue.AppointmentDate: cleared.
// @Description  - TriageQueue.AssignedDoctorID / DoctorAssignedAt: cleared.
// @Description  - Referral.Status: -> ACCEPTED (so it is eligible for batch / manual scheduling again).
// @Description
// @Description  **Side Effects:**
// @Description  - The slot count for the original date drops by 1 on the next read because the row is no longer Expected/Arrived/Admitted.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid ID / record not MISSED
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        id path string true "TriageQueue ID (UUID)"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/return-to-triage [post]
func (h *SpecialistHandler) ReturnToTriage(c *gin.Context) {
	triageQueueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid triage queue id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(uuid.UUID); ok {
		deptID = dID
	} else if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}

	if err := h.arrivalUC.ReturnToTriage(c.Request.Context(), triageQueueID, userID, hospID, deptID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient successfully returned to triage"})
}

// ScheduleOptions godoc
// @Summary      Get next N routine scheduling options for a referral
// @Description  Returns up to N upcoming days starting at today + system_configs.buffer_days where the referral's target department still has routine capacity. Each entry includes max_slots, booked_slots, available_slots, overbook_limit, and whether an active capacity override is in effect.
// @Description
// @Description  **Roles:** RECEIVING_SPECIALIST, REFERRING_DOCTOR
// @Description
// @Description  **Prerequisites:** Referral must exist and have a target hospital + department set.
// @Description
// @Description  **Side Effects:** None. Read-only. The same rule is applied when actually booking, so callers can rely on the dates to be schedulable at the moment of the call (subject to concurrent bookings).
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid referral ID / days
// @Description  - 401 Unauthorized
// @Description  - 404 referral not found
// @Description  - 500 Internal Server Error
// @Tags         Specialist
// @Produce      json
// @Param        id   path  string true  "Referral ID (UUID)"
// @Param        days query int    false "Number of days to scan starting at today + buffer_days (1-60, default 14)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.BaseResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/specialist/referrals/{id}/schedule-options [get]
func (h *SpecialistHandler) ScheduleOptions(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid referral ID"})
		return
	}

	days := 14
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}

	options, err := h.schedUC.ListScheduleOptions(c.Request.Context(), referralID, days)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.BaseResponse{Success: false, Message: "referral not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"referral_id": referralID,
		"days_window": days,
		"data":        options,
	})
}
