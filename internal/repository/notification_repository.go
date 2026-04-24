package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

// notificationRepository manages outbound SMS notifications and delivery status.
// Embeds BaseRepository for standard CRUD.
type notificationRepository struct {
	*BaseRepository[entity.Notification]
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) irepository.NotificationRepository {
	return &notificationRepository{
		BaseRepository: NewBaseRepository[entity.Notification](db),
		db:             db,
	}
}

func (r *notificationRepository) Create(ctx context.Context, n *entity.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepository) GetQueued(ctx context.Context, limit int) ([]entity.Notification, error) {
	var ns []entity.Notification
	err := r.db.WithContext(ctx).
		Where("delivery_status = ?", entity.DeliveryQueued).
		Limit(limit).
		Find(&ns).Error
	return ns, err
}

func (r *notificationRepository) UpdateDelivery(ctx context.Context, id uuid.UUID, status entity.DeliveryStatus, messageID *string) error {
	updates := map[string]interface{}{"delivery_status": status}
	if messageID != nil {
		updates["provider_message_id"] = *messageID
	}
	if status == entity.DeliverySent {
		updates["sent_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&entity.Notification{}).Where("id = ?", id).Updates(updates).Error
}
