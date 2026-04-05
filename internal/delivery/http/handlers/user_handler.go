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

type UserHandler struct {
	userUseCase iusecase.UserUseCase
}

func NewUserHandler(userUseCase iusecase.UserUseCase) *UserHandler {
	return &UserHandler{userUseCase: userUseCase}
}

// --- Request / Response DTOs ---

type CreateUserRequest struct {
	Email        string          `json:"email" binding:"required,email" example:"analyst@moh.gov.et"`
	Password     string          `json:"password" binding:"required,min=8" example:"password123"`
	FirstName    string          `json:"first_name" binding:"required" example:"MoH"`
	LastName     string          `json:"last_name" binding:"required" example:"Analyst"`
	NationalID   string          `json:"national_id" example:"MOH-001"`
	Role         entity.UserRole `json:"role" binding:"required" example:"MOH_ANALYST"`
	HospitalID   *string         `json:"hospital_id" example:"0f74f069-d52d-4482-9ba5-41b007fdc1e5"`
	DepartmentID *string         `json:"department_id" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
}

type UpdateUserRequest struct {
	Email        *string          `json:"email" binding:"omitempty,email" example:"analyst@moh.gov.et"`
	FirstName    *string          `json:"first_name" example:"MoH"`
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
		ID:         u.ID.String(),
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		NationalID: u.NationalID,
		Role:       u.Role,
		IsActive:   u.IsActive,
		CreatedAt:  u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if u.HospitalID != nil {
		s := u.HospitalID.String()
		resp.HospitalID = &s
	}
	if u.Hospital != nil {
		resp.HospitalName = u.Hospital.Name
	}
	if u.DepartmentID != nil {
		s := u.DepartmentID.String()
		resp.DepartmentID = &s
	}
	if u.Department != nil {
		resp.DepartmentName = u.Department.Name
	}
	return resp
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Admin-only endpoint to create a new user account
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body body CreateUserRequest true "User creation payload"
// @Success      201 {object} dto.UserResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [post]
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
		case usecase.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to create user"})
		}
		return
	}

	c.JSON(http.StatusCreated, dto.UserResponse{
		ID:             user.ID.String(),
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		NationalID:     user.NationalID,
		Role:           user.Role,
		IsActive:       user.IsActive,
		CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "User created successfully",
		},
	})
}

// ListUsers godoc
// @Summary      List users
// @Description  Admin-only endpoint to list users with optional filters
// @Tags         Users
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        role      query string false "Filter by role"
// @Param        hospital_id query string false "Filter by hospital ID"
// @Param        is_active query bool   false "Filter by active status"
// @Param        search    query string false "Search by name or email"
// @Success      200 {object} dto.UserListResponse
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
	if hospID := c.Query("hospital_id"); hospID != "" {
		filter.HospitalID = &hospID
	}
	if active := c.Query("is_active"); active != "" {
		a := active == "true"
		filter.IsActive = &a
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	users, total, err := h.userUseCase.ListUsers(c.Request.Context(), filter)
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
// @Description  Admin-only endpoint to retrieve a user by their ID
// @Tags         Users
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} dto.UserResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid user ID",
		})
		return
	}

	user, err := h.userUseCase.GetUserByID(c.Request.Context(), id)
	if err != nil {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:             user.ID.String(),
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		NationalID:     user.NationalID,
		Role:           user.Role,
		IsActive:       user.IsActive,
		CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "User details retrieved successfully",
		},
	})
}

// UpdateUser godoc
// @Summary      Update a user
// @Description  Admin-only endpoint to update a user's information
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path string            true "User ID"
// @Param        body body UpdateUserRequest  true "User update payload"
// @Success      200 {object} dto.UserResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [put]
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
	existing, err := h.userUseCase.GetUserByID(c.Request.Context(), id)
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

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:             existing.ID.String(),
		Email:          existing.Email,
		FirstName:      existing.FirstName,
		LastName:       existing.LastName,
		NationalID:     existing.NationalID,
		Role:           existing.Role,
		IsActive:       existing.IsActive,
		CreatedAt:      existing.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      existing.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "User updated successfully",
		},
	})
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Admin-only endpoint to soft-delete a user
// @Tags         Users
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [delete]
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
// @Description  Returns the profile of the currently authenticated user
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

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:             user.ID.String(),
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		NationalID:     user.NationalID,
		Role:           user.Role,
		IsActive:       user.IsActive,
		CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Profile retrieved successfully",
		},
	})
}

// AssignRole godoc
// @Summary      Assign role to user
// @Description  Admin-only endpoint to change a user's role
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path string           true "User ID"
// @Param        body body AssignRoleRequest true "Role assignment payload"
// @Success      200 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/role [patch]
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
