package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/usecase"
)

type DepartmentHandler struct {
	deptUseCase iusecase.DepartmentUseCase
}

func NewDepartmentHandler(deptUseCase iusecase.DepartmentUseCase) *DepartmentHandler {
	return &DepartmentHandler{deptUseCase: deptUseCase}
}

// --- Request / Response DTOs ---

type CreateDepartmentRequest struct {
	Name        string  `json:"name" binding:"required" example:"Cardiology"`
	Description *string `json:"description" example:"Heart and blood vessel diseases"`
}

type UpdateDepartmentRequest struct {
	Name        *string `json:"name" example:"Cardiology"`
	Description *string `json:"description" example:"Heart and blood vessel diseases"`
}

type LinkDepartmentRequest struct {
	DepartmentID string `json:"department_id" binding:"required" example:"dfc2b777-a5d5-424b-911a-976b2e8d8614"`
	DailyLimit   int    `json:"daily_limit" example:"20"`
}

func toDepartmentResponse(d *entity.Department) dto.DepartmentResponse {
	resp := dto.DepartmentResponse{
		ID:        d.ID.String(),
		Name:      d.Name,
		CreatedAt: d.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: d.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if d.Description != nil {
		resp.Description = *d.Description
	}
	return resp
}

func toHospitalDepartmentResponse(hd *entity.HospitalDepartment) dto.HospitalDepartmentResponse {
	return toHospitalDepartmentResponseWithHead(hd, nil)
}

func toHospitalDepartmentResponseWithHead(hd *entity.HospitalDepartment, head *entity.User) dto.HospitalDepartmentResponse {
	resp := dto.HospitalDepartmentResponse{
		ID:                 hd.ID.String(),
		HospitalID:         hd.HospitalID.String(),
		DepartmentID:       hd.DepartmentID.String(),
		Department:         toDepartmentResponse(&hd.Department),
		StandardDailyLimit: hd.StandardDailyLimit,
		IsActive:           hd.IsActive,
		CreatedAt:          hd.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if head != nil {
		resp.DepartmentHead = &dto.DepartmentHeadSummary{
			ID:       head.ID.String(),
			FullName: userFullName(head),
		}
	}
	return resp
}

func userFullName(u *entity.User) string {
	if u == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	for _, p := range []string{u.FirstName, u.MiddleName, u.LastName} {
		if s := strings.TrimSpace(p); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// CreateDepartment godoc
// @Summary      Create a new department
// @Description  Admin-only endpoint to create a department.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 401 Unauthorized
// @Description  - 403 Forbidden
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        body body CreateDepartmentRequest true "Department creation payload"
// @Success      201 {object} dto.DepartmentResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/departments [post]
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	dept := &entity.Department{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.deptUseCase.CreateDepartment(c.Request.Context(), dept); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to create department",
		})
		return
	}

	c.JSON(http.StatusCreated, dto.DepartmentResponse{
		ID:             dept.ID.String(),
		Name:           dept.Name,
		CreatedAt:      dept.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      dept.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Department created successfully",
		},
	})
}

// ListDepartments godoc
// @Summary      List departments
// @Description  List departments with optional search.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Departments
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        search    query string false "Search by name"
// @Success      200 {object} dto.DepartmentListResponse
// @Security     BearerAuth
// @Router       /api/v1/departments [get]
func (h *DepartmentHandler) ListDepartments(c *gin.Context) {
	filter := irepository.DepartmentListFilter{
		Page:     1,
		PageSize: 20,
	}

	if p, err := strconv.Atoi(c.Query("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil {
		filter.PageSize = ps
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	departments, total, err := h.deptUseCase.ListDepartments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to list departments",
		})
		return
	}

	var resp []dto.DepartmentResponse
	for i := range departments {
		resp = append(resp, toDepartmentResponse(&departments[i]))
	}

	c.JSON(http.StatusOK, dto.DepartmentListResponse{
		Data:  resp,
		Total: total,
		Page:  filter.Page,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Departments retrieved successfully",
		},
	})
}

// GetDepartment godoc
// @Summary      Get department by ID
// @Description  Retrieve a department by its ID.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 400 invalid ID format
// @Description  - 404 Not Found
// @Tags         Departments
// @Produce      json
// @Param        id path string true "Department ID"
// @Success      200 {object} dto.DepartmentResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/departments/{id} [get]
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	dept, err := h.deptUseCase.GetDepartmentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Department not found"})
		return
	}

	c.JSON(http.StatusOK, dto.DepartmentResponse{
		ID:             dept.ID.String(),
		Name:           dept.Name,
		CreatedAt:      dept.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      dept.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Department details retrieved successfully",
		},
	})
}

// UpdateDepartment godoc
// @Summary      Update a department
// @Description  Admin-only endpoint to update department information.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 404 Not Found
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        id   path string                  true "Department ID"
// @Param        body body UpdateDepartmentRequest  true "Department update payload"
// @Success      200 {object} dto.DepartmentResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/departments/{id} [put]
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	existing, err := h.deptUseCase.GetDepartmentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Department not found"})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = req.Description
	}

	if err := h.deptUseCase.UpdateDepartment(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to update department",
		})
		return
	}

	c.JSON(http.StatusOK, dto.DepartmentResponse{
		ID:             existing.ID.String(),
		Name:           existing.Name,
		CreatedAt:      existing.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      existing.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Department updated successfully",
		},
	})
}

// DeleteDepartment godoc
// @Summary      Delete a department
// @Description  Admin-only endpoint to delete a department.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid ID format
// @Description  - 404 Not Found
// @Tags         Departments
// @Produce      json
// @Param        id path string true "Department ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/departments/{id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	if err := h.deptUseCase.DeleteDepartment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "Department not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Department deleted successfully",
	})
}

// LinkDepartmentToHospital godoc
// @Summary      Link a department to a hospital
// @Description  Admin-only endpoint to associate a department with a hospital.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **State Transition:** Creates a link and initializes the daily schedule.
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 404 hospital/department not found
// @Description  - 409 link already exists
// @Tags         Hospitals
// @Accept       json
// @Produce      json
// @Param        id   path string              true "Hospital ID"
// @Param        body body LinkDepartmentRequest true "Link payload"
// @Success      201 {object} dto.BaseResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id}/departments [post]
func (h *DepartmentHandler) LinkDepartmentToHospital(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid hospital ID"})
		return
	}

	var req LinkDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	deptID, err := uuid.Parse(req.DepartmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid department_id",
		})
		return
	}

	if err := h.deptUseCase.LinkDepartmentToHospital(c.Request.Context(), hospitalID, deptID, req.DailyLimit); err != nil {
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

	c.JSON(http.StatusCreated, dto.BaseResponse{
		Success: true,
		Message: "Department linked to hospital successfully",
	})
}

// UnlinkDepartmentFromHospital godoc
// @Summary      Unlink a department from a hospital
// @Description  Admin-only endpoint to remove a department-hospital association.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 404 link not found
// @Tags         Hospitals
// @Produce      json
// @Param        id     path string true "Hospital ID"
// @Param        deptId path string true "Department ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id}/departments/{deptId} [delete]
func (h *DepartmentHandler) UnlinkDepartmentFromHospital(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid hospital ID"})
		return
	}

	deptID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid department ID"})
		return
	}

	if err := h.deptUseCase.UnlinkDepartmentFromHospital(c.Request.Context(), hospitalID, deptID); err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Department unlinked from hospital successfully",
	})
}

// ListHospitalDepartments godoc
// @Summary      List departments of a hospital
// @Description  Retrieve all departments linked to a specific hospital.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 404 hospital not found
// @Tags         Hospitals
// @Produce      json
// @Param        id path string true "Hospital ID"
// @Success      200 {object} dto.HospitalDepartmentListResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id}/departments [get]
func (h *DepartmentHandler) ListHospitalDepartments(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid hospital ID"})
		return
	}

	links, err := h.deptUseCase.ListHospitalDepartments(c.Request.Context(), hospitalID)
	if err != nil {
		if err == usecase.ErrHospitalNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Success: false, Error: "Hospital not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: "Failed to list hospital departments"})
		return
	}

	var resp []dto.HospitalDepartmentResponse
	for i := range links {
		resp = append(resp, toHospitalDepartmentResponse(&links[i]))
	}

	c.JSON(http.StatusOK, dto.HospitalDepartmentListResponse{
		Data: resp,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospital departments retrieved successfully",
		},
	})
}
