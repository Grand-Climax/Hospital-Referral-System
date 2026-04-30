package dto

import (
	"time"

	"github.com/google/uuid"
	"Hospital-Referral-System/internal/domain/entity"
)

type NotificationSendSummary struct {
	TotalProcessed int `json:"total_processed"`
	SentCount      int `json:"sent_count"`
	FailedCount    int `json:"failed_count"`
}

type NotificationStatusSummary struct {
	TotalChecked    int `json:"total_checked"`
	DeliveredCount  int `json:"delivered_count"`
	FailedCount     int `json:"failed_count"`
	StillProcessing int `json:"still_processing"`
}

type NotificationResponse struct {
	ID             uuid.UUID  `json:"id"`
	RecipientPhone string     `json:"recipient_phone"`
	Message        string     `json:"message"`
	DeliveryStatus string     `json:"delivery_status"`
	SentAt         *time.Time `json:"sent_at"`
}

type NotificationListResponse struct {
	BaseResponse
	Data  []entity.Notification `json:"data"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
}
