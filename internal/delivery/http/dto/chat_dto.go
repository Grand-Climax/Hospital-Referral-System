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
	Type              string  `json:"type"` // "direct" or "referral"
	OtherUserID       string  `json:"other_user_id"`
	OtherUserName     string  `json:"other_user_name"`
	OtherUserRole     string  `json:"other_user_role"`
	OtherUserHospital string  `json:"other_user_hospital"`
	LastMessage       string  `json:"last_message"`
	LastMessageAt     string  `json:"last_message_at"`
	UnreadCount       int64   `json:"unread_count"`
	ReferralID        *string `json:"referral_id,omitempty"`
	ReferralStatus    *string `json:"referral_status,omitempty"` // "ACCEPTED", "SUBMITTED", etc. or nil if direct
	IsReadOnly        bool    `json:"is_read_only"`
	IsDisabled        bool    `json:"is_disabled"`
	DisabledReason    string  `json:"disabled_reason,omitempty"`
}

type ChatMessageResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	ReferralID     *string `json:"referral_id,omitempty"`
	Type           string  `json:"type"` // "direct" or "referral"
	SenderID       string  `json:"sender_id"`
	SenderName     string  `json:"sender_name,omitempty"`
	SenderRole     string  `json:"sender_role,omitempty"`
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

func ToChatMessageResponse(m entity.ChatMessage, receiverID string) ChatMessageResponse {
	var refID *string
	convType := "direct"
	if m.Conversation != nil {
		if m.Conversation.ReferralID != nil {
			s := m.Conversation.ReferralID.String()
			refID = &s
			convType = "referral"
		}
	}

	var rID string
	if receiverID != "" {
		rID = receiverID
	} else if m.Conversation != nil {
		for _, p := range m.Conversation.Participants {
			if p.UserID != m.SenderID {
				rID = p.UserID.String()
				break
			}
		}
	}

	var senderName string
	var senderRole string
	if m.Sender != nil {
		senderName = m.Sender.FirstName + " " + m.Sender.LastName
		senderRole = string(m.Sender.Role)
	}

	return ChatMessageResponse{
		ID:             m.ID.String(),
		ConversationID: m.ConversationID.String(),
		ReferralID:     refID,
		Type:           convType,
		SenderID:       m.SenderID.String(),
		SenderName:     senderName,
		SenderRole:     senderRole,
		ReceiverID:     rID,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ToChatMessageResponseSlice(messages []entity.ChatMessage) []ChatMessageResponse {
	responses := make([]ChatMessageResponse, 0, len(messages))
	for _, m := range messages {
		responses = append(responses, ToChatMessageResponse(m, ""))
	}
	return responses
}

type ChatUnreadCountResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	UnreadCount int64  `json:"unread_count"`
}

type ContactResponse struct {
	UserID         string `json:"user_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Role           string `json:"role"`
	HospitalName   string `json:"hospital_name"`
	DepartmentName string `json:"department_name"`
}

type PaginatedContactResponse struct {
	BaseResponse
	Data     []ContactResponse `json:"data"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}
