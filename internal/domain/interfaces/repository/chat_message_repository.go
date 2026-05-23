package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type ChatMessageRepository interface {
	BaseRepository[entity.ChatMessage]
	Create(ctx context.Context, msg *entity.ChatMessage) error
	GetOrCreateDirectConversation(ctx context.Context, userA, userB uuid.UUID) (*entity.Conversation, error)
	GetOrCreateReferralConversation(ctx context.Context, referralID, targetHospitalID uuid.UUID) (*entity.Conversation, error)
	GetConversations(ctx context.Context, userID uuid.UUID, filterType string, limit, offset int) ([]entity.Conversation, int64, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]entity.ChatMessage, int64, error)
	MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	HasConversation(ctx context.Context, userA, userB uuid.UUID) (bool, error)
	EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) error
	GetConversationUnreadCount(ctx context.Context, conversationID, userID uuid.UUID) (int64, error)

	// Admin Actions
	ToggleDisabled(ctx context.Context, conversationID uuid.UUID, isDisabled bool, reason string, adminID *uuid.UUID) error
	SoftDeleteConversation(ctx context.Context, conversationID uuid.UUID) error
	GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*entity.Conversation, error)
}
