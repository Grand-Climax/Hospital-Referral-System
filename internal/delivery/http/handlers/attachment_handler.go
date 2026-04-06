package handlers

import (
	"net/http"

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
// @Summary      Register Referral Attachments (Bulk)
// @Description  Tie multiple Cloudinary-uploaded files explicitly to a referral with metadata extraction. Limited to Referral Doctors on Draft/NeedRevision referrals.
// @Tags         Attachments
// @Accept       json
// @Produce      json
// @Param        id   path string true "Referral ID"
// @Param        body body dto.BulkAttachmentRequest true "Bulk attachment registration payload"
// @Success      201 {object} dto.AttachmentListResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/referrals/{id}/attachments [post]
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid Referral UUID"})
		return
	}

	userIDStr := c.GetString("user_id")
	doctorID, _ := uuid.Parse(userIDStr)

	var req dto.BulkAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	attachments, err := h.attachmentUC.AddAttachmentsToReferral(c.Request.Context(), referralID, doctorID, req.Attachments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to save attachments: " + err.Error(),
		})
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

	userIDStr := c.GetString("user_id")
	doctorID, _ := uuid.Parse(userIDStr)

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
	// 1. Identify Hospital (from user context)
	// In a real app, you'd get this from the JWT claims or user record.
	// For now, we'll try to find it or use a default if it's a doctor's request.
	hospitalIDStr := c.GetString("hospital_id")
	if hospitalIDStr == "" {
		// Fallback for demo or if not set in middleware yet
		// In production, this should always be available for authenticated staff
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Hospital context missing"})
		return
	}

	hospitalID, _ := uuid.Parse(hospitalIDStr)

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
