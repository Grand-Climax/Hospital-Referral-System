package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/delivery/http/dto"
	deliveryWS "Hospital-Referral-System/internal/delivery/ws"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	"Hospital-Referral-System/internal/infrastructure/ws"
	"Hospital-Referral-System/internal/pkg/auth"
	"Hospital-Referral-System/internal/usecase"
)

type MockChatMessageRepo struct {
	mock.Mock
}

func (m *MockChatMessageRepo) Create(ctx context.Context, msg *entity.ChatMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatMessageRepo) FindByID(ctx context.Context, id interface{}) (*entity.ChatMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ChatMessage), args.Error(1)
}

func (m *MockChatMessageRepo) FindAll(ctx context.Context) ([]entity.ChatMessage, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.ChatMessage), args.Error(1)
}

func (m *MockChatMessageRepo) Update(ctx context.Context, msg *entity.ChatMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatMessageRepo) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatMessageRepo) GetConversations(ctx context.Context, userID uuid.UUID, filterType string, limit, offset int) ([]entity.Conversation, int64, error) {
	args := m.Called(ctx, userID, filterType, limit, offset)
	return args.Get(0).([]entity.Conversation), args.Get(1).(int64), args.Error(2)
}

func (m *MockChatMessageRepo) GetMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]entity.ChatMessage, int64, error) {
	args := m.Called(ctx, conversationID, limit, offset)
	return args.Get(0).([]entity.ChatMessage), args.Get(1).(int64), args.Error(2)
}

func (m *MockChatMessageRepo) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	args := m.Called(ctx, conversationID, userID)
	return args.Error(0)
}

func (m *MockChatMessageRepo) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockChatMessageRepo) HasConversation(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	args := m.Called(ctx, userA, userB)
	return args.Bool(0), args.Error(1)
}

func (m *MockChatMessageRepo) EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) error {
	args := m.Called(ctx, conversationID, userID)
	return args.Error(0)
}

func (m *MockChatMessageRepo) GetConversationUnreadCount(ctx context.Context, conversationID, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, conversationID, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockChatMessageRepo) ToggleDisabled(ctx context.Context, conversationID uuid.UUID, isDisabled bool, reason string, adminID *uuid.UUID) error {
	args := m.Called(ctx, conversationID, isDisabled, reason, adminID)
	return args.Error(0)
}

func (m *MockChatMessageRepo) SoftDeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	args := m.Called(ctx, conversationID)
	return args.Error(0)
}

func (m *MockChatMessageRepo) GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*entity.Conversation, error) {
	args := m.Called(ctx, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Conversation), args.Error(1)
}

func (m *MockChatMessageRepo) GetOrCreateDirectConversation(ctx context.Context, userA, userB uuid.UUID) (*entity.Conversation, error) {
	args := m.Called(ctx, userA, userB)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Conversation), args.Error(1)
}

func (m *MockChatMessageRepo) GetOrCreateReferralConversation(ctx context.Context, referralID, targetHospitalID uuid.UUID) (*entity.Conversation, error) {
	args := m.Called(ctx, referralID, targetHospitalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Conversation), args.Error(1)
}

type MockAuditLogRepo struct {
	mock.Mock
}

func (m *MockAuditLogRepo) Create(ctx context.Context, log *entity.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockAuditLogRepo) LogWithContext(ctx context.Context, userID uuid.UUID, action entity.ActionType, referralID *uuid.UUID, oldValue, newValue interface{}) error {
	args := m.Called(ctx, userID, action, referralID, oldValue, newValue)
	return args.Error(0)
}

func (m *MockAuditLogRepo) ListByHospital(ctx context.Context, hospitalID uuid.UUID, filter irepository.AuditLogFilter) ([]entity.AuditLog, int64, error) {
	args := m.Called(ctx, hospitalID, filter)
	return args.Get(0).([]entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditLogRepo) ListByReferralAndActions(ctx context.Context, referralID uuid.UUID, actions []entity.ActionType, limit int) ([]entity.AuditLog, error) {
	args := m.Called(ctx, referralID, actions, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.AuditLog), args.Error(1)
}

// Custom mock JWT Auth handler for testing multiple sessions
type MockMultiJWTAuth struct {
	sessions map[string]*auth.TokenPayload
}

func (m *MockMultiJWTAuth) ValidateToken(tokenStr string) (*auth.TokenPayload, error) {
	if payload, ok := m.sessions[tokenStr]; ok {
		return payload, nil
	}
	return nil, assert.AnError
}

func TestWebSocketSecureChatFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Define users
	doctorID := uuid.New()
	specialistID := uuid.New()

	doctorHospitalID := uuid.New()
	specialistHospitalID := uuid.New()

	doctorUser := &entity.User{
		ID:         doctorID,
		Role:       entity.RoleReferringDoctor,
		IsActive:   true,
		IsDeleted:  false,
		FirstName:  "John",
		LastName:   "Doc",
		HospitalID: &doctorHospitalID,
	}

	specialistUser := &entity.User{
		ID:         specialistID,
		Role:       entity.RoleReceivingSpecialist,
		IsActive:   true,
		IsDeleted:  false,
		FirstName:  "Jane",
		LastName:   "Spec",
		HospitalID: &specialistHospitalID,
	}

	// 1. Setup Hub, Mocks, and ChatUseCase
	hub := ws.NewHub()

	mockChatRepo := new(MockChatMessageRepo)
	mockReferralRepo := new(MockReferralRepo)
	mockUserRepo := new(MockUserRepo)
	mockReferralAccessRepo := new(MockReferralAccessRepo)
	mockNetRepo := new(MockNetworkRepo)
	mockAuditRepo := new(MockAuditLogRepo)

	chatUseCase := usecase.NewChatUseCase(
		mockChatRepo,
		mockReferralRepo,
		mockUserRepo,
		mockReferralAccessRepo,
		mockNetRepo,
		nil,
		hub,
		mockAuditRepo,
	)

	// Mock User Lookups
	mockUserRepo.On("FindByID", mock.Anything, doctorID).Return(doctorUser, nil)
	mockUserRepo.On("FindByID", mock.Anything, specialistID).Return(specialistUser, nil)
	mockChatRepo.On("EnsureParticipant", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtAuth := &MockMultiJWTAuth{
		sessions: map[string]*auth.TokenPayload{
			"doctor_token": {
				UserID: doctorID,
				Role:   "REFERRING_DOCTOR",
				HospID: &doctorHospitalID,
			},
			"specialist_token": {
				UserID: specialistID,
				Role:   "RECEIVING_SPECIALIST",
				HospID: &specialistHospitalID,
			},
		},
	}

	wsHandler := deliveryWS.NewHandler(hub, jwtAuth, chatUseCase)

	router := gin.New()
	router.GET("/ws", wsHandler.ServeWS)

	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("WebSocket Chat - Bidirectional Message Delivery Success", func(t *testing.T) {
		// Mock direct conversation setup
		directConv := &entity.Conversation{
			ID: uuid.New(),
		}
		mockChatRepo.On("HasConversation", mock.Anything, doctorID, specialistID).Return(true, nil)
		mockChatRepo.On("GetOrCreateDirectConversation", mock.Anything, doctorID, specialistID).Return(directConv, nil)
		mockChatRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		// Connect Doctor
		doctorWSUrl := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=doctor_token"
		docConn, docResp, err := websocket.DefaultDialer.Dial(doctorWSUrl, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer docConn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, docResp.StatusCode)

		// Connect Specialist
		specWSUrl := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=specialist_token"
		specConn, specResp, err := websocket.DefaultDialer.Dial(specWSUrl, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer specConn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, specResp.StatusCode)

		// Send chat packet from Doctor to Specialist
		msgPayload := map[string]interface{}{
			"type":        "chat",
			"receiver_id": specialistID.String(),
			"content":     "Hello receiving specialist!",
		}
		msgBytes, _ := json.Marshal(msgPayload)
		err = docConn.WriteMessage(websocket.TextMessage, msgBytes)
		assert.NoError(t, err)

		// Wait and read on Specialist connection
		var wsMsg dto.WebSocketMessage
		err = specConn.ReadJSON(&wsMsg)
		if assert.NoError(t, err) {
			assert.Equal(t, "chat", wsMsg.Type)
			dataMap, ok := wsMsg.Data.(map[string]interface{})
			if assert.True(t, ok) {
				assert.Equal(t, doctorID.String(), dataMap["sender_id"])
				assert.Equal(t, specialistID.String(), dataMap["receiver_id"])
				assert.Equal(t, "Hello receiving specialist!", dataMap["content"])
			}
		}
	})

	t.Run("WebSocket Chat - Rejection when Self-Chatting", func(t *testing.T) {
		// Connect Doctor
		doctorWSUrl := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=doctor_token"
		docConn, _, err := websocket.DefaultDialer.Dial(doctorWSUrl, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer docConn.Close()

		// Send chat packet to self
		msgPayload := map[string]interface{}{
			"type":        "chat",
			"receiver_id": doctorID.String(),
			"content":     "Self text",
		}
		msgBytes, _ := json.Marshal(msgPayload)
		err = docConn.WriteMessage(websocket.TextMessage, msgBytes)
		assert.NoError(t, err)

		// Expect error message frame on doctor's own connection
		_, docRespBytes, err := docConn.ReadMessage()
		if assert.NoError(t, err) {
			var errFrame struct {
				Type string `json:"type"`
				Data struct {
					Message string `json:"message"`
				} `json:"data"`
			}
			err = json.Unmarshal(docRespBytes, &errFrame)
			if assert.NoError(t, err) {
				assert.Equal(t, "error", errFrame.Type)
				assert.Contains(t, errFrame.Data.Message, "Cannot send a message to yourself.")
			}
		}
	})

	t.Run("WebSocket Chat - Rejection when Scoped to Completed Referral (Read‑Only)", func(t *testing.T) {
		referralID := uuid.New()
		referral := &entity.Referral{
			ID:                referralID,
			ReferringDoctorID: doctorID,
			SpecialistID:      &specialistID,
			Status:            entity.StatusCompleted, // Completed!
		}

		mockReferralRepo.On("GetReferralByID", mock.Anything, referralID).Return(referral, nil)

		// Connect Doctor
		doctorWSUrl := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=doctor_token"
		docConn, _, err := websocket.DefaultDialer.Dial(doctorWSUrl, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer docConn.Close()

		// Send chat message with Referral ID
		refIDStr := referralID.String()
		msgPayload := map[string]interface{}{
			"type":        "chat",
			"receiver_id": specialistID.String(),
			"referral_id": &refIDStr,
			"content":     "Referral scoped text",
		}
		msgBytes, _ := json.Marshal(msgPayload)
		err = docConn.WriteMessage(websocket.TextMessage, msgBytes)
		assert.NoError(t, err)

		// Expect error message frame on doctor's connection
		_, docRespBytes, err := docConn.ReadMessage()
		if assert.NoError(t, err) {
			var errFrame struct {
				Type string `json:"type"`
				Data struct {
					Message string `json:"message"`
				} `json:"data"`
			}
			err = json.Unmarshal(docRespBytes, &errFrame)
			if assert.NoError(t, err) {
				assert.Equal(t, "error", errFrame.Type)
				assert.Contains(t, errFrame.Data.Message, "This referral has been completed. The conversation is now read")
			}
		}
	})

	t.Run("WebSocket Chat - Admin Conversation Locking", func(t *testing.T) {
		// Reset shared mockChatRepo expectations to prevent interference from previous subtests
		mockChatRepo.ExpectedCalls = nil
		mockChatRepo.Calls = nil

		// Re-register required mock behaviors
		mockChatRepo.On("EnsureParticipant", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockChatRepo.On("HasConversation", mock.Anything, doctorID, specialistID).Return(true, nil)

		// Mock conversation that is disabled
		disabledConv := &entity.Conversation{
			ID:         uuid.New(),
			IsDisabled: true,
		}
		mockChatRepo.On("GetOrCreateDirectConversation", mock.Anything, doctorID, specialistID).Return(disabledConv, nil)

		// Connect Doctor
		doctorWSUrl := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=doctor_token"
		docConn, _, err := websocket.DefaultDialer.Dial(doctorWSUrl, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer docConn.Close()

		// Send chat packet
		msgPayload := map[string]interface{}{
			"type":        "chat",
			"receiver_id": specialistID.String(),
			"content":     "Locked chat test",
		}
		msgBytes, _ := json.Marshal(msgPayload)
		err = docConn.WriteMessage(websocket.TextMessage, msgBytes)
		assert.NoError(t, err)

		// Expect lockout rejection error
		_, docRespBytes, err := docConn.ReadMessage()
		if assert.NoError(t, err) {
			var errFrame struct {
				Type string `json:"type"`
				Data struct {
					Message string `json:"message"`
				} `json:"data"`
			}
			err = json.Unmarshal(docRespBytes, &errFrame)
			if assert.NoError(t, err) {
				assert.Equal(t, "error", errFrame.Type)
				assert.Contains(t, errFrame.Data.Message, "This chat room has been administrative-locked.")
			}
		}
	})

	t.Run("WebSocket Chat - Referral Redirections (Target Hospital Isolation)", func(t *testing.T) {
		referralID := uuid.New()
		hospA := uuid.New()
		hospB := uuid.New()

		referral := &entity.Referral{
			ID:                referralID,
			ReferringDoctorID: doctorID,
			SpecialistID:      &specialistID,
			Status:            entity.StatusAccepted,
			TargetHospitalID:  hospA,
		}

		convA := &entity.Conversation{
			ID:               uuid.New(),
			ReferralID:       &referralID,
			TargetHospitalID: &hospA,
		}

		convB := &entity.Conversation{
			ID:               uuid.New(),
			ReferralID:       &referralID,
			TargetHospitalID: &hospB,
		}

		// Reset mock and configure for redirections
		mockReferralRepo2 := new(MockReferralRepo)
		mockChatRepo2 := new(MockChatMessageRepo)

		chatUseCase2 := usecase.NewChatUseCase(
			mockChatRepo2,
			mockReferralRepo2,
			mockUserRepo,
			mockReferralAccessRepo,
			mockNetRepo,
			nil,
			hub,
			mockAuditRepo,
		)

		mockReferralRepo2.On("GetReferralByID", mock.Anything, referralID).Return(referral, nil)
		mockChatRepo2.On("HasConversation", mock.Anything, doctorID, specialistID).Return(true, nil)
		mockChatRepo2.On("GetOrCreateReferralConversation", mock.Anything, referralID, hospA).Return(convA, nil)
		mockChatRepo2.On("GetOrCreateReferralConversation", mock.Anything, referralID, hospB).Return(convB, nil)
		mockChatRepo2.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockChatRepo2.On("EnsureParticipant", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// 1. Send under target hospital A
		msg1, err := chatUseCase2.SendMessage(context.Background(), doctorID, specialistID, &referralID, "Referral message 1")
		assert.NoError(t, err)
		assert.Equal(t, convA.ID, msg1.ConversationID)

		// 2. Redirect referral to target hospital B
		referral.TargetHospitalID = hospB

		// 3. Send under target hospital B -> must resolve new isolated conversation B
		msg2, err := chatUseCase2.SendMessage(context.Background(), doctorID, specialistID, &referralID, "Referral message 2")
		assert.NoError(t, err)
		assert.Equal(t, convB.ID, msg2.ConversationID)
		assert.NotEqual(t, msg1.ConversationID, msg2.ConversationID) // Guaranteed Isolation!
	})
}
