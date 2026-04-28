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
	schedUC    iusecase.SchedulingUseCase
}

func NewDepartmentHeadHandler(capacityUC iusecase.CapacityManagementUseCase, schedUC iusecase.SchedulingUseCase) *DepartmentHeadHandler {
	return &DepartmentHeadHandler{
		capacityUC: capacityUC,
		schedUC:    schedUC,
	}
}

// BatchSchedule godoc
// @Summary      Run Batch Scheduling
// @Description  Triggers the automated batch scheduling process for all WAITING referrals in priority order.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Prerequisites:** Reads all WAITING entries for the department.
// @Description  **State Transition:** Assigns appointment dates to referrals and creates status history records.
// @Description  **Gatekeepers:** Respects buffer days and never overbooks.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
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
// @Summary      List Capacity Overrides
// @Description  Get all active and upcoming capacity overrides for the department.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
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
// @Summary      Create Capacity Override
// @Description  Sets a temporary capacity override for a specific date.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Side Effect:** Automatically synchronizes the daily schedule for that date.
// @Description  **Common Errors:**
// @Description  - 400 (date in past)
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Accept       json
// @Produce      json
// @Param        body body dto.CreateOverrideRequest true "Override details"
// @Success      201 {object} dto.BaseResponse
// @Failure      400 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
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

// UpdateOverride godoc
// @Summary      Update Capacity Override
// @Description  Updates an existing capacity override's limit and reason.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 400 invalid ID or input
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Accept       json
// @Produce      json
// @Param        id path string true "Override ID"
// @Param        body body dto.UpdateOverrideRequest true "Update details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/department-head/capacity/overrides/{id} [put]
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

// DeleteOverride godoc
// @Summary      Delete Capacity Override
// @Description  Removes a capacity override, reverting the daily schedule to standard limits.
// @Description  **Roles:** DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 400 invalid ID
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Department Head
// @Produce      json
// @Param        id path string true "Override ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.BaseResponse
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

