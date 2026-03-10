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
// @Description  Lookup hospitals with optional tier filter
// @Tags         References
// @Produce      json
// @Param        tier query string false "Hospital Tier"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/references/hospitals [get]
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
// @Description  Lookup all global departments
// @Tags         References
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/references/departments [get]
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
// @Description  Lookup ICD codes by string query
// @Tags         References
// @Produce      json
// @Param        q query string false "Search query"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/references/icd [get]
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
// @Description  Lookup networked hospitals based on requesting hospital ID
// @Tags         References
// @Produce      json
// @Param        X-Hospital-ID header string true "Sender Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/references/networked-hospitals [get]
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
// @Description  Lookup specific departments mapped to a hospital
// @Tags         References
// @Produce      json
// @Param        id path string true "Hospital ID" default(62af3d82-52ce-4e8f-af29-2c5e509e1e24)
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /api/v1/references/hospitals/{id}/departments [get]
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
