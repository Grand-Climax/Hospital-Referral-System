package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
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

// LookupOrCreate godoc
// @Summary      Lookup or Create Patient
// @Description  Try to find a patient by National ID. If not found/provided, fallback to Phone + First Name. If still not found, auto-creates a new patient record. Roles: REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        body body dto.LookupPatientRequest true "Patient lookup/create payload"
// @Success      200 {object} map[string]interface{} "Existing patient found"
// @Success      201 {object} map[string]interface{} "New patient record created"
// @Failure      400 {object} map[string]string "Validation error"
// @Failure      500 {object} map[string]string "Server error"
// @Security     BearerAuth
// @Router       /api/v1/patients/lookup [post]
func (h *PatientHandler) LookupOrCreate(c *gin.Context) {
	var req dto.LookupPatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	patient, isNew, err := h.patientUC.LookupOrCreate(c.Request.Context(), req)
	if err != nil {
		// Differentiate validation errors from actual server faults if needed
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "provide either") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to look up or create patient"})
		return
	}

	if isNew {
		c.JSON(http.StatusCreated, gin.H{
			"message": "New patient record created",
			"data":    patient,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Existing patient found",
		"data":    patient,
	})
}

