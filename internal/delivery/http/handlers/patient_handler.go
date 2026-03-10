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

// GetByNationalID looks up a patient by their unencrypted National ID string.
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
