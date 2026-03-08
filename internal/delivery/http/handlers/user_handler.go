package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
	"Hospital-Referral-System/internal/usecase"
)

type UserHandler struct {
	userUseCase usecase.UserUseCase
}

func NewUserHandler(userUseCase usecase.UserUseCase) *UserHandler {
	return &UserHandler{userUseCase: userUseCase}
}

// --- Request / Response DTOs ---

type CreateUserRequest struct {
	Email        string          `json:"email" binding:"required,email"`
	Password     string          `json:"password" binding:"required,min=8"`
	FirstName    string          `json:"first_name" binding:"required"`
	LastName     string          `json:"last_name" binding:"required"`
	NationalID   string          `json:"national_id"`
	Role         entity.UserRole `json:"role" binding:"required"`
	HospitalID   *string         `json:"hospital_id"`
	DepartmentID *string         `json:"department_id"`
}

type UpdateUserRequest struct {
	Email        *string          `json:"email" binding:"omitempty,email"`
	FirstName    *string          `json:"first_name"`
	LastName     *string          `json:"last_name"`
	NationalID   *string          `json:"national_id"`
	Role         *entity.UserRole `json:"role"`
	HospitalID   *string          `json:"hospital_id"`
	DepartmentID *string          `json:"department_id"`
	IsActive     *bool            `json:"is_active"`
}

type AssignRoleRequest struct {
	Role entity.UserRole `json:"role" binding:"required"`
}

type UserResponse struct {
	ID           string          `json:"id"`
	Email        string          `json:"email"`
	FirstName    string          `json:"first_name"`
	LastName     string          `json:"last_name"`
	NationalID   string          `json:"national_id"`
	Role         entity.UserRole `json:"role"`
	HospitalID   *string         `json:"hospital_id,omitempty"`
	DepartmentID *string         `json:"department_id,omitempty"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

func toUserResponse(u *entity.User) UserResponse {
	resp := UserResponse{
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
	if u.DepartmentID != nil {
		s := u.DepartmentID.String()
		resp.DepartmentID = &s
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
// @Success      201 {object} UserResponse
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hospital_id"})
			return
		}
		user.HospitalID = &id
	}
	if req.DepartmentID != nil {
		id, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
			return
		}
		user.DepartmentID = &id
	}

	if err := h.userUseCase.CreateUser(c.Request.Context(), user, req.Password); err != nil {
		switch err {
		case usecase.ErrEmailExists, usecase.ErrNationalIDExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case usecase.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		}
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
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
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	filter := repository.UserListFilter{
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	var resp []UserResponse
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  resp,
		"total": total,
		"page":  filter.Page,
	})
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Admin-only endpoint to retrieve a user by their ID
// @Tags         Users
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} UserResponse
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.userUseCase.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// UpdateUser godoc
// @Summary      Update a user
// @Description  Admin-only endpoint to update a user's information
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path string            true "User ID"
// @Param        body body UpdateUserRequest  true "User update payload"
// @Success      200 {object} UserResponse
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch existing user first
	existing, err := h.userUseCase.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hospital_id"})
			return
		}
		existing.HospitalID = &hid
	}
	if req.DepartmentID != nil {
		did, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
			return
		}
		existing.DepartmentID = &did
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.userUseCase.UpdateUser(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(existing))
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Admin-only endpoint to soft-delete a user
// @Tags         Users
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.userUseCase.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// GetMyProfile godoc
// @Summary      Get current user's profile
// @Description  Returns the profile of the currently authenticated user
// @Tags         Users
// @Produce      json
// @Success      200 {object} UserResponse
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/users/me [get]
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
		return
	}

	user, err := h.userUseCase.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// AssignRole godoc
// @Summary      Assign role to user
// @Description  Admin-only endpoint to change a user's role
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path string           true "User ID"
// @Param        body body AssignRoleRequest true "Role assignment payload"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/role [patch]
func (h *UserHandler) AssignRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userUseCase.AssignRole(c.Request.Context(), id, req.Role); err != nil {
		switch err {
		case usecase.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		case usecase.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign role"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}
