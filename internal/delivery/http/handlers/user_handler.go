package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/pkg/utils"
	"Hospital-Referral-System/internal/usecase"
)

type UserHandler struct {
	userUseCase iusecase.UserUseCase
}

func NewUserHandler(userUseCase iusecase.UserUseCase) *UserHandler {
	return &UserHandler{userUseCase: userUseCase}
}

func (h *UserHandler) getRequesterID(c *gin.Context) uuid.UUID {
	id, exists := c.Get("userID")
	if !exists {
		return uuid.Nil
	}
	return id.(uuid.UUID)
}

// --- Request / Response DTOs ---

type CreateUserRequest struct {
	Email        string          `json:"email" binding:"required,email" example:"analyst@moh.gov.et"`
	Password     string          `json:"password" binding:"required,min=8" example:"password123"`
	FirstName    string          `json:"first_name" binding:"required" example:"Abebe"`
	MiddleName   string          `json:"middle_name" binding:"required" example:"Kebede"`
	LastName     string          `json:"last_name" binding:"required" example:"Analyst"`
	NationalID   string          `json:"national_id" example:"MOH-001"`
	Role         entity.UserRole `json:"role" binding:"required" example:"MOH_ANALYST"`
	HospitalID   *string         `json:"hospital_id" example:"0f74f069-d52d-4482-9ba5-41b007fdc1e5"`
	DepartmentID *string         `json:"department_id" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
}

type UpdateUserRequest struct {
	Email        *string          `json:"email" binding:"omitempty,email" example:"analyst@moh.gov.et"`
	FirstName    *string          `json:"first_name" example:"Abebe"`
	MiddleName   *string          `json:"middle_name" example:"Kebede"`
	LastName     *string          `json:"last_name" example:"Analyst"`
	NationalID   *string          `json:"national_id" example:"MOH-001"`
	Role         *entity.UserRole `json:"role" example:"MOH_ANALYST"`
	HospitalID   *string         `json:"hospital_id" example:"0f74f069-d52d-4482-9ba5-41b007fdc1e5"`
	DepartmentID *string         `json:"department_id" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
	IsActive     *bool            `json:"is_active" example:"true"`
}

type AssignRoleRequest struct {
	Role entity.UserRole `json:"role" binding:"required" example:"SYSTEM_SUPER_ADMIN"`
}

func toUserResponse(u *entity.User) dto.UserResponse {
	resp := dto.UserResponse{
		ID:              u.ID.String(),
		Email:           u.Email,
		FirstName:       u.FirstName,
		MiddleName:      u.MiddleName,
		LastName:        u.LastName,
		NationalID:      u.NationalID,
		Role:            u.Role,
		IsActive:        u.IsActive,
		ProfileImageURL: utils.OptimizeCloudinaryURL(u.ProfileImageURL),
		CreatedAt:       u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       u.UpdatedAt.Format(time.RFC3339),
	}
	if u.HospitalID != nil {
		s := u.HospitalID.String()
		resp.HospitalID = &s
	}
	if u.Hospital != nil {
		resp.Hospital = &dto.HospitalResponse{
			ID:           u.Hospital.ID.String(),
			Name:         u.Hospital.Name,
			TierLevel:    string(u.Hospital.TierLevel),
			Region:       u.Hospital.Region,
			IsActive:     u.Hospital.IsActive,
			CreatedAt:    u.Hospital.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    u.Hospital.UpdatedAt.Format(time.RFC3339),
		}
		if u.Hospital.Address != nil {
			resp.Hospital.Address = *u.Hospital.Address
		}
		if u.Hospital.ContactPhone != nil {
			resp.Hospital.ContactPhone = *u.Hospital.ContactPhone
		}
	}
	if u.DepartmentID != nil {
		s := u.DepartmentID.String()
		resp.DepartmentID = &s
	}
	if u.Department != nil {
		resp.Department = &dto.DepartmentResponse{
			ID:        u.Department.ID.String(),
			Name:      u.Department.Name,
			CreatedAt: u.Department.CreatedAt.Format(time.RFC3339),
			UpdatedAt: u.Department.UpdatedAt.Format(time.RFC3339),
		}
		if u.Department.Description != nil {
			resp.Department.Description = *u.Department.Description
		}
	}
	return resp
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Admin-only endpoint to create a new user account.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 409 conflict (email/national_id)
// @Tags         SystemAdmin
// @Accept       json
// @Produce      json
// @Param        body body CreateUserRequest true "User creation payload"
// @Success      201 {object} dto.UserResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/system-admin/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
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

	if req.HospitalID != nil {
		id, err := uuid.Parse(*req.HospitalID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Error:   "invalid hospital_id",
			})
			return
		}
		user.HospitalID = &id
	}
	if req.DepartmentID != nil {
		id, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Error:   "invalid department_id",
			})
			return
		}
		user.DepartmentID = &id
	}

	if err := h.userUseCase.CreateUser(c.Request.Context(), user, req.Password); err != nil {
		switch err {
		case usecase.ErrEmailExists, usecase.ErrNationalIDExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse{Success: false, Error: err.Error()})
		case usecase.ErrInvalidRole, usecase.ErrInvalidDepartment:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to create user"})
		}
		return
	}

	resp := toUserResponse(user)
	resp.BaseResponse = dto.BaseResponse{
		Success: true,
		Message: "User created successfully",
	}

	c.JSON(http.StatusCreated, resp)
}

// ListUsers godoc
// @Summary      List users (Hospital-Scoped)
// @Description  Retrieve a list of users within your own hospital.
// @Description  **Roles:** Hospital Admins, Doctors, Specialists, Liaisons
// @Description  **Visibility:** MOH Analysts are strictly forbidden. Users see only others in the same hospital.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden for MOH Analysts
// @Tags         Users
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        role      query string false "Filter by role"
// @Param        name      query string false "Tokenized name search (e.g. 'Doe John')"
// @Param        email     query string false "Filter by partial email"
// @Param        dept_id   query string false "Filter by Department ID"
// @Param        is_active query bool   false "Filter by active status"
// @Success      200 {object} dto.UserListResponse
// @Failure      403 {object} dto.ErrorResponse "Forbidden for MOH Analysts"
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	filter := irepository.UserListFilter{
		Page:     1,
		PageSize: 20,
	}

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

	users, total, err := h.userUseCase.ListUsers(c.Request.Context(), filter, h.getRequesterID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to list users",
		})
		return
	}

	var resp []dto.UserResponse
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}

	c.JSON(http.StatusOK, dto.UserListResponse{
		Data:  resp,
		Total: total,
		Page:  filter.Page,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Users retrieved successfully",
		},
	})
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Retrieve detailed profile of a user with nuanced visibility rules.
// @Description  **Roles:** All authenticated roles (with visibility rules).
// @Description  **Visibility Rules:**
// @Description  - System Admins: Can view any profile.
// @Description  - Clinical roles: Can view each other across hospitals (for referrals).
// @Description  - Clinical roles: Cannot view SystemAdmins or Receptionists from other hospitals.
// @Description  - Receptionists: Can ONLY view users within their own hospital.
// @Description  - MOH Analysts: Cannot view any user except their own.
// @Description  **Common Errors:**
// @Description  - 403 visibility violation
// @Description  - 404 Not Found
// @Tags         Users
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} dto.UserResponse
// @Failure      403 {object} dto.ErrorResponse "Forbidden due to visibility restrictions"
// @Failure      404 {object} dto.ErrorResponse "User not found"
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	user, err := h.userUseCase.GetUserByID(c.Request.Context(), id, h.getRequesterID(c))
	if err != nil {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	resp := toUserResponse(user)
	resp.BaseResponse = dto.BaseResponse{
		Success: true,
		Message: "User details retrieved successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateUser godoc
// @Summary      Update a user
// @Description  Admin-only endpoint to update a user's information.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 404 Not Found
// @Tags         SystemAdmin
// @Accept       json
// @Produce      json
// @Param        id   path string            true "User ID"
// @Param        body body UpdateUserRequest  true "User update payload"
// @Success      200 {object} dto.UserResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/system-admin/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user ID",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Fetch existing user first
	existing, err := h.userUseCase.GetUserByID(c.Request.Context(), id, h.getRequesterID(c))
	if err != nil {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	// Apply partial updates
	if req.Email != nil {
		existing.Email = *req.Email
	}
	if req.FirstName != nil {
		existing.FirstName = *req.FirstName
	}
	if req.MiddleName != nil {
		existing.MiddleName = *req.MiddleName
	}
	if req.LastName != nil {
		existing.LastName = *req.LastName
	}
	if req.NationalID != nil {
		existing.NationalID = *req.NationalID
	}
	if req.Role != nil {
		existing.Role = *req.Role
	}
	if req.HospitalID != nil {
		hid, err := uuid.Parse(*req.HospitalID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Error:   "invalid hospital_id",
			})
			return
		}
		existing.HospitalID = &hid
	}
	if req.DepartmentID != nil {
		did, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Error:   "invalid department_id",
			})
			return
		}
		existing.DepartmentID = &did
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.userUseCase.UpdateUser(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to update user",
		})
		return
	}

	resp := toUserResponse(existing)
	resp.BaseResponse = dto.BaseResponse{
		Success: true,
		Message: "User updated successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Admin-only endpoint to soft-delete a user.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 404 Not Found
// @Tags         SystemAdmin
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/system-admin/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user ID",
		})
		return
	}

	if err := h.userUseCase.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

// GetMyProfile godoc
// @Summary      Get current user's profile
// @Description  Returns the profile of the currently authenticated user.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Tags         Users
// @Produce      json
// @Success      200 {object} dto.UserResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/me [get]
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid user ID in context",
		})
		return
	}

	user, err := h.userUseCase.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	resp := toUserResponse(user)
	resp.BaseResponse = dto.BaseResponse{
		Success: true,
		Message: "Profile retrieved successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// SystemAdminListUsers godoc
// @Summary      Global User List (System Admin Only)
// @Description  Administrative-only endpoint for global user discovery across all hospitals.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Visibility:** Global access to all users.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (not super admin)
// @Tags         SystemAdmin
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        name      query string false "Tokenized name search"
// @Param        email     query string false "Filter by email"
// @Param        hospital_id query string false "Filter by Hospital ID"
// @Param        dept_id   query string false "Filter by Department ID"
// @Param        role      query string false "Filter by role"
// @Param        is_active query bool   false "Filter by active status"
// @Success      200 {object} dto.UserListResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/system-admin/users [get]
func (h *UserHandler) SystemAdminListUsers(c *gin.Context) {
	filter := irepository.UserListFilter{
		Page:     1,
		PageSize: 20,
	}

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
	if hospID := c.Query("hospital_id"); hospID != "" {
		filter.HospitalID = &hospID
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

	users, total, err := h.userUseCase.ListUsers(c.Request.Context(), filter, h.getRequesterID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to list users",
		})
		return
	}

	var resp []dto.UserResponse
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}

	c.JSON(http.StatusOK, dto.UserListResponse{
		Data:  resp,
		Total: total,
		Page:  filter.Page,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Global user list retrieved successfully",
		},
	})
}

// AssignRole godoc
// @Summary      Assign role to user
// @Description  Admin-only endpoint to change a user's role.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid role
// @Description  - 404 Not Found
// @Tags         SystemAdmin
// @Accept       json
// @Produce      json
// @Param        id   path string           true "User ID"
// @Param        body body AssignRoleRequest true "Role assignment payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/system-admin/users/{id}/role [patch]
func (h *UserHandler) AssignRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user ID",
		})
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	if err := h.userUseCase.AssignRole(c.Request.Context(), id, req.Role); err != nil {
		switch err {
		case usecase.ErrUserNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "User not found"})
		case usecase.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to assign role"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "User role assigned successfully",
	})
}

// @Summary      Update Profile Image
// @Description  Update the current user's profile image by uploading a file (Max 5MB: JPEG, PNG, WEBP).
// @Description  **Roles:** Any authenticated user.
// @Description  **Constraints:** Max 5MB, format must be JPEG, PNG, or WEBP.
// @Description  **Common Errors:**
// @Description  - 400 invalid size/format
// @Tags         Users
// @Accept       mpfd
// @Produce      json
// @Param        image formData file true "Profile image file"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/profile/image [put]
func (h *UserHandler) UpdateProfileImage(c *gin.Context) {
	userID := h.getRequesterID(c)

	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Profile image file is required"})
		return
	}

	// 1. Validate Size (Max 5MB)
	if fileHeader.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Image size exceeds 5MB limit"})
		return
	}

	// 2. Validate Format
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to read image file"})
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to read image header"})
		return
	}

	contentType := http.DetectContentType(buffer)
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}

	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid image format. Allowed: JPEG, PNG, WEBP"})
		return
	}

	// Reset file pointer after reading header
	file, _ = fileHeader.Open()
	defer file.Close()

	if err := h.userUseCase.UpdateProfileImage(c.Request.Context(), userID, file); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to update profile image: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Profile image updated successfully",
	})
}

// DeleteMyProfileImage godoc
// @Summary      Delete own Profile Image
// @Description  Remove the current user's profile image (unsets URL and PublicID).
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Tags         Users
// @Produce      json
// @Success      200 {object} dto.BaseResponse
// @Failure      401 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/profile/image [delete]
func (h *UserHandler) DeleteMyProfileImage(c *gin.Context) {
	userID := h.getRequesterID(c)
	if err := h.userUseCase.DeleteProfileImage(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Profile image removed successfully",
	})
}

// ModerateProfileImage godoc
// @Summary      Moderate Profile Image
// @Description  Remove a user's profile image (SystemAdmin or HospitalAdmin only).
// @Description  **Roles:** SYSTEM_SUPER_ADMIN, HOSPITAL_ADMIN
// @Description  **Prerequisites:** Hospital Admin can only moderate users in their own hospital.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (out of scope)
// @Tags         SystemAdmin
// @Produce      json
// @Param        id path string true "User ID to moderate"
// @Success      200 {object} dto.BaseResponse
// @Failure      403 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/system-admin/users/{id}/profile/image [delete]
func (h *UserHandler) ModerateProfileImage(c *gin.Context) {
	moderatorID := h.getRequesterID(c)
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "Invalid user ID"})
		return
	}

	if err := h.userUseCase.ModerateProfileImage(c.Request.Context(), userID, moderatorID); err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Profile image removed by moderator",
	})
}
