package dto

import (
	"time"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

type InAppNotificationResponse struct {
	ID         uuid.UUID  `json:"id"`
	ReferralID *uuid.UUID `json:"referral_id,omitempty"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	EventType  string     `json:"event_type"`
	IsRead     bool       `json:"is_read"`
	CreatedAt  time.Time  `json:"created_at"`
}

type PaginatedNotificationResponse struct {
	BaseResponse
	Data        []InAppNotificationResponse `json:"data"`
	Total       int64                       `json:"total"`
	UnreadCount int64                       `json:"unread_count"`
	Page        int                         `json:"page"`
	PageSize    int                         `json:"page_size"`
}

type UnreadCountResponse struct {
	BaseResponse
	UnreadCount int64 `json:"unread_count"`
}

func ToInAppNotificationResponse(n entity.InAppNotification) InAppNotificationResponse {
	return InAppNotificationResponse{
		ID:         n.ID,
		ReferralID: n.ReferralID,
		Title:      n.Title,
		Message:    n.Message,
		EventType:  n.EventType,
		IsRead:     n.IsRead,
		CreatedAt:  n.CreatedAt,
	}
}

func ToInAppNotificationResponseSlice(ns []entity.InAppNotification) []InAppNotificationResponse {
	res := make([]InAppNotificationResponse, len(ns))
	for i, n := range ns {
		res[i] = ToInAppNotificationResponse(n)
	}
	return res
}
