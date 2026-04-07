package handlers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type WebhookHandler struct {
	attachmentUC iusecase.AttachmentUseCase
	storage      iinfra.StorageService
}

func NewWebhookHandler(attachUC iusecase.AttachmentUseCase, storage iinfra.StorageService) *WebhookHandler {
	return &WebhookHandler{
		attachmentUC: attachUC,
		storage:      storage,
	}
}

// HandleCloudinaryWebhook godoc
// @Summary      Handle Cloudinary Upload Notifications
// @Description  Endpoint for Cloudinary to notify the backend about successful direct uploads. Signature verification is required.
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Router       /api/v1/webhooks/cloudinary [post]
func (h *WebhookHandler) HandleCloudinaryWebhook(c *gin.Context) {
	// 1. Read raw body for verification
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Failed to read body"})
		return
	}
	// Restore body for subsequent parsing if needed (though we use the slice here)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 2. Verify Signature
	headers := map[string]string{
		"X-Cld-Signature": c.GetHeader("X-Cld-Signature"),
		"X-Cld-Timestamp": c.GetHeader("X-Cld-Timestamp"),
	}

	isValid, err := h.storage.VerifyWebhookSignature(headers, body)
	if err != nil || !isValid {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "Invalid webhook signature"})
		return
	}

	// 3. Delegate to UseCase
	// We pass the raw body as a generic map to the usecase for flexible metadata extraction
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid payload"})
		return
	}

	if err := h.attachmentUC.ProcessWebhookAttachment(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Webhook processed successfully",
	})
}
