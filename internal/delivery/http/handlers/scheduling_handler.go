package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type SchedulingHandler struct {
	schedUC iusecase.SchedulingUseCase
}

func NewSchedulingHandler(schedUC iusecase.SchedulingUseCase) *SchedulingHandler {
	return &SchedulingHandler{schedUC: schedUC}
}

func (h *SchedulingHandler) GetCapacity(c *gin.Context) {
	hospID, _ := uuid.Parse(c.GetString("hosp_id"))
	deptID, _ := uuid.Parse(c.GetString("dept_id"))
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))

	status, err := h.schedUC.GetCapacityStatus(c.Request.Context(), hospID, deptID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func (h *SchedulingHandler) Schedule(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid referral ID"})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.SchedulingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	if err := h.schedUC.ScheduleAppointment(c.Request.Context(), referralID, userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Appointment scheduled successfully"})
}

func (h *SchedulingHandler) BatchSchedule(c *gin.Context) {
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

	result, err := h.schedUC.BatchSchedule(c.Request.Context(), hospID, deptID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
