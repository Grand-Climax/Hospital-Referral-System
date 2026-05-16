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

type HospitalHandler struct {
	hospitalUseCase iusecase.HospitalUseCase
}

func NewHospitalHandler(hospitalUseCase iusecase.HospitalUseCase) *HospitalHandler {
	return &HospitalHandler{hospitalUseCase: hospitalUseCase}
}

// --- Request / Response DTOs ---

type CreateHospitalRequest struct {
	Name         string              `json:"name" binding:"required" example:"Tikur Anbessa Specialized Hospital"`
	TierLevel    entity.HospitalTier `json:"tier_level" binding:"required" example:"SPECIALIZED"`
	Region       string              `json:"region" binding:"required" example:"Addis Ababa"`
	Address      *string             `json:"address" example:"Churchill Road, Addis Ababa, Ethiopia"`
	ContactPhone *string             `json:"contact_phone" example:"+251 11 111 2233"`
}

type UpdateHospitalRequest struct {
	Name         *string              `json:"name" example:"Tikur Anbessa Specialized Hospital"`
	TierLevel    *entity.HospitalTier `json:"tier_level" example:"SPECIALIZED"`
	Region       *string              `json:"region" example:"Addis Ababa"`
	Address      *string              `json:"address" example:"Churchill Road, Addis Ababa, Ethiopia"`
	ContactPhone *string              `json:"contact_phone" example:"+251 11 111 2233"`
	IsActive     *bool                `json:"is_active" example:"true"`
}

func toHospitalResponse(h *entity.Hospital) dto.HospitalResponse {
	resp := dto.HospitalResponse{
		ID:        h.ID.String(),
		Name:      h.Name,
		TierLevel: string(h.TierLevel),
		Region:    string(h.Region),
		IsActive:  h.IsActive,
		CreatedAt: h.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: h.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if h.Address != nil {
		resp.Address = *h.Address
	}
	if h.ContactPhone != nil {
		resp.ContactPhone = *h.ContactPhone
	}
	return resp
}

// CreateHospital godoc
// @Summary      Create a new hospital
// @Description  Admin-only endpoint to create a hospital.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 401 Unauthorized
// @Description  - 403 Forbidden
// @Tags         Hospitals
// @Accept       json
// @Produce      json
// @Param        body body CreateHospitalRequest true "Hospital creation payload"
// @Success      201 {object} dto.HospitalResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals [post]
func (h *HospitalHandler) CreateHospital(c *gin.Context) {
	var req CreateHospitalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	hospital := &entity.Hospital{
		Name:         req.Name,
		TierLevel:    req.TierLevel,
		Region:       entity.EthiopianRegion(req.Region),
		Address:      req.Address,
		ContactPhone: req.ContactPhone,
	}

	if err := h.hospitalUseCase.CreateHospital(c.Request.Context(), hospital); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to create hospital",
		})
		return
	}

	c.JSON(http.StatusCreated, dto.HospitalResponse{
		ID:           hospital.ID.String(),
		Name:         hospital.Name,
		TierLevel:    string(hospital.TierLevel),
		Region:       string(hospital.Region),
		IsActive:     hospital.IsActive,
		CreatedAt:    hospital.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    hospital.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospital created successfully",
		},
	})
}

// ListHospitals godoc
// @Summary      List hospitals
// @Description  List hospitals with optional filters.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         Hospitals
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        tier      query string false "Filter by tier level"
// @Param        region    query string false "Filter by region"
// @Param        is_active query bool   false "Filter by active status"
// @Param        search    query string false "Search by name"
// @Success      200 {object} dto.HospitalListResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals [get]
func (h *HospitalHandler) ListHospitals(c *gin.Context) {
	filter := irepository.HospitalListFilter{
		Page:     1,
		PageSize: 20,
	}

	if p, err := strconv.Atoi(c.Query("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil {
		filter.PageSize = ps
	}
	if tier := c.Query("tier"); tier != "" {
		t := entity.HospitalTier(tier)
		filter.Tier = &t
	}
	if region := c.Query("region"); region != "" {
		filter.Region = &region
	}
	if active := c.Query("is_active"); active != "" {
		a := active == "true"
		filter.IsActive = &a
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	hospitals, total, err := h.hospitalUseCase.ListHospitals(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to list hospitals",
		})
		return
	}

	var resp []dto.HospitalResponse
	for i := range hospitals {
		resp = append(resp, toHospitalResponse(&hospitals[i]))
	}

	c.JSON(http.StatusOK, dto.HospitalListResponse{
		Data:  resp,
		Total: total,
		Page:  filter.Page,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospitals retrieved successfully",
		},
	})
}

// GetHospital godoc
// @Summary      Get hospital by ID
// @Description  Retrieve a hospital by its ID.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 400 invalid ID format
// @Description  - 404 Not Found
// @Tags         Hospitals
// @Produce      json
// @Param        id path string true "Hospital ID"
// @Success      200 {object} dto.HospitalResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id} [get]
func (h *HospitalHandler) GetHospital(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid hospital ID"})
		return
	}

	hospital, err := h.hospitalUseCase.GetHospitalByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "Hospital not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.HospitalResponse{
		ID:           hospital.ID.String(),
		Name:         hospital.Name,
		TierLevel:    string(hospital.TierLevel),
		Region:       string(hospital.Region),
		IsActive:     hospital.IsActive,
		CreatedAt:    hospital.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    hospital.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospital details retrieved successfully",
		},
	})
}

// UpdateHospital godoc
// @Summary      Update a hospital
// @Description  Admin-only endpoint to update hospital information.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 404 Not Found
// @Tags         Hospitals
// @Accept       json
// @Produce      json
// @Param        id   path string               true "Hospital ID"
// @Param        body body UpdateHospitalRequest  true "Hospital update payload"
// @Success      200 {object} dto.HospitalResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id} [put]
func (h *HospitalHandler) UpdateHospital(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid hospital ID"})
		return
	}

	var req UpdateHospitalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	existing, err := h.hospitalUseCase.GetHospitalByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "Hospital not found",
		})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.TierLevel != nil {
		existing.TierLevel = *req.TierLevel
	}
	if req.Region != nil {
		existing.Region = entity.EthiopianRegion(*req.Region)
	}
	if req.Address != nil {
		existing.Address = req.Address
	}
	if req.ContactPhone != nil {
		existing.ContactPhone = req.ContactPhone
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.hospitalUseCase.UpdateHospital(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to update hospital",
		})
		return
	}

	c.JSON(http.StatusOK, dto.HospitalResponse{
		ID:           existing.ID.String(),
		Name:         existing.Name,
		TierLevel:    string(existing.TierLevel),
		Region:       string(existing.Region),
		IsActive:     existing.IsActive,
		CreatedAt:    existing.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    existing.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospital updated successfully",
		},
	})
}

// DeleteHospital godoc
// @Summary      Delete a hospital
// @Description  Admin-only endpoint to soft-delete a hospital.
// @Description  **Roles:** SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid ID format
// @Description  - 404 Not Found
// @Tags         Hospitals
// @Produce      json
// @Param        id path string true "Hospital ID"
// @Success      200 {object} dto.BaseResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id} [delete]
func (h *HospitalHandler) DeleteHospital(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "invalid hospital ID",
		})
		return
	}

	if err := h.hospitalUseCase.DeleteHospital(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "Hospital not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{
		Success: true,
		Message: "Hospital deleted successfully",
	})
}
