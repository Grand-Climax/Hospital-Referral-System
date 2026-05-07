package handlers

import (
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

	var req dto.HospitalAdminLinkDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	deptID, err := uuid.Parse(req.DepartmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department_id"})
		return
	}

	if err := h.departmentUseCase.LinkDepartmentToHospital(c.Request.Context(), hospID, deptID, req.DailyLimit); err != nil {
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
// @Param        deptId path string true "Department ID"
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
	if err := c.ShouldBindJSON(&req); err != nil || req.IsActive == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid request payload"})
		return
	}

	if err := h.departmentUseCase.SetHospitalDepartmentActive(c.Request.Context(), hospID, deptID, *req.IsActive); err != nil {
		if err == usecase.ErrHospitalDeptLinkNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to update department activation"})
		return
	}

	msg := "Department deactivated successfully"
	if *req.IsActive {
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
// @Param        deptId path string true "Department ID"
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

	deptID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	var req dto.HospitalAdminAssignDepartmentHeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	staffID, err := uuid.Parse(req.StaffID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff_id"})
		return
	}

	// Ensure department is linked to admin's hospital before assigning a head.
	links, err := h.departmentUseCase.ListHospitalDepartments(c.Request.Context(), hospID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to validate department scope"})
		return
	}
	linked := false
	for _, l := range links {
		if l.DepartmentID == deptID {
			linked = true
			break
		}
	}
	if !linked {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "department is not linked to this hospital"})
		return
	}

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
