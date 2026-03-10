package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/usecase"
)

type PatientHandler struct {
	patientUC usecase.PatientUseCase
}

func NewPatientHandler(patientUC usecase.PatientUseCase) *PatientHandler {
	return &PatientHandler{
		patientUC: patientUC,
	}
}

// GetByNationalID godoc
// @Summary      Get patient by National ID
// @Description  Lookup patient data by plain text National ID
// @Tags         Patients
// @Produce      json
// @Param        id path string true "National ID" default(NAT-12345)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/patients/national-id/{id} [get]
func (h *PatientHandler) GetByNationalID(c *gin.Context) {
	nationalID := strings.TrimSpace(c.Param("id"))
	if nationalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "national ID is required"})
		return
	}

	patient, err := h.patientUC.GetByNationalID(c.Request.Context(), nationalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to look up patient"})
		return
	}

	if patient == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Patient not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": patient})
}
