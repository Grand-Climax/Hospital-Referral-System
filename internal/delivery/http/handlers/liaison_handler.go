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

type LiaisonHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewLiaisonHandler(referralUC iusecase.ReferralUseCase) *LiaisonHandler {
	return &LiaisonHandler{referralUC: referralUC}
}

// ListReferrals godoc
// @Summary      List Referrals for Liaison
// @Description  Get a paginated list of referrals forwarded to or managed by the liaison's hospital.
// @Tags         Liaison Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals [get]
func (h *LiaisonHandler) ListReferrals(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user scopes"})
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

	referrals, total, err := h.referralUC.ListForLiaison(c.Request.Context(), hospID, limit, page, statusFilter)
	if err != nil {
		log.Printf("[LiaisonHandler.List] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No referrals found for your hospital",
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
// @Summary      Get Referral Details for Liaison
// @Description  Get detailed information about a specific referral.
// @Tags         Liaison Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} entity.Referral
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id} [get]
func (h *LiaisonHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id format"})
		return
	}

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	ref, err := h.referralUC.GetDetailsForLiaison(c.Request.Context(), id, hospID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessPayload(ref, "Referral details retrieved successfully"))
}

// Read godoc
// @Summary      Mark Referral as Read
// @Description  Acknowledge receipt and mark the referral as read by the liaison.
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
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonRead(c.Request.Context(), id, liaisonID, hospID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Referral marked as read",
	})
}

// Forward godoc
// @Summary      Forward Referral to Specialist
// @Description  Forward an approved referral to the specialists within the hospital.
// @Tags         Liaison Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/forward [post]
func (h *LiaisonHandler) Forward(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonForward(c.Request.Context(), id, liaisonID, hospID, ""); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Referral forwarded to specialist",
	})
}

// Reject godoc
// @Summary      Reject Referral
// @Description  Reject an incoming referral.
// @Tags         Liaison Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Rejection Reason"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/reject [post]
func (h *LiaisonHandler) Reject(c *gin.Context) {
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
	liaisonID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonReject(c.Request.Context(), id, liaisonID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Referral rejected",
	})
}

// Revise godoc
// @Summary      Request Referral Revision
// @Description  Send a referral back to the draft stage to ask the sender for revisions.
// @Tags         Liaison Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.ReviseDTO true "Revision Reason"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/liaison/referrals/{id}/revise [post]
func (h *LiaisonHandler) Revise(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid format"})
		return
	}

	var req dto.ReviseDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	liaisonID, _ := userIdVal.(uuid.UUID)
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	if err := h.referralUC.LiaisonRevise(c.Request.Context(), id, liaisonID, hospID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Revision requested for referral",
	})
}
