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

type AdminHandler struct {
	referralUC iusecase.ReferralUseCase
}

func NewAdminHandler(referralUC iusecase.ReferralUseCase) *AdminHandler {
	return &AdminHandler{referralUC: referralUC}
}

// SystemAdminList godoc
// @Summary      System Admin Global Listing
// @Description  Get a global paginated list of all referrals with optional status filtering.
// @Tags         Admin Referrals
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
// @Router       /api/v1/system-admin/referrals [get]
func (h *AdminHandler) SystemAdminList(c *gin.Context) {
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

	referrals, total, err := h.referralUC.ListForSystemAdmin(c.Request.Context(), filter)
	if err != nil {
		log.Printf("[AdminHandler.SystemAdminList] error: %v", err)
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
				Message: "No referrals found in the system",
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
			Message: "Global referrals retrieved successfully",
		},
		Data:         responseData,
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

// HospitalAdminLogs godoc
// @Summary      Get Referral Logs for Hospital
// @Description  Get audit logs of all referral status transitions connected to the hospital.
// @Tags         Admin Referrals
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Success      200 {object} dto.PaginatedLogResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals-log [get]
func (h *AdminHandler) HospitalAdminLogs(c *gin.Context) {
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

	logs, total, err := h.referralUC.GetHospitalLogsForAdmin(c.Request.Context(), hospID, limit, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedLogResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No referral logs found",
			},
			Data:     []dto.LogResponseDTO{},
			Total:    0,
			Page:     page,
			PageSize: limit,
		})
		return
	}

	var responseData []dto.LogResponseDTO
	for _, l := range logs {
		var from *string
		if l.FromStatus != nil {
			f := string(*l.FromStatus)
			from = &f
		}
		
		responseData = append(responseData, dto.LogResponseDTO{
			HistoryID:   l.ID,
			ReferralID:  l.ReferralID,
			ChangedByID: l.ChangedByID,
			Role:        "System",
			FromStatus:  from,
			ToStatus:    string(l.ToStatus),
			CreatedAt:   l.ChangedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.PaginatedLogResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospital referral logs retrieved successfully",
		},
		Data:     responseData,
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}
