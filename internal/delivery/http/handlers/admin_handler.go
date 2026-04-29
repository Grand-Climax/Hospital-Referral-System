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

type AdminHandler struct {
	referralUC iusecase.ReferralUseCase
	auditRepo  irepository.AuditLogRepository
}

func NewAdminHandler(referralUC iusecase.ReferralUseCase) *AdminHandler {
	return &AdminHandler{referralUC: referralUC}
}

func NewAdminHandlerWithAudit(referralUC iusecase.ReferralUseCase, auditRepo irepository.AuditLogRepository) *AdminHandler {
	return &AdminHandler{referralUC: referralUC, auditRepo: auditRepo}
}

func getHospitalScopeFromContext(c *gin.Context) (uuid.UUID, bool) {
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
		return uuid.Nil, false
	}
	return hospID, true
}

func parseReferralFilter(c *gin.Context) irepository.ReferralFilter {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	return irepository.ReferralFilter{
		Status:      c.Query("status"),
		Region:      c.Query("region"),
		PatientName: c.Query("patient_name"),
		Sort:        c.Query("sort"),
		Limit:       limit,
		Page:        page,
	}
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
		Data:     responseData,
		Total:    total,
		Page:     page,
		PageSize: limit,
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
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
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

// HospitalAdminInboundReferrals godoc
// @Summary      List inbound referrals (Hospital Admin)
// @Description  List referrals where the admin's hospital is the target hospital.
// @Tags         Hospital Admin - Referral Oversight
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/inbound [get]
func (h *AdminHandler) HospitalAdminInboundReferrals(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	filter := parseReferralFilter(c)
	if filter.Status != "" && !h.referralUC.IsValidStatus(filter.Status) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "forbidden: unknown or invalid referral status"})
		return
	}

	referrals, total, err := h.referralUC.ListInboundForHospitalAdmin(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	responseData := toListReferralResponseSlice(referrals)
	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Inbound referrals retrieved successfully"},
		Data:         responseData,
		Total:        total,
		Page:         filter.Page,
		PageSize:     filter.Limit,
	})
}

// HospitalAdminOutboundReferrals godoc
// @Summary      List outbound referrals (Hospital Admin)
// @Description  List referrals where the admin's hospital is the sender hospital.
// @Tags         Hospital Admin - Referral Oversight
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/outbound [get]
func (h *AdminHandler) HospitalAdminOutboundReferrals(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	filter := parseReferralFilter(c)
	if filter.Status != "" && !h.referralUC.IsValidStatus(filter.Status) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "forbidden: unknown or invalid referral status"})
		return
	}

	referrals, total, err := h.referralUC.ListOutboundForHospitalAdmin(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	responseData := toListReferralResponseSlice(referrals)
	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Outbound referrals retrieved successfully"},
		Data:         responseData,
		Total:        total,
		Page:         filter.Page,
		PageSize:     filter.Limit,
	})
}

// HospitalAdminPendingApprovals godoc
// @Summary      List pending approvals (Hospital Admin)
// @Description  List hospital referrals currently waiting in referral approval/review stages.
// @Tags         Hospital Admin - Referral Oversight
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/pending-approvals [get]
func (h *AdminHandler) HospitalAdminPendingApprovals(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	filter := parseReferralFilter(c)
	filter.Status = ""

	referrals, total, err := h.referralUC.ListPendingApprovalsForHospitalAdmin(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	responseData := toListReferralResponseSlice(referrals)
	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Pending approvals retrieved successfully"},
		Data:         responseData,
		Total:        total,
		Page:         filter.Page,
		PageSize:     filter.Limit,
	})
}

// HospitalAdminRejectedRedirected godoc
// @Summary      List rejected or redirected referrals (Hospital Admin)
// @Description  List hospital referrals that were rejected or redirected for revision.
// @Tags         Hospital Admin - Referral Oversight
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/rejected-redirected [get]
func (h *AdminHandler) HospitalAdminRejectedRedirected(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	filter := parseReferralFilter(c)
	filter.Status = ""

	referrals, total, err := h.referralUC.ListRejectedRedirectedForHospitalAdmin(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	responseData := toListReferralResponseSlice(referrals)
	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Rejected/redirected referrals retrieved successfully"},
		Data:         responseData,
		Total:        total,
		Page:         filter.Page,
		PageSize:     filter.Limit,
	})
}

// HospitalAdminReferralDetails godoc
// @Summary      Get referral details (Hospital Admin)
// @Description  Read-only referral details for referrals connected to admin's hospital.
// @Tags         Hospital Admin - Referral Oversight
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/{id} [get]
func (h *AdminHandler) HospitalAdminReferralDetails(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral ID"})
		return
	}

	ref, err := h.referralUC.GetDetailsForHospitalAdmin(c.Request.Context(), hospID, referralID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Referral not found"})
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

// HospitalAdminReferralStatusCounts godoc
// @Summary      Get referral counts by status (Hospital Admin)
// @Description  Aggregate referral counts by status for referrals connected to admin's hospital.
// @Tags         Hospital Admin - Referral Oversight
// @Produce      json
// @Success      200 {object} dto.ReferralStatusCountListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/stats/by-status [get]
func (h *AdminHandler) HospitalAdminReferralStatusCounts(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}

	counts, err := h.referralUC.GetReferralStatusCountsForHospitalAdmin(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.ReferralStatusCountResponse, 0, len(counts))
	for _, cnt := range counts {
		resp = append(resp, dto.ReferralStatusCountResponse{
			Status: string(cnt.Status),
			Count:  cnt.Count,
		})
	}

	c.JSON(http.StatusOK, dto.ReferralStatusCountListResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referral status counts retrieved successfully"},
	})
}

// HospitalAdminAuditLogs godoc
// @Summary      View hospital audit logs (Hospital Admin)
// @Description  Read-only hospital-scoped audit log viewer.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Param        action_type query string false "Filter by action type"
// @Param        start_date query string false "Start date (YYYY-MM-DD)"
// @Param        end_date query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.AuditLogListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      503 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/audit-logs [get]
func (h *AdminHandler) HospitalAdminAuditLogs(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	if h.auditRepo == nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Success: false, Error: "audit repository is unavailable"})
		return
	}

	filter := irepository.AuditLogFilter{Page: 1, PageSize: 20}
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil {
		filter.PageSize = ps
	}
	if actionType := c.Query("action_type"); actionType != "" {
		at := entity.ActionType(actionType)
		filter.ActionType = &at
	}
	if s := c.Query("start_date"); s != "" {
		filter.StartDate = &s
	}
	if e := c.Query("end_date"); e != "" {
		filter.EndDate = &e
	}

	logs, total, err := h.auditRepo.ListByHospital(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.AuditLogResponse, 0, len(logs))
	for _, lg := range logs {
		var refID *string
		if lg.ReferralID != nil {
			id := lg.ReferralID.String()
			refID = &id
		}
		resp = append(resp, dto.AuditLogResponse{
			ID:         lg.ID.String(),
			UserID:     lg.UserID.String(),
			ReferralID: refID,
			ActionType: lg.ActionType,
			Resource:   lg.Resource,
			ResourceID: lg.ResourceID,
			IPAddress:  lg.IPAddress,
			UserAgent:  lg.UserAgent,
			Timestamp:  lg.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.AuditLogListResponse{
		Data:         resp,
		Total:        total,
		Page:         filter.Page,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Audit logs retrieved successfully"},
	})
}

// HospitalAdminMonthlyReferralTotals godoc
// @Summary      Monthly referral totals (Hospital Admin)
// @Description  Monthly referral totals for hospital-connected referrals.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Param        months query int false "Number of months" default(6)
// @Success      200 {object} dto.MonthlyReferralTotalsResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/reports/monthly-referrals [get]
func (h *AdminHandler) HospitalAdminMonthlyReferralTotals(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	months, _ := strconv.Atoi(c.DefaultQuery("months", "6"))
	if months <= 0 {
		months = 6
	}

	totals, err := h.referralUC.GetMonthlyReferralTotalsForHospitalAdmin(c.Request.Context(), hospID, months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.MonthlyReferralTotalResponse, 0, len(totals))
	for _, t := range totals {
		resp = append(resp, dto.MonthlyReferralTotalResponse{Month: t.Month, Count: t.Count})
	}
	c.JSON(http.StatusOK, dto.MonthlyReferralTotalsResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Monthly referral totals retrieved successfully"},
	})
}

// HospitalAdminAcceptanceRejectionRate godoc
// @Summary      Acceptance and rejection rate (Hospital Admin)
// @Description  Acceptance and rejection rates for hospital-connected referrals.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Success      200 {object} dto.AcceptanceRejectionRateResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/reports/acceptance-rejection-rate [get]
func (h *AdminHandler) HospitalAdminAcceptanceRejectionRate(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}

	acceptance, rejection, err := h.referralUC.GetAcceptanceRejectionRateForHospitalAdmin(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AcceptanceRejectionRateResponse{
		AcceptanceRate: acceptance,
		RejectionRate:  rejection,
		BaseResponse:   dto.BaseResponse{Success: true, Message: "Acceptance/rejection rate retrieved successfully"},
	})
}

// HospitalAdminMissedAppointmentRate godoc
// @Summary      Missed appointment rate (Hospital Admin)
// @Description  Missed appointment rate for hospital inbound appointment-tracked referrals.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Success      200 {object} dto.MissedAppointmentRateResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/reports/missed-appointment-rate [get]
func (h *AdminHandler) HospitalAdminMissedAppointmentRate(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}

	rate, err := h.referralUC.GetMissedAppointmentRateForHospitalAdmin(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MissedAppointmentRateResponse{
		MissedAppointmentRate: rate,
		BaseResponse:          dto.BaseResponse{Success: true, Message: "Missed appointment rate retrieved successfully"},
	})
}

// HospitalAdminBusiestDepartments godoc
// @Summary      Busiest departments (Hospital Admin)
// @Description  Ranked departments by inbound referral volume.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Param        limit query int false "Result size" default(5)
// @Success      200 {object} dto.DepartmentLoadListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/reports/busiest-departments [get]
func (h *AdminHandler) HospitalAdminBusiestDepartments(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	if limit <= 0 {
		limit = 5
	}

	rows, err := h.referralUC.GetBusiestDepartmentsForHospitalAdmin(c.Request.Context(), hospID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.DepartmentLoadResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, dto.DepartmentLoadResponse{
			DepartmentID: r.DepartmentID.String(),
			Count:        r.Count,
		})
	}
	c.JSON(http.StatusOK, dto.DepartmentLoadListResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Busiest departments retrieved successfully"},
	})
}

// HospitalAdminAverageWaitTime godoc
// @Summary      Average wait time (Hospital Admin)
// @Description  Average wait time derived from referral waiting-hours weight.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Success      200 {object} dto.AverageWaitTimeResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/reports/average-wait-time [get]
func (h *AdminHandler) HospitalAdminAverageWaitTime(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}

	wait, err := h.referralUC.GetAverageWaitTimeForHospitalAdmin(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AverageWaitTimeResponse{
		AverageWaitTime: wait,
		BaseResponse:    dto.BaseResponse{Success: true, Message: "Average wait time retrieved successfully"},
	})
}

// HospitalAdminTopReferringHospitals godoc
// @Summary      Top referring hospitals (Hospital Admin)
// @Description  Ranked sender hospitals by referrals sent to this hospital.
// @Tags         Hospital Admin - Audit & Reports
// @Produce      json
// @Param        limit query int false "Result size" default(5)
// @Success      200 {object} dto.TopReferringHospitalListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/reports/top-referring-hospitals [get]
func (h *AdminHandler) HospitalAdminTopReferringHospitals(c *gin.Context) {
	hospID, ok := getHospitalScopeFromContext(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	if limit <= 0 {
		limit = 5
	}

	rows, err := h.referralUC.GetTopReferringHospitalsForHospitalAdmin(c.Request.Context(), hospID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.TopReferringHospitalResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, dto.TopReferringHospitalResponse{
			HospitalID:   r.HospitalID.String(),
			HospitalName: r.HospitalName,
			Count:        r.Count,
		})
	}
	c.JSON(http.StatusOK, dto.TopReferringHospitalListResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Top referring hospitals retrieved successfully"},
	})
}
