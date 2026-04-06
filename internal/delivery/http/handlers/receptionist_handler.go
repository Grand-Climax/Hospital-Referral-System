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
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name (any order)"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals [get]
func (h *ReceptionistHandler) ListReferrals(c *gin.Context) {
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

	referrals, total, err := h.referralUC.ListForReceptionist(c.Request.Context(), hospID, filter)
	if err != nil {
		log.Printf("[ReceptionistHandler.List] error: %v", err)
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
				Message: "No referrals found for your hospital",
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
// @Summary      Get Referral Details for Receptionist
// @Description  Get detailed information about a specific referral.
// @Tags         Receptionist Referrals
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id} [get]
func (h *ReceptionistHandler) GetReferral(c *gin.Context) {
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
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	ref, err := h.referralUC.GetDetailsForReceptionist(c.Request.Context(), id, hospID)
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

// ConfirmAttendance godoc
// @Summary      Confirm Referral Attendance
// @Description  Confirm that the patient has attended their referral appointment.
// @Tags         Receptionist Referrals
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body map[string]string true "Status (key: status)"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/confirm-attendance [post]
func (h *ReceptionistHandler) ConfirmAttendance(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid format",
		})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Patient attendance confirmed successfully",
	})
}
