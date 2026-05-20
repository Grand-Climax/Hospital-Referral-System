package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatMessage struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ConversationID uuid.UUID      `gorm:"type:uuid;not null;index:idx_chat_message_conv" json:"conversation_id"`
	SenderID       uuid.UUID      `gorm:"type:uuid;not null;index:idx_chat_message_sender" json:"sender_id"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	IsRead         bool           `gorm:"default:false" json:"is_read"`
	CreatedAt      time.Time      `gorm:"default:now();index" json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Sender       *User         `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Conversation *Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
}

func (cm *ChatMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	return
}
