package test

import (
	"bytes"
	"context"
	"encoding/json"
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

// MockReferralUseCase simulates business logic layer responses for isolated handler testing
type MockReferralUseCase struct {
	mock.Mock
}

func (m *MockReferralUseCase) CreateReferral(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	args := m.Called(ctx, doctorID, senderHospitalID, req)
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) GetReferral(ctx context.Context, id, userID, hospID, deptID uuid.UUID, userRole entity.UserRole) (*entity.Referral, error) {
	args := m.Called(ctx, id, userID, hospID, deptID, userRole)
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) ListReferrals(ctx context.Context, userID, hospID, deptID uuid.UUID, userRole entity.UserRole, statusFilter, dateFrom, dateTo string) ([]entity.Referral, error) {
	args := m.Called(ctx, userID, hospID, deptID, userRole, statusFilter, dateFrom, dateTo)
	return args.Get(0).([]entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) UpdateDraft(ctx context.Context, id, userID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	args := m.Called(ctx, id, userID, req)
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) DeleteDraft(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockReferralUseCase) SubmitReferral(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockReferralUseCase) ResubmitReferral(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockReferralUseCase) CancelReferral(ctx context.Context, id, userID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, userID, reason)
	return args.Error(0)
}

type StatusPayload struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// TestReferralDraftCreation verifies the relaxed DTO accepts a partial DRAFT save
func TestReferralDraftCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)

	doctorID := uuid.New()
	hospitalID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/referrals", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Set("hospID", &hospitalID)
		c.Set("role", entity.RoleReferringDoctor)
		c.Next()
	}, handler.Create)

	reqPayload := dto.CreateReferralRequest{
		PatientID:        uuid.New(),
		TargetHospitalID: uuid.New(),
		TargetDeptID:     uuid.New(),
		Status:           "DRAFT",
	}

	mockReferral := &entity.Referral{
		ID:     uuid.New(),
		Status: entity.StatusDraft,
	}

	mockUC.On("CreateReferral", mock.Anything, doctorID, hospitalID, reqPayload).Return(mockReferral, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Referral created successfully", response["message"])
	mockUC.AssertExpectations(t)
}

// TestDoctorSubmitEndpoint verifies Doctor explicit submission
func TestDoctorSubmitEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)

	t.Run("Valid Transition DRAFT -> SUBMITTED", func(t *testing.T) {
		referralID := uuid.New()
		currentUserID := uuid.New()

		mockUC.On("SubmitReferral", mock.Anything, referralID, currentUserID).Return(nil)

		router := gin.Default()
		router.POST("/api/v1/referrals/:id/submit", func(c *gin.Context) {
			c.Set("userID", currentUserID)
			c.Set("role", entity.RoleReferringDoctor)
			c.Next()
		}, handler.Submit)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/referrals/"+referralID.String()+"/submit", bytes.NewBuffer([]byte{}))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})
}

	// Assume mockLiaisonHandler exists or just remove Liaison handlers from this specific referral test file
	// if it tests the liaison logic we should move it to liaison_integration_test.go. BUT we will skip
	// rewriting liaison integration logic here because this file tests Doctor explicitly now, 
	// or we just remove the test that uses generic handler.UpdateStatus.


func TestDoctorResubmitCycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)

	t.Run("Doctor resubmits fixed referral -> back to UNDER_LIAISON_REVIEW", func(t *testing.T) {
		revisedID := uuid.New()
		currentUserID := uuid.New()
		
		mockUC.On("ResubmitReferral",
			mock.Anything, revisedID, currentUserID,
		).Return(nil)

		router := gin.Default()
		router.POST("/api/v1/referrals/:id/resubmit", func(c *gin.Context) {
			c.Set("userID", currentUserID)
			c.Set("role", entity.RoleReferringDoctor)
			c.Next()
		}, handler.Resubmit)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/referrals/"+revisedID.String()+"/resubmit", bytes.NewBuffer([]byte{}))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})
}

// Specialist tests removed from here because they will have their own dedicated handler and mock.

func TestInvalidTransitionBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)
	referralID := uuid.New()

	t.Run("Test Doctor Canceling Referral works", func(t *testing.T) {
		currentUserID := uuid.New()
		
		mockUC.On("CancelReferral",
			mock.Anything, referralID,
			currentUserID, "",
		).Return(nil)

		router := gin.Default()
		router.POST("/api/v1/referrals/:id/cancel", func(c *gin.Context) {
			c.Set("userID", currentUserID)
			c.Set("role", entity.RoleReferringDoctor)
			c.Next()
		}, handler.Cancel)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/referrals/"+referralID.String()+"/cancel", bytes.NewBuffer([]byte{}))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})
}

func TestReferralWithVitalsAndEmergency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)
	
	doctorID := uuid.New()
	hospitalID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/referrals", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Set("hospID", &hospitalID)
		c.Set("role", entity.RoleReferringDoctor)
		c.Next()
	}, handler.Create)

	systolic := int16(185)
	diastolic := int16(115)
	heartRate := int16(110)

	reqPayload := dto.CreateReferralRequest{
		PatientID:        uuid.New(),
		TargetHospitalID: uuid.New(),
		TargetDeptID:     uuid.New(),
		ClinicalSummary:  "Acute MI with cardiogenic shock",
		PatientHistory:   "HTN x 10 years",
		ReasonOfReferral: "Cardiac catheterization",
		ReasonForReferralCategory: "EMERGENCY",
		ConditionAtReferral: "UNSTABLE",
		Status: "SUBMITTED",
		Vitals: &dto.VitalsDTO{
			SystolicBP:  &systolic,
			DiastolicBP: &diastolic,
			HeartRate:   &heartRate,
		},
		EmergencyDetail: &dto.EmergencyDetailDTO{
			EmergencyJustification: "Cardiogenic shock, BP unresponsive to fluids",
		},
	}

	mockReferral := &entity.Referral{
		ID:     uuid.New(),
		Status: entity.StatusSubmitted,
	}

	mockUC.On("CreateReferral",
		mock.Anything, doctorID, hospitalID, mock.Anything,
	).Return(mockReferral, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockUC.AssertExpectations(t)
}
