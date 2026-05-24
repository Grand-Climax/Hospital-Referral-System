package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/usecase"
)

type DepartmentHeadHandler struct {
	capacityUC iusecase.CapacityManagementUseCase
	schedUC    iusecase.SchedulingUseCase
	triageUC   iusecase.TriageUseCase
}

func NewDepartmentHeadHandler(capacityUC iusecase.CapacityManagementUseCase, schedUC iusecase.SchedulingUseCase, triageUC iusecase.TriageUseCase) *DepartmentHeadHandler {
	return &DepartmentHeadHandler{
		capacityUC: capacityUC,
		schedUC:    schedUC,
		triageUC:   triageUC,
	}
}

// GetTriageQueue godoc
// @Summary      Get department triage queue
// @Description  Returns triage queue entries scoped to the authenticated department head's hospital and department.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Visibility:** Department-scoped queue only.
// @Tags         Department Head
// @Produce      json
// @Param        limit query int false "Pagination limit" default(50)
// @Param        page query int false "Page number" default(1)
// @Success      200 {object} dto.DeptHeadTriageQueueResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/triage-queue [get]
func (h *DepartmentHeadHandler) GetTriageQueue(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}

	if hospID == uuid.Nil || deptID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes (hospital/department missing)"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	offset := (page - 1) * limit

	queues, total, err := h.triageUC.ListForTriageByDepartment(c.Request.Context(), hospID, deptID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    queues,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// BatchSchedule godoc
// @Summary      Run batch scheduling (Schedule-on-Demand, no overbook)
// @Description  Walks the department's waiting queue and books each patient into the earliest available slot in [today + buffer_days, today + buffer_days + max_horizon_days].
// @Description
// @Description  **Roles:** DEPT_HEAD (scoped to caller's hospital + department).
// @Description
// @Description  **Prerequisites:**
// @Description  - WAITING TriageQueue rows for the department (AppointmentDate IS NULL, ArrivalStatus = EXPECTED).
// @Description  - system_configs.buffer_days (default 2) and system_configs.max_horizon_days (default 30).
// @Description
// @Description  **Capacity Rule:** Routine - booked < maxSlots. Overbook capacity is never used here.
// @Description  For each candidate date the helper re-evaluates getEffectiveCapacity, so concurrent emergency bookings are honored.
// @Description
// @Description  **State Transitions per booking:**
// @Description  - TriageQueue.AppointmentDate set, ArrivalStatus = EXPECTED.
// @Description  - Referral.Status -> SCHEDULED with a ReferralStatusHistory row.
// @Description  - DailySchedule snapshot row created/updated for the target date.
// @Description
// @Description  **Side Effects:**
// @Description  - Per-booking SMS via NotifyScheduling (when auto_notify or sendNotifications=true).
// @Description  - One audit row with the aggregated result.
// @Description  - One in-app BATCH_SCHEDULE_COMPLETED event when at least one patient was placed.
// @Description
// @Description  **Soft lease:** A 5-minute lease is acquired via SchedulerCheckpointRepository before any work begins. If another batch run is in progress for the same (hospital, department), this call returns 200 with `{ scheduled_count: 0, waiting_count: 0, message: "Batch already running for this department; try again in a few minutes" }` rather than racing.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized / scope missing
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} dto.DeptHeadBatchScheduleResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/schedule/batch [post]
func (h *DepartmentHeadHandler) BatchSchedule(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	result, err := h.schedUC.BatchSchedule(c.Request.Context(), hospID, deptID, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ListOverrides godoc
// @Summary      List capacity overrides (active + historical)
// @Description  Returns every CapacityOverride row recorded for the department, including inactive ones (IsActive=false). Useful for audit and for deciding whether you must delete an override before creating a new one.
// @Description
// @Description  **Roles:** DEPT_HEAD (scope inferred from token).
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized / scope missing
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} dto.DeptHeadOverrideListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/overrides [get]
func (h *DepartmentHeadHandler) ListOverrides(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}

	overrides, err := h.capacityUC.GetOverrides(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overrides,
	})
}

// CreateOverride godoc
// @Summary      Create capacity override (immutable)
// @Description  Sets a fixed daily capacity for a future date under the Schedule-on-Demand model.
// @Description  Overrides are IMMUTABLE - to change a value, delete the existing override and create a new one.
// @Description
// @Description  **Roles:** DEPT_HEAD (scoped to the authenticated user's hospital + department)
// @Description
// @Description  **Prerequisites:**
// @Description  - target_date must be >= today + system_configs.buffer_days + 1 (default buffer is 2 days)
// @Description  - new_limit must be >= 0
// @Description  - no other active override may exist for the same (hospital, department, date)
// @Description
// @Description  **Side Effects:**
// @Description  - On the next ScheduleAppointment / ManualEmergencySchedule / BatchSchedule for that date, getEffectiveCapacity will use new_limit instead of HospitalDepartment.StandardDailyLimit.
// @Description  - No DailySchedule rows are created here; one is logged lazily on the first booking for that date.
// @Description  - Emits a CAPACITY_OVERRIDE_CREATED in-app notification and an audit log row.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid date format / inside booking buffer / negative new_limit
// @Description  - 401 Unauthorized
// @Description  - 409 active override already exists for that date - delete it first
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Accept       json
// @Produce      json
// @Param        body body dto.CreateOverrideRequest true "Override details"
// @Success      201 {object} dto.BaseResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      409 {object} dto.DeptHeadErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/overrides [post]
func (h *DepartmentHeadHandler) CreateOverride(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	var req dto.CreateOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid date format, use YYYY-MM-DD"})
		return
	}

	if err := h.capacityUC.CreateOverride(c.Request.Context(), hospID, deptID, date, req.NewLimit, req.Reason, userID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrOverrideInsideBuffer):
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		case errors.Is(err, usecase.ErrOverrideExists):
			c.JSON(http.StatusConflict, dto.BaseResponse{Success: false, Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, dto.BaseResponse{Success: true, Message: "Capacity override created successfully"})
}

// DeleteOverride godoc
// @Summary      Delete (deactivate) capacity override
// @Description  Deactivates an override (IsActive=false) so subsequent capacity decisions for that date fall back to HospitalDepartment.StandardDailyLimit. The row is preserved for historical audit.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Valid override ID.
// @Description
// @Description  **State Transition:** CapacityOverride.IsActive: true -> false.
// @Description
// @Description  **Side Effects:** No DailySchedule rows are touched - capacity is recomputed live on next booking.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid ID
// @Description  - 401 Unauthorized
// @Description  - 404 override not found
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        id path string true "Override ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.DeptHeadErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/overrides/{id} [delete]
func (h *DepartmentHeadHandler) DeleteOverride(c *gin.Context) {
	overrideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid override ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	if err := h.capacityUC.DeleteOverride(c.Request.Context(), overrideID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Capacity override deleted successfully"})
}

// ListOverridesByMonth godoc
// @Summary      List capacity overrides filtered by year and optional month
// @Description  Returns overrides for the department whose target_date falls in the given year (required) and optional month. Both active and inactive rows are returned, ordered ascending so a calendar UI can render them in date order.
// @Description
// @Description  **Roles:** DEPT_HEAD (scope inferred from token).
// @Description
// @Description  **Prerequisites:** year must be a positive integer (e.g. 2026). month is optional (1-12).
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid or missing year / month out of range
// @Description  - 401 Unauthorized / scope missing
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        year  query int true  "Year (e.g. 2026)"
// @Param        month query int false "Month (1-12); omit for the entire year"
// @Success      200 {object} dto.DeptHeadOverrideByMonthResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/overrides/by-month [get]
func (h *DepartmentHeadHandler) ListOverridesByMonth(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}

	yearStr := c.Query("year")
	if yearStr == "" {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "year query parameter is required"})
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "year must be a positive integer"})
		return
	}

	month := 0
	if mStr := c.Query("month"); mStr != "" {
		m, err := strconv.Atoi(mStr)
		if err != nil || m < 1 || m > 12 {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "month must be between 1 and 12"})
			return
		}
		month = m
	}

	overrides, err := h.capacityUC.ListOverridesByYearMonth(c.Request.Context(), hospID, deptID, year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"year":    year,
		"month":   month,
		"data":    overrides,
	})
}

// GetCapacityDetail godoc
// @Summary      Get rich capacity detail for a date
// @Description  Returns the live max/booked/overbook capacity for a date along with the soft staff hint (HospitalDepartment.MaxCapacityOfStaff) and the distinct count of doctors currently assigned to triage rows. The staff fields are advisory only; the booking engine does NOT gate on them.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session with hospital + department scope. date must be YYYY-MM-DD.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid or missing date
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        date query string true "Target date (YYYY-MM-DD)"
// @Success      200 {object} dto.DeptHeadCapacityDetailResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/detail [get]
func (h *DepartmentHeadHandler) GetCapacityDetail(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "date query parameter is required (YYYY-MM-DD)"})
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid date format; expected YYYY-MM-DD"})
		return
	}

	detail, err := h.capacityUC.GetCapacityDetail(c.Request.Context(), hospID, deptID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetScheduledPatients godoc
// @Summary      List scheduled patients for a date
// @Description  Returns the triage rows scheduled for the given date in any "occupying-the-slot" arrival status (Expected, Arrived, Admitted). Referral + Patient are eagerly loaded so the UI can render names without a follow-up call.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session. date must be YYYY-MM-DD.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid or missing date
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        date query string true "Target date (YYYY-MM-DD)"
// @Success      200 {object} dto.DeptHeadScheduledPatientsResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/schedule/patients [get]
func (h *DepartmentHeadHandler) GetScheduledPatients(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "date query parameter is required (YYYY-MM-DD)"})
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid date format; expected YYYY-MM-DD"})
		return
	}

	patients, err := h.capacityUC.GetScheduledPatientsForDate(c.Request.Context(), hospID, deptID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"date":    dateStr,
		"total":   len(patients),
		"data":    patients,
	})
}

// GetCapacityCalendar godoc
// @Summary      Per-day capacity rollup for a calendar month
// @Description  Returns one entry per day of the requested (year, month) with the live max / overbook / booked / available slots and flags indicating whether an active CapacityOverride or a DailySchedule log exists for the day.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** year required, month required (1-12).
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid or missing year / month
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        year  query int true "Year (e.g. 2026)"
// @Param        month query int true "Month (1-12)"
// @Success      200 {object} dto.DeptHeadCapacityCalendarResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/calendar [get]
func (h *DepartmentHeadHandler) GetCapacityCalendar(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	yearStr := c.Query("year")
	monthStr := c.Query("month")
	if yearStr == "" || monthStr == "" {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "year and month query parameters are required"})
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year <= 0 {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "year must be a positive integer"})
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "month must be between 1 and 12"})
		return
	}

	calendar, err := h.capacityUC.BuildCapacityCalendar(c.Request.Context(), hospID, deptID, year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"year":    year,
		"month":   month,
		"data":    calendar,
	})
}

// UpdateStaffCapacity godoc
// @Summary      Update HospitalDepartment.MaxCapacityOfStaff (soft hint)
// @Description  Persists the staffing "soft hint" for the caller's hospital + department. This value is returned via /capacity/detail but is NOT enforced by the booking engine - it exists purely so the UI can warn dept heads when staff_assigned >= staff_capacity.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** max_capacity_of_staff must be >= 0.
// @Description
// @Description  **Side Effects:**
// @Description  - Updates HospitalDepartment.MaxCapacityOfStaff.
// @Description  - Emits an audit log row (UPDATE_SYSTEM_CONFIG with the new value).
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid payload / negative value
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Accept       json
// @Produce      json
// @Param        body body dto.UpdateStaffCapacityRequest true "Soft staff hint"
// @Success      200 {object} dto.DeptHeadStaffCapacityUpdateResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/staff-capacity [put]
func (h *DepartmentHeadHandler) UpdateStaffCapacity(c *gin.Context) {
	hospID, deptID, ok := h.scopedHospDept(c)
	if !ok {
		return
	}

	var req dto.UpdateStaffCapacityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	userIDVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else if uid, ok := userIDVal.(*uuid.UUID); ok && uid != nil {
		userID = *uid
	}

	if err := h.capacityUC.UpdateStaffCapacity(c.Request.Context(), hospID, deptID, req.MaxCapacityOfStaff, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "staff capacity (soft hint) updated"})
}

// scopedHospDept is a tiny helper that pulls hospID + deptID off the
// gin context, fails the response with 401 if either is missing, and
// returns (hospID, deptID, ok). All the dept-head endpoints that scope
// to the caller's department use this to avoid copy-pasting the same
// type assertions.
func (h *DepartmentHeadHandler) scopedHospDept(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
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

// GetOverride godoc
// @Summary      Get a single capacity override by ID
// @Description  Fetches one CapacityOverride row. Useful for the UI before deciding to delete + recreate (overrides are immutable).
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Valid override UUID.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 400 invalid ID
// @Description  - 401 Unauthorized
// @Description  - 404 override not found
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        id path string true "Override ID (UUID)"
// @Success      200 {object} dto.DeptHeadOverrideDetailResponse
// @Failure      400 {object} dto.DeptHeadErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.DeptHeadErrorResponse
// @Failure      500 {object} dto.DeptHeadErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/overrides/{id} [get]
func (h *DepartmentHeadHandler) GetOverride(c *gin.Context) {
	overrideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid override ID"})
		return
	}

	override, err := h.capacityUC.GetOverride(c.Request.Context(), overrideID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.BaseResponse{Success: false, Message: "override not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    override,
	})
}
