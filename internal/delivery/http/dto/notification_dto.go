package dto

import (
	"time"

	"github.com/google/uuid"
)

type NotificationSendSummary struct {
	TotalProcessed int `json:"total_processed"`
	SentCount      int `json:"sent_count"`
	FailedCount    int `json:"failed_count"`
}

type NotificationResponse struct {
	ID             uuid.UUID `json:"id"`
	RecipientPhone string    `json:"recipient_phone"`
	Message        string    `json:"message"`
	DeliveryStatus string    `json:"delivery_status"`
	SentAt         *time.Time `json:"sent_at"`
}
