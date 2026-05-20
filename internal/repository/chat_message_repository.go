package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type chatMessageRepository struct {
	*BaseRepository[entity.ChatMessage]
	db *gorm.DB
}

func NewChatMessageRepository(db *gorm.DB) irepository.ChatMessageRepository {
	return &chatMessageRepository{
		BaseRepository: NewBaseRepository[entity.ChatMessage](db),
		db:             db,
	}
}

func (r *chatMessageRepository) Create(ctx context.Context, msg *entity.ChatMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}

		preview := msg.Content
		if len(preview) > 100 {
			preview = preview[:97] + "..."
		}

		// Update the parent conversation's last message previews
		err := tx.Model(&entity.Conversation{}).
			Where("id = ?", msg.ConversationID).
			Updates(map[string]interface{}{
				"last_message_content": preview,
				"last_message_at":      msg.CreatedAt,
				"updated_at":           time.Now(),
			}).Error
		return err
	})
}

func (r *chatMessageRepository) GetOrCreateDirectConversation(ctx context.Context, userA, userB uuid.UUID) (*entity.Conversation, error) {
	var convID uuid.UUID

	// Query to find a direct conversation with exactly these two participants
	query := `
		SELECT cp1.conversation_id 
		FROM conversation_participants cp1
		JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
		JOIN conversations c ON c.id = cp1.conversation_id
		WHERE cp1.user_id = ? AND cp2.user_id = ? AND c.referral_id IS NULL AND c.deleted_at IS NULL
		  AND (SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = cp1.conversation_id) = 2
		LIMIT 1
	`

	err := r.db.WithContext(ctx).Raw(query, userA, userB).Scan(&convID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if convID != uuid.Nil {
		var conversation entity.Conversation
		if err := r.db.WithContext(ctx).Preload("Participants").First(&conversation, "id = ?", convID).Error; err != nil {
			return nil, err
		}
		return &conversation, nil
	}

	// Create new direct conversation
	conversation := &entity.Conversation{
		ID: uuid.New(),
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			return err
		}

		p1 := entity.ConversationParticipant{
			ConversationID: conversation.ID,
			UserID:         userA,
		}
		p2 := entity.ConversationParticipant{
			ConversationID: conversation.ID,
			UserID:         userB,
		}

		if err := tx.Create(&p1).Error; err != nil {
			return err
		}
		if err := tx.Create(&p2).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Preload participants for returning
	r.db.WithContext(ctx).Preload("Participants").First(conversation, "id = ?", conversation.ID)
	return conversation, nil
}

func (r *chatMessageRepository) GetOrCreateReferralConversation(ctx context.Context, referralID, targetHospitalID uuid.UUID) (*entity.Conversation, error) {
	var conversation entity.Conversation
	err := r.db.WithContext(ctx).
		Preload("Participants").
		Where("referral_id = ? AND target_hospital_id = ?", referralID, targetHospitalID).
		First(&conversation).Error

	if err == nil {
		return &conversation, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create new referral-scoped conversation
	conversation = entity.Conversation{
		ID:               uuid.New(),
		ReferralID:       &referralID,
		TargetHospitalID: &targetHospitalID,
		LastMessageAt:    time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&conversation).Error; err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *chatMessageRepository) GetConversations(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Conversation, int64, error) {
	var conversations []entity.Conversation
	var total int64

	// Count conversations where user is participant
	countQuery := r.db.WithContext(ctx).Model(&entity.Conversation{}).
		Joins("JOIN conversation_participants cp ON cp.conversation_id = conversations.id").
		Where("cp.user_id = ?", userID)

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []entity.Conversation{}, 0, nil
	}

	// Fetch with preloads, pagination and last_message_at sorting
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_participants cp ON cp.conversation_id = conversations.id").
		Where("cp.user_id = ? AND conversations.deleted_at IS NULL", userID).
		Order("conversations.last_message_at DESC").
		Limit(limit).
		Offset(offset).
		Preload("Participants.User.Hospital").
		Preload("Referral").
		Find(&conversations).Error

	if err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

func (r *chatMessageRepository) GetMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]entity.ChatMessage, int64, error) {
	var messages []entity.ChatMessage
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.ChatMessage{}).
		Where("conversation_id = ?", conversationID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Preload("Sender").
		Find(&messages).Error

	if err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (r *chatMessageRepository) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	now := time.Now()
	// Update participant's LastReadAt
	if err := r.db.WithContext(ctx).Model(&entity.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_at", now).Error; err != nil {
		return err
	}

	// Update is_read on ChatMessage table for all messages in the conversation sent by other users
	return r.db.WithContext(ctx).Model(&entity.ChatMessage{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = ?", conversationID, userID, false).
		Update("is_read", true).Error
}

func (r *chatMessageRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	// Find all conversations user participates in
	var participantRecords []entity.ConversationParticipant
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&participantRecords).Error; err != nil {
		return 0, err
	}

	if len(participantRecords) == 0 {
		return 0, nil
	}

	var totalUnread int64
	for _, part := range participantRecords {
		var count int64
		// Count messages in this conversation created after participant's LastReadAt, which are not sent by user
		err := r.db.WithContext(ctx).Model(&entity.ChatMessage{}).
			Where("conversation_id = ? AND sender_id != ? AND created_at > ?", part.ConversationID, userID, part.LastReadAt).
			Count(&count).Error
		if err == nil {
			totalUnread += count
		}
	}

	return totalUnread, nil
}

func (r *chatMessageRepository) HasConversation(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	var count int64
	query := `
		SELECT COUNT(*)
		FROM conversation_participants cp1
		JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
		JOIN conversations c ON c.id = cp1.conversation_id
		WHERE cp1.user_id = ? AND cp2.user_id = ? AND c.referral_id IS NULL AND c.deleted_at IS NULL
	`
	err := r.db.WithContext(ctx).Raw(query, userA, userB).Scan(&count).Error
	return count > 0, err
}

func (r *chatMessageRepository) ToggleDisabled(ctx context.Context, conversationID uuid.UUID, isDisabled bool, reason string, adminID *uuid.UUID) error {
	now := time.Now()
	updates := map[string]interface{}{
		"is_disabled":     isDisabled,
		"disabled_reason": reason,
		"updated_at":      now,
	}
	if isDisabled {
		updates["disabled_at"] = &now
		updates["disabled_by_id"] = adminID
	} else {
		updates["disabled_at"] = nil
		updates["disabled_by_id"] = nil
	}

	return r.db.WithContext(ctx).Model(&entity.Conversation{}).
		Where("id = ?", conversationID).
		Updates(updates).Error
}

func (r *chatMessageRepository) SoftDeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Conversation{ID: conversationID}).Error
}

func (r *chatMessageRepository) GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*entity.Conversation, error) {
	var conversation entity.Conversation
	err := r.db.WithContext(ctx).
		Preload("Participants.User.Hospital").
		Preload("Referral").
		First(&conversation, "id = ?", conversationID).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *chatMessageRepository) EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) error {
	var part entity.ConversationParticipant
	return r.db.WithContext(ctx).FirstOrCreate(&part, entity.ConversationParticipant{
		ConversationID: conversationID,
		UserID:         userID,
	}).Error
}

func (r *chatMessageRepository) GetConversationUnreadCount(ctx context.Context, conversationID, userID uuid.UUID) (int64, error) {
	var part entity.ConversationParticipant
	err := r.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&part).Error
	if err != nil {
		return 0, err
	}
	var count int64
	err = r.db.WithContext(ctx).Model(&entity.ChatMessage{}).
		Where("conversation_id = ? AND sender_id != ? AND created_at > ?", conversationID, userID, part.LastReadAt).
		Count(&count).Error
	return count, err
}
