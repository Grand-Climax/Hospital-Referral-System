package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeliveryStatus string

const (
	DeliveryQueued    DeliveryStatus = "QUEUED"
	DeliverySent      DeliveryStatus = "SENT"
	DeliveryDelivered DeliveryStatus = "DELIVERED"
	DeliveryFailed    DeliveryStatus = "FAILED"
)

type NotificationType string

const (
	NotifyAcceptance NotificationType = "ACCEPTANCE"
	NotifyScheduling NotificationType = "SCHEDULING"
	NotifyReminder   NotificationType = "REMINDER"
	NotifyReschedule NotificationType = "RESCHEDULE"
)

type Notification struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID       uuid.UUID        `gorm:"type:uuid;not null;index" json:"referral_id"`
	PhoneNumber      string           `gorm:"type:varchar(20);not null" json:"phone_number"`
	Content          string           `gorm:"type:text;not null" json:"content"`
	DeliveryStatus   DeliveryStatus   `gorm:"type:deliverystatus;not null;default:'QUEUED';index" json:"delivery_status"`
	FailureReason    *string          `gorm:"type:text" json:"failure_reason,omitempty"`
	SentAt           *time.Time       `json:"sent_at,omitempty"`
	CreatedAt        time.Time        `gorm:"default:now()" json:"created_at"`
	NotificationType NotificationType `gorm:"type:notificationtype" json:"notification_type"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return
}
