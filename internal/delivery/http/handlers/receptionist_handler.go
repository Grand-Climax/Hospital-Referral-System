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
	referralUC  iusecase.ReferralUseCase
	arrivalUC   iusecase.ArrivalUseCase
	patientUC   iusecase.PatientUseCase
	userUseCase iusecase.UserUseCase
	triageUC    iusecase.TriageUseCase
}

func NewReceptionistHandler(referralUC iusecase.ReferralUseCase, arrivalUC iusecase.ArrivalUseCase, patientUC iusecase.PatientUseCase, userUC iusecase.UserUseCase, triageUC iusecase.TriageUseCase) *ReceptionistHandler {
	return &ReceptionistHandler{
		referralUC:  referralUC,
		arrivalUC:   arrivalUC,
		patientUC:   patientUC,
		userUseCase: userUC,
		triageUC:    triageUC,
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


// GetTriageQueue godoc
// @Summary      Get Triage Queue (receptionist view, filterable)
// @Description  Returns the triage queue for the receptionist's hospital with reduced clinical fields. Same filter/sort matrix as the specialist endpoint, but the response intentionally omits clinical text and severity scores; it surfaces only what a receptionist needs (name, time, arrival_status, assigned doctor).
// @Description
// @Description  **Roles:** RECEPTIONIST
// @Description  **Scope:** Hospital-wide (caller's hospital from JWT).
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Receptionist
// @Produce      json
// @Param        limit query int false "Pagination limit (1-100)" default(20)
// @Param        page query int false "Page number (1-based)" default(1)
// @Param        department_id query string false "Filter by HospitalDepartment ID"
// @Param        arrival_status query string false "Comma-separated: EXPECTED,ARRIVED,ADMITTED,MISSED"
// @Param        referral_status query string false "Comma-separated: ACCEPTED,SCHEDULED"
// @Param        has_doctor_assigned query bool false "Filter by treating-doctor assignment"
// @Param        patient_id query string false "Filter by patient UUID"
// @Param        national_id query string false "Filter by patient national ID (hashed server-side)"
// @Param        sort_by query string false "composite_score|appointment_date|created_at" default(composite_score)
// @Param        sort_order query string false "asc|desc" default(desc)
// @Param        include_terminal query bool false "Include terminal-status referrals" default(false)
// @Success      200 {object} dto.TriageListEnvelope
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/triage-queue [get]
func (h *ReceptionistHandler) GetTriageQueue(c *gin.Context) {
	hospID, _ := h.getHospitalAndDept(c)
	if hospID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid hospital scope"})
		return
	}

	filter := parseTriageListFilter(c)
	filter.HospitalID = hospID

	items, total, err := h.triageUC.ListTriageFiltered(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TriageListEnvelope{
		Success: true,
		Data:    items,
		Total:   total,
		Page:    pageFromOffset(filter.Limit, filter.Offset),
		Limit:   filter.Limit,
		HasMore: hasMorePage(filter.Offset, filter.Limit, total),
	})
}

// GetTriageDetail godoc
// @Summary      Get Triage Detail (receptionist view)
// @Description  Returns the redacted operational detail a receptionist needs: patient identity, appointment date, arrival_status, assigned doctor, arrival_history timeline, and the role-aware available_actions. Clinical text, ML severity, ICD codes, and the consulting-doctors list are intentionally omitted.
// @Description
// @Description  **Roles:** RECEPTIONIST
// @Description  **Path param:** {id} = REFERRAL UUID (not the queue UUID).
// @Description  **Common Errors:**
// @Description  - 400 Invalid referral id
// @Description  - 404 Referral not in triage queue
// @Tags         Receptionist
// @Produce      json
// @Param        id path string true "Referral UUID"
// @Success      200 {object} dto.TriageDetailReceptionistResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/triage-detail [get]
func (h *ReceptionistHandler) GetTriageDetail(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id"})
		return
	}
	userIDVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else if uid, ok := userIDVal.(*uuid.UUID); ok && uid != nil {
		userID = *uid
	}

	resp, err := h.triageUC.GetTriageDetailForReceptionist(c.Request.Context(), referralID, userID)
	if err != nil {
		// gorm.ErrRecordNotFound from the queue lookup means the
		// referral is not currently in the triage queue.
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "referral not in triage queue"})
		return
	}
	c.JSON(http.StatusOK, resp)
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
// @Router       /api/v1/receptionist/referrals/upcoming [get]
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

// ListDoctors godoc
// @Summary      List Available Doctors for Assignment
// @Description  Returns a list of active REFERRING_DOCTORs belonging to the receptionist's hospital and department.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Scope:** Strictly filtered to the receptionist's own department (from JWT). Cross‑department doctors are not returned.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized (hospital/department missing from token)
// @Tags         Receptionist
// @Produce      json
// @Success      200 {object} dto.DoctorListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/doctors [get]
func (h *ReceptionistHandler) ListDoctors(c *gin.Context) {
	hospID, deptID := h.getHospitalAndDept(c)
	if hospID == uuid.Nil || deptID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes (hospital/department missing)"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	role := entity.RoleReferringDoctor
	active := true
	hospStr := hospID.String()
	deptStr := deptID.String()

	doctors, _, err := h.userUseCase.ListUsers(c.Request.Context(), irepository.UserListFilter{
		Role:         &role,
		HospitalID:   &hospStr,
		DepartmentID: &deptStr,
		IsActive:     &active,
		Page:         1,
		PageSize:     200,
	}, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "failed to fetch doctors"})
		return
	}

	doctorInfos := make([]dto.DoctorInfo, 0, len(doctors))
	for _, d := range doctors {
		doctorInfos = append(doctorInfos, dto.DoctorInfo{
			ID:        d.ID,
			FirstName: d.FirstName,
			LastName:  d.LastName,
		})
	}

	c.JSON(http.StatusOK, dto.DoctorListResponse{
		BaseResponse: dto.BaseResponse{Success: true, Message: "Doctors retrieved successfully"},
		Data:         doctorInfos,
	})
}

// GetOfflineData godoc
// @Summary      Get Offline Data for Receptionist
// @Description  Returns all data needed for offline operation: today's and tomorrow's scheduled patients and the list of available doctors.
// @Description  **Roles:** RECEPTIONIST
// @Tags         Receptionist
// @Produce      json
// @Success      200 {object} dto.ReceptionistOfflineDataResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/offline-data [get]
func (h *ReceptionistHandler) GetOfflineData(c *gin.Context) {
	hospID, deptID := h.getHospitalAndDept(c)
	if hospID == uuid.Nil || deptID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "invalid user scopes (hospital/department missing)"})
		return
	}

	schedule, err := h.arrivalUC.GetTodayAndTomorrowSchedule(c.Request.Context(), hospID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	// Fetch available doctors in the hospital and department
	role := entity.RoleReferringDoctor
	active := true
	hospIDStr := hospID.String()
	deptIDStr := deptID.String()
	doctors, _, err := h.userUseCase.ListUsers(c.Request.Context(), irepository.UserListFilter{
		HospitalID:   &hospIDStr,
		DepartmentID: &deptIDStr,
		Role:         &role,
		IsActive:     &active,
		Page:         1,
		PageSize:     200,
	}, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "failed to fetch hospital doctors"})
		return
	}

	doctorInfos := make([]dto.DoctorInfo, 0, len(doctors))
	for _, d := range doctors {
		doctorInfos = append(doctorInfos, dto.DoctorInfo{
			ID:        d.ID,
			FirstName: d.FirstName,
			LastName:  d.LastName,
		})
	}

	c.JSON(http.StatusOK, dto.ReceptionistOfflineDataResponse{
		Schedule: schedule,
		Doctors:  doctorInfos,
	})
}

// ConfirmArrival godoc
// @Summary      Mark Patient Arrival
// @Description  Mark a patient as arrived. Resolve TriageQueue internally from referral ID.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** TriageQueue entry exists, arrival_status = EXPECTED.
// @Description  **State Transition:** arrival_status → ARRIVED.
// @Tags         Receptionist
// @Produce      json
// @Param        id path string true "Referral ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/arrive [post]
func (h *ReceptionistHandler) ConfirmArrival(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	queue, err := h.arrivalUC.GetTriageQueueByReferralID(c.Request.Context(), referralID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "no triage queue found for this referral"})
		return
	}

	hospID, deptID := h.getHospitalAndDept(c)
	if err := h.arrivalUC.ConfirmArrival(c.Request.Context(), queue.ID, userID, hospID, deptID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient arrival confirmed"})
}

// AssignDoctor godoc
// @Summary      Assign Treating Doctor
// @Description  Assign a treating doctor (referring doctor role) to the patient. Resolve TriageQueue internally from referral ID.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** queue must be ARRIVED; doctor must be REFERRING_DOCTOR in same hospital.
// @Description  **Reassignment:** If a doctor is already assigned, all previous accesses are revoked before new assignment.
// @Description  **Side Effect:** Creates ReferralAccess grant and grants clinical access to the assigned doctor.
// @Tags         Receptionist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.AssignDoctorRequest true "Assignment details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/assign-doctor [post]
func (h *ReceptionistHandler) AssignDoctor(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
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

	queue, err := h.arrivalUC.GetTriageQueueByReferralID(c.Request.Context(), referralID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "no triage queue found for this referral"})
		return
	}

	hospID, deptID := h.getHospitalAndDept(c)
	if err := h.arrivalUC.AssignDoctor(c.Request.Context(), queue.ID, req.DoctorID, userID, req.Reason, hospID, deptID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Doctor assigned successfully"})
}

// RevokeDoctor godoc
// @Summary      Revoke Assigned Doctor
// @Description  Removes the assigned treating doctor from a patient, revoking their clinical access. Resolve TriageQueue internally from referral ID.
// @Description  **Roles:** RECEPTIONIST
// @Tags         Receptionist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        request body dto.RevokeDoctorRequest true "Revoke reason"
// @Success      200 {object} dto.BaseResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/revoke-doctor [post]
func (h *ReceptionistHandler) RevokeDoctor(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
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

	queue, err := h.arrivalUC.GetTriageQueueByReferralID(c.Request.Context(), referralID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "no triage queue found for this referral"})
		return
	}

	hospID, deptID := h.getHospitalAndDept(c)
	if err := h.arrivalUC.RevokeDoctorAssignment(c.Request.Context(), queue.ID, userID, req.Reason, hospID, deptID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Doctor assignment revoked successfully"})
}



// MarkMissed godoc
// @Summary      Mark Appointment as Missed
// @Description  Mark an appointment as missed. Resolve TriageQueue internally from referral ID.
// @Description  **Roles:** RECEPTIONIST
// @Description  **Prerequisites:** queue entry must exist.
// @Description  **State Transition:** arrival_status → MISSED; creates ClinicalUpdate for re‑evaluation.
// @Tags         Receptionist
// @Accept       json
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        body body dto.MarkMissedRequest true "Miss reason details"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/miss [post]
func (h *ReceptionistHandler) MarkMissed(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral id format"})
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

	queue, err := h.arrivalUC.GetTriageQueueByReferralID(c.Request.Context(), referralID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "no triage queue found for this referral"})
		return
	}

	hospID, deptID := h.getHospitalAndDept(c)
	if err := h.arrivalUC.MarkMissed(c.Request.Context(), queue.ID, entity.MissReason(req.MissReason), userID, hospID, deptID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient marked as missed"})
}

// ReturnToTriage godoc
// @Summary      Return Missed Patient to Triage
// @Description  Allows a receptionist to reset a missed patient back to the waiting queue (EXPECTED, no appointment date).
// @Description  **Detailed Behavior:**
// @Description  - Resets the patient's queue record arrival status from 'MISSED' back to 'EXPECTED' (placing the patient back in the active triage pool).
// @Description  - Wipes out the missed appointment date ('AppointmentDate' = nil).
// @Description  - Clears the missed reasons and any active doctor assignment details ('AssignedDoctorID' = nil, 'DoctorAssignedAt' = nil).
// @Description  - Transactionally updates the underlying Referral status back to 'ACCEPTED' so that the patient is eligible to be scheduled or manually triaged/rescheduled.
// @Description  **Roles:** RECEPTIONIST
// @Tags         Receptionist
// @Produce      json
// @Param        id path string true "TriageQueue ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/receptionist/referrals/{id}/return-to-triage [post]
func (h *ReceptionistHandler) ReturnToTriage(c *gin.Context) {
	triageQueueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid triage queue id format"})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID := uuid.Nil
	if uID, ok := userIdVal.(uuid.UUID); ok {
		userID = uID
	} else if uID, ok := userIdVal.(*uuid.UUID); ok && uID != nil {
		userID = *uID
	}

	hospID, deptID := h.getHospitalAndDept(c)
	if err := h.arrivalUC.ReturnToTriage(c.Request.Context(), triageQueueID, userID, hospID, deptID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Patient successfully returned to triage"})
}
