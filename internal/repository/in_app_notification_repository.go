package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type inAppNotificationRepository struct {
	*BaseRepository[entity.InAppNotification]
	db *gorm.DB
}

func NewInAppNotificationRepository(db *gorm.DB) irepository.InAppNotificationRepository {
	return &inAppNotificationRepository{
		BaseRepository: NewBaseRepository[entity.InAppNotification](db),
		db:             db,
	}
}

func (r *inAppNotificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, filter irepository.InAppNotificationFilter, limit, offset int) ([]entity.InAppNotification, int64, error) {
	var notifications []entity.InAppNotification
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.InAppNotification{}).Where("user_id = ?", userID)

	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}

	if filter.IsRead != nil {
		query = query.Where("is_read = ?", *filter.IsRead)
	}

	if filter.ReferralID != nil {
		query = query.Where("referral_id = ?", *filter.ReferralID)
	}

	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("(title ILIKE ? OR message ILIKE ?)", searchTerm, searchTerm)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error
	if err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *inAppNotificationRepository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.InAppNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

func (r *inAppNotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.InAppNotification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (r *inAppNotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.InAppNotification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}
