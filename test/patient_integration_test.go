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

func (m *MockPatientUseCase) LookupPatient(ctx context.Context, nationalID, phone string) (*entity.Patient, error) {
	args := m.Called(ctx, nationalID, phone)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) CreatePatient(ctx context.Context, req dto.CreatePatientRequest) (*entity.Patient, error) {
	args := m.Called(ctx, req)
	if patient := args.Get(0); patient != nil {
		return patient.(*entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) SearchPatients(ctx context.Context, query string) ([]entity.Patient, error) {
	args := m.Called(ctx, query)
	if patients := args.Get(0); patients != nil {
		return patients.([]entity.Patient), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPatientUseCase) LookupByNationalID(ctx context.Context, nationalID string) (*uuid.UUID, error) {
	args := m.Called(ctx, nationalID)
	if id := args.Get(0); id != nil {
		return id.(*uuid.UUID), args.Error(1)
	}
	return nil, args.Error(1)
}


func setupPatientRouter(mockUC *MockPatientUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewPatientHandler(mockUC)
	router := gin.Default()
	router.GET("/api/v1/patients/lookup", handler.LookupPatient)
	router.POST("/api/v1/patients", handler.CreatePatient)
	return router
}

func TestPatientLookupByPhoneAndName_Found(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	mockPatient := &entity.Patient{
		ID:             uuid.New(),
		FirstNamePlain: "Liya",
	}

	phone := "+251911000002"
	mockUC.On("LookupPatient", mock.Anything, "", phone).Return(mockPatient, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/patients/lookup?phone_number=%2B251911000002", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Liya")
}

func TestPatientLookup_MissingParams(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/patients/lookup", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "required")
}

func TestCreatePatient_Success(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	reqPayload := dto.CreatePatientRequest{
		PhoneNumber: "+251999999999",
		FirstName:   "New",
		LastName:    "Guy",
		Sex:         "male",
	}

	mockPatient := &entity.Patient{
		ID:             uuid.New(),
		FirstNamePlain: "New",
		PhonePlain:     "+251999999999",
	}

	mockUC.On("CreatePatient", mock.Anything, mock.AnythingOfType("dto.CreatePatientRequest")).Return(mockPatient, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Patient record created successfully")
}

func TestCreatePatient_DuplicateConflict(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	reqPayload := dto.CreatePatientRequest{
		PhoneNumber: "+251911000001",
		FirstName:   "Abebe",
		LastName:    "Kebede",
		Sex:         "male",
	}

	// Simulated Usecase rejecting because patient exists
	mockUC.On("CreatePatient", mock.Anything, mock.AnythingOfType("dto.CreatePatientRequest")).Return((*entity.Patient)(nil), errors.New("already exists"))

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already exists")
}

func TestCreatePatient_MissingRequiredFields(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	// Missing LastName and Sex
	reqPayload := dto.CreatePatientRequest{
		PhoneNumber: "+251999999999",
		FirstName:   "New",
	}

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// DTO binding enforce LastName and Sex
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request payload")
}

func ptr(s string) *string {
	return &s
}
