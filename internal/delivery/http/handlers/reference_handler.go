package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ReferenceHandler struct {
	referenceUseCase iusecase.ReferenceUseCase
}

func NewReferenceHandler(uc iusecase.ReferenceUseCase) *ReferenceHandler {
	return &ReferenceHandler{referenceUseCase: uc}
}

// GetHospitals godoc
// @Summary      Get Hospitals List
// @Description  Returns all hospitals, optionally filtered by tier. Accessible by all authenticated roles.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Param        tier query string false "Hospital Tier (PRIMARY, GENERAL, SPECIALIZED)"
// @Success      200 {object} dto.HospitalListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/hospitals [get]
func (h *ReferenceHandler) GetHospitals(c *gin.Context) {
	tier := c.Query("tier")
	hospitals, err := h.referenceUseCase.GetHospitals(c.Request.Context(), tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch hospitals",
		})
		return
	}

	var data []dto.HospitalResponse
	for _, hosp := range hospitals {
		addr := ""
		if hosp.Address != nil {
			addr = *hosp.Address
		}
		phone := ""
		if hosp.ContactPhone != nil {
			phone = *hosp.ContactPhone
		}
		data = append(data, dto.HospitalResponse{
			ID:           hosp.ID.String(),
			Name:         hosp.Name,
			TierLevel:    string(hosp.TierLevel),
			Address:      addr,
			ContactPhone: phone,
			Region:       string(hosp.Region),
			IsActive:     hosp.IsActive,
			CreatedAt:    hosp.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.HospitalListResponse{
		Data: data,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospitals retrieved successfully",
		},
	})
}

// GetDepartments godoc
// @Summary      Get Departments List
// @Description  Returns all global departments (not scoped to a hospital). Accessible by all authenticated roles.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Success      200 {object} dto.DepartmentListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/departments [get]
func (h *ReferenceHandler) GetDepartments(c *gin.Context) {
	depts, err := h.referenceUseCase.GetDepartments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch departments",
		})
		return
	}

	var data []dto.DepartmentResponse
	for _, d := range depts {
		desc := ""
		if d.Description != nil {
			desc = *d.Description
		}
		data = append(data, dto.DepartmentResponse{
			ID:          d.ID.String(),
			Name:        d.Name,
			Description: desc,
			CreatedAt:   d.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.DepartmentListResponse{
		Data: data,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Departments retrieved successfully",
		},
	})
}

// ListICDCodes godoc
// @Summary      List all ICD-10 Codes
// @Description  Returns all available ICD-10 codes. Used by doctors and specialists when filling in diagnoses.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Param        search query string false "Search by code or description"
// @Success      200 {object} dto.ICDCodeListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/icd-codes [get]
func (h *ReferenceHandler) ListICDCodes(c *gin.Context) {
	codes, err := h.referenceUseCase.ListICDCodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch ICD codes",
		})
		return
	}
	c.JSON(http.StatusOK, dto.ICDCodeListResponse{
		Data: codes,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "ICD codes retrieved successfully",
		},
	})
}

// GetNetworkedHospitals godoc
// @Summary      Get Networked Hospitals
// @Description  Returns hospitals in the referral network that can receive from the requesting hospital. Used by doctors/liaison when selecting a referral target.
// @Description  **Roles:** Any authenticated user with a hospital scope.
// @Description  **Prerequisites:** Authenticated user must belong to a sender hospital.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (no hospital scope)
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Success      200 {object} dto.HospitalListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/networked-hospitals [get]
func (h *ReferenceHandler) GetNetworkedHospitals(c *gin.Context) {
	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   "No hospital assigned to user",
		})
		return
	}
	hospIDPtr, ok := hospIDVal.(*uuid.UUID)
	if !ok || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   "No hospital assigned to user",
		})
		return
	}

	hospitals, err := h.referenceUseCase.GetNetworkedHospitals(c.Request.Context(), *hospIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch networked hospitals",
		})
		return
	}

	var data []dto.HospitalResponse
	for _, hosp := range hospitals {
		addr := ""
		if hosp.Address != nil {
			addr = *hosp.Address
		}
		phone := ""
		if hosp.ContactPhone != nil {
			phone = *hosp.ContactPhone
		}
		data = append(data, dto.HospitalResponse{
			ID:           hosp.ID.String(),
			Name:         hosp.Name,
			TierLevel:    string(hosp.TierLevel),
			Address:      addr,
			ContactPhone: phone,
			Region:       string(hosp.Region),
			IsActive:     hosp.IsActive,
			CreatedAt:    hosp.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.HospitalListResponse{
		Data: data,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Networked hospitals retrieved successfully",
		},
	})
}

// GetHospitalDepartments godoc
// @Summary      Get Hospital Departments
// @Description  Returns departments available at a specific target hospital. Used by doctors when selecting a department to refer to.
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 400 Invalid hospital ID
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Param        id path string true "Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Success      200 {object} dto.DepartmentListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/hospitals/{id}/departments [get]
func (h *ReferenceHandler) GetHospitalDepartments(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid target hospital ID format",
		})
		return
	}

	depts, err := h.referenceUseCase.GetHospitalDepartments(c.Request.Context(), hospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch hospital departments",
		})
		return
	}

	var data []dto.DepartmentResponse
	for _, d := range depts {
		desc := ""
		if d.Description != nil {
			desc = *d.Description
		}
		data = append(data, dto.DepartmentResponse{
			ID:          d.ID.String(),
			Name:        d.Name,
			Description: desc,
			CreatedAt:   d.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.DepartmentListResponse{
		Data: data,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Hospital departments retrieved successfully",
		},
	})
}

// GetLiaisons godoc
// @Summary      Get Liaisons for Current Hospital
// @Description  Returns all active liaison officers belonging to the authenticated user's hospital. Hospital ID is extracted from the JWT token.
// @Description  **Roles:** Any authenticated user with a hospital scope.
// @Description  **Prerequisites:** Authenticated user must belong to a hospital.
// @Description  **Common Errors:**
// @Description  - 403 Forbidden (no hospital scope)
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Success      200 {object} dto.LiaisonListResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/liaisons [get]
func (h *ReferenceHandler) GetLiaisons(c *gin.Context) {
	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   "No hospital assigned to user",
		})
		return
	}
	hospIDPtr, ok := hospIDVal.(*uuid.UUID)
	if !ok || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Error:   "No hospital assigned to user",
		})
		return
	}

	liaisons, err := h.referenceUseCase.GetLiaisonsByHospital(c.Request.Context(), *hospIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch liaison officers",
		})
		return
	}

	var result []dto.LiaisonItem
	for _, l := range liaisons {
		result = append(result, dto.LiaisonItem{
			ID:        l.ID.String(),
			FirstName: l.FirstName,
			LastName:  l.LastName,
			Email:     l.Email,
		})
	}
	c.JSON(http.StatusOK, dto.LiaisonListResponse{
		Data: result,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Liaisons retrieved successfully",
		},
	})
}
