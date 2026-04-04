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

func (h *AdminHandler) SystemAdminList(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	statusFilter := c.Query("status")

	referrals, total, err := h.referralUC.ListForSystemAdmin(c.Request.Context(), limit, offset, statusFilter)
	if err != nil {
		log.Printf("[AdminHandler.SystemAdminList] error: %v", err)
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
		responseData = append(responseData, dto.ListReferralResponse{
			ID:                  r.ID,
			PatientFirstName:    r.Patient.FirstName,
			PatientMiddleName:   r.Patient.MiddleName,
			PatientLastName:     r.Patient.LastName,
			Department:          r.TargetDeptID.String(),
			Date:                r.CreatedAt.Format("2006-01-02"),
			Status:              string(r.Status),
			ICDCode:             icd,
			Diagnosis:           diag,
			ConditionAtReferral: r.ReferralForm.ConditionAtReferral,
		})
	}

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		Data:     responseData,
		Total:    total,
		Page:     offset/limit + 1,
		PageSize: limit,
	})
}

func (h *AdminHandler) HospitalAdminLogs(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user scopes"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, total, err := h.referralUC.GetHospitalLogsForAdmin(c.Request.Context(), hospID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	// We can reuse pagination response style or just return custom map
	c.JSON(http.StatusOK, gin.H{
		"data":      responseData,
		"total":     total,
		"page":      offset/limit + 1,
		"page_size": limit,
	})
}
