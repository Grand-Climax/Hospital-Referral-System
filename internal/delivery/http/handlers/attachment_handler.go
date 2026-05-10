package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/pkg/utils"
)

type AttachmentHandler struct {
	attachmentUC iusecase.AttachmentUseCase
}

func NewAttachmentHandler(uc iusecase.AttachmentUseCase) *AttachmentHandler {
	return &AttachmentHandler{attachmentUC: uc}
}

func toAttachmentResponse(a *entity.Attachment) dto.AttachmentResponse {
	return dto.AttachmentResponse{
		ID:                 a.ID.String(),
		ReferralID:         a.ReferralID.String(),
		FileName:           a.FileName,
		FileType:           a.FileType,
		FileSize:           a.FileSize,
		Category:           a.Category,
		StoragePath:        a.StoragePath,
		VerificationStatus: a.VerificationStatus,
		RejectionReason:    a.RejectionReason,
		RejectedAt:         a.RejectedAt,
		Metadata:           a.Metadata,
		UploadedAt:         a.UploadedAt,
		BaseResponse: dto.BaseResponse{
			Success: true,
		},
	}
}

// UploadAttachment godoc
// @Summary      Upload Attachment to Referral
// @Description  Upload a single clinical file to an existing referral. The file is immediately verified and stored permanently.
// @Description
// @Description  **Request format:** multipart/form-data
// @Description  - **file**: The file to upload (max 20MB).
// @Description  - **category**: Category for the attachment (e.g., RADIOLOGY, LAB_REPORT, DICOM_XRAY). Defaults to GENERAL_CLINICAL.
// @Description
// @Description  **Verification:** DICOM and PDF files are checked for valid metadata.
// @Description  - On success: attachment status set to VERIFIED.
// @Description  - On failure: attachment status set to REJECTED and referral moves to NEED_REVISION.
// @Description
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Referral must be in DRAFT or NEED_REVISION status.
// @Description  **Common Errors:**
// @Description  - 400 Invalid ID or file too large (> 20MB)
// @Description  - 403 Forbidden (not the creator or invalid status)
// @Tags         Attachments
// @Accept       mpfd
// @Produce      json
// @Param        id       path     string true "Referral ID"
// @Param        file     formData file   true "File to upload"
// @Param        category formData string false "Category (e.g. RADIOLOGY, LAB_REPORT, DICOM_XRAY)"
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

	// --- Multipart File Upload ---
	if strings.Contains(contentType, "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "File field is required for multipart upload"})
			return
		}

		// 20MB Limit
		if fileHeader.Size > 20*1024*1024 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "File size exceeds 20MB limit"})
			return
		}

		if _, err := utils.ValidateFileExtension(fileHeader.Filename); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}

		category := c.PostForm("category")
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to open file"})
			return
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to read file"})
			return
		}

		att, err := h.attachmentUC.UploadAndAddAttachment(c.Request.Context(), referralID, doctorID, fileBytes, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), category, fileHeader.Size)
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

	c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Only multipart/form-data is supported for file upload"})
}

// GetAttachment godoc
// @Summary      Get Attachment Details
// @Description  Retrieve metadata for a specific attachment by its ID.
// @Description  **Roles:** All authenticated roles with referral access.
// @Description  **Common Errors:**
// @Description  - 404 Not Found
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
// @Description  Remove attachment from referral.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Only allowed for Draft/NeedRevision status by the referring doctor.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (invalid status or not owner)
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


// GetReferralAttachments godoc
// @Summary      List Referral Attachments
// @Description  Get all attachments associated with a specific referral.
// @Description  **Roles:** All authenticated roles with referral access.
// @Description  **Common Errors:**
// @Description  - 404 Referral not found
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
