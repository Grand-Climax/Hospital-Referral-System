package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
)

type MockPatientUseCase struct {
	mock.Mock
}

func (m *MockPatientUseCase) GetByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error) {
	args := m.Called(ctx, nationalID)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) LookupOrCreate(ctx context.Context, req dto.LookupPatientRequest) (*entity.Patient, bool, error) {
	args := m.Called(ctx, req)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

func setupPatientRouter(mockUC *MockPatientUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewPatientHandler(mockUC)
	router := gin.Default()
	router.POST("/api/v1/patients/lookup", handler.LookupOrCreate)
	return router
}

func TestPatientLookupByNationalID(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	reqPayload := dto.LookupPatientRequest{
		NationalID: "NAT-SEED-001",
		LastName:   "Placeholder", // required
		Sex:        "male",        // required
	}

	mockPatient := &entity.Patient{
		ID:        uuid.New(),
		FirstName: "Abebe",
	}

	mockUC.On("LookupOrCreate", mock.Anything, mock.AnythingOfType("dto.LookupPatientRequest")).Return(mockPatient, false, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients/lookup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Abebe")
	assert.Contains(t, w.Body.String(), "Existing patient found")
}

func TestPatientAutoCreate(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	reqPayload := dto.LookupPatientRequest{
		PhoneNumber: "+251999999999",
		FirstName:   "New",
		LastName:    "Guy",
		Sex:         "male",
	}

	mockPatient := &entity.Patient{
		ID:          uuid.New(),
		FirstName:   "New",
		PhoneNumber: ptr("+251999999999"),
	}

	mockUC.On("LookupOrCreate", mock.Anything, mock.AnythingOfType("dto.LookupPatientRequest")).Return(mockPatient, true, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients/lookup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "New patient record created")
}

func TestPatientLookupMissingRequiredFields(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	// Missing LastName and Sex
	reqPayload := dto.LookupPatientRequest{
		PhoneNumber: "+251999999999",
		FirstName:   "New",
	}

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients/lookup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// DTO binding enforce LastName and Sex
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request payload")
}

func TestPatientLookupLogicErrors(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	reqPayload := dto.LookupPatientRequest{
		LastName: "Guy",
		Sex:      "male",
	}

	// Simulated Usecase rejecting because no lookup keys provided
	mockUC.On("LookupOrCreate", mock.Anything, mock.AnythingOfType("dto.LookupPatientRequest")).Return((*entity.Patient)(nil), false, errors.New("must provide either national_id OR"))

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients/lookup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "must provide either")
}

func ptr(s string) *string {
	return &s
}
