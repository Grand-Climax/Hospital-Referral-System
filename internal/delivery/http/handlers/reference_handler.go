package handlers

import (
	"net/http"
	"strconv"

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
// @Description  Returns all available ICD-10 codes with optional pagination, category filtering, and search matching. Used by doctors and specialists when filling in diagnoses.
// @Description  **Valid Chapter Categories:**
// @Description  - "Blood & Immune Disorders"
// @Description  - "Cancers & Tumors"
// @Description  - "Circulatory System Diseases"
// @Description  - "Conditions Originating in Perinatal Period"
// @Description  - "Congenital Malformations & Chromosomal Abnormalities"
// @Description  - "Digestive System Diseases"
// @Description  - "Ear & Mastoid Diseases"
// @Description  - "Endocrine, Nutritional & Metabolic Diseases"
// @Description  - "External Causes of Morbidity & Mortality"
// @Description  - "Eye & Adnexa Diseases"
// @Description  - "Genitourinary System Diseases"
// @Description  - "Infectious & Parasitic Diseases"
// @Description  - "Injury, Poisoning & External Causes"
// @Description  - "Mental & Behavioral Disorders"
// @Description  - "Musculoskeletal & Connective Tissue Diseases"
// @Description  - "Nervous System Diseases"
// @Description  - "Pregnancy, Childbirth & Puerperium"
// @Description  - "Respiratory System Diseases"
// @Description  - "Skin & Subcutaneous Tissue Diseases"
// @Description  - "Symptoms, Signs & Abnormal Findings"
// @Description  - "Unknown"
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Param        search query string false "Search by code or description"
// @Param        category query string false "Filter by category"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Number of items per page" default(30)
// @Success      200 {object} dto.PaginatedICDCodeResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/icd-codes [get]
func (h *ReferenceHandler) ListICDCodes(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "30"))

	codes, total, err := h.referenceUseCase.ListICDCodes(c.Request.Context(), search, category, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch ICD codes",
		})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedICDCodeResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "ICD codes retrieved successfully",
		},
		Data:     codes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListICDCategories godoc
// @Summary      List ICD-10 Categories
// @Description  Returns all unique chapter categories present in the ICD-10 dataset (with a robust static fallback).
// @Description  **Categories Included:**
// @Description  - "Blood & Immune Disorders"
// @Description  - "Cancers & Tumors"
// @Description  - "Circulatory System Diseases"
// @Description  - "Conditions Originating in Perinatal Period"
// @Description  - "Congenital Malformations & Chromosomal Abnormalities"
// @Description  - "Digestive System Diseases"
// @Description  - "Ear & Mastoid Diseases"
// @Description  - "Endocrine, Nutritional & Metabolic Diseases"
// @Description  - "External Causes of Morbidity & Mortality"
// @Description  - "Eye & Adnexa Diseases"
// @Description  - "Genitourinary System Diseases"
// @Description  - "Infectious & Parasitic Diseases"
// @Description  - "Injury, Poisoning & External Causes"
// @Description  - "Mental & Behavioral Disorders"
// @Description  - "Musculoskeletal & Connective Tissue Diseases"
// @Description  - "Nervous System Diseases"
// @Description  - "Pregnancy, Childbirth & Puerperium"
// @Description  - "Respiratory System Diseases"
// @Description  - "Skin & Subcutaneous Tissue Diseases"
// @Description  - "Symptoms, Signs & Abnormal Findings"
// @Description  - "Unknown"
// @Description  **Roles:** Any authenticated user.
// @Description  **Common Errors:**
// @Description  - 401 Unauthorized
// @Description  - 500 Internal Server Error
// @Tags         References
// @Produce      json
// @Success      200 {object} dto.RegionListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/icd-categories [get]
func (h *ReferenceHandler) ListICDCategories(c *gin.Context) {
	categories, err := h.referenceUseCase.ListICDCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch ICD categories",
		})
		return
	}

	c.JSON(http.StatusOK, dto.RegionListResponse{
		Data: categories,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "ICD categories retrieved successfully",
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

// GetRegions godoc
// @Summary      Get Ethiopian Regions List
// @Description  Returns all valid Ethiopian regions from the hardcoded enum. Accessible by all authenticated roles.
// @Description  **Roles:** Any authenticated user.
// @Tags         References
// @Produce      json
// @Success      200 {object} dto.RegionListResponse
// @Security     BearerAuth
// @Router       /api/v1/reference/regions [get]
func (h *ReferenceHandler) GetRegions(c *gin.Context) {
	regions := []string{
		"Addis Ababa",
		"Afar",
		"Amhara",
		"Benishangul-Gumuz",
		"Dire Dawa",
		"Gambela",
		"Harari",
		"Oromia",
		"Sidama",
		"Somali",
		"South Ethiopia",
		"South West Ethiopia",
		"Central Ethiopia",
		"Tigray",
	}

	c.JSON(http.StatusOK, dto.RegionListResponse{
		Data: regions,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Regions retrieved successfully",
		},
	})
}

