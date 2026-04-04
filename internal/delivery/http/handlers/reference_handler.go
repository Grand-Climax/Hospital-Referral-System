package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
// @Tags         References
// @Produce      json
// @Param        tier query string false "Hospital Tier (PRIMARY, GENERAL, SPECIALIZED)"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/hospitals [get]
func (h *ReferenceHandler) GetHospitals(c *gin.Context) {
	tier := c.Query("tier")
	hospitals, err := h.referenceUseCase.GetHospitals(c.Request.Context(), tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch hospitals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hospitals retrieved successfully",
		"data":    hospitals,
	})
}

// GetDepartments godoc
// @Summary      Get Departments List
// @Description  Returns all global departments (not scoped to a hospital). Accessible by all authenticated roles.
// @Tags         References
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/departments [get]
func (h *ReferenceHandler) GetDepartments(c *gin.Context) {
	depts, err := h.referenceUseCase.GetDepartments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Departments retrieved successfully",
		"data":    depts,
	})
}

// ListICDCodes godoc
// @Summary      List all ICD-10 Codes
// @Description  Returns all available ICD-10 codes. Used by doctors and specialists when filling in diagnoses.
// @Tags         References
// @Produce      json
// @Param        search query string false "Search by code or description"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/icd-codes [get]
func (h *ReferenceHandler) ListICDCodes(c *gin.Context) {
	codes, err := h.referenceUseCase.ListICDCodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch ICD codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ICD codes retrieved successfully",
		"data":    codes,
	})
}

// GetNetworkedHospitals godoc
// @Summary      Get Networked Hospitals
// @Description  Returns hospitals in the referral network that can receive from the requesting hospital. Used by doctors/liaison when selecting a referral target. Accessible by all authenticated roles.
// @Tags         References
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/networked-hospitals [get]
func (h *ReferenceHandler) GetNetworkedHospitals(c *gin.Context) {
	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "No hospital assigned to user"})
		return
	}
	hospIDPtr, ok := hospIDVal.(*uuid.UUID)
	if !ok || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "No hospital assigned to user"})
		return
	}

	hospitals, err := h.referenceUseCase.GetNetworkedHospitals(c.Request.Context(), *hospIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch networked hospitals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Networked hospitals retrieved successfully",
		"data":    hospitals,
	})
}

// GetHospitalDepartments godoc
// @Summary      Get Hospital Departments
// @Description  Returns departments available at a specific target hospital. Used by doctors when selecting a department to refer to. Accessible by all authenticated roles.
// @Tags         References
// @Produce      json
// @Param        id path string true "Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/hospitals/{id}/departments [get]
func (h *ReferenceHandler) GetHospitalDepartments(c *gin.Context) {
	hospitalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid target hospital ID format"})
		return
	}

	depts, err := h.referenceUseCase.GetHospitalDepartments(c.Request.Context(), hospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch hospital departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hospital departments retrieved successfully",
		"data":    depts,
	})
}

// GetLiaisons godoc
// @Summary      Get Liaisons for Current Hospital
// @Description  Returns all active liaison officers belonging to the authenticated user's hospital. Hospital ID is extracted from the JWT token.
// @Tags         References
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/reference/liaisons [get]
func (h *ReferenceHandler) GetLiaisons(c *gin.Context) {
	hospIDVal, exists := c.Get("hospID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "No hospital assigned to user"})
		return
	}
	hospIDPtr, ok := hospIDVal.(*uuid.UUID)
	if !ok || hospIDPtr == nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "No hospital assigned to user"})
		return
	}

	liaisons, err := h.referenceUseCase.GetLiaisonsByHospital(c.Request.Context(), *hospIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch liaison officers"})
		return
	}

	type liaisonItem struct {
		ID        string `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}
	var result []liaisonItem
	for _, l := range liaisons {
		result = append(result, liaisonItem{
			ID:        l.ID.String(),
			FirstName: l.FirstName,
			LastName:  l.LastName,
			Email:     l.Email,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Liaisons retrieved successfully",
		"data":    result,
	})
}
