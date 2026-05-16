package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type NotificationHandler struct {
	notifUC iusecase.NotificationUseCase
}

func NewNotificationHandler(notifUC iusecase.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{notifUC: notifUC}
}

// ListNotifications godoc
// @Summary      List Notifications
// @Description  Retrieve a paginated list of notifications with various filters.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD, RECEPTIONIST
// @Description  **Scoping:**
// @Description  - SYSTEM_SUPER_ADMIN: Global access, can filter by any hospital/department.
// @Description  - HOSPITAL_ADMIN: Scoped to their hospital.
// @Description  - DEPT_HEAD / RECEPTIONIST: Scoped to their hospital and department.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 403 Forbidden
// @Description  - 500 Internal Server Error
// @Tags         SMS Notifications
// @Param        page            query int    false "Page number" default(1)
// @Param        page_size       query int    false "Page size"   default(20)
// @Param        referral_id     query string false "Filter by Referral ID"
// @Param        hospital_id     query string false "Filter by Hospital ID (Ignored if not Super Admin)"
// @Param        department_id   query string false "Filter by Department ID (Ignored if not Super Admin/Hospital Admin)"
// @Param        notif_type      query string false "Filter by Notification Type" Enums(ACCEPTANCE, SCHEDULING, REMINDER, RESCHEDULE)
// @Param        delivery_status query string false "Filter by Delivery Status" Enums(QUEUED, MANUAL_REQUIRED, SENT, FAILED)
// @Description  Notification types: ACCEPTANCE (referral accepted), SCHEDULING (appointment scheduled), REMINDER (appointment reminder), RESCHEDULE (appointment changed).
// @Description  Delivery status: QUEUED (awaiting auto-send), MANUAL_REQUIRED (needs manual trigger), SENT (sent to provider), FAILED (provider error).
// @Produce      json
// @Success      200 {object} dto.NotificationListResponse
// @Security     BearerAuth
// @Router       /api/v1/internal/notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	role, _ := c.Get("role")
	userRole := role.(entity.UserRole)

	filter := irepository.NotificationListFilter{
		Page:     1,
		PageSize: 20,
	}

	if p, err := strconv.Atoi(c.Query("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil {
		filter.PageSize = ps
	}

	if refIDStr := c.Query("referral_id"); refIDStr != "" {
		if id, err := uuid.Parse(refIDStr); err == nil {
			filter.ReferralID = &id
		}
	}

	// Scoping logic
	if userRole != entity.RoleSystemSuperAdmin {
		hospIdVal, _ := c.Get("hospID")
		if hID, ok := hospIdVal.(uuid.UUID); ok && hID != uuid.Nil {
			filter.HospitalID = &hID
		} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil && *hID != uuid.Nil {
			filter.HospitalID = hID
		}

		if userRole == entity.RoleDeptHead || userRole == entity.RoleReceptionist {
			deptIdVal, _ := c.Get("deptID")
			if dID, ok := deptIdVal.(uuid.UUID); ok && dID != uuid.Nil {
				filter.DepartmentID = &dID
			} else if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil && *dID != uuid.Nil {
				filter.DepartmentID = dID
			}
		}
	}

	// If Super Admin, allow manual hospital/dept filtering
	if userRole == entity.RoleSystemSuperAdmin {
		if hospIDStr := c.Query("hospital_id"); hospIDStr != "" {
			if id, err := uuid.Parse(hospIDStr); err == nil {
				filter.HospitalID = &id
			}
		}
		if deptIDStr := c.Query("department_id"); deptIDStr != "" {
			if id, err := uuid.Parse(deptIDStr); err == nil {
				filter.DepartmentID = &id
			}
		}
	} else if userRole == entity.RoleHospitalAdmin {
		// Hospital Admin can still filter by department within their hospital
		if deptIDStr := c.Query("department_id"); deptIDStr != "" {
			if id, err := uuid.Parse(deptIDStr); err == nil {
				filter.DepartmentID = &id
			}
		}
	}

	if notifType := c.Query("notif_type"); notifType != "" {
		nt := entity.NotificationType(notifType)
		filter.NotificationType = &nt
	}
	if status := c.Query("delivery_status"); status != "" {
		ds := entity.DeliveryStatus(status)
		filter.DeliveryStatus = &ds
	}

	notifications, total, err := h.notifUC.ListNotifications(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.NotificationListResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Notifications retrieved successfully"},
		Data:         notifications,
		Total:        total,
		Page:         filter.Page,
	})
}

// TriggerManualSend godoc
// @Summary      Manual Trigger Notification Send
// @Description  Processes up to 50 queued SMS notifications. Optionally filter by hospital/department.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **Prerequisites:** Authenticated administrative session.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         SMS Notifications
// @Param        hospital_id   query string false "Filter by Hospital ID"
// @Param        department_id query string false "Filter by Department ID"
// @Produce      json
// @Success      200 {object} dto.NotificationSendSummary
// @Security     BearerAuth
// @Router       /api/v1/internal/notifications/send [post]
func (h *NotificationHandler) TriggerManualSend(c *gin.Context) {
	var hospID, deptID *uuid.UUID

	if hIDStr := c.Query("hospital_id"); hIDStr != "" {
		if id, err := uuid.Parse(hIDStr); err == nil {
			hospID = &id
		}
	}
	if dIDStr := c.Query("department_id"); dIDStr != "" {
		if id, err := uuid.Parse(dIDStr); err == nil {
			deptID = &id
		}
	}

	summary, err := h.notifUC.TriggerManualSend(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// Resend godoc
// @Summary      Manual Resend Notification
// @Description  Resends a failed or queued notification.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 400 invalid notification ID
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         SMS Notifications
// @Param        id path string true "Notification ID"
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/internal/notifications/{id}/resend [post]
func (h *NotificationHandler) Resend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Success: false, Message: "Invalid notification ID"})
		return
	}

	n, err := h.notifUC.ResendNotification(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification resent successfully",
		"data":    n,
	})
}

// UpdateStatus godoc
// @Summary      Manual Sync Delivery Status
// @Description  Sync delivery status of SENT notifications with the SMS provider (AfroMessage).
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         SMS Notifications
// @Produce      json
// @Success      200 {object} dto.NotificationStatusSummary
// @Security     BearerAuth
// @Router       /api/v1/internal/notifications/update-status [post]
func (h *NotificationHandler) UpdateStatus(c *gin.Context) {
	summary, err := h.notifUC.UpdateStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// SMSWebhook godoc
// @Summary      SMS Webhook
// @Description  Endpoint for AfroMessage to push status updates.
// @Description  **Roles:** SMS_PROVIDER (Public)
// @Description  **Common Errors:**
// @Description  - 500 Internal Server Error
// @Tags         SMS Notifications
// @Param        message_id query string true "Provider Message ID"
// @Param        status     query string true "Delivery Status"
// @Success      200 {object} dto.BaseResponse
// @Router       /api/v1/internal/notifications/webhook [get]
func (h *NotificationHandler) SMSWebhook(c *gin.Context) {
	messageID := c.Query("message_id")
	status := c.Query("status")

	if err := h.notifUC.HandleSMSWebhook(c.Request.Context(), messageID, status); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Webhook processed"})
}
