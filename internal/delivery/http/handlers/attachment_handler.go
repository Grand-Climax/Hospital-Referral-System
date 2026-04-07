package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type AttachmentHandler struct {
	attachmentUC iusecase.AttachmentUseCase
}

func NewAttachmentHandler(uc iusecase.AttachmentUseCase) *AttachmentHandler {
	return &AttachmentHandler{attachmentUC: uc}
}

func toAttachmentResponse(a *entity.Attachment) dto.AttachmentResponse {
	return dto.AttachmentResponse{
		ID:          a.ID.String(),
		ReferralID:  a.ReferralID.String(),
		FileName:    a.FileName,
		FileType:    a.FileType,
		FileSize:    a.FileSize,
		Category:    a.Category,
		StoragePath: a.StoragePath,
		Metadata:    a.Metadata,
		UploadedAt:  a.UploadedAt,
	}
}

// UploadAttachment godoc
// @Summary      Register Referral Attachments
// @Description  Supports two modes:
// @Description  1. **Bulk JSON**: Register multiple Cloudinary-uploaded files.
// @Description  2. **Direct File**: Upload a single file (Max 20MB) directly via `multipart/form-data`.
// @Description  Limited to Referral Doctors on Draft/NeedRevision referrals.
// @Tags         Attachments
// @Accept       json
// @Accept       mpfd
// @Produce      json
// @Param        id   path string true "Referral ID"
// @Param        body body dto.BulkAttachmentRequest false "Bulk JSON payload"
// @Param        file formData file false "Direct file upload"
// @Param        category formData string false "Attachment category (e.g. LAB_REPORT, DICOM_XRAY)"
// @Success      201 {object} dto.AttachmentListResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/attachments [post]
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid Referral UUID"})
		return
	}

	userIDVal, _ := c.Get("userID")
	doctorID, _ := userIDVal.(uuid.UUID)

	contentType := c.Request.Header.Get("Content-Type")

	// --- Mode 1: Multipart File Upload ---
	if strings.Contains(contentType, "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "File field is required for multipart upload"})
			return
		}

		// 20MB Limit for direct-to-backend referral attachments
		if fileHeader.Size > 20*1024*1024 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "File size exceeds 20MB limit"})
			return
		}

		category := c.PostForm("category")
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to open file"})
			return
		}
		defer file.Close()

		att, err := h.attachmentUC.UploadAndAddAttachment(c.Request.Context(), referralID, doctorID, file, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), category, fileHeader.Size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}

		c.JSON(http.StatusCreated, dto.AttachmentListResponse{
			Data: []dto.AttachmentResponse{toAttachmentResponse(att)},
			BaseResponse: dto.BaseResponse{
				Success: true,
				Message: "Attachment uploaded and registered successfully",
			},
		})
		return
	}

	// --- Mode 2: Bulk JSON Registration (Hybrid Pattern) ---
	var req dto.BulkAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid JSON or Multipart payload: " + err.Error()})
		return
	}

	attachments, err := h.attachmentUC.AddAttachmentsToReferral(c.Request.Context(), referralID, doctorID, req.Attachments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	var data []dto.AttachmentResponse
	for i := range attachments {
		data = append(data, toAttachmentResponse(&attachments[i]))
	}

	c.JSON(http.StatusCreated, dto.AttachmentListResponse{
		Data: data,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Attachments registered successfully",
		},
	})
}

// GetAttachment godoc
// @Summary      Get Attachment Details
// @Description  Retrieve metadata for a specific attachment by its ID.
// @Tags         Attachments
// @Produce      json
// @Param        id path string true "Attachment ID"
// @Success      200 {object} dto.AttachmentResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/attachments/{id} [get]
func (h *AttachmentHandler) GetAttachment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid ID"})
		return
	}

	attachment, err := h.attachmentUC.GetAttachment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Attachment not found"})
		return
	}

	resp := toAttachmentResponse(attachment)
	resp.Success = true
	c.JSON(http.StatusOK, resp)
}

// DeleteFromReferral godoc
// @Summary      Delete Attachment From Referral
// @Description  Remove attachment from referral. Only allowed for Draft/NeedRevision status by the referring doctor.
// @Tags         Attachments
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        attachment_id path string true "Attachment ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/attachments/{attachment_id} [delete]
func (h *AttachmentHandler) DeleteFromReferral(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid Referral ID"})
		return
	}
	attachmentID, err := uuid.Parse(c.Param("attachment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid Attachment ID"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "User ID not found in context"})
		return
	}
	doctorID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid user ID type"})
		return
	}

	if err := h.attachmentUC.DeleteAttachmentFromReferral(c.Request.Context(), referralID, attachmentID, doctorID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Attachment deleted successfully",
	})
}

// GetUploadSignature godoc
// @Summary      Get Secure Upload Signature
// @Description  Request a cryptographic signature from the backend to upload files directly to Cloudinary.
// @Tags         Attachments
// @Produce      json
// @Success      200 {object} dto.UploadSignatureResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/attachments/signature [get]
func (h *AttachmentHandler) GetUploadSignature(c *gin.Context) {
	// 1. Identify Hospital (from user context set by middleware)
	hospIdVal, exists := c.Get("hospID")
	if !exists || hospIdVal == nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Hospital context missing"})
		return
	}

	var hospitalID uuid.UUID
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospitalID = hID
	} else if hIDPtr, ok := hospIdVal.(*uuid.UUID); ok && hIDPtr != nil {
		hospitalID = *hIDPtr
	} else {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid hospital context type"})
		return
	}

	data, err := h.attachmentUC.GenerateSignature(hospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.UploadSignatureResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Upload signature generated successfully",
		},
		Signature: data["signature"].(string),
		Timestamp: data["timestamp"].(int64),
		APIKey:    data["api_key"].(string),
		CloudName: data["cloud_name"].(string),
		Folder:    data["folder"].(string),
	})
}

// GetReferralAttachments godoc
// @Summary      List Referral Attachments
// @Description  Get all attachments associated with a specific referral
// @Tags         Attachments
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.AttachmentListResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/attachments [get]
func (h *AttachmentHandler) GetReferralAttachments(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid ID"})
		return
	}

	attachments, err := h.attachmentUC.GetAttachmentsByReferralID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	var data []dto.AttachmentResponse
	for i := range attachments {
		data = append(data, toAttachmentResponse(&attachments[i]))
	}

	c.JSON(http.StatusOK, dto.AttachmentListResponse{
		Data: data,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Attachments retrieved successfully",
		},
	})
}
