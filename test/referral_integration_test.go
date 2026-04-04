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

// --- Doctor Actions ---
func (m *MockReferralUseCase) CreateDraftOrSubmit(ctx context.Context, doctorID uuid.UUID, senderHospitalID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	args := m.Called(ctx, doctorID, senderHospitalID, req)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReferralUseCase) ListForDoctor(ctx context.Context, doctorID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, doctorID, limit, offset, statusFilter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}
func (m *MockReferralUseCase) GetDetailsForDoctor(ctx context.Context, id, doctorID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, doctorID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReferralUseCase) UpdateAndResubmit(ctx context.Context, id, doctorID uuid.UUID, req dto.CreateReferralRequest) (*entity.Referral, error) {
	args := m.Called(ctx, id, doctorID, req)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReferralUseCase) CancelReferral(ctx context.Context, id, doctorID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, doctorID, reason)
	return args.Error(0)
}

// --- Liaison Actions ---
func (m *MockReferralUseCase) ListForLiaison(ctx context.Context, hospID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, limit, offset, statusFilter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}
func (m *MockReferralUseCase) GetDetailsForLiaison(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, hospID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReferralUseCase) LiaisonRead(ctx context.Context, id, liaisonID, hospID uuid.UUID) error {
	args := m.Called(ctx, id, liaisonID, hospID)
	return args.Error(0)
}
func (m *MockReferralUseCase) LiaisonForward(ctx context.Context, id, liaisonID, hospID uuid.UUID, comment string) error {
	args := m.Called(ctx, id, liaisonID, hospID, comment)
	return args.Error(0)
}
func (m *MockReferralUseCase) LiaisonReject(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, liaisonID, hospID, reason)
	return args.Error(0)
}
func (m *MockReferralUseCase) LiaisonRevise(ctx context.Context, id, liaisonID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, liaisonID, hospID, reason)
	return args.Error(0)
}

// --- Specialist Actions ---
func (m *MockReferralUseCase) ListForSpecialist(ctx context.Context, hospID, specialistID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, specialistID, limit, offset, statusFilter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}
func (m *MockReferralUseCase) GetDetailsForSpecialist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, hospID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReferralUseCase) SpecialistRead(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	args := m.Called(ctx, id, specialistID, hospID)
	return args.Error(0)
}
func (m *MockReferralUseCase) SpecialistAccept(ctx context.Context, id, specialistID, hospID uuid.UUID, severityScore *float64) error {
	args := m.Called(ctx, id, specialistID, hospID, severityScore)
	return args.Error(0)
}
func (m *MockReferralUseCase) SpecialistReject(ctx context.Context, id, specialistID, hospID uuid.UUID, reason string) error {
	args := m.Called(ctx, id, specialistID, hospID, reason)
	return args.Error(0)
}
func (m *MockReferralUseCase) SpecialistRerunML(ctx context.Context, id, specialistID, hospID uuid.UUID) error {
	args := m.Called(ctx, id, specialistID, hospID)
	return args.Error(0)
}

// --- Receptionist Actions ---
func (m *MockReferralUseCase) ListForReceptionist(ctx context.Context, hospID uuid.UUID, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, hospID, limit, offset, statusFilter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}
func (m *MockReferralUseCase) GetDetailsForReceptionist(ctx context.Context, id, hospID uuid.UUID) (*entity.Referral, error) {
	args := m.Called(ctx, id, hospID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Referral), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReferralUseCase) ConfirmAttendance(ctx context.Context, id, receptionistID, hospID uuid.UUID, status string) error {
	args := m.Called(ctx, id, receptionistID, hospID, status)
	return args.Error(0)
}

// --- Admin Actions ---
func (m *MockReferralUseCase) ListForSystemAdmin(ctx context.Context, limit, offset int, statusFilter string) ([]entity.Referral, int64, error) {
	args := m.Called(ctx, limit, offset, statusFilter)
	return args.Get(0).([]entity.Referral), args.Get(1).(int64), args.Error(2)
}
func (m *MockReferralUseCase) GetHospitalLogsForAdmin(ctx context.Context, hospID uuid.UUID, limit, offset int) ([]entity.ReferralStatusHistory, int64, error) {
	args := m.Called(ctx, hospID, limit, offset)
	return args.Get(0).([]entity.ReferralStatusHistory), args.Get(1).(int64), args.Error(2)
}

// --- DOCTOR TESTS ---
func TestDoctorOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewDoctorHandler(mockUC)
	doctorID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/doctor/referrals", func(c *gin.Context) {
		c.Set("userID", doctorID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.CreateOrSubmit)

	t.Run("Create Draft", func(t *testing.T) {
		liaisonID := uuid.New()
		reqPayload := dto.CreateReferralRequest{
			Status: "DRAFT", 
			PatientID: uuid.New(),
			TargetHospitalID: uuid.New(),
			TargetDeptID: uuid.New(),
			LiaisonOfficerID: &liaisonID,
		}
		mockUC.On("CreateDraftOrSubmit", mock.Anything, doctorID, hospID, reqPayload).Return(&entity.Referral{ID: uuid.New()}, nil)

		body, _ := json.Marshal(reqPayload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/doctor/referrals", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

// --- LIAISON TESTS ---
func TestLiaisonOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewLiaisonHandler(mockUC)
	liaisonID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/liaison/referrals/:id/forward", func(c *gin.Context) {
		c.Set("userID", liaisonID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.Forward)

	t.Run("Forward Referral", func(t *testing.T) {
		refID := uuid.New()
		mockUC.On("LiaisonForward", mock.Anything, refID, liaisonID, hospID, "").Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/liaison/referrals/"+refID.String()+"/forward", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- SPECIALIST TESTS ---
func TestSpecialistOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewSpecialistHandler(mockUC)
	specialistID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/specialist/referrals/:id/accept", func(c *gin.Context) {
		c.Set("userID", specialistID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.Accept)

	t.Run("Accept Referral", func(t *testing.T) {
		refID := uuid.New()
		mockUC.On("SpecialistAccept", mock.Anything, refID, specialistID, hospID, (*float64)(nil)).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/specialist/referrals/"+refID.String()+"/accept", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- RECEPTIONIST TESTS ---
func TestReceptionistOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewReceptionistHandler(mockUC)
	receptionistID := uuid.New()
	hospID := uuid.New()

	router := gin.Default()
	router.POST("/api/v1/receptionist/referrals/:id/confirm-attendance", func(c *gin.Context) {
		c.Set("userID", receptionistID)
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.ConfirmAttendance)

	t.Run("Confirm Attendance", func(t *testing.T) {
		refID := uuid.New()
		mockUC.On("ConfirmAttendance", mock.Anything, refID, receptionistID, hospID, "ASSIGNED").Return(nil)

		payload := map[string]string{"status": "ASSIGNED"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/receptionist/referrals/"+refID.String()+"/confirm-attendance", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- ADMIN TESTS ---
func TestAdminOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := new(MockReferralUseCase)
	handler := handlers.NewAdminHandler(mockUC)
	hospID := uuid.New()

	router := gin.Default()
	router.GET("/api/v1/system-admin/referrals", handler.SystemAdminList)
	router.GET("/api/v1/hospital-admin/referrals-log", func(c *gin.Context) {
		c.Set("hospID", &hospID)
		c.Next()
	}, handler.HospitalAdminLogs)

	t.Run("System Admin List", func(t *testing.T) {
		mockUC.On("ListForSystemAdmin", mock.Anything, 20, 0, "").Return([]entity.Referral{}, int64(0), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/system-admin/referrals", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Hospital Admin Logs", func(t *testing.T) {
		mockUC.On("GetHospitalLogsForAdmin", mock.Anything, hospID, 20, 0).Return([]entity.ReferralStatusHistory{}, int64(0), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/hospital-admin/referrals-log", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
