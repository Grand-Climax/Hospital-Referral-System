package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type LiaisonHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewLiaisonHandler(referralUC iusecase.ReferralUseCase) *LiaisonHandler {
	return &LiaisonHandler{referralUC: referralUC}
}

// ListOutgoing godoc
// @Summary      List Outgoing Referrals for Liaison
// @Description  Get a paginated list of referrals sent FROM the liaison's hospital.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** Must be a liaison at the sender hospital.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Liaison Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
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

// DEPRECATED: ListIncoming is now handled via specialists or specific monitoring tools.
// ListIncoming godoc
// @Summary      List Incoming Referrals for Liaison
// @Description  Get a paginated list of referrals sent TO the liaison's hospital for monitoring.
// @Description  **Visibility:** Includes FORWARDED, UNDER_SPECIALIST_REVIEW, ACCEPTED, SCHEDULED, ASSIGNED, COMPLETED, REJECTED_BY_SPECIALIST, MISSED, RESCHEDULED, REDIRECTED, REJECTED_AFTER_SEND, and ADMITTED.
// @Tags         Liaison Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
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
		Status:      c.Query("status"),
		Region:      c.Query("region"),
		PatientName: c.Query("patient_name"),
		Sort:        c.Query("sort"),
		Limit:       limit,
		Page:        page,
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
				patientRegion = *r.Patient.HomeRegion
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
// @Tags         Liaison Referrals
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

// Read godoc
// @Summary      Mark Referral as Read
// @Description  Acknowledge receipt and mark the referral as read by the liaison.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** referral.status must be SUBMITTED.
// @Description  **State Transition:** SUBMITTED → UNDER_LIAISON_REVIEW.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (not sender hospital)
// @Tags         Liaison Referrals
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
// @Description  **Prerequisites:** status = SUBMITTED or UNDER_LIAISON_REVIEW; all attachments must be VERIFIED.
// @Description  **State Transition:** → FORWARDED.
// @Description  **Gatekeepers:** Attachment verification (not PENDING/REJECTED).
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (wrong hospital)
// @Description  - 422 (attachments not verified)
// @Tags         Liaison Referrals
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
// @Tags         Liaison Referrals
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

// Revise godoc
// @Summary      Request Referral Revision
// @Description  Send a referral back to the doctor for additional information or clarification.
// @Description  **Roles:** LIAISON_OFFICER
// @Description  **Prerequisites:** status = SUBMITTED or UNDER_LIAISON_REVIEW.
// @Description  **State Transition:** → NEED_REVISION.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 403 (wrong hospital)
// @Tags         Liaison Referrals
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
// @Tags         Liaison Referrals
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
