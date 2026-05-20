package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	deliveryWS "Hospital-Referral-System/internal/delivery/ws"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	"Hospital-Referral-System/internal/infrastructure/ws"
	"Hospital-Referral-System/internal/pkg/auth"
	"Hospital-Referral-System/internal/usecase"
)

type MockJWTAuth struct {
	claims *auth.TokenPayload
	err    error
}

func (m *MockJWTAuth) ValidateToken(tokenStr string) (*auth.TokenPayload, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claims, nil
}

func TestWebSocketFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := ws.NewHub()
	mockClaims := &auth.TokenPayload{
		UserID: uuid.New(),
		Role:   "REFERRING_DOCTOR",
	}
	mockJWT := &MockJWTAuth{claims: mockClaims}

	wsHandler := deliveryWS.NewHandler(hub, mockJWT, nil)
	pushHandler := handlers.NewPushHandler(hub)

	router := gin.New()
	router.GET("/ws", wsHandler.ServeWS)
	router.POST("/api/v1/internal/push", pushHandler.PushToUser)

	s := httptest.NewServer(router)
	defer s.Close()

	t.Run("ServeWS - Missing Token", func(t *testing.T) {
		u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
		_, resp, err := websocket.DefaultDialer.Dial(u, nil)
		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("ServeWS & PushToUser Success Flow", func(t *testing.T) {
		// 1. Establish WebSocket Connection
		u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws?token=valid_token"
		conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

		// Set env variable for push authorization
		os.Setenv("WS_PUSH_SECRET", "super_secret")
		defer os.Unsetenv("WS_PUSH_SECRET")

		// 2. Perform Push via PushToUser endpoint
		pushURL := s.URL + "/api/v1/internal/push"
		reqBody := `{"user_id":"` + mockClaims.UserID.String() + `","type":"test_event","data":{"msg":"hello"}}`
		
		req, err := http.NewRequest(http.MethodPost, pushURL, strings.NewReader(reqBody))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer super_secret")
		req.Header.Set("Content-Type", "application/json")

		respPush, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, respPush.StatusCode)

		// 3. Read message from WebSocket client conn
		_, msg, err := conn.ReadMessage()
		assert.NoError(t, err)
		assert.Contains(t, string(msg), "test_event")
		assert.Contains(t, string(msg), "hello")
	})
}

type MockInAppNotifRepo struct {
	mock.Mock
}

func (m *MockInAppNotifRepo) Create(ctx context.Context, notif *entity.InAppNotification) error {
	args := m.Called(ctx, notif)
	return args.Error(0)
}

func (m *MockInAppNotifRepo) FindByID(ctx context.Context, id interface{}) (*entity.InAppNotification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.InAppNotification), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInAppNotifRepo) FindAll(ctx context.Context) ([]entity.InAppNotification, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.InAppNotification), args.Error(1)
}

func (m *MockInAppNotifRepo) Update(ctx context.Context, notif *entity.InAppNotification) error {
	args := m.Called(ctx, notif)
	return args.Error(0)
}

func (m *MockInAppNotifRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInAppNotifRepo) ListByUser(ctx context.Context, userID uuid.UUID, filter irepository.InAppNotificationFilter, limit, offset int) ([]entity.InAppNotification, int64, error) {
	args := m.Called(ctx, userID, filter, limit, offset)
	return args.Get(0).([]entity.InAppNotification), args.Get(1).(int64), args.Error(2)
}

func (m *MockInAppNotifRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockInAppNotifRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockInAppNotifRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func TestWebSocketNotificationPush(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := ws.NewHub()
	userID := uuid.New()
	mockClaims := &auth.TokenPayload{
		UserID: userID,
		Role:   "REFERRING_DOCTOR",
	}
	mockJWT := &MockJWTAuth{claims: mockClaims}

	wsHandler := deliveryWS.NewHandler(hub, mockJWT, nil)

	router := gin.New()
	router.GET("/ws", wsHandler.ServeWS)

	s := httptest.NewServer(router)
	defer s.Close()

	// 1. Establish WebSocket Connection
	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws?token=valid_token"
	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// 2. Set up Mock Repositories and Use Case
	mockNotifRepo := new(MockInAppNotifRepo)
	mockUserRepo := new(MockUserRepo)
	mockReferralRepo := new(MockReferralRepo)

	inAppUseCase := usecase.NewInAppNotificationUseCase(
		mockNotifRepo,
		mockUserRepo,
		mockReferralRepo,
		hub,
	)

	referralID := uuid.New()
	actorID := uuid.New()

	referral := &entity.Referral{
		ID:                referralID,
		ReferringDoctorID: userID, // recipient of REFERRAL_ACCEPTED
		Patient: &entity.Patient{
			FirstNamePlain: "John",
			LastNamePlain:  "Doe",
		},
		ReceiverHospital: &entity.Hospital{
			Name: "Mayo Clinic",
		},
	}

	mockReferralRepo.On("GetReferralByID", mock.Anything, referralID).Return(referral, nil)
	mockNotifRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// 3. Trigger Notification Creation
	err = inAppUseCase.CreateForEvent(context.Background(), "REFERRAL_ACCEPTED", referralID, actorID)
	assert.NoError(t, err)

	// 4. Read notification from WebSocket client
	var wsMsg dto.WebSocketMessage
	err = conn.ReadJSON(&wsMsg)
	assert.NoError(t, err)

	assert.Equal(t, "notification", wsMsg.Type)
	
	// Assert on the payload data
	dataMap, ok := wsMsg.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Referral Accepted", dataMap["title"])
	assert.Contains(t, dataMap["message"], "John Doe")
	assert.Contains(t, dataMap["message"], "Mayo Clinic")
	assert.Equal(t, "REFERRAL_ACCEPTED", dataMap["event_type"])
	assert.Equal(t, referralID.String(), dataMap["referral_id"])
}

func TestWebSocketSystemNotificationPush(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := ws.NewHub()
	userID := uuid.New()
	hospitalID := uuid.New()

	mockClaims := &auth.TokenPayload{
		UserID: userID,
		Role:   "HOSPITAL_ADMIN",
		HospID: &hospitalID,
	}
	mockJWT := &MockJWTAuth{claims: mockClaims}

	wsHandler := deliveryWS.NewHandler(hub, mockJWT, nil)

	router := gin.New()
	router.GET("/ws", wsHandler.ServeWS)

	s := httptest.NewServer(router)
	defer s.Close()

	// 1. Establish WebSocket Connection
	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws?token=valid_token"
	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// 2. Set up Mock Repositories and Use Case
	mockNotifRepo := new(MockInAppNotifRepo)
	mockUserRepo := new(MockUserRepo)
	mockReferralRepo := new(MockReferralRepo)

	inAppUseCase := usecase.NewInAppNotificationUseCase(
		mockNotifRepo,
		mockUserRepo,
		mockReferralRepo,
		hub,
	)

	actorID := uuid.New()
	actorUser := &entity.User{
		ID:         actorID,
		Role:       entity.RoleHospitalAdmin,
		HospitalID: &hospitalID,
	}

	adminRecipient := entity.User{
		ID:         userID,
		Role:       entity.RoleHospitalAdmin,
		HospitalID: &hospitalID,
	}

	// Mock actor user lookup
	mockUserRepo.On("FindByID", mock.Anything, actorID).Return(actorUser, nil)

	// Mock recipient admins list query
	mockUserRepo.On("ListUsers", mock.Anything, mock.Anything).Return([]entity.User{adminRecipient}, int64(1), nil)

	mockNotifRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// 3. Trigger System Notification (STAFF_ADDED) with uuid.Nil referralID
	err = inAppUseCase.CreateForEvent(context.Background(), "STAFF_ADDED", uuid.Nil, actorID)
	assert.NoError(t, err)

	// 4. Read notification from WebSocket client
	var wsMsg dto.WebSocketMessage
	err = conn.ReadJSON(&wsMsg)
	assert.NoError(t, err)

	assert.Equal(t, "notification", wsMsg.Type)
	
	// Assert on the payload data
	dataMap, ok := wsMsg.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "New Staff Added", dataMap["title"])
	assert.Contains(t, dataMap["message"], "registered at your hospital")
	assert.Equal(t, "STAFF_ADDED", dataMap["event_type"])
	assert.Nil(t, dataMap["referral_id"]) // Referral ID must be omitted (nil in JSON map) for system notification due to omitempty!
}


