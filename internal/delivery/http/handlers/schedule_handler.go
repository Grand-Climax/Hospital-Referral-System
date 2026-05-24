package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ScheduleHandler struct {
	capacityUC iusecase.CapacityManagementUseCase
}

func NewScheduleHandler(capacityUC iusecase.CapacityManagementUseCase) *ScheduleHandler {
	return &ScheduleHandler{capacityUC: capacityUC}
}

// GetSchedule godoc
// @Summary      View department schedule history log
// @Description  Returns the immutable DailySchedule history log for an inclusive date range. Under the Schedule-on-Demand model each row is a snapshot captured at the first booking for that date; only BookedSlots is updated thereafter and MaxSlots / OverbookLimit are frozen.
// @Description
// @Description  **Roles:** DEPT_HEAD
// @Description
// @Description  **Prerequisites:** Authenticated DEPT_HEAD session - hospital and department are inferred from the token.
// @Description
// @Description  **Side Effects:** None. Read-only.
// @Description
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized / scope missing
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        start_date query string false "Start date (YYYY-MM-DD); defaults to today"
// @Param        end_date   query string false "End date (YYYY-MM-DD); defaults to start_date + 30 days"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/schedule [get]
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
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

	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		start = time.Now()
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		end = start.AddDate(0, 0, 30)
	}

	schedules, err := h.capacityUC.GetSchedule(c.Request.Context(), hospID, deptID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	// Single-day request with no matching log: return a clear "no log yet"
	// payload so the UI can render a friendly placeholder. Under the
	// Schedule-on-Demand model a missing row simply means no booking has
	// happened for that date, not an error.
	if start.Equal(end) {
		if len(schedules) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success":      true,
				"data":         nil,
				"has_schedule": false,
				"message":      "No schedule log for this date yet",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"data":         schedules[0],
			"has_schedule": true,
		})
		return
	}

	// Range request - empty range still gets a 200 with a guidance message
	// rather than a silent empty list.
	if len(schedules) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"data":         []interface{}{},
			"has_schedule": false,
			"message":      "No schedule logs for the requested period",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"data":         schedules,
		"has_schedule": true,
	})
}
