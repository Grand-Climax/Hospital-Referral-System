package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

// DepartmentHeadDashboardHandler exposes the read-only dashboard /
// widget endpoints scoped to the caller's (hospital, department). All
// scoping is taken from JWT claims set by the auth middleware - no
// query parameter overrides are accepted.
type DepartmentHeadDashboardHandler struct {
	uc iusecase.DepartmentHeadDashboardUseCase
}

func NewDepartmentHeadDashboardHandler(uc iusecase.DepartmentHeadDashboardUseCase) *DepartmentHeadDashboardHandler {
	return &DepartmentHeadDashboardHandler{uc: uc}
}

// scopedDashboardHospDept pulls hospID + deptID off the gin context and
// fails the response with 401 if either is missing. The dashboard does
// not run cross-dept, so a missing scope is always a bug or a misissued
// token, not a use-case condition.
func (h *DepartmentHeadDashboardHandler) scopedHospDept(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
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
	if hospID == uuid.Nil || deptID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes (hospital/department missing)"})
		return uuid.Nil, uuid.Nil, false
	}
	return hospID, deptID, true
}

// GetDashboardStats godoc
// @Summary      Department Head dashboard landing summary
// @Description  Aggregated, live-computed snapshot for the dept-head landing page. Each field is read from the same source the booking engine uses, so the numbers always match what /capacity/detail and /schedule/patients would return.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session with hospital + department scope.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized / scope missing
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} dto.DeptHeadDashboardStatsResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/dashboard/stats [get]
func (h *DepartmentHeadDashboardHandler) GetDashboardStats(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	stats, err := h.uc.GetDashboardStats(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetTrends godoc
// @Summary      Department Head capacity utilization trends
// @Description  Per-day capacity utilization for the last N days ending today (inclusive). Each entry reuses the same EffectiveCapacity helper used by the booking engine, so trends and live capacity always agree.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** days must be 1-90; defaults to 14.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid days
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        days query int false "Lookback window in days (1-90, default 14)"
// @Success      200 {object} dto.DeptHeadTrendsResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/dashboard/trends [get]
func (h *DepartmentHeadDashboardHandler) GetTrends(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	days := 14
	if d := c.Query("days"); d != "" {
		v, err := strconv.Atoi(d)
		if err != nil || v <= 0 {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "days must be a positive integer"})
			return
		}
		days = v
	}

	trends, err := h.uc.GetTrends(c.Request.Context(), hospID, deptID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"days":    days,
		"data":    trends,
	})
}

// GetPriorityBuckets godoc
// @Summary      Priority bucket counts for the dept's waiting queue
// @Description  Breaks the dept's current waiting queue down by clinical condition (critical/urgent/stable) and ML severity tier (HIGH/MEDIUM/LOW), and surfaces the top 5 highest-composite-score patients still waiting. The composite score is the same number the scheduling engine uses to prioritize batch bookings.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} dto.DeptHeadPriorityBucketsResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/triage-queue/buckets [get]
func (h *DepartmentHeadDashboardHandler) GetPriorityBuckets(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	resp, err := h.uc.GetPriorityBuckets(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// GetStaffSummary godoc
// @Summary      Department staff summary (active/inactive + roster)
// @Description  Per-role active/inactive counts for the dept's REFERRING_DOCTOR and RECEPTIONIST users, the soft staff-capacity hint (MaxCapacityOfStaff), the distinct count of doctors currently assigned to triage rows for today, and a lightweight roster of the active staff (id/name/email/role/profile image).
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} dto.DeptHeadStaffSummaryResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/staff/summary [get]
func (h *DepartmentHeadDashboardHandler) GetStaffSummary(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	resp, err := h.uc.GetStaffSummary(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// GetActivity godoc
// @Summary      Recent dept-head activity stream
// @Description  Returns the most recent audit-log rows for the dept head's hospital, filtered to a curated set of actions (batch scheduling, override mutations, emergency scheduling, doctor assignments, arrival confirmations, missed marks, etc.). Each row is summarized for direct display in the activity feed.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        limit query int false "Max number of rows to return (1-100, default 20)"
// @Param        start_date query string false "Inclusive start date (YYYY-MM-DD)"
// @Param        end_date   query string false "Inclusive end date (YYYY-MM-DD)"
// @Success      200 {object} dto.DeptHeadActivityResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/activity [get]
func (h *DepartmentHeadDashboardHandler) GetActivity(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	var start, end *time.Time
	if s := c.Query("start_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid start_date; expected YYYY-MM-DD"})
			return
		}
		start = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid end_date; expected YYYY-MM-DD"})
			return
		}
		end = &t
	}

	items, err := h.uc.GetActivity(c.Request.Context(), hospID, deptID, limit, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(items),
		"data":    items,
	})
}
