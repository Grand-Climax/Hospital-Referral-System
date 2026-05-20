package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Conversation struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID         *uuid.UUID     `gorm:"type:uuid;index:idx_conversation_referral" json:"referral_id,omitempty"`
	TargetHospitalID   *uuid.UUID     `gorm:"type:uuid;index:idx_conversation_target_hospital" json:"target_hospital_id,omitempty"`
	Title              string         `gorm:"type:varchar(255)" json:"title,omitempty"`
	IsDisabled         bool           `gorm:"default:false" json:"is_disabled"`
	DisabledReason     string         `gorm:"type:text" json:"disabled_reason,omitempty"`
	DisabledAt         *time.Time     `json:"disabled_at,omitempty"`
	DisabledByID       *uuid.UUID     `gorm:"type:uuid" json:"disabled_by_id,omitempty"`
	LastMessageContent string         `gorm:"type:text" json:"last_message_content,omitempty"`
	LastMessageAt      time.Time      `gorm:"default:now()" json:"last_message_at"`
	CreatedAt          time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"default:now()" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Referral     *Referral                 `gorm:"foreignKey:ReferralID" json:"referral,omitempty"`
	DisabledBy   *User                     `gorm:"foreignKey:DisabledByID" json:"disabled_by,omitempty"`
	Participants []ConversationParticipant `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"participants,omitempty"`
	Messages     []ChatMessage             `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"messages,omitempty"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

type ConversationParticipant struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index:idx_participant_conv_user,priority:1" json:"conversation_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index:idx_participant_conv_user,priority:2;index:idx_participant_user" json:"user_id"`
	JoinedAt       time.Time `gorm:"default:now()" json:"joined_at"`
	LastReadAt     time.Time `gorm:"default:now()" json:"last_read_at"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (cp *ConversationParticipant) BeforeCreate(tx *gorm.DB) (err error) {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	return
}
