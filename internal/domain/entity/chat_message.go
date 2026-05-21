package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatMessage stores a single message within a Conversation.
//
// Read-tracking design:
//   Per-message IsRead flags are intentionally absent. Read state is tracked
//   via ConversationParticipant.LastReadAt (a per-user timestamp cursor),
//   which scales to any number of participants and avoids N writes per read.
//   Any message with created_at > participant.last_read_at AND sender_id ≠ userID
//   is considered unread for that user.
type ChatMessage struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ConversationID uuid.UUID      `gorm:"type:uuid;not null;index:idx_chat_message_conv" json:"conversation_id"`
	SenderID       uuid.UUID      `gorm:"type:uuid;not null;index:idx_chat_message_sender" json:"sender_id"`
	Content        string         `gorm:"type:text;not null" json:"content"`
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
