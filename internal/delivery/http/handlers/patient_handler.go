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

// GetByPhoneAndName godoc
// @Summary      Get patient by Phone and First Name
// @Description  Fallback lookup using phone number and first name. Roles: REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Tags         Patients
// @Produce      json
// @Param        phone_number query string true "Phone Number (E.164)" default(+251911000002)
// @Param        first_name query string true "First Name" default(Liya)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/patients/lookup/phone [get]
func (h *PatientHandler) GetByPhoneAndName(c *gin.Context) {
	phone := strings.TrimSpace(c.Query("phone_number"))
	firstName := strings.TrimSpace(c.Query("first_name"))

	if phone == "" || firstName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone_number and first_name are required"})
		return
	}

	patient, err := h.patientUC.GetByPhoneAndName(c.Request.Context(), phone, firstName)
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

