package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type LiaisonHandler struct {
	referralUC iusecase.ReferralUseCase
	patientUC  iusecase.PatientUseCase
}

func NewLiaisonHandler(referralUC iusecase.ReferralUseCase, patientUC iusecase.PatientUseCase) *LiaisonHandler {
	return &LiaisonHandler{
		referralUC: referralUC,
		patientUC:  patientUC,
	}
}

// ListOutgoing godoc
// @Summary      List Outgoing Referrals for Liaison
// @Description  Get a paginated list of referrals sent FROM the liaison's hospital.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** Must be a liaison at the sender hospital.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Liaison
// @Produce      json
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
// @Router       /api/v1/liaison/referrals [get]
func (h *LiaisonHandler) ListOutgoing(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes"})
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

	referrals, total, err := h.referralUC.ListOutgoingForLiaison(c.Request.Context(), hospID, filter)
	if err != nil {
		log.Printf("[LiaisonHandler.ListOutgoing] error: %v", err)
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
				Message: "No outgoing referrals found",
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
			Message: "Outgoing referrals retrieved successfully",
		},
		Data:         responseData,
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

// ListApprovedReferrals godoc
// @Summary      List Approved Referrals for Liaison
// @Description  Returns outgoing referrals from the hospital with status ACCEPTED, SCHEDULED, or COMPLETED.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Statuses:** ACCEPTED, SCHEDULED, COMPLETED (pre-applied filter).
// @Tags         Liaison
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
// @Router       /api/v1/liaison/referrals/approved [get]
func (h *LiaisonHandler) ListApprovedReferrals(c *gin.Context) {
	h.listFilteredReferrals(c, []entity.ReferralStatus{
		entity.StatusAccepted,
		entity.StatusScheduled,
		entity.StatusCompleted,
	})
}

// ListRejectedReferrals godoc
// @Summary      List Rejected Referrals for Liaison
// @Description  Returns outgoing referrals from the hospital with status REJECTED_BY_LIAISON, REJECTED_BY_SPECIALIST, or REJECTED_AFTER_SEND.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Statuses:** REJECTED_BY_LIAISON, REJECTED_BY_SPECIALIST, REJECTED_AFTER_SEND (pre-applied filter).
// @Tags         Liaison
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
// @Router       /api/v1/liaison/referrals/rejected [get]
func (h *LiaisonHandler) ListRejectedReferrals(c *gin.Context) {
	h.listFilteredReferrals(c, []entity.ReferralStatus{
		entity.StatusRejectedByLiaison,
		entity.StatusRejectedBySpecialist,
		entity.StatusRejectedAfterSend,
	})
}

func (h *LiaisonHandler) listFilteredReferrals(c *gin.Context, statuses []entity.ReferralStatus) {
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
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes"})
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

	referrals, total, err := h.referralUC.ListOutgoingForLiaison(c.Request.Context(), hospID, filter)
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

// DEPRECATED: ListIncoming is now handled via specialists or specific monitoring tools.
// ListIncoming godoc
// @Summary      List Incoming Referrals for Liaison
// @Description  Get a paginated list of referrals sent TO the liaison's hospital for monitoring.
// @Description  **Visibility:** Includes referral statuses FORWARDED, UNDER_SPECIALIST_REVIEW, ACCEPTED, SCHEDULED, COMPLETED, REJECTED_BY_SPECIALIST, REDIRECTED, REJECTED_AFTER_SEND, CANCELLED, and DECEASED. (MISSED / ARRIVED / ADMITTED are arrival_status values, surfaced separately on each row.)
// @Tags         Liaison
// @Produce      json
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
// @Router       /api/v1/liaison/referrals/incoming [get]
func (h *LiaisonHandler) ListIncoming(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes"})
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

	referrals, total, err := h.referralUC.ListIncomingForLiaison(c.Request.Context(), hospID, filter)
	if err != nil {
		log.Printf("[LiaisonHandler.ListIncoming] error: %v", err)
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
				Message: "No incoming referrals found",
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
		patientRegion := ""
		if r.Patient != nil {
			patientNameFirst = r.Patient.FirstNamePlain
			patientNameMiddle = r.Patient.MiddleNamePlain
			patientNameLast = r.Patient.LastNamePlain
			if r.Patient.HomeRegion != nil {
				patientRegion = string(*r.Patient.HomeRegion)
			}
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
			PatientRegion:       patientRegion,
			Department:          r.TargetDeptID.String(),
			Status:              string(r.Status),
			ICDCode:             icd,
			Diagnosis:           diag,
			ConditionAtReferral: condition,
			CreatedAt:           r.CreatedAt,
			UpdatedAt:           r.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Incoming referrals retrieved successfully",
		},
		Data:         responseData,
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

// GetReferral godoc
// @Summary      Get Referral Details for Liaison
// @Description  Get detailed information about an outgoing referral.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** Referral must be SUBMITTED or later (cannot view DRAFT).
// @Description  **Common Errors:**
// @Description  - 400 Invalid ID format
// @Description  - 403 Forbidden (DRAFT status or wrong hospital)
// @Tags         Liaison
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id} [get]
func (h *LiaisonHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid id format",
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

	ref, err := h.referralUC.GetDetailsForLiaison(c.Request.Context(), id, hospID)
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

// GetReviewChecklist godoc
// @Summary      Get Review Checklist
// @Description  Get the current state of the liaison review checklist for a referral.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** Referral must belong to the liaison's hospital.
// @Description  **Common Errors:**
// @Description  - 400 Invalid format
// @Description  - 401 Unauthorized
// @Tags         Liaison
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReviewChecklistResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/review-checklist [get]
func (h *LiaisonHandler) GetReviewChecklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	hospID := h.getHospitalID(c)
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "unauthorized"})
		return
	}

	checklist, err := h.referralUC.GetReviewChecklist(c.Request.Context(), id, hospID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, checklist)
}

// UpdateReviewChecklist godoc
// @Summary      Update Review Checklist
// @Description  Update specific items in the liaison review checklist.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** Referral must belong to the liaison's hospital; Status must be SUBMITTED or UNDER_LIAISON_REVIEW.
// @Description  **Common Errors:**
// @Description  - 400 Invalid format / validation error / invalid status
// @Description  - 401 Unauthorized
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.ReviewChecklistRequest true "Checklist Updates"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/review-checklist [put]
func (h *LiaisonHandler) UpdateReviewChecklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req dto.ReviewChecklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userID := h.getUserID(c)
	hospID := h.getHospitalID(c)
	if hospID == uuid.Nil || userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "unauthorized"})
		return
	}

	if err := h.referralUC.UpdateReviewChecklist(c.Request.Context(), id, userID, hospID, req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Review checklist updated"})
}

func (h *LiaisonHandler) getHospitalID(c *gin.Context) uuid.UUID {
	val, _ := c.Get("hospID")
	if id, ok := val.(uuid.UUID); ok {
		return id
	}
	if id, ok := val.(*uuid.UUID); ok && id != nil {
		return *id
	}
	return uuid.Nil
}

func (h *LiaisonHandler) getUserID(c *gin.Context) uuid.UUID {
	val, _ := c.Get("userID")
	if id, ok := val.(uuid.UUID); ok {
		return id
	}
	if id, ok := val.(*uuid.UUID); ok && id != nil {
		return *id
	}
	return uuid.Nil
}

// Read godoc
// @Summary      Mark Referral as Read
// @Description  Acknowledge receipt and mark the referral as read by the liaison.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** referral.status must be SUBMITTED.
// @Description  **State Transition:** SUBMITTED → UNDER_LIAISON_REVIEW.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (not sender hospital)
// @Tags         Liaison
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/read [post]
func (h *LiaisonHandler) Read(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		liaisonID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		liaisonID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonRead(c.Request.Context(), id, liaisonID, hospID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral marked as read",
	})
}

// Forward godoc
// @Summary      Forward Referral to Specialist
// @Description  Forward the referral to specialists at the target hospital.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** status = SUBMITTED or UNDER_LIAISON_REVIEW; all attachments must be VERIFIED; checklist must be COMPLETE (condition-based).
// @Description  **State Transition:** → FORWARDED.
// @Description  **Gatekeepers:** Attachment verification; Review Checklist completion.
// @Description  **Common Errors:**
// @Description  - 400 invalid format / checklist incomplete
// @Description  - 403 (wrong hospital)
// @Description  - 422 (attachments not verified)
// @Tags         Liaison
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/forward [post]
func (h *LiaisonHandler) Forward(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		liaisonID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		liaisonID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonForward(c.Request.Context(), id, liaisonID, hospID, ""); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral forwarded to specialist",
	})
}

// Reject godoc
// @Summary      Reject Referral
// @Description  Reject the referral back to the referring doctor.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** status = SUBMITTED or UNDER_LIAISON_REVIEW.
// @Description  **State Transition:** → REJECTED_BY_LIAISON.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (wrong hospital)
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Rejection Reason"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/reject [post]
func (h *LiaisonHandler) Reject(c *gin.Context) {
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
	liaisonID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		liaisonID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		liaisonID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonReject(c.Request.Context(), id, liaisonID, hospID, req.Reason); err != nil {
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

// RejectAfterSend godoc
// @Summary      Reject Referral After Sending
// @Description  Cancel a referral that has already been submitted but not yet scheduled.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** Status must be SUBMITTED, UNDER_LIAISON_REVIEW, FORWARDED, UNDER_SPECIALIST_REVIEW, or ACCEPTED.
// @Description  **State Transition:** → REJECTED_AFTER_SEND. Removes from triage queue.
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Rejection Reason"
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/reject-after-send [post]
func (h *LiaisonHandler) RejectAfterSend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}
	userIdVal, _ := c.Get("userID")
	liaisonID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		liaisonID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		liaisonID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	var req dto.RejectDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	if err := h.referralUC.RejectAfterSend(c.Request.Context(), id, liaisonID, hospID, entity.RoleLiaisonOfficer, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Referral rejected after send"})
}

// Revise godoc
// @Summary      Request Referral Revision
// @Description  Send a referral back to the doctor for additional information or clarification.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** status = SUBMITTED or UNDER_LIAISON_REVIEW.
// @Description  **State Transition:** → NEED_REVISION.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (wrong hospital)
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.ReviseDTO true "Revision Reason"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/revise [post]
func (h *LiaisonHandler) Revise(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req dto.ReviseDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		liaisonID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		liaisonID = *uID
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonRevise(c.Request.Context(), id, liaisonID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Revision requested for referral",
	})
}

// DEPRECATED: UnassignSpecialist is no longer directly exposed to Liaisons in the current workflow.
/*
// UnassignSpecialist godoc
// @Summary      Unassign Specialist from Referral
// @Description  Allows a Liaison to break the lock on a referral if a specialist has gone inactive. Only available for incoming referrals.
// @Tags         Liaison
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Reason for Unassignment (Reason field reused)"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/incoming/{id}/unassign [post]
*/
func (h *LiaisonHandler) UnassignSpecialist(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid format"})
		return
	}

	var req dto.RejectDTO // Reuse RejectDTO for the reason
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonUnassignSpecialist(c.Request.Context(), id, liaisonID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Specialist successfully unassigned from referral",
	})
}

// GetDashboardStats godoc
// @Summary      Get Liaison Dashboard Stats
// @Description  Returns aggregated dashboard statistics for the liaison's hospital.
// @Description  **Metrics:**
// @Description  - **Total Referrals**: All non‑DRAFT referrals from the sender hospital (last 30 days).
// @Description  - **Pending Review**: Referrals in SUBMITTED or UNDER_LIAISON_REVIEW status.
// @Description  - **Approved Today**: Referrals that were accepted or completed today.
// @Description  - **Rejected**: Referrals rejected by liaison, specialist, or after sending.
// @Description  **Percentage changes** compare the last 30 days with the previous 30‑day period.
// @Tags         Liaison
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/dashboard/stats [get]
func (h *LiaisonHandler) GetDashboardStats(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid hospital scope"})
		return
	}

	stats, err := h.referralUC.GetLiaisonDashboardStats(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
