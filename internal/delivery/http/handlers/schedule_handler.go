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
// @Summary      View department schedule
// @Description  Returns daily schedule records for the next 30 days for the department.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        start_date query string false "Start date (YYYY-MM-DD)"
// @Param        end_date   query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
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
		end = start.AddDate(0, 0, 30) // Default 30-day window
	}

	schedules, err := h.capacityUC.GetSchedule(c.Request.Context(), hospID, deptID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schedules,
	})
}

// UpdateMaxSlots godoc
// @Summary      Update Daily Max Slots
// @Description  Manually adjust the maximum slots for a specific day.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 400 invalid schedule ID or input
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Accept       json
// @Produce      json
// @Param        id path string true "Schedule ID"
// @Param        body body dto.UpdateMaxSlotsRequest true "Update details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.BaseResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/schedule/{id}/max-slots [put]
func (h *ScheduleHandler) UpdateMaxSlots(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid schedule ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	var req dto.UpdateMaxSlotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	if err := h.capacityUC.UpdateMaxSlots(c.Request.Context(), scheduleID, req.MaxSlots, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Max slots updated successfully"})
}
