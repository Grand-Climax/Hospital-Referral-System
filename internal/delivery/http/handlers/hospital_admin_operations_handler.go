package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/usecase"
)

type HospitalAdminOperationsHandler struct {
	userUseCase       iusecase.UserUseCase
	hospitalUseCase   iusecase.HospitalUseCase
	departmentUseCase iusecase.DepartmentUseCase
}

func NewHospitalAdminOperationsHandler(userUC iusecase.UserUseCase, hospitalUC iusecase.HospitalUseCase, deptUC iusecase.DepartmentUseCase) *HospitalAdminOperationsHandler {
	return &HospitalAdminOperationsHandler{
		userUseCase:       userUC,
		hospitalUseCase:   hospitalUC,
		departmentUseCase: deptUC,
	}
}

// GetMyHospitalProfile godoc
// @Summary      Get own hospital profile (Hospital Admin)
// @Description  Retrieve hospital profile for the authenticated hospital admin scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Visibility:** Admin's own hospital.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Produce      json
// @Success      200 {object} dto.HospitalResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/hospital/profile [get]
func (h *HospitalAdminOperationsHandler) GetMyHospitalProfile(c *gin.Context) {
	_, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	hospital, err := h.hospitalUseCase.GetHospitalByID(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Hospital not found"})
		return
	}

	resp := toHospitalResponse(hospital)
	resp.BaseResponse = dto.BaseResponse{Success: true, Message: "Hospital profile retrieved successfully"}
	c.JSON(http.StatusOK, resp)
}

// UpdateMyHospitalProfile godoc
// @Summary      Update own hospital profile (Hospital Admin)
// @Description  Update name, contact phone, and address for the admin's own hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** Hospital profile fields updated in DB.
// @Description  **Common Errors:**
// @Description  - 400 Invalid input
// @Description  - 403 Forbidden
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        body body dto.HospitalAdminUpdateHospitalProfileRequest true "Hospital profile update payload"
// @Success      200 {object} dto.HospitalResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/hospital/profile [patch]
func (h *HospitalAdminOperationsHandler) UpdateMyHospitalProfile(c *gin.Context) {
	_, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	var req dto.HospitalAdminUpdateHospitalProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	hospital, err := h.hospitalUseCase.GetHospitalByID(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Hospital not found"})
		return
	}

	if req.Name != nil {
		hospital.Name = *req.Name
	}
	if req.ContactPhone != nil {
		hospital.ContactPhone = req.ContactPhone
	}
	if req.Address != nil {
		hospital.Address = req.Address
	}

	if err := h.hospitalUseCase.UpdateHospital(c.Request.Context(), hospital); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to update hospital profile"})
		return
	}

	resp := toHospitalResponse(hospital)
	resp.BaseResponse = dto.BaseResponse{Success: true, Message: "Hospital profile updated successfully"}
	c.JSON(http.StatusOK, resp)
}

// LinkDepartmentToMyHospital godoc
// @Summary      Add department to own hospital (Hospital Admin)
// @Description  Link an existing department to the admin's hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** New HospitalDepartment link created.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Description  - 409 Link already exists
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        body body dto.HospitalAdminLinkDepartmentRequest true "Department link payload"
// @Success      201 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/departments [post]
func (h *HospitalAdminOperationsHandler) LinkDepartmentToMyHospital(c *gin.Context) {
	_, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	deptIDStr, dailyLimit := dto.ResolveLinkDepartmentFromBody(raw)
	if deptIDStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "department_id is required in the request body (accepted keys: department_id, departmentId, id)",
		})
		return
	}

	deptID, err := uuid.Parse(deptIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department_id"})
		return
	}

	if err := h.departmentUseCase.LinkDepartmentToHospital(c.Request.Context(), hospID, deptID, dailyLimit); err != nil {
		switch err {
		case usecase.ErrHospitalNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Hospital not found"})
		case usecase.ErrDepartmentNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Department not found"})
		case usecase.ErrHospitalDeptLinkExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to link department"})
		}
		return
	}

	c.JSON(http.StatusCreated, dto.BaseResponse{Success: true, Message: "Department linked successfully"})
}

// ListMyHospitalDepartments godoc
// @Summary      View own hospital departments (Hospital Admin)
// @Description  List departments linked to the admin's hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Visibility:** Hospital scoped department listing.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Tags         Hospital Admin
// @Produce      json
// @Success      200 {object} dto.HospitalDepartmentListResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/departments [get]
func (h *HospitalAdminOperationsHandler) ListMyHospitalDepartments(c *gin.Context) {
	_, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	links, err := h.departmentUseCase.ListHospitalDepartments(c.Request.Context(), hospID)
	if err != nil {
		if err == usecase.ErrHospitalNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Hospital not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to list hospital departments"})
		return
	}

	resp := make([]dto.HospitalDepartmentResponse, 0, len(links))
	for i := range links {
		resp = append(resp, toHospitalDepartmentResponse(&links[i]))
	}

	c.JSON(http.StatusOK, dto.HospitalDepartmentListResponse{
		Data:         resp,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Hospital departments retrieved successfully"},
	})
}

// SetDepartmentActive godoc
// @Summary      Activate or deactivate department link (Hospital Admin)
// @Description  Toggle active state of a hospital-department link in own hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** HospitalDepartment.IsActive toggled.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        deptId path string true "Hospital-department link ID (id from GET /departments) or department_id"
// @Param        body body dto.HospitalAdminSetDepartmentActiveRequest true "Activation payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/departments/{deptId}/activation [patch]
func (h *HospitalAdminOperationsHandler) SetDepartmentActive(c *gin.Context) {
	_, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	deptID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	var req dto.HospitalAdminSetDepartmentActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid request payload"})
		return
	}
	isActive := req.ResolvedIsActive()
	if isActive == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "is_active is required (accepted keys: is_active, isActive)"})
		return
	}

	if err := h.departmentUseCase.SetHospitalDepartmentActive(c.Request.Context(), hospID, deptID, *isActive); err != nil {
		if err == usecase.ErrHospitalDeptLinkNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to update department activation"})
		return
	}

	msg := "Department deactivated successfully"
	if *isActive {
		msg = "Department activated successfully"
	}
	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: msg})
}

// AssignDepartmentHead godoc
// @Summary      Assign or change department head (Hospital Admin)
// @Description  Assign a staff user as DEPT_HEAD and bind them to the department.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** User.Role = DEPT_HEAD, User.DepartmentID updated.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Description  - 404 Department/Staff not found
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        deptId path string true "Hospital-department link ID (id from GET /departments) or department_id"
// @Param        body body dto.HospitalAdminAssignDepartmentHeadRequest true "Department head payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/departments/{deptId}/head [patch]
func (h *HospitalAdminOperationsHandler) AssignDepartmentHead(c *gin.Context) {
	adminID, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	pathID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid request body"})
		return
	}

	staffIDStr := dto.ResolveStaffID(bodyBytes, c.Request.URL.Query())
	if staffIDStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "staff_id is required (body keys: staff_id, staffId, user_id, userId, head_id, headId, departmentHeadId, staff, head, id; or query param staff_id)",
		})
		return
	}

	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff_id"})
		return
	}

	link, err := h.departmentUseCase.GetHospitalDepartmentLink(c.Request.Context(), hospID, pathID)
	if err != nil {
		if err == usecase.ErrHospitalDeptLinkNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to validate department scope"})
		return
	}
	deptID := link.DepartmentID

	if err := h.userUseCase.HospitalAdminReassignStaffDepartment(c.Request.Context(), adminID, staffID, &deptID); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	if err := h.userUseCase.HospitalAdminChangeStaffRole(c.Request.Context(), adminID, staffID, entity.RoleDeptHead); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Department head assigned successfully"})
}

// GetPersonnelWidgetStats godoc
// @Summary      Get personnel widget stats (Hospital Admin)
// @Description  Retrieve dashboard metrics for hospital personnel (Total Registered Users, Active Duty, Inactive).
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Visibility:** Scoped to the authenticated hospital admin's hospital.
// @Description  **Metrics Returned:**
// @Description  - **TOTAL PERSONNEL:** All registered system users in the hospital (excluding soft-deleted users `is_deleted = true`).
// @Description  - **ACTIVE DUTY (ONLINE):** Currently active accounts (`is_active = true`, `is_deleted = false`).
// @Description  - **INACTIVE (AWAY):** Disabled or on leave accounts (`is_active = false`, `is_deleted = false`).
// @Description  - **ACCESS REQUESTS (PENDING):** Placeholder `0` for now.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (not a hospital admin or no hospital assigned)
// @Description  - 404 Hospital admin not found
// @Tags         Hospital Admin
// @Produce      json
// @Success      200 {object} dto.HospitalAdminPersonnelWidgetResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/dashboard/personnel-widget [get]
func (h *HospitalAdminOperationsHandler) GetPersonnelWidgetStats(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}

	resp, err := h.userUseCase.GetPersonnelWidgetStats(c.Request.Context(), adminID)
	if err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp.BaseResponse = dto.BaseResponse{Success: true, Message: "Personnel widget stats retrieved successfully"}
	c.JSON(http.StatusOK, resp)
}

