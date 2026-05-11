package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/pkg/utils"
)

type DoctorHandler struct {
	referralUC   iusecase.ReferralUseCase
	attachmentUC iusecase.AttachmentUseCase
}

func NewDoctorHandler(referralUC iusecase.ReferralUseCase, attachmentUC iusecase.AttachmentUseCase) *DoctorHandler {
	return &DoctorHandler{referralUC: referralUC, attachmentUC: attachmentUC}
}

func extractUUID(val interface{}) uuid.UUID {
	if uID, ok := val.(uuid.UUID); ok {
		return uID
	}
	if uID, ok := val.(*uuid.UUID); ok && uID != nil {
		return *uID
	}
	return uuid.Nil
}

// ListReferrals godoc
// @Summary      List Referrals for Doctor
// @Description  Get a paginated list of referrals created by the authenticated doctor.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Authenticated session as a doctor.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Doctor
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_name query string false "Filter by patient name (any order)"
// @Param        sort query string false "Sort order (asc/desc)"
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals [get]
func (h *DoctorHandler) ListReferrals(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}
	if doctorID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}

	filter := irepository.ReferralFilter{
		Status:      c.Query("status"),
		Region:      c.Query("region"),
		PatientName: c.Query("patient_name"),
		Sort:        c.Query("sort"),
		Limit:       limit,
		Page:        page,
	}

	if filter.Status != "" && !h.referralUC.IsValidStatus(filter.Status) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "forbidden: unknown or invalid referral status",
		})
		return
	}

	referrals, total, err := h.referralUC.ListForDoctor(c.Request.Context(), doctorID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if total == 0 {
		c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
			BaseResponse: dto.BaseResponse{
				Success: false,
				Message: "No referrals found matching your criteria",
			},
			Data:     []dto.ListReferralResponse{},
			Total:    0,
			Page:     page,
			PageSize: limit,
		})
		return
	}

	responseData := toListReferralResponseSlice(referrals)

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referrals retrieved successfully",
		},
		Data:         responseData,
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

// GetReferral godoc
// @Summary      Get Referral Details for Doctor
// @Description  Get full details of a specific referral created by the doctor.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Must be the original creator of the referral.
// @Description  **Common Errors:**
// @Description  - 400 Invalid ID format
// @Description  - 403 Forbidden (not the creator)
// @Tags         Doctor
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id} [get]
func (h *DoctorHandler) GetReferral(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid id format",
		})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}

	ref, err := h.referralUC.GetDetailsForDoctor(c.Request.Context(), id, doctorID)
	if err != nil {
		log.Printf("[DoctorHandler.GetReferral] error: %v", err)
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ReferralDetailResponse{
		Referral: *ref,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Referral details retrieved successfully",
		},
		Redirections: toRedirectionResponseSlice(ref.Redirections),
	})
}

// GetStats godoc
// @Summary      Get Doctor Dashboard Stats
// @Description  Dashboard statistics including Total, Pending, Accepted, and Critical counts.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Doctor
// @Produce      json
// @Success      200 {object} dto.DoctorDashboardStatsResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/stats [get]
func (h *DoctorHandler) GetStats(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}

	stats, err := h.referralUC.GetDoctorDashboardStats(c.Request.Context(), doctorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.DoctorDashboardStatsResponse{
		DoctorDashboardStats: *stats,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Dashboard statistics retrieved successfully",
		},
	})
}

// GetLatestPending godoc
// @Summary      Get Latest Pending Referrals
// @Description  Get the most recent pending referrals for the doctor's dashboard.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Doctor
// @Produce      json
// @Param        limit query int false "Number of records to fetch" default(5)
// @Success      200 {object} dto.LatestPendingReferralsResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/latest-pending [get]
func (h *DoctorHandler) GetLatestPending(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}
	if doctorID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user session",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		// If negative or non-numeric, default to 5 as per user request to "handle" it
		limit = 5
	}

	referrals, err := h.referralUC.GetLatestPendingReferrals(c.Request.Context(), doctorID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if len(referrals) == 0 {
		c.JSON(http.StatusOK, dto.BaseResponse{
			Success: false,
			Message: "No pending referrals found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.LatestPendingReferralsResponse{
		Data: referrals,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Latest pending referrals retrieved successfully",
		},
	})
}

// CreateOrSubmit godoc
// @Summary      Create or Submit Referral (Multipart)
// @Description  Create a referral in DRAFT or SUBMITTED state and upload attachment files in a single synchronous call.
// @Description  
// @Description  ### Request format: multipart/form-data
// @Description  - **referral**: JSON string matching the `CreateReferralRequest` schema. Example:
// @Description    ```json
// @Description    {
// @Description      "accompanying_person_name": "Sarah Kebede",
// @Description      "accompanying_person_phone": "+251922334455",
// @Description      "clinical_summary": "Patient complains of severe chest pain for 2 hours",
// @Description      "condition_at_referral": "UNSTABLE",
// @Description      "diagnoses": [
// @Description        {
// @Description          "diagnosis_certainty": "SUSPECTED",
// @Description          "icd_code": "J18.9",
// @Description          "is_primary": true
// @Description        }
// @Description      ],
// @Description      "emergency_detail": {
// @Description        "emergency_justification": "Patient requires immediate intubation and bypass surgery"
// @Description      },
// @Description      "investigation_results": "ECG shows ST elevation",
// @Description      "liaison_officer_id": "d1000000-0000-0000-0000-000000000002",
// @Description      "medication_on_transfer": "IV Nitroglycerin",
// @Description      "mode_of_transport": "AMBULANCE",
// @Description      "patient_history": "Hypertension diagnosed 5 years ago",
// @Description      "patient_id": "e0000000-0000-0000-0000-000000000001",
// @Description      "physical_examination_findings": "BP 180/110, HR 105",
// @Description      "reason_for_referral_category": "EMERGENCY",
// @Description      "reason_of_referral": "Requires immediate cardiological intervention",
// @Description      "status": "SUBMITTED",
// @Description      "target_dept_id": "b5000000-0000-0000-0000-000000000005",
// @Description      "target_hospital_id": "a3000000-0000-0000-0000-000000000003",
// @Description      "treatment_given_before_referral": "Aspirin 300mg",
// @Description      "vitals": {
// @Description        "diastolic_bp": 200,
// @Description        "gcs_score": 15,
// @Description        "heart_rate": 300,
// @Description        "respiratory_rate": 60,
// @Description        "sp_o2": 100,
// @Description        "systolic_bp": 300,
// @Description        "temperature": 45
// @Description      }
// @Description    }
// @Description    ```
// @Description  - **attachments**: One or more files (optional).
// @Description  - **attachment_category**: Category for all attachments (default: GENERAL_CLINICAL).
// @Description  
// @Description  **Validation:**
// @Description  - DICOM/PDF files are verified for metadata.
// @Description  - If a file is rejected during a SUBMITTED request, the referral moves to NEED_REVISION.
// @Description  
// @Description  **Roles:** REFERRING_DOCTOR
// @Tags         Doctor
// @Accept       mpfd
// @Produce      json
// @Param        referral formData string true "Referral JSON payload"
// @Param        attachments formData file false "Attachment files"
// @Param        attachment_category formData string false "Category for attachments"
// @Success      201 {object} dto.ReferralCreationResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals [post]
func (h *DoctorHandler) CreateOrSubmit(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	doctorID := extractUUID(userIDVal)
	hospIDVal, _ := c.Get("hospID")
	hospID := extractUUID(hospIDVal)

	// 1. Parse multipart form (large limit for attachments)
	if err := c.Request.ParseMultipartForm(200 << 20); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Failed to parse form: " + err.Error()})
		return
	}

	// 2. Extract referral JSON
	referralStr := c.PostForm("referral")
	if referralStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "referral field is required"})
		return
	}
	var req dto.CreateReferralRequest
	if err := json.Unmarshal([]byte(referralStr), &req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral JSON: " + err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = string(entity.StatusSubmitted)
	}
	if req.Status != string(entity.StatusDraft) && req.Status != string(entity.StatusSubmitted) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid status: only DRAFT or SUBMITTED allowed"})
		return
	}

	// 3. Generate referral ID
	refID := uuid.New()
	if req.ID != nil {
		refID = *req.ID
	}

	// 4. Process attachments
	fileHeaders := c.Request.MultipartForm.File["attachments"]
	category := c.PostForm("attachment_category")
	if category == "" {
		category = "GENERAL_CLINICAL"
	}

	var uploads []iusecase.UploadedFileData

	for _, fh := range fileHeaders {
		// Validate file extension
		if _, err := utils.ValidateFileExtension(fh.Filename); err != nil {
			// Clean up already-uploaded files
			for _, u := range uploads {
				_ = h.attachmentUC.Storage().DeleteFile(c.Request.Context(), u.PublicID)
			}
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}

		file, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to open file"})
			return
		}
		fileBytes, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to read file"})
			return
		}

		// Upload to Cloudinary
		folder := fmt.Sprintf("referrals/%s/attachments", refID.String())
		reader := bytes.NewReader(fileBytes)
		url, publicID, err := h.attachmentUC.Storage().UploadFile(c.Request.Context(), reader, folder)
		if err != nil {
			for _, u := range uploads {
				_ = h.attachmentUC.Storage().DeleteFile(c.Request.Context(), u.PublicID)
			}
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Cloudinary upload failed: " + err.Error()})
			return
		}

		// Metadata verification (first 256 KB)
		metaBuf := fileBytes
		if len(metaBuf) > 256*1024 {
			metaBuf = metaBuf[:256*1024]
		}
		att := &entity.Attachment{FileName: fh.Filename, FileType: fh.Header.Get("Content-Type"), FileSize: fh.Size}
		status, meta, reason := h.attachmentUC.VerifyAttachmentBytes(att, metaBuf)

		uploads = append(uploads, iusecase.UploadedFileData{
			URL: url, PublicID: publicID, FileName: fh.Filename, FileType: fh.Header.Get("Content-Type"),
			Category: category, FileSize: fh.Size, Status: status, Metadata: meta, Reason: reason,
		})
	}

	// 5. Create referral + attachments in a transaction
	ref, err := h.referralUC.CreateReferralWithAttachments(c.Request.Context(), doctorID, hospID, req, refID, uploads)
	if err != nil {
		for _, u := range uploads {
			_ = h.attachmentUC.Storage().DeleteFile(c.Request.Context(), u.PublicID)
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	msg := "Referral created (DRAFT)"
	if ref.Referral.Status == entity.StatusSubmitted {
		msg = "Referral submitted (SUBMITTED)"
	}
	ref.Message = msg
	c.JSON(http.StatusCreated, ref)
}

// UpdateDraft godoc
// @Summary      Update Referral Draft
// @Description  Updates existing clinical data or forms for a referral in DRAFT or NEED_REVISION status.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Status must be DRAFT or NEED_REVISION.
// @Description  **State Transition:** None (stays in DRAFT/NEED_REVISION).
// @Description  **Common Errors:**
// @Description  - 400 Invalid input or status field included
// @Description  - 403 Forbidden (not the creator)
// @Description  - 422 Invalid state transition (already submitted)
// @Tags         Doctor
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.UpdateReferralRequest true "Updated Referral Details"
// @Success      200 {object} dto.ReferralCreationResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id} [put]
func (h *DoctorHandler) UpdateDraft(c *gin.Context) {
	h.handleUpdate(c, false)
}

// SubmitReferral godoc
// @Summary      Submit Referral for Review
// @Description  Finalizes and submits an existing draft (or a referral needing revision) into the hospital review pipeline.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Status must be DRAFT or NEED_REVISION. Must have at least one diagnosis, clinical summary, and patient history.
// @Description  **State Transition:** Status becomes SUBMITTED. Visible to Liaisons.
// @Description  **Common Errors:**
// @Description  - 400 Missing clinical data
// @Description  - 403 Forbidden (not the creator)
// @Tags         Doctor
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.UpdateReferralRequest true "Submission Details"
// @Success      200 {object} dto.ReferralCreationResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id}/submit [put]
func (h *DoctorHandler) SubmitReferral(c *gin.Context) {
	h.handleUpdate(c, true)
}

func (h *DoctorHandler) handleUpdate(c *gin.Context, submit bool) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}

	var rawBody map[string]interface{}
	if err := c.ShouldBindJSON(&rawBody); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if _, exists := rawBody["status"]; exists {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "status field is not allowed in manually in update; use the /submit route to finalize",
		})
		return
	}

	jsonData, _ := json.Marshal(rawBody)
	var req dto.UpdateReferralRequest
	_ = json.Unmarshal(jsonData, &req)

	ref, err := h.referralUC.UpdateAndResubmit(c.Request.Context(), id, doctorID, req, submit)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	msg := "Referral details updated successfully"
	if submit {
		msg = "Referral officially submitted for review"
	}
	ref.Message = msg
	c.JSON(http.StatusOK, ref)
}

// Cancel godoc
// @Summary      Cancel Referral
// @Description  Cancel an active referral that has not yet been processed.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Status must be DRAFT or NEED_REVISION.
// @Description  **State Transition:** Status becomes CANCELLED. Referral becomes read-only.
// @Description  **Common Errors:**
// @Description  - 400 Invalid status
// @Description  - 403 Forbidden
// @Tags         Doctor
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.CancelReferralRequest true "Cancellation Reason"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id}/cancel [post]
func (h *DoctorHandler) Cancel(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}

	var req dto.CancelReferralRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.referralUC.CancelReferral(c.Request.Context(), id, doctorID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Referral cancelled successfully",
	})
}

// RejectAfterSend godoc
// @Summary      Reject Referral After Sending
// @Description  Cancel a referral that has already been submitted but not yet scheduled.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Status must be SUBMITTED, UNDER_LIAISON_REVIEW, FORWARDED, UNDER_SPECIALIST_REVIEW, or ACCEPTED.
// @Description  **State Transition:** → REJECTED_AFTER_SEND. Removes from triage queue.
// @Tags         Doctor
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RejectDTO true "Rejection Reason"
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id}/reject-after-send [post]
func (h *DoctorHandler) RejectAfterSend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}
	userID, _ := c.Get("userID")
	hospID, _ := c.Get("hospID")
	var req dto.RejectDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	if err := h.referralUC.RejectAfterSend(c.Request.Context(), id, userID.(uuid.UUID), extractUUID(hospID), entity.RoleReferringDoctor, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Referral rejected after send"})
}

// DeleteAttachments godoc
// @Summary      Delete All Attachments
// @Description  Bulk delete all attachments associated with a referral.
// @Description  **Roles:** REFERRING_DOCTOR
// @Description  **Prerequisites:** Status must be DRAFT or NEED_REVISION.
// @Description  **Common Errors:**
// @Description  - 400 Invalid status
// @Description  - 403 Forbidden
// @Tags         Doctor
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/doctor/referrals/{id}/attachments [delete]
func (h *DoctorHandler) DeleteAttachments(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	doctorID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		doctorID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		doctorID = *uID
	}

	if err := h.referralUC.DeleteAttachmentsByReferralID(c.Request.Context(), id, doctorID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "All attachments removed successfully",
	})
}
