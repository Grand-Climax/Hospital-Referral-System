package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type CronHandler struct {
	attachmentUC iusecase.AttachmentUseCase
}

func NewCronHandler(attachUC iusecase.AttachmentUseCase) *CronHandler {
	return &CronHandler{attachmentUC: attachUC}
}

// ValidateAttachments godoc
// @Summary      Batch Verification Cron
// @Description  Internal endpoint triggered by GCP Cloud Scheduler. Extracts metadata from PENDING attachments and promotes valid files to permanent storage.
// @Description  **Roles:** INTERNAL_CRON (Protected by GCP OIDC)
// @Description  **Common Errors:**
// @Description  - 500 Internal Server Error
// @Tags         Cron
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/cron/verify-attachments [post]
func (h *CronHandler) ValidateAttachments(c *gin.Context) {
	// In production, you would verify an OIDC token from Cloud Scheduler here
	if err := h.attachmentUC.VerifyPendingAttachments(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Validation cron job completed"})
}

// CleanupTemp godoc
// @Summary      Cleanup Temporary Folders
// @Description  Internal endpoint triggered by GCP Cloud Scheduler. Deletes empty temp folders in Cloudinary to keep the storage clean.
// @Description  **Roles:** INTERNAL_CRON (Protected by GCP OIDC)
// @Description  **Common Errors:**
// @Description  - 500 Internal Server Error
// @Tags         Cron
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/cron/cleanup-temp [post]
func (h *CronHandler) CleanupTemp(c *gin.Context) {
	// In production, verify OIDC token here
	if err := h.attachmentUC.CleanupTempAttachments(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Cleanup cron job completed"})
}
