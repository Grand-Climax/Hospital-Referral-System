package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type PatientHandler struct {
	patientUC iusecase.PatientUseCase
}

func NewPatientHandler(patientUC iusecase.PatientUseCase) *PatientHandler {
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

// LookupPatient godoc
// @Summary      Lookup patient
// @Description  Intelligent secure search by strictly providing National ID OR (Phone + First Name). Roles: REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Tags         Patients
// @Produce      json
// @Param        national_id query string false "National ID"
// @Param        phone_number query string false "Phone Number (E.164)"
// @Param        first_name query string false "First Name"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/patients/lookup [get]
func (h *PatientHandler) LookupPatient(c *gin.Context) {
	nationalID := strings.TrimSpace(c.Query("national_id"))
	phone := strings.TrimSpace(c.Query("phone_number"))
	firstName := strings.TrimSpace(c.Query("first_name"))

	if nationalID == "" && (phone == "" || firstName == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either national_id OR both phone_number and first_name are required"})
		return
	}

	patient, err := h.patientUC.LookupPatient(c.Request.Context(), nationalID, phone, firstName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if patient == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Patient not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": patient})
}

// CreatePatient godoc
// @Summary      Create Patient
// @Description  Explicitly create a new patient record. Validates uniqueness covering National ID or Phone+Name. Roles: REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        body body dto.CreatePatientRequest true "New patient payload"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string "Conflict - patient already exists"
// @Security     BearerAuth
// @Router       /api/v1/patients [post]
func (h *PatientHandler) CreatePatient(c *gin.Context) {
	var req dto.CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	patient, err := h.patientUC.CreatePatient(c.Request.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create patient"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Patient created successfully",
		"data":    patient,
	})
}

