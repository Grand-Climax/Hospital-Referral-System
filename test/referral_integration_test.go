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

func (m *MockReferralUseCase) GetReferral(ctx context.Context, id uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) ListReferrals(ctx context.Context, userRole entity.UserRole, userHospitalID uuid.UUID, statusFilter, dateFrom, dateTo string) ([]entity.Referral, error) {
	args := m.Called(ctx, userRole, userHospitalID, statusFilter, dateFrom, dateTo)
	return args.Get(0).([]entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) UpdateDraft(ctx context.Context, id uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(*entity.Referral), args.Error(1)
}

func (m *MockReferralUseCase) DeleteDraft(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReferralUseCase) UpdateReferralStatus(ctx context.Context, id uuid.UUID, newStatus entity.ReferralStatus, userID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, newStatus, userID, reason)
	return args.Error(0)
}

// TestReferralDraftCreation verifies the relaxed DTO accepts a partial DRAFT save
func TestReferralDraftCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)

	router := gin.Default()
	router.POST("/api/v1/referrals", handler.Create)

	// Simulated incoming payload (partial form allowed for drafts)
	reqPayload := dto.CreateReferralRequest{
		FirstName:        "Test",
		LastName:         "Patient",
		TargetHospitalID: uuid.New(),
		TargetDeptID:     uuid.New(),
		Status:           "DRAFT", // Signals partial
	}

	mockReferral := &entity.Referral{
		ID:     uuid.New(),
		Status: entity.StatusDraft,
	}

	mockUC.On("CreateReferral", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.Anything).Return(mockReferral, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Inject required context headers
	req.Header.Set("X-Doctor-ID", uuid.New().String())
	req.Header.Set("X-Hospital-ID", uuid.New().String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Referral created successfully", response["message"])
	mockUC.AssertExpectations(t)
}

// TestValidTransitions blocks incorrect status hops
func TestValidTransitionsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReferralHandler(mockUC)

	router := gin.Default()
	router.PATCH("/api/v1/referrals/:id/status", handler.UpdateStatus)

	referralID := uuid.New()

	type StatusPayload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	
	// Test transitioning Draft -> Submitted appropriately
	t.Run("Valid Transition DRAFT -> SUBMITTED", func(t *testing.T) {
		payload := StatusPayload{Status: "SUBMITTED"}
		mockUC.On("UpdateReferralStatus", mock.Anything, referralID, entity.StatusSubmitted, mock.AnythingOfType("uuid.UUID"), "").Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/referrals/"+referralID.String()+"/status", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Doctor-ID", uuid.New().String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
