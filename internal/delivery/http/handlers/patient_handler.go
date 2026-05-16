package handlers

import (
	"net/http"
	"strings"
	"log"

	"github.com/gin-gonic/gin"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
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
// @Description  Lookup patient data by plain text National ID.
// @Description  **Roles:** REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid format
// @Description  - 404 Not Found
// @Tags         Patients
// @Produce      json
// @Param        id path string true "National ID" default(NAT-12345)
// @Success      200 {object} dto.PatientResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/patients/national-id/{id} [get]
func (h *PatientHandler) GetByNationalID(c *gin.Context) {
	nationalID := strings.TrimSpace(c.Param("id"))
	if nationalID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "national ID is required",
		})
		return
	}

	patient, err := h.patientUC.GetByNationalID(c.Request.Context(), nationalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to look up patient",
		})
		return
	}

	if patient == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "Patient not found",
		})
		return
	}

	c.JSON(http.StatusOK, mapPatientToResponse(patient, "Patient retrieved successfully"))
}

// LookupPatient godoc
// @Summary      Lookup patient
// @Description  Intelligent secure search by strictly providing National ID OR Phone Number.
// @Description  **Roles:** REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Description  **Common Errors:**
// @Description  - 400 invalid query parameters
// @Description  - 404 Not Found
// @Tags         Patients
// @Produce      json
// @Param        national_id query string false "National ID"
// @Param        phone_number query string false "Phone Number (E.164)"
// @Success      200 {object} dto.PatientResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/patients/lookup [get]
func (h *PatientHandler) LookupPatient(c *gin.Context) {
	nationalID := strings.TrimSpace(c.Query("national_id"))
	phone := strings.TrimSpace(c.Query("phone_number"))

	if nationalID == "" && phone == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "either national_id OR phone_number are required",
		})
		return
	}

	patient, err := h.patientUC.LookupPatient(c.Request.Context(), nationalID, phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if patient == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error:   "Patient not found",
		})
		return
	}

	c.JSON(http.StatusOK, mapPatientToResponse(patient, "Patient found successfully"))
}

// CreatePatient godoc
// @Summary      Create Patient
// @Description  Explicitly create a new patient record. Validates uniqueness covering National ID or Phone+Name.
// @Description  **Roles:** REFERRING_DOCTOR, RECEPTIONIST, SYSTEM_SUPER_ADMIN
// @Description  **Gatekeepers:** Duplicate check on (NationalID) or (Phone + FirstName).
// @Description  **Common Errors:**
// @Description  - 400 invalid input
// @Description  - 409 Conflict (patient already exists)
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        body body dto.CreatePatientRequest true "New patient payload"
// @Success      201 {object} dto.PatientResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse "Conflict - patient already exists"
// @Security     BearerAuth
// @Router       /api/v1/patients [post]
func (h *PatientHandler) CreatePatient(c *gin.Context) {
	var req dto.CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Error:   "Invalid request payload: " + err.Error(),
		})
		return
	}

	patient, err := h.patientUC.CreatePatient(c.Request.Context(), req)
	if err != nil {
		log.Printf("[PatientHandler.CreatePatient] error: %v", err)
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error:   "Failed to create patient: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, mapPatientToResponse(patient, "Patient record created successfully"))
}

func mapPatientToResponse(patient *entity.Patient, message string) dto.PatientResponse {
	dob := ""
	if patient.DateOfBirth != nil {
		dob = patient.DateOfBirth.Format("2006-01-02")
	}
	phone := patient.PhonePlain
	region := ""
	if patient.HomeRegion != nil {
		region = string(*patient.HomeRegion)
	}
	nid := patient.NationalIDPlain

	return dto.PatientResponse{
		ID:          patient.ID.String(),
		FirstName:   patient.FirstNamePlain,
		MiddleName:  patient.MiddleNamePlain,
		LastName:    patient.LastNamePlain,
		NationalID:  nid,
		Sex:         patient.Sex,
		DateOfBirth: dob,
		PhoneNumber: phone,
		HomeRegion:  region,
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: message,
		},
	}
}
