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
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
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
	statusFilter := c.Query("status")

	referrals, total, err := h.referralUC.ListForSystemAdmin(c.Request.Context(), limit, page, statusFilter)
	if err != nil {
		log.Printf("[AdminHandler.SystemAdminList] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
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
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals-log [get]
func (h *AdminHandler) HospitalAdminLogs(c *gin.Context) {
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

	logs, total, err := h.referralUC.GetHospitalLogsForAdmin(c.Request.Context(), hospID, limit, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":   false,
			"message":   "No referral logs found",
			"data":      []dto.LogResponseDTO{},
			"total":     0,
			"page":      page,
			"page_size": limit,
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

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Hospital referral logs retrieved successfully",
		"data":      responseData,
		"total":     total,
		"page":      page,
		"page_size": limit,
	})
}
