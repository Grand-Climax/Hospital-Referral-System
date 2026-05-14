package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/handlers"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/pkg/auth"
)

// The original script manually tested DB connections and Bcrypt hashing.
// Here we map that into an integration test.
func TestDatabaseAuthHashing(t *testing.T) {
	// Only run this if a local DB is actually available for integration testing
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=6628 dbname=referral port=5432 sslmode=disable TimeZone=Africa/Addis_Ababa"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping integration test; DB unavailable: %v", err)
		return
	}

	var user entity.User
	if err := db.Where("email = ?", "superadmin@moh.gov.et").First(&user).Error; err != nil {
		t.Skipf("Skipping test; User not found in DB (seeder not run?): %v", err)
		return
	}

	// The superadmin seeded password is "password123"
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123"))
	assert.NoError(t, err, "Bcrypt hash in database does not match the seeded password 'password123'")
}

// Mocking the UseCase for isolated HTTP Handler tests conforms strictly to usecase.AuthUseCase
type MockAuthUseCase struct {
	mock.Mock
}

func (m *MockAuthUseCase) Login(ctx context.Context, email, password string) (*iusecase.LoginResult, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*iusecase.LoginResult), args.Error(1)
}

func (m *MockAuthUseCase) VerifyMFA(ctx context.Context, userID uuid.UUID, code, ipAddress, userAgent string) (*auth.TokenPair, error) {
	args := m.Called(ctx, userID, code, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockAuthUseCase) SetupMFA(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthUseCase) ResetMFA(ctx context.Context, adminUserID, targetUserID uuid.UUID) error {
	args := m.Called(ctx, adminUserID, targetUserID)
	return args.Error(0)
}

func (m *MockAuthUseCase) Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*auth.TokenPair, error) {
	args := m.Called(ctx, refreshToken, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockAuthUseCase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	args := m.Called(ctx, accessToken, refreshToken)
	return args.Error(0)
}

func TestAuthHandlerLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(MockAuthUseCase)
	handler := handlers.NewAuthHandler(mockUC)

	router := gin.Default()
	router.POST("/api/v1/auth/login", handler.Login)

	reqPayload := map[string]string{
		"email":    "superadmin@moh.gov.et",
		"password": "password123",
	}

	mockResp := &iusecase.LoginResult{
		MFAToken:         "mock-mfa-token",
		MFASetupRequired: false,
	}

	mockUC.On("Login", mock.Anything, "superadmin@moh.gov.et", "password123").Return(mockResp, nil)

	body, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:8000"
	req.Header.Set("User-Agent", "TestAgent")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Ensure the response JSON format directly maps the mock.
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "mock-mfa-token", response["mfa_token"])
	mockUC.AssertExpectations(t)
}
