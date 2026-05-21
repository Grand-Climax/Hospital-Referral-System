package dto

import (
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type SendMessageRequest struct {
	ReceiverID uuid.UUID  `json:"receiver_id" binding:"required"`
	ReferralID *uuid.UUID `json:"referral_id"`
	Content    string     `json:"content" binding:"required,min=1,max=5000"`
}

type ConversationResponse struct {
	ConversationID    string  `json:"conversation_id"`
	OtherUserID       string  `json:"other_user_id"`
	OtherUserName     string  `json:"other_user_name"`
	OtherUserRole     string  `json:"other_user_role"`
	OtherUserHospital string  `json:"other_user_hospital"`
	LastMessage       string  `json:"last_message"`
	LastMessageAt     string  `json:"last_message_at"`
	UnreadCount       int64   `json:"unread_count"`
	ReferralID        *string `json:"referral_id,omitempty"`
	IsReadOnly        bool    `json:"is_read_only"`
	IsDisabled        bool    `json:"is_disabled"`
	DisabledReason    string  `json:"disabled_reason,omitempty"`
}

type ChatMessageResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	ReferralID     *string `json:"referral_id,omitempty"`
	SenderID       string  `json:"sender_id"`
	ReceiverID     string  `json:"receiver_id,omitempty"`
	Content        string  `json:"content"`
	CreatedAt      string  `json:"created_at"`
}

type PaginatedChatResponse struct {
	BaseResponse
	Data     []ChatMessageResponse `json:"data"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type PaginatedConversationResponse struct {
	BaseResponse
	Data     []ConversationResponse `json:"data"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

func ToChatMessageResponseSlice(messages []entity.ChatMessage) []ChatMessageResponse {
	responses := make([]ChatMessageResponse, 0, len(messages))
	for _, m := range messages {
		var refID *string
		if m.Conversation != nil && m.Conversation.ReferralID != nil {
			s := m.Conversation.ReferralID.String()
			refID = &s
		}
		var receiverID string
		if m.Conversation != nil {
			for _, p := range m.Conversation.Participants {
				if p.UserID != m.SenderID {
					receiverID = p.UserID.String()
					break
				}
			}
		}
		responses = append(responses, ChatMessageResponse{
			ID:             m.ID.String(),
			ConversationID: m.ConversationID.String(),
			ReferralID:     refID,
			SenderID:       m.SenderID.String(),
			ReceiverID:     receiverID,
			Content:        m.Content,
			CreatedAt:      m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return responses
}

type ChatUnreadCountResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	UnreadCount int64  `json:"unread_count"`
}
