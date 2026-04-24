package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type CapacityHandler struct {
	capacityUC iusecase.CapacityManagementUseCase
}

func NewCapacityHandler(capacityUC iusecase.CapacityManagementUseCase) *CapacityHandler {
	return &CapacityHandler{capacityUC: capacityUC}
}


func (h *CapacityHandler) ListOverrides(c *gin.Context) {
	hospIDVal, _ := c.Get("hospID")
	hID, _ := hospIDVal.(*uuid.UUID)
	deptIDVal, _ := c.Get("deptID")
	dID, _ := deptIDVal.(*uuid.UUID)

	if hID == nil || dID == nil {
		c.JSON(http.StatusForbidden, dto.BaseResponse{Success: false, Message: "Hospital or Department ID missing from context"})
		return
	}

	overrides, err := h.capacityUC.GetOverrides(c.Request.Context(), *hID, *dID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overrides,
	})
}

func (h *CapacityHandler) CreateOverride(c *gin.Context) {
	hospID, _ := c.Get("hospID")
	hID, _ := hospID.(*uuid.UUID)
	deptID, _ := c.Get("deptID")
	dID, _ := deptID.(*uuid.UUID)
	
	if hID == nil || dID == nil {
		c.JSON(http.StatusForbidden, dto.BaseResponse{Success: false, Message: "Hospital or Department ID missing from context"})
		return
	}

	userID, _ := c.Get("userID")
	uID, _ := userID.(uuid.UUID)

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

	if err := h.capacityUC.CreateOverride(c.Request.Context(), *hID, *dID, date, req.NewLimit, req.Reason, uID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.BaseResponse{Success: true, Message: "Capacity override created successfully"})
}

func (h *CapacityHandler) UpdateOverride(c *gin.Context) {
	overrideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid override ID"})
		return
	}

	userID, _ := c.Get("userID")
	uID, _ := userID.(uuid.UUID)

	var req dto.UpdateOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	if err := h.capacityUC.UpdateOverride(c.Request.Context(), overrideID, req.NewLimit, req.Reason, uID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Capacity override updated successfully"})
}

func (h *CapacityHandler) DeleteOverride(c *gin.Context) {
	overrideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "invalid override ID"})
		return
	}

	userID, _ := c.Get("userID")
	uID, _ := userID.(uuid.UUID)

	if err := h.capacityUC.DeleteOverride(c.Request.Context(), overrideID, uID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Capacity override deleted successfully"})
}
