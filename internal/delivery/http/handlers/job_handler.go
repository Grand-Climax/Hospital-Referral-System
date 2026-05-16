package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type JobHandler struct {
	capacityUC   iusecase.CapacityManagementUseCase
	notifUC      iusecase.NotificationUseCase
	dailyWeightUC iusecase.DailyWeightUseCase
	schedulerUC   iusecase.SchedulerServiceUseCase
	schedulingUC  iusecase.SchedulingUseCase
}

func NewJobHandler(
	capacityUC iusecase.CapacityManagementUseCase,
	notifUC iusecase.NotificationUseCase,
	dailyWeightUC iusecase.DailyWeightUseCase,
	schedulerUC iusecase.SchedulerServiceUseCase,
	schedulingUC iusecase.SchedulingUseCase,
) *JobHandler {
	return &JobHandler{
		capacityUC:    capacityUC,
		notifUC:       notifUC,
		dailyWeightUC: dailyWeightUC,
		schedulerUC:   schedulerUC,
		schedulingUC:  schedulingUC,
	}
}

// ExtendDailySchedule godoc
// @Summary      Extend Daily Schedule Window
// @Description  Triggers the expansion of the rolling capacity window. Typically called by a nightly cron job.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **Prerequisites:** Authenticated administrative session.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Automation Jobs
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/internal/jobs/extend-daily-schedule [post]
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

// SendReminders godoc
// @Summary      Batch Queue Appointment Reminders
// @Description  Finds appointments for tomorrow and queues SMS reminders for patients.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **Prerequisites:** Authenticated administrative session.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Automation Jobs
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/internal/jobs/send-reminders [post]
func (h *JobHandler) SendReminders(c *gin.Context) {
	count, err := h.notifUC.QueueReminders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{
			Success: false,
			Message: "Failed to queue reminders: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Appointment reminders queued successfully",
		"data": map[string]int{
			"queued_count": count,
		},
	})
}

// UpdateWaitingWeights godoc
// @Summary      Manual Trigger for Waiting Weight Updates
// @Description  Increments waiting_hours_weight for all WAITING referrals.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **Constraints:** Idempotent (runs once per calendar day).
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Automation Jobs
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/internal/jobs/update-waiting-weights [post]
func (h *JobHandler) UpdateWaitingWeights(c *gin.Context) {
	userID, _ := c.Get("userID")

	message, err := h.dailyWeightUC.Execute(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{
			Success: false,
			Message: "Failed to update waiting weights: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: message,
	})
}

// RunSchedulerCycle godoc
// @Summary      Run Automated Batch Scheduling Cycle
// @Description  Run one sharded scheduler cycle (processes one department).
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **State Transition:** Moves WAITING referrals to SCHEDULED if capacity exists.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Automation Jobs
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/internal/jobs/run-scheduler-cycle [post]
func (h *JobHandler) RunSchedulerCycle(c *gin.Context) {
	leaseHolder := c.Query("lease_holder")
	if leaseHolder == "" {
		leaseHolder = "scheduler-" + uuid.New().String()[:8]
	}

	result, err := h.schedulerUC.RunSchedulerCycle(c.Request.Context(), leaseHolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{
			Success: false,
			Message: "Failed to run scheduler cycle: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ProcessPendingSMS godoc
// @Summary      Process Pending SMS
// @Description  Sends queued SMS notifications in bulk.
// @Tags         Automation Jobs
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/internal/jobs/process-pending-sms [post]
func (h *JobHandler) ProcessPendingSMS(c *gin.Context) {
	summary, err := h.notifUC.ProcessPendingSMS(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{
			Success: false,
			Message: "Failed to process SMS: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// ProcessMissedAppointments godoc
// @Summary      Process Missed Appointments
// @Description  Marks past expected appointments as missed and flags for review.
// @Tags         Automation Jobs
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/internal/jobs/process-missed [post]
func (h *JobHandler) ProcessMissedAppointments(c *gin.Context) {
	err := h.schedulingUC.ProcessMissedAppointments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{
			Success: false,
			Message: "Failed to process missed appointments: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Missed appointments processed successfully",
	})
}
