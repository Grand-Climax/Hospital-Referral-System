package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/usecase"
)

type AttachmentHandler struct {
	attachmentUC usecase.AttachmentUseCase
}

func NewAttachmentHandler(uc usecase.AttachmentUseCase) *AttachmentHandler {
	return &AttachmentHandler{attachmentUC: uc}
}

func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	idStr := c.Param("id")
	referralID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Referral UUID format"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload required", "details": err.Error()})
		return
	}

	// Basic Ext/Type validation (could be expanded)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".dcm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only PDF, Image, and DICOM formats are allowed."})
		return
	}

	// Generate UUID filename
	newFileName := uuid.New().String() + ext
	storagePath := filepath.Join("storage", "attachments", newFileName)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create storage directory"})
		return
	}

	// Save to local disk (simulation for S3)
	if err := c.SaveUploadedFile(file, storagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file physically", "details": err.Error()})
		return
	}

	// Save metadata to DB
	attachment, err := h.attachmentUC.SaveAttachment(c.Request.Context(), referralID, file.Filename, ext, storagePath)
	if err != nil {
		// Cleanup physical file if DB fails
		_ = os.Remove(storagePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to map attachment to database", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Attachment uploaded successfully",
		"data":    attachment,
	})
}

func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	idStr := c.Param("id")
	attachmentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Attachment UUID format"})
		return
	}

	attachment, err := h.attachmentUC.GetAttachment(c.Request.Context(), attachmentID)
	if err != nil || attachment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment reference not found"})
		return
	}

	if _, err := os.Stat(attachment.StoragePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Physical file missing from storage"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.FileName))
	c.File(attachment.StoragePath)
}
