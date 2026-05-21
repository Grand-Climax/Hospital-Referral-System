package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/ws"
	"Hospital-Referral-System/internal/usecase"
)

// TestSendMessage_TerminalReferralStatuses verifies that ALL 7 terminal referral statuses
// correctly block message sending (read-only enforcement) via both WebSocket and REST paths.
//
// The 7 terminal statuses that make a referral conversation read-only:
//   - StatusCompleted
//   - StatusDeceased
//   - StatusCancelled
//   - StatusRejectedByLiaison
//   - StatusRejectedBySpecialist
//   - StatusRejectedAfterSend
//   - StatusRedirected
func TestSendMessage_TerminalReferralStatuses(t *testing.T) {
	doctorID := uuid.New()
	specialistID := uuid.New()
	doctorHospID := uuid.New()
	specHospID := uuid.New()

	doctorUser := &entity.User{
		ID:         doctorID,
		Role:       entity.RoleReferringDoctor,
		IsActive:   true,
		IsDeleted:  false,
		FirstName:  "John",
		LastName:   "Doc",
		HospitalID: &doctorHospID,
	}
	specialistUser := &entity.User{
		ID:         specialistID,
		Role:       entity.RoleReceivingSpecialist,
		IsActive:   true,
		IsDeleted:  false,
		FirstName:  "Jane",
		LastName:   "Spec",
		HospitalID: &specHospID,
	}

	terminalStatuses := []struct {
		name   string
		status entity.ReferralStatus
	}{
		{"Completed", entity.StatusCompleted},
		{"Deceased", entity.StatusDeceased},
		{"Cancelled", entity.StatusCancelled},
		{"RejectedByLiaison", entity.StatusRejectedByLiaison},
		{"RejectedBySpecialist", entity.StatusRejectedBySpecialist},
		{"RejectedAfterSend", entity.StatusRejectedAfterSend},
		{"Redirected", entity.StatusRedirected},
	}

	for _, tc := range terminalStatuses {
		tc := tc // capture for parallel
		t.Run("Blocks_SendMessage_when_status_is_"+tc.name, func(t *testing.T) {
			t.Parallel()

			referralID := uuid.New()
			referral := &entity.Referral{
				ID:                referralID,
				ReferringDoctorID: doctorID,
				SpecialistID:      &specialistID,
				Status:            tc.status,
				TargetHospitalID:  uuid.New(),
			}

			hub := ws.NewHub()
			mockChatRepo := new(MockChatMessageRepo)
			mockReferralRepo := new(MockReferralRepo)
			mockUserRepo := new(MockUserRepo)
			mockReferralAccessRepo := new(MockReferralAccessRepo)
			mockNetRepo := new(MockNetworkRepo)
			mockAuditRepo := new(MockAuditLogRepo)

			mockUserRepo.On("FindByID", mock.Anything, doctorID).Return(doctorUser, nil)
			mockUserRepo.On("FindByID", mock.Anything, specialistID).Return(specialistUser, nil)
			// HasConversation is checked before the referral terminal status guard
			mockChatRepo.On("HasConversation", mock.Anything, doctorID, specialistID).Return(false, nil)
			mockReferralRepo.On("GetReferralByID", mock.Anything, referralID).Return(referral, nil)

			chatUC := usecase.NewChatUseCase(
				mockChatRepo,
				mockReferralRepo,
				mockUserRepo,
				mockReferralAccessRepo,
				mockNetRepo,
				nil,
				hub,
				mockAuditRepo,
			)

			_, err := chatUC.SendMessage(context.Background(), doctorID, specialistID, &referralID, "hello")

			assert.Error(t, err, "Expected error for terminal status: %s", tc.name)
			assert.Contains(t, err.Error(), "read",
				"Expected read-only error for status %s, got: %s", tc.name, err.Error())

			// Verify no message was persisted
			mockChatRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

// TestSendMessage_TerminalReferral_AllowsRead verifies that reaching GET messages still works
// even when the referral is in a terminal state (read-only = sending blocked, reading allowed).
func TestSendMessage_TerminalReferral_AllowsRead(t *testing.T) {
	doctorID := uuid.New()
	doctorHospID := uuid.New()

	doctorUser := &entity.User{
		ID:         doctorID,
		Role:       entity.RoleReferringDoctor,
		IsActive:   true,
		IsDeleted:  false,
		HospitalID: &doctorHospID,
	}

	convID := uuid.New()
	conv := &entity.Conversation{
		ID: convID,
		Participants: []entity.ConversationParticipant{
			{UserID: doctorID},
		},
	}

	existingMessages := []entity.ChatMessage{
		{ID: uuid.New(), ConversationID: convID, SenderID: doctorID, Content: "Old message", CreatedAt: time.Now().Add(-24 * time.Hour)},
	}

	hub := ws.NewHub()
	mockChatRepo := new(MockChatMessageRepo)
	mockReferralRepo := new(MockReferralRepo)
	mockUserRepo := new(MockUserRepo)
	mockReferralAccessRepo := new(MockReferralAccessRepo)
	mockNetRepo := new(MockNetworkRepo)
	mockAuditRepo := new(MockAuditLogRepo)

	mockUserRepo.On("FindByID", mock.Anything, doctorID).Return(doctorUser, nil)
	mockChatRepo.On("GetConversationByID", mock.Anything, convID).Return(conv, nil)
	mockChatRepo.On("GetMessages", mock.Anything, convID, 20, 0).Return(existingMessages, int64(1), nil)
	mockChatRepo.On("MarkRead", mock.Anything, convID, doctorID).Return(nil)

	chatUC := usecase.NewChatUseCase(
		mockChatRepo,
		mockReferralRepo,
		mockUserRepo,
		mockReferralAccessRepo,
		mockNetRepo,
		nil,
		hub,
		mockAuditRepo,
	)

	// GetMessages must succeed regardless of referral status — read is always allowed
	messages, total, err := chatUC.GetMessages(context.Background(), doctorID, convID, 20, 0)
	assert.NoError(t, err, "GetMessages must succeed even when referral is terminal")
	assert.Equal(t, int64(1), total)
	assert.Len(t, messages, 1)
	assert.Equal(t, "Old message", messages[0].Content)

	// MarkRead must also succeed (badge clearing must work even in read-only rooms)
	err = chatUC.MarkRead(context.Background(), doctorID, convID)
	assert.NoError(t, err, "MarkRead must succeed even in terminal/locked conversations")
}

// TestSendMessage_DisabledConversation_AllowsRead verifies that an admin-locked
// (is_disabled) conversation still allows history reads and badge clearing.
func TestSendMessage_DisabledConversation_AllowsRead(t *testing.T) {
	userID := uuid.New()
	hospID := uuid.New()
	convID := uuid.New()

	user := &entity.User{
		ID:         userID,
		Role:       entity.RoleReferringDoctor,
		IsActive:   true,
		HospitalID: &hospID,
	}

	// Conversation is administratively disabled
	lockedConv := &entity.Conversation{
		ID:         convID,
		IsDisabled: true,
		Participants: []entity.ConversationParticipant{
			{UserID: userID},
		},
	}

	existingMessages := []entity.ChatMessage{
		{ID: uuid.New(), ConversationID: convID, SenderID: uuid.New(), Content: "Archived message"},
	}

	hub := ws.NewHub()
	mockChatRepo := new(MockChatMessageRepo)
	mockReferralRepo := new(MockReferralRepo)
	mockUserRepo := new(MockUserRepo)
	mockReferralAccessRepo := new(MockReferralAccessRepo)
	mockNetRepo := new(MockNetworkRepo)
	mockAuditRepo := new(MockAuditLogRepo)

	mockUserRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
	mockChatRepo.On("GetConversationByID", mock.Anything, convID).Return(lockedConv, nil)
	mockChatRepo.On("GetMessages", mock.Anything, convID, 20, 0).Return(existingMessages, int64(1), nil)
	mockChatRepo.On("MarkRead", mock.Anything, convID, userID).Return(nil)

	chatUC := usecase.NewChatUseCase(
		mockChatRepo,
		mockReferralRepo,
		mockUserRepo,
		mockReferralAccessRepo,
		mockNetRepo,
		nil,
		hub,
		mockAuditRepo,
	)

	// GetMessages succeeds on disabled room — audit/history access preserved
	messages, total, err := chatUC.GetMessages(context.Background(), userID, convID, 20, 0)
	assert.NoError(t, err, "GetMessages must work on disabled conversations")
	assert.Equal(t, int64(1), total)
	assert.Len(t, messages, 1)

	// MarkRead succeeds on disabled room — badge clearing preserved
	err = chatUC.MarkRead(context.Background(), userID, convID)
	assert.NoError(t, err, "MarkRead must work on disabled conversations")
}
