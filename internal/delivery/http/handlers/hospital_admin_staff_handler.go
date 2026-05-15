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
	"Hospital-Referral-System/internal/usecase"
)

type HospitalAdminStaffHandler struct {
	userUseCase     iusecase.UserUseCase
	referralUseCase iusecase.ReferralUseCase
	deptUseCase     iusecase.DepartmentUseCase
}

func NewHospitalAdminStaffHandler(userUC iusecase.UserUseCase, referralUC iusecase.ReferralUseCase, deptUseCase iusecase.DepartmentUseCase) *HospitalAdminStaffHandler {
	return &HospitalAdminStaffHandler{
		userUseCase:     userUC,
		referralUseCase: referralUC,
		deptUseCase:     deptUseCase,
	}
}

func getAdminContext(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Success: false, Error: "User not authenticated"})
		return uuid.Nil, uuid.Nil, false
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Invalid user context"})
		return uuid.Nil, uuid.Nil, false
	}

	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: "No hospital assigned to user"})
		return uuid.Nil, uuid.Nil, false
	}
	hospIDPtr, ok := hospIDVal.(*uuid.UUID)
	if !ok || hospIDPtr == nil || *hospIDPtr == uuid.Nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: "No hospital assigned to user"})
		return uuid.Nil, uuid.Nil, false
	}

	return userID, *hospIDPtr, true
}

func mapHospitalAdminStaffError(err error) int {
	switch err {
	case usecase.ErrUserNotFound:
		return http.StatusNotFound
	case usecase.ErrInvalidRole:
		return http.StatusBadRequest
	case usecase.ErrInvalidDepartment:
		return http.StatusBadRequest
	case usecase.ErrEmailExists, usecase.ErrNationalIDExists:
		return http.StatusConflict
	case usecase.ErrForbiddenStaffScope, usecase.ErrInvalidAdminScope, usecase.ErrCannotManageUser, usecase.ErrCannotManageSelf:
		return http.StatusForbidden
	case usecase.ErrSecurityUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// CreateStaff godoc
// @Summary      Create staff (Hospital Admin)
// @Description  Create a new staff user in the current hospital scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Prerequisites:** Hospital Admin must have an assigned hospital.
// @Description  **Common Errors:**
// @Description  - 400 Invalid input
// @Description  - 403 Forbidden (wrong hospital or not admin)
// @Description  - 409 Email/NationalID already exists
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        body body dto.HospitalAdminCreateStaffRequest true "Staff creation payload"
// @Success      201 {object} dto.UserResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff [post]
func (h *HospitalAdminStaffHandler) CreateStaff(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}

	var req dto.HospitalAdminCreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	user := &entity.User{
		Email:      req.Email,
		FirstName:  req.FirstName,
		MiddleName: req.MiddleName,
		LastName:   req.LastName,
		NationalID: req.NationalID,
		Role:       req.Role,
	}
	if req.DepartmentID != nil {
		deptID, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department_id"})
			return
		}
		user.DepartmentID = &deptID
	}

	if err := h.userUseCase.HospitalAdminCreateStaff(c.Request.Context(), adminID, user, req.Password); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := toUserResponse(user)
	resp.BaseResponse = dto.BaseResponse{Success: true, Message: "Staff created successfully"}
	c.JSON(http.StatusCreated, resp)
}

// ListStaff godoc
// @Summary      List staff (Hospital Admin)
// @Description  List staff users scoped to the admin's hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Visibility:** Hospital scoped staff listing.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Description  - 500 Internal Server Error
// @Tags         Hospital Admin
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Param        role query string false "Filter by role"
// @Param        name query string false "Filter by name"
// @Param        email query string false "Filter by email"
// @Param        dept_id query string false "Filter by department ID"
// @Param        is_active query bool false "Filter by active status"
// @Success      200 {object} dto.UserListResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff [get]
func (h *HospitalAdminStaffHandler) ListStaff(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}

	filter := irepository.UserListFilter{Page: 1, PageSize: 20}
	if p, err := strconv.Atoi(c.Query("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil {
		filter.PageSize = ps
	}
	if role := c.Query("role"); role != "" {
		r := entity.UserRole(role)
		filter.Role = &r
	}
	if deptID := c.Query("dept_id"); deptID != "" {
		filter.DepartmentID = &deptID
	}
	if email := c.Query("email"); email != "" {
		filter.Email = &email
	}
	if active := c.Query("is_active"); active != "" {
		a := active == "true"
		filter.IsActive = &a
	}
	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}

	users, total, err := h.userUseCase.HospitalAdminListStaff(c.Request.Context(), adminID, filter)
	if err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}

	c.JSON(http.StatusOK, dto.UserListResponse{
		Data:         resp,
		Total:        total,
		Page:         filter.Page,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Staff retrieved successfully"},
	})
}

// GetStaff godoc
// @Summary      Get staff by ID (Hospital Admin)
// @Description  Get a staff profile scoped to the admin's hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 Invalid ID
// @Description  - 403 Forbidden (trying to access staff from another hospital)
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Success      200 {object} dto.UserResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id} [get]
func (h *HospitalAdminStaffHandler) GetStaff(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	staff, err := h.userUseCase.HospitalAdminGetStaffByID(c.Request.Context(), adminID, staffID)
	if err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := toUserResponse(staff)
	resp.BaseResponse = dto.BaseResponse{Success: true, Message: "Staff details retrieved successfully"}
	c.JSON(http.StatusOK, resp)
}

// ChangeStaffRole godoc
// @Summary      Change staff role (Hospital Admin)
// @Description  Update the role of a staff user within the same hospital scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** User.Role updated.
// @Description  **Common Errors:**
// @Description  - 400 Invalid role
// @Description  - 403 Forbidden (managing self or outside scope)
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Param        body body dto.HospitalAdminChangeRoleRequest true "Role change payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id}/role [patch]
func (h *HospitalAdminStaffHandler) ChangeStaffRole(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	var req dto.HospitalAdminChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.userUseCase.HospitalAdminChangeStaffRole(c.Request.Context(), adminID, staffID, req.Role); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Staff role updated successfully"})
}

// DeleteStaff godoc
// @Summary      Soft delete staff (Hospital Admin)
// @Description  Soft delete a staff user within the same hospital scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** User.IsDeleted = true.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (managing self or outside scope)
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id} [delete]
func (h *HospitalAdminStaffHandler) DeleteStaff(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	if err := h.userUseCase.HospitalAdminSoftDeleteStaff(c.Request.Context(), adminID, staffID); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Staff soft-deleted successfully"})
}

// SetStaffActive godoc
// @Summary      Activate or deactivate staff (Hospital Admin)
// @Description  Toggle staff active status within the same hospital scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** User.IsActive toggled.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (managing self or outside scope)
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Param        body body dto.HospitalAdminSetStaffActiveRequest true "Activation payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id}/activation [patch]
func (h *HospitalAdminStaffHandler) SetStaffActive(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}

	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	var req dto.HospitalAdminSetStaffActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.IsActive == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid request payload"})
		return
	}

	if err := h.userUseCase.HospitalAdminSetStaffActive(c.Request.Context(), adminID, staffID, *req.IsActive); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	msg := "Staff deactivated successfully"
	if *req.IsActive {
		msg = "Staff activated successfully"
	}
	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: msg})
}

// ReassignDepartment godoc
// @Summary      Reassign staff department (Hospital Admin)
// @Description  Update staff department within the same hospital scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** User.DepartmentID updated.
// @Description  **Common Errors:**
// @Description  - 400 Invalid department
// @Description  - 403 Forbidden (outside scope)
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Param        body body dto.HospitalAdminReassignDepartmentRequest true "Department reassignment payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Description  **Validation:** The department must belong to the admin's hospital (checked via HospitalDepartment link).
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id}/department [patch]
func (h *HospitalAdminStaffHandler) ReassignDepartment(c *gin.Context) {
	adminID, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	var req dto.HospitalAdminReassignDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid request payload"})
		return
	}

	var departmentID *uuid.UUID
	if req.DepartmentID != nil {
		parsed, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department_id"})
			return
		}
		departmentID = &parsed

		// Validate department belongs to the admin's hospital
		if err := h.deptUseCase.ValidateDepartmentForHospital(c.Request.Context(), hospID, *departmentID); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
	}

	if err := h.userUseCase.HospitalAdminReassignStaffDepartment(c.Request.Context(), adminID, staffID, departmentID); err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Staff department reassigned successfully"})
}

// ListActiveStaffSessions godoc
// @Summary      List active staff sessions (Hospital Admin)
// @Description  View active sessions for hospital staff, optionally filtered by staff ID.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Visibility:** Hospital scoped sessions.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden
// @Tags         Hospital Admin
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Param        staff_id query string false "Staff user ID filter"
// @Success      200 {object} dto.HospitalAdminSessionListResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/sessions [get]
func (h *HospitalAdminStaffHandler) ListActiveStaffSessions(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}

	filter := iusecase.HospitalAdminSessionFilter{Page: 1, PageSize: 20}
	if p, err := strconv.Atoi(c.Query("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil {
		filter.PageSize = ps
	}
	if staffIDParam := c.Query("staff_id"); staffIDParam != "" {
		staffID, err := uuid.Parse(staffIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff_id"})
			return
		}
		filter.StaffID = &staffID
	}

	sessions, total, err := h.userUseCase.HospitalAdminListActiveStaffSessions(c.Request.Context(), adminID, filter)
	if err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.HospitalAdminSessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		resp = append(resp, dto.HospitalAdminSessionResponse{
			ID:        sess.ID.String(),
			UserID:    sess.UserID.String(),
			IPAddress: sess.IPAddress,
			UserAgent: sess.UserAgent,
			ExpiresAt: sess.ExpiresAt.Format("2006-01-02 15:04:05"),
			CreatedAt: sess.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.HospitalAdminSessionListResponse{
		Data:  resp,
		Total: total,
		Page:  filter.Page,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Active staff sessions retrieved successfully",
		},
	})
}

// ForceLogoutStaff godoc
// @Summary      Force logout staff (Hospital Admin)
// @Description  Revoke active sessions for a staff user within hospital scope.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **State Transition:** All active sessions for the user are blacklisted/deleted.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (managing self or outside scope)
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Success      200 {object} dto.HospitalAdminForceLogoutResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id}/force-logout [post]
func (h *HospitalAdminStaffHandler) ForceLogoutStaff(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	revoked, err := h.userUseCase.HospitalAdminForceLogoutStaff(c.Request.Context(), adminID, staffID)
	if err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.HospitalAdminForceLogoutResponse{
		RevokedSessions: revoked,
		BaseResponse:    dto.BaseResponse{Success: true, Message: "Staff sessions revoked successfully"},
	})
}

// ReplaceStaff godoc
// @Summary      Replace staff in-place (Hospital Admin)
// @Description  In-place replacement updates identity and password while keeping the same user ID.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Prerequisites:** Staff member must be in the same hospital.
// @Description  **State Transition:** User profile (name, email) and credentials updated.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (managing self or outside scope)
// @Description  - 409 Conflict (email exists)
// @Tags         Hospital Admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Staff user ID"
// @Param        body body dto.HospitalAdminReplaceStaffRequest true "Replacement payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/staff/{id}/replace [post]
func (h *HospitalAdminStaffHandler) ReplaceStaff(c *gin.Context) {
	adminID, _, ok := getAdminContext(c)
	if !ok {
		return
	}
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid staff ID"})
		return
	}

	var req dto.HospitalAdminReplaceStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	err = h.userUseCase.HospitalAdminReplaceStaff(c.Request.Context(), adminID, staffID, iusecase.HospitalAdminReplacementInput{
		FirstName:  req.FirstName,
		MiddleName: req.MiddleName,
		LastName:   req.LastName,
		Email:      req.Email,
		Password:   req.Password,
		Reason:     req.Reason,
	})
	if err != nil {
		c.JSON(mapHospitalAdminStaffError(err), dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Staff replaced in-place successfully"})
}

// GetReferralStatusHistory godoc
// @Summary      Get referral status history (Hospital Admin)
// @Description  Read-only status history for a referral linked to the admin's hospital.
// @Description  **Roles:** HOSPITAL_ADMIN
// @Description  **Visibility:** Hospital connected referral history.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (not connected)
// @Description  - 404 Not Found
// @Tags         Hospital Admin
// @Produce      json
// @Param        id path string true "Referral ID"
// @Param        limit query int false "Pagination limit" default(20)
// @Param        page query int false "Page number" default(1)
// @Success      200 {object} dto.PaginatedLogResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospital-admin/referrals/{id}/status-history [get]
func (h *HospitalAdminStaffHandler) GetReferralStatusHistory(c *gin.Context) {
	_, hospID, ok := getAdminContext(c)
	if !ok {
		return
	}

	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral ID"})
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

	history, total, err := h.referralUseCase.GetReferralStatusHistoryForHospitalAdmin(c.Request.Context(), hospID, referralID, limit, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	resp := make([]dto.LogResponseDTO, 0, len(history))
	for _, hst := range history {
		var from *string
		if hst.FromStatus != nil {
			f := string(*hst.FromStatus)
			from = &f
		}
		resp = append(resp, dto.LogResponseDTO{
			HistoryID:   hst.ID,
			ReferralID:  hst.ReferralID,
			ChangedByID: hst.ChangedByID,
			Role:        "System",
			FromStatus:  from,
			ToStatus:    string(hst.ToStatus),
			CreatedAt:   hst.ChangedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.PaginatedLogResponse{
		Data:         resp,
		Total:        total,
		Page:         page,
		PageSize:     limit,
		BaseResponse: dto.BaseResponse{Success: true, Message: "Referral status history retrieved successfully"},
	})
}
