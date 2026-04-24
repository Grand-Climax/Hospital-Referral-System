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

func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIDStr := c.Query("dept_id")
	deptID, _ := uuid.Parse(deptIDStr)
	
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		start = time.Now()
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		end = start.AddDate(0, 0, 14)
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

func (h *ScheduleHandler) UpdateMaxSlots(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid schedule ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

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

func (h *ScheduleHandler) BatchSchedule(c *gin.Context) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}
	deptIDStr := c.Query("dept_id")
	deptID, _ := uuid.Parse(deptIDStr)
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	result, err := h.capacityUC.BatchSchedule(c.Request.Context(), hospID, deptID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
