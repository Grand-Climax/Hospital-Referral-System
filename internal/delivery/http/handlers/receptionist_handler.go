package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ReceptionistHandler struct {
	referralUC iusecase.ReferralUseCase
	arrivalUC  iusecase.ArrivalUseCase
	patientUC  iusecase.PatientUseCase
}

func NewReceptionistHandler(referralUC iusecase.ReferralUseCase, arrivalUC iusecase.ArrivalUseCase, patientUC iusecase.PatientUseCase) *ReceptionistHandler {
	return &ReceptionistHandler{
		referralUC: referralUC,
		arrivalUC:  arrivalUC,
		patientUC:  patientUC,
	}
}

func (h *ReceptionistHandler) getHospitalAndDept(c *gin.Context) (uuid.UUID, uuid.UUID) {
	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	deptIdVal, _ := c.Get("deptID")
	deptID := uuid.Nil
	if dID, ok := deptIdVal.(uuid.UUID); ok {
		deptID = dID
	} else if dID, ok := deptIdVal.(*uuid.UUID); ok && dID != nil {
		deptID = *dID
	}

	return hospID, deptID
}

// ListReferrals godoc
// @Summary      List Referrals for Receptionist
// @Description  Get a paginated list of accepted/scheduled referrals for the receptionist's hospital.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Visibility:** ACCEPTED, SCHEDULED.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Receptionist
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Param        status query string false "Filter by status"
// @Param        region query string false "Filter by patient region"
// @Param        patient_id query string false "Filter by patient ID"
// @Param        national_id query string false "Filter by patient national ID"
// @Param        sort_by query string false "Sort by field (created_at, updated_at)" default(created_at)
// @Param        sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success      200 {object} dto.PaginatedReferralResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals [get]
func (h *ReceptionistHandler) ListReferrals(c *gin.Context) {
	hospID, _ := h.getHospitalAndDept(c)
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid hospital scope"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	filter := irepository.ReferralFilter{
		Status:    c.Query("status"),
		Region:    c.Query("region"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Limit:     limit,
		Page:      page,
	}

	if pID := c.Query("patient_id"); pID != "" {
		parsedID, err := uuid.Parse(pID)
		if err == nil {
			filter.PatientID = &parsedID
		}
	}

	if nID := c.Query("national_id"); nID != "" && filter.PatientID == nil {
		pID, err := h.patientUC.LookupByNationalID(c.Request.Context(), nID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "failed to lookup patient"})
			return
		}
		if pID == nil {
			c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
				BaseResponse: dto.BaseResponse{Success: false, Message: "Patient not found"},
				Data:         []dto.ListReferralResponse{},
				Total:        0,
				Page:         page,
				PageSize:     limit,
			})
			return
		}
		filter.PatientID = pID
	}

	referrals, total, err := h.referralUC.ListForReceptionist(c.Request.Context(), hospID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedReferralResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referrals retrieved successfully"},
		Data:         toListReferralResponseSlice(referrals),
		Total:        total,
		Page:         page,
		PageSize:     limit,
	})
}

// ListMissedReferrals godoc
// @Summary      List Missed Referrals for Receptionist
// @Description  Returns referrals where the patient missed their scheduled appointment (arrival status = MISSED).
// @Description  **Roles:** RECEPTIONIST
// @Tags         Receptionist
// @Produce      json
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Router       /api/v1/receptionist/referrals/missed [get]
func (h *ReceptionistHandler) ListMissedReferrals(c *gin.Context) {
	hospID, _ := h.getHospitalAndDept(c)
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid hospital scope"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	offset := (page - 1) * limit

	missed, total, err := h.arrivalUC.ListMissedByHospital(c.Request.Context(), hospID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Missed referrals retrieved successfully",
		"data":      missed,
		"total":     total,
		"page":      page,
		"page_size": limit,
	})
}


// GetReferral godoc
// @Summary      Get Referral Details for Receptionist
// @Description  Get detailed information about an accepted or scheduled referral.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** Status must be ACCEPTED or SCHEDULED.
// @Description  **Common Errors:**
// @Description  - 400 Invalid ID format
// @Description  - 403 Forbidden (wrong hospital or invalid status)
// @Tags         Receptionist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.ReferralDetailResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id} [get]
func (h *ReceptionistHandler) GetReferral(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid id format"})
		return
	}

	hospID, _ := h.getHospitalAndDept(c)
	ref, err := h.referralUC.GetDetailsForReceptionist(c.Request.Context(), id, hospID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ReferralDetailResponse{
		Referral: *ref,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referral details retrieved successfully"},
	})
}

// GetSchedule godoc
// @Summary      Get Receptionist Schedule
// @Description  Returns all scheduled triage records for the next 48 hours for the receptionist's hospital and department.
// @Description  **Access Scope:** Receptionists can view scheduled/operational queue items only; full triage prioritization queue is restricted to RECEIVING_SPECIALIST and DEPT_HEAD roles.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Receptionist
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/schedule [get]
func (h *ReceptionistHandler) GetSchedule(c *gin.Context) {
	hospID, deptID := h.getHospitalAndDept(c)
	if hospID == uuid.Nil || deptID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes (hospital/department missing)"})
		return
	}

	schedules, err := h.arrivalUC.GetTodayAndTomorrowSchedule(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    schedules,
	})
}

// ConfirmArrival godoc
// @Summary      Mark Patient Arrival
// @Description  Mark a patient as arrived.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** TriageQueue entry exists, arrival_status = EXPECTED.
// @Description  **State Transition:** arrival_status → ARRIVED.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 409 already arrived
// @Tags         Receptionist
// @Produce      json
// @Param        id path string true "TriageQueue ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/arrive [post]
func (h *ReceptionistHandler) ConfirmArrival(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	if err := h.arrivalUC.ConfirmArrival(c.Request.Context(), queueID, userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient arrival confirmed"})
}

// AssignDoctor godoc
// @Summary      Assign Treating Doctor
// @Description  Assign a treating doctor (referring doctor role) to the patient.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** queue must be ARRIVED; doctor must be REFERRING_DOCTOR in same hospital.
// @Description  **Reassignment:** If a doctor is already assigned, all previous accesses are revoked before new assignment.
// @Description  **Side Effect:** Creates ReferralAccess grant and grants clinical access to the assigned doctor.
// @Tags         Receptionist
// @Accept       json
// @Produce      json
// @Param        id path string true "TriageQueue ID"
// @Param        body body dto.AssignDoctorRequest true "Assignment details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/assign-doctor [post]
func (h *ReceptionistHandler) AssignDoctor(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	var req dto.AssignDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	if err := h.arrivalUC.AssignDoctor(c.Request.Context(), queueID, req.DoctorID, userID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Doctor assigned successfully"})
}

// RevokeDoctor godoc
// @Summary      Revoke Assigned Doctor
// @Description  Removes the assigned treating doctor from a patient, revoking their clinical access. Only the receptionist who assigned the doctor (or any receptionist in the same hospital) can call this.
// @Description  **Roles:** RECEPTIONIST
// @Tags         Receptionist
// @Accept       json
// @Produce      json
// @Param        id path string true "TriageQueue ID"
// @Param        request body dto.RevokeDoctorRequest true "Revoke reason"
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/revoke-doctor [post]
func (h *ReceptionistHandler) RevokeDoctor(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	var req dto.RevokeDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	if err := h.arrivalUC.RevokeDoctorAssignment(c.Request.Context(), queueID, userID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Doctor assignment revoked successfully"})
}



// MarkMissed godoc
// @Summary      Mark Appointment as Missed
// @Description  Mark an appointment as missed.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** queue entry must exist.
// @Description  **State Transition:** arrival_status → MISSED; creates ClinicalUpdate for re‑evaluation.
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Tags         Receptionist
// @Accept       json
// @Produce      json
// @Param        id path string true "TriageQueue ID"
// @Param        body body dto.MarkMissedRequest true "Miss reason details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/miss [post]
func (h *ReceptionistHandler) MarkMissed(c *gin.Context) {
	queueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid queue id format"})
		return
	}

	var req dto.MarkMissedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	if err := h.arrivalUC.MarkMissed(c.Request.Context(), queueID, entity.MissReason(req.MissReason), userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient marked as missed"})
}
