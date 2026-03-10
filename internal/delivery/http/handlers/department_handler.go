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

type DepartmentHandler struct {
	deptUseCase usecase.DepartmentUseCase
}

func NewDepartmentHandler(deptUseCase usecase.DepartmentUseCase) *DepartmentHandler {
	return &DepartmentHandler{deptUseCase: deptUseCase}
}

// --- Request / Response DTOs ---

type CreateDepartmentRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

type UpdateDepartmentRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type LinkDepartmentRequest struct {
	DepartmentID string `json:"department_id" binding:"required"`
	DailyLimit   int    `json:"daily_limit"`
}

type DepartmentResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type HospitalDepartmentResponse struct {
	ID                 string             `json:"id"`
	HospitalID         string             `json:"hospital_id"`
	DepartmentID       string             `json:"department_id"`
	Department         DepartmentResponse `json:"department"`
	StandardDailyLimit int                `json:"standard_daily_limit"`
	IsActive           bool               `json:"is_active"`
	CreatedAt          string             `json:"created_at"`
}

func toDepartmentResponse(d *entity.Department) DepartmentResponse {
	resp := DepartmentResponse{
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

func toHospitalDepartmentResponse(hd *entity.HospitalDepartment) HospitalDepartmentResponse {
	return HospitalDepartmentResponse{
		ID:                 hd.ID.String(),
		HospitalID:         hd.HospitalID.String(),
		DepartmentID:       hd.DepartmentID.String(),
		Department:         toDepartmentResponse(&hd.Department),
		StandardDailyLimit: hd.StandardDailyLimit,
		IsActive:           hd.IsActive,
		CreatedAt:          hd.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// CreateDepartment godoc
// @Summary      Create a new department
// @Description  Admin-only endpoint to create a department
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        body body CreateDepartmentRequest true "Department creation payload"
// @Success      201 {object} DepartmentResponse
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/departments [post]
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dept := &entity.Department{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.deptUseCase.CreateDepartment(c.Request.Context(), dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create department"})
		return
	}

	c.JSON(http.StatusCreated, toDepartmentResponse(dept))
}

// ListDepartments godoc
// @Summary      List departments
// @Description  List departments with optional search
// @Tags         Departments
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        search    query string false "Search by name"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/departments [get]
func (h *DepartmentHandler) ListDepartments(c *gin.Context) {
	filter := repository.DepartmentListFilter{
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list departments"})
		return
	}

	var resp []DepartmentResponse
	for i := range departments {
		resp = append(resp, toDepartmentResponse(&departments[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  resp,
		"total": total,
		"page":  filter.Page,
	})
}

// GetDepartment godoc
// @Summary      Get department by ID
// @Description  Retrieve a department by its ID
// @Tags         Departments
// @Produce      json
// @Param        id path string true "Department ID"
// @Success      200 {object} DepartmentResponse
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/departments/{id} [get]
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department ID"})
		return
	}

	dept, err := h.deptUseCase.GetDepartmentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	c.JSON(http.StatusOK, toDepartmentResponse(dept))
}

// UpdateDepartment godoc
// @Summary      Update a department
// @Description  Admin-only endpoint to update department information
// @Tags         Departments
// @Accept       json
// @Produce      json
// @Param        id   path string                  true "Department ID"
// @Param        body body UpdateDepartmentRequest  true "Department update payload"
// @Success      200 {object} DepartmentResponse
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/departments/{id} [put]
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department ID"})
		return
	}

	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.deptUseCase.GetDepartmentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = req.Description
	}

	if err := h.deptUseCase.UpdateDepartment(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update department"})
		return
	}

	c.JSON(http.StatusOK, toDepartmentResponse(existing))
}

// DeleteDepartment godoc
// @Summary      Delete a department
// @Description  Admin-only endpoint to delete a department
// @Tags         Departments
// @Produce      json
// @Param        id path string true "Department ID"
// @Success      200 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/departments/{id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department ID"})
		return
	}

	if err := h.deptUseCase.DeleteDepartment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Department deleted successfully"})
}

// LinkDepartmentToHospital godoc
// @Summary      Link a department to a hospital
// @Description  Admin-only endpoint to associate a department with a hospital
// @Tags         Hospitals
// @Accept       json
// @Produce      json
// @Param        id   path string              true "Hospital ID"
// @Param        body body LinkDepartmentRequest true "Link payload"
// @Success      201 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id}/departments [post]
func (h *DepartmentHandler) LinkDepartmentToHospital(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hospital ID"})
		return
	}

	var req LinkDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deptID, err := uuid.Parse(req.DepartmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
		return
	}

	if err := h.deptUseCase.LinkDepartmentToHospital(c.Request.Context(), hospitalID, deptID, req.DailyLimit); err != nil {
		switch err {
		case usecase.ErrHospitalNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Hospital not found"})
		case usecase.ErrDepartmentNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		case usecase.ErrHospitalDeptLinkExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link department"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Department linked to hospital successfully"})
}

// UnlinkDepartmentFromHospital godoc
// @Summary      Unlink a department from a hospital
// @Description  Admin-only endpoint to remove a department-hospital association
// @Tags         Hospitals
// @Produce      json
// @Param        id     path string true "Hospital ID"
// @Param        deptId path string true "Department ID"
// @Success      200 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id}/departments/{deptId} [delete]
func (h *DepartmentHandler) UnlinkDepartmentFromHospital(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hospital ID"})
		return
	}

	deptID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department ID"})
		return
	}

	if err := h.deptUseCase.UnlinkDepartmentFromHospital(c.Request.Context(), hospitalID, deptID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Department unlinked from hospital successfully"})
}

// ListHospitalDepartments godoc
// @Summary      List departments of a hospital
// @Description  Retrieve all departments linked to a specific hospital
// @Tags         Hospitals
// @Produce      json
// @Param        id path string true "Hospital ID"
// @Success      200 {array} HospitalDepartmentResponse
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id}/departments [get]
func (h *DepartmentHandler) ListHospitalDepartments(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hospital ID"})
		return
	}

	links, err := h.deptUseCase.ListHospitalDepartments(c.Request.Context(), hospitalID)
	if err != nil {
		if err == usecase.ErrHospitalNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Hospital not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list hospital departments"})
		return
	}

	var resp []HospitalDepartmentResponse
	for i := range links {
		resp = append(resp, toHospitalDepartmentResponse(&links[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
