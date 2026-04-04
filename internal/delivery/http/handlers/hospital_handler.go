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

type HospitalResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TierLevel    string `json:"tier_level"`
	Region       string `json:"region"`
	Address      string `json:"address,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toHospitalResponse(h *entity.Hospital) HospitalResponse {
	resp := HospitalResponse{
		ID:        h.ID.String(),
		Name:      h.Name,
		TierLevel: string(h.TierLevel),
		Region:    h.Region,
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
// @Description  Admin-only endpoint to create a hospital
// @Tags         Hospitals
// @Accept       json
// @Produce      json
// @Param        body body CreateHospitalRequest true "Hospital creation payload"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals [post]
func (h *HospitalHandler) CreateHospital(c *gin.Context) {
	var req CreateHospitalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	hospital := &entity.Hospital{
		Name:         req.Name,
		TierLevel:    req.TierLevel,
		Region:       req.Region,
		Address:      req.Address,
		ContactPhone: req.ContactPhone,
	}

	if err := h.hospitalUseCase.CreateHospital(c.Request.Context(), hospital); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create hospital"})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessPayload(toHospitalResponse(hospital), "Hospital created successfully"))
}

// ListHospitals godoc
// @Summary      List hospitals
// @Description  List hospitals with optional filters
// @Tags         Hospitals
// @Produce      json
// @Param        page      query int    false "Page number" default(1)
// @Param        page_size query int    false "Page size"   default(20)
// @Param        tier      query string false "Filter by tier level"
// @Param        region    query string false "Filter by region"
// @Param        is_active query bool   false "Filter by active status"
// @Param        search    query string false "Search by name"
// @Success      200 {object} map[string]interface{}
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
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to list hospitals"})
		return
	}

	var resp []HospitalResponse
	for i := range hospitals {
		resp = append(resp, toHospitalResponse(&hospitals[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hospitals retrieved successfully",
		"data":    resp,
		"total":   total,
		"page":    filter.Page,
	})
}

// GetHospital godoc
// @Summary      Get hospital by ID
// @Description  Retrieve a hospital by its ID
// @Tags         Hospitals
// @Produce      json
// @Param        id path string true "Hospital ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id} [get]
func (h *HospitalHandler) GetHospital(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid hospital ID"})
		return
	}

	hospital, err := h.hospitalUseCase.GetHospitalByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Hospital not found"})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessPayload(toHospitalResponse(hospital), "Hospital details retrieved successfully"))
}

// UpdateHospital godoc
// @Summary      Update a hospital
// @Description  Admin-only endpoint to update hospital information
// @Tags         Hospitals
// @Accept       json
// @Produce      json
// @Param        id   path string               true "Hospital ID"
// @Param        body body UpdateHospitalRequest  true "Hospital update payload"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id} [put]
func (h *HospitalHandler) UpdateHospital(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid hospital ID"})
		return
	}

	var req UpdateHospitalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	existing, err := h.hospitalUseCase.GetHospitalByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Hospital not found"})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.TierLevel != nil {
		existing.TierLevel = *req.TierLevel
	}
	if req.Region != nil {
		existing.Region = *req.Region
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
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update hospital"})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessPayload(toHospitalResponse(existing), "Hospital updated successfully"))
}

// DeleteHospital godoc
// @Summary      Delete a hospital
// @Description  Admin-only endpoint to soft-delete a hospital
// @Tags         Hospitals
// @Produce      json
// @Param        id path string true "Hospital ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/hospitals/{id} [delete]
func (h *HospitalHandler) DeleteHospital(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid hospital ID"})
		return
	}

	if err := h.hospitalUseCase.DeleteHospital(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Hospital not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hospital deleted successfully",
	})
}
