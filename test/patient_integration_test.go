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

func (m *MockPatientUseCase) GetByPhoneAndName(ctx context.Context, phone, firstName string) (*entity.Patient, error) {
	args := m.Called(ctx, phone, firstName)
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


func setupPatientRouter(mockUC *MockPatientUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewPatientHandler(mockUC)
	router := gin.Default()
	router.GET("/api/v1/patients/lookup/phone", handler.GetByPhoneAndName)
	router.POST("/api/v1/patients", handler.CreatePatient)
	return router
}

func TestPatientLookupByPhoneAndName_Found(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	mockPatient := &entity.Patient{
		ID:        uuid.New(),
		FirstName: "Liya",
	}

	mockUC.On("GetByPhoneAndName", mock.Anything, "+251911000002", "Liya").Return(mockPatient, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/patients/lookup/phone?phone_number=%2B251911000002&first_name=Liya", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Liya")
}

func TestPatientLookupByPhoneAndName_MissingParams(t *testing.T) {
	mockUC := new(MockPatientUseCase)
	router := setupPatientRouter(mockUC)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/patients/lookup/phone?phone_number=%2B251911000002", nil)
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
		ID:          uuid.New(),
		FirstName:   "New",
		PhoneNumber: ptr("+251999999999"),
	}

	mockUC.On("CreatePatient", mock.Anything, mock.AnythingOfType("dto.CreatePatientRequest")).Return(mockPatient, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/patients", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Patient created successfully")
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
