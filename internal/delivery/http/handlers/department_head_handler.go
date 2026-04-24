package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)



type DepartmentHeadHandler struct {
	capacityUC iusecase.CapacityManagementUseCase
}

func NewDepartmentHeadHandler(capacityUC iusecase.CapacityManagementUseCase) *DepartmentHeadHandler {
	return &DepartmentHeadHandler{capacityUC: capacityUC}
}

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


func (h *DepartmentHeadHandler) GetSchedule(c *gin.Context) {
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

	// Validate date is not in the past
	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "cannot set override for past dates"})
		return
	}

	if err := h.capacityUC.CreateOverride(c.Request.Context(), hospID, deptID, date, req.NewLimit, req.Reason, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.BaseResponse{Success: true, Message: "Capacity override created successfully"})
}

func (h *DepartmentHeadHandler) UpdateOverride(c *gin.Context) {
	overrideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid override ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	var req dto.UpdateOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	if err := h.capacityUC.UpdateOverride(c.Request.Context(), overrideID, req.NewLimit, req.Reason, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Capacity override updated successfully"})
}

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

func (h *DepartmentHeadHandler) UpdateMaxSlots(c *gin.Context) {
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
