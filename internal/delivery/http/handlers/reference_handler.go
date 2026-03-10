package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/usecase"
)

type ReferenceHandler struct {
	referenceUseCase usecase.ReferenceUseCase
}

func NewReferenceHandler(uc usecase.ReferenceUseCase) *ReferenceHandler {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hospitals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hospitals})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": depts})
}

// SearchICD godoc
// @Summary      Search ICD-10 Codes
// @Description  Search ICD-10 codes by keyword. Used by doctors and specialists when filling in diagnoses. Accessible by all authenticated roles.
// @Tags         References
// @Produce      json
// @Param        q query string false "Search query (e.g. Cholera)"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/icd-codes [get]
func (h *ReferenceHandler) SearchICD(c *gin.Context) {
	q := c.Query("q")
	codes, err := h.referenceUseCase.SearchICDCodes(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search ICD codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": codes})
}

// GetNetworkedHospitals godoc
// @Summary      Get Networked Hospitals
// @Description  Returns hospitals in the referral network that can receive from the requesting hospital. Used by doctors/liaison when selecting a referral target. Accessible by all authenticated roles.
// @Tags         References
// @Produce      json
// @Param        X-Hospital-ID header string true "Sender Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/reference/networked-hospitals [get]
func (h *ReferenceHandler) GetNetworkedHospitals(c *gin.Context) {
	senderHospitalID, err := uuid.Parse(c.GetHeader("X-Hospital-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid requesting hospital ID"})
		return
	}

	hospitals, err := h.referenceUseCase.GetNetworkedHospitals(c.Request.Context(), senderHospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch networked hospitals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hospitals})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target hospital ID format"})
		return
	}

	depts, err := h.referenceUseCase.GetHospitalDepartments(c.Request.Context(), hospitalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hospital departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": depts})
}
