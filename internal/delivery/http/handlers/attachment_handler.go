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
	}
}

// UploadAttachment godoc
// @Summary      Register or Upload Referral Attachments
// @Description  Supports two modes for associating clinical data with a referral:
// @Description  1. **Hybrid/Bulk JSON**: Register multiple files already uploaded to Cloudinary.
// @Description     - Requires the `referral_id` pre-minted from the `/attachments/signature` endpoint.
// @Description     - Files MUST be at the `temp/{referral_id}/` path in Cloudinary.
// @Description  2. **Direct File**: Upload a single file (Max 20MB) directly to the backend.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Allowed only when referral status is DRAFT or NEED_REVISION.
// @Description  **Common Errors:**
// @Description  - 400 Invalid ID or file too large
// @Description  - 403 Forbidden (not the creator or invalid status)
// @Tags         Attachments
// @Accept       json
// @Accept       mpfd
// @Produce      json
// @Param        id   path string true "Referral ID (Pre-minted or Existing)"
// @Param        body body dto.BulkAttachmentRequest false "Bulk JSON payload for Hybrid Flow"
// @Param        file formData file false "Direct file upload payload"
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

// GetUploadSignature godoc
// @Summary      Get Secure Upload Signature (Pre-Minted Flow)
// @Description  The first step in creating/updating a referral with attachments.
// @Description  1. Generates a unique **Pre-Minted Referral ID**.
// @Description  2. Provides a cryptographic signature for Cloudinary.
// @Description  3. Frontend MUST upload files to the folder path: `temp/{referral_id}/`.
// @Description  4. Use the returned `referral_id` when calling the Referral Creation API.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Common Errors:**
// @Description  - 500 Internal Server Error
// @Tags         Attachments
// @Produce      json
// @Param        referral_id query string false "Existing Referral ID (for updates)"
// @Success      200 {object} dto.UploadSignatureResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/attachments/signature [get]
func (h *AttachmentHandler) GetUploadSignature(c *gin.Context) {
	var refIDPtr *uuid.UUID
	refIDStr := c.Query("referral_id")
	if refIDStr != "" {
		if parsed, err := uuid.Parse(refIDStr); err == nil {
			refIDPtr = &parsed
		}
	}

	data, referralID, err := h.attachmentUC.GenerateSignature(refIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Upload signature generated successfully",
		"referral_id": referralID.String(),
		"signature":   data["signature"].(string),
		"timestamp":   data["timestamp"].(int64),
		"api_key":     data["api_key"].(string),
		"cloud_name":  data["cloud_name"].(string),
		"folder":      data["folder"].(string),
	})
}

// ManualVerifyAttachment godoc
// @Summary      Manual Verification & Promotion
// @Description  Immediately triggers metadata extraction and folder promotion for a specific attachment.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD
// @Description  **State Transition:** Verified files are moved from `temp/` to permanent hospital storage. Failed files mark referral as NEED_REVISION.
// @Description  **Common Errors:**
// @Description  - 500 Internal Server Error
// @Tags         Attachments
// @Produce      json
// @Param        id path string true "Attachment ID"
// @Success      200 {object} dto.AttachmentListResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/attachments/{id}/verify [post]
func (h *AttachmentHandler) ManualVerifyAttachment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid Attachment ID"})
		return
	}

	att, err := h.attachmentUC.VerifyAttachment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AttachmentListResponse{
		Data: []dto.AttachmentResponse{toAttachmentResponse(att)},
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Manual verification triggered successfully",
		},
	})
}

// ManualVerifyReferralAttachments godoc
// @Summary      Verify All Referral Attachments
// @Description  Triggers verification and Cloudinary promotion for ALL pending attachments of a specific referral.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN, DEPT_HEAD, REFERRING_DOCTOR
// @Description  **State Transition:** Ensures the system moves the referral out of PENDING states before Liaison review.
// @Description  **Common Errors:**
// @Description  - 500 Internal Server Error
// @Tags         Attachments
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/verify-attachments [post]
func (h *AttachmentHandler) ManualVerifyReferralAttachments(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid Referral ID"})
		return
	}

	if err := h.attachmentUC.VerifyReferralAttachments(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral attachments verification triggered successfully",
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
