package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type NotificationHandler struct {
	notifUC iusecase.NotificationUseCase
}

func NewNotificationHandler(notifUC iusecase.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{notifUC: notifUC}
}

func (h *NotificationHandler) TriggerManualSend(c *gin.Context) {
	summary, err := h.notifUC.TriggerManualSend(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

func (h *NotificationHandler) SMSWebhook(c *gin.Context) {
	messageID := c.Query("message_id")
	status := c.Query("status")

	if err := h.notifUC.HandleSMSWebhook(c.Request.Context(), messageID, status); err != nil {
		c.JSON(http.StatusInternalServerError, dto.BaseResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Webhook processed"})
}
