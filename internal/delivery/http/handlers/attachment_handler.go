package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type AttachmentHandler struct {
	attachmentUC iusecase.AttachmentUseCase
}

func NewAttachmentHandler(uc iusecase.AttachmentUseCase) *AttachmentHandler {
	return &AttachmentHandler{attachmentUC: uc}
}

// UploadAttachment godoc
// @Summary      Upload Referral Attachment
// @Description  Submit physical files (Images/PDFs/DICOM) explicitly tied to a referral
// @Tags         Attachments
// @Accept       multipart/form-data
// @Produce      json
// @Param        id   path string true "Referral ID"
// @Param        file formData file true "File to upload"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/attachments [post]
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	idStr := c.Param("id")
	referralID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid Referral UUID format"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "File upload required", "details": err.Error()})
		return
	}

	// Basic Ext/Type validation (could be expanded)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".dcm" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid file type. Only PDF, Image, and DICOM formats are allowed."})
		return
	}

	// Generate UUID filename
	newFileName := uuid.New().String() + ext
	storagePath := filepath.Join("storage", "attachments", newFileName)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create storage directory"})
		return
	}

	// Save to local disk (simulation for S3)
	if err := c.SaveUploadedFile(file, storagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to save file physically", "details": err.Error()})
		return
	}

	// Save metadata to DB
	attachment, err := h.attachmentUC.SaveAttachment(c.Request.Context(), referralID, file.Filename, ext, storagePath)
	if err != nil {
		// Cleanup physical file if DB fails
		_ = os.Remove(storagePath)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to map attachment to database", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "File uploaded and attached to referral successfully",
		"data":    attachment,
	})
}

// DownloadAttachment godoc
// @Summary      Download Attachment
// @Description  Stream actual binary file data for an attachment ID
// @Tags         Attachments
// @Produce      application/octet-stream
// @Param        id path string true "Attachment ID"
// @Success      200 {file} file
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/attachments/{id}/download [get]
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	idStr := c.Param("id")
	attachmentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid Attachment UUID format"})
		return
	}

	attachment, err := h.attachmentUC.GetAttachment(c.Request.Context(), attachmentID)
	if err != nil || attachment == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Attachment reference not found"})
		return
	}

	if _, err := os.Stat(attachment.StoragePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Physical file missing from storage"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.FileName))
	c.File(attachment.StoragePath)
}
