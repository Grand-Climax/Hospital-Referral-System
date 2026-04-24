package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type JobHandler struct {
	capacityUC iusecase.CapacityManagementUseCase
}

func NewJobHandler(capacityUC iusecase.CapacityManagementUseCase) *JobHandler {
	return &JobHandler{capacityUC: capacityUC}
}

// ExtendDailySchedule triggers the expansion of the rolling capacity window.
// Typically called by a nightly cron job.
func (h *JobHandler) ExtendDailySchedule(c *gin.Context) {
	err := h.capacityUC.ExtendSchedules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{
			Success: false,
			Message: "Failed to extend schedules: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Rolling schedule window extended successfully",
	})
}
