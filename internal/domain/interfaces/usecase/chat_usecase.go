package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
)

type ChatUseCase interface {
	SendMessage(ctx context.Context, senderID, receiverID uuid.UUID, referralID *uuid.UUID, content string) (*entity.ChatMessage, error)
	ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int64, error)
	GetMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, limit, offset int) ([]entity.ChatMessage, int64, error)
	MarkRead(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	GetConversationID(ctx context.Context, userID, otherUserID uuid.UUID, referralID *uuid.UUID) (uuid.UUID, error)

	// Admin Actions
	ToggleDisabled(ctx context.Context, adminID uuid.UUID, conversationID uuid.UUID, isDisabled bool, reason string) error
	SoftDeleteConversation(ctx context.Context, adminID uuid.UUID, conversationID uuid.UUID) error
}
