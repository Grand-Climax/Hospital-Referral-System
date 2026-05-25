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

func (r *notificationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	var n entity.Notification
	err := r.db.WithContext(ctx).First(&n, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) GetQueued(ctx context.Context, limit int) ([]entity.Notification, error) {
	return r.GetQueuedByFilter(ctx, nil, nil, limit)
}

func (r *notificationRepository) GetQueuedByFilter(ctx context.Context, hospitalID, deptID *uuid.UUID, limit int) ([]entity.Notification, error) {
	var ns []entity.Notification
	query := r.db.WithContext(ctx).
		Table("notifications").
		Select("notifications.*").
		Joins("JOIN referrals ON notifications.referral_id = referrals.id").
		Where("notifications.delivery_status = ?", entity.DeliveryQueued)

	if hospitalID != nil && *hospitalID != uuid.Nil {
		query = query.Where("referrals.target_hospital_id = ?", *hospitalID)
	}
	if deptID != nil && *deptID != uuid.Nil {
		query = query.Where("referrals.target_dept_id = ?", *deptID)
	}

	err := query.Limit(limit).Find(&ns).Error
	return ns, err
}

func (r *notificationRepository) GetPendingByFilter(ctx context.Context, hospitalID, deptID *uuid.UUID, statuses []entity.DeliveryStatus, limit int) ([]entity.Notification, error) {
	var ns []entity.Notification
	query := r.db.WithContext(ctx).
		Table("notifications").
		Select("notifications.*").
		Joins("JOIN referrals ON notifications.referral_id = referrals.id").
		Where("notifications.delivery_status IN ?", statuses)

	if hospitalID != nil && *hospitalID != uuid.Nil {
		query = query.Where("referrals.target_hospital_id = ?", *hospitalID)
	}
	if deptID != nil && *deptID != uuid.Nil {
		query = query.Where("referrals.target_dept_id = ?", *deptID)
	}

	err := query.Limit(limit).Find(&ns).Error
	return ns, err
}

func (r *notificationRepository) GetSent(ctx context.Context, limit int) ([]entity.Notification, error) {
	var ns []entity.Notification
	err := r.db.WithContext(ctx).
		Where("delivery_status = ?", entity.DeliverySent).
		Where("provider_message_id IS NOT NULL AND provider_message_id != ?", "").
		Limit(limit).
		Find(&ns).Error
	return ns, err
}

func (r *notificationRepository) UpdateDelivery(ctx context.Context, id uuid.UUID, status entity.DeliveryStatus, messageID *string, failureReason *string) error {
	updates := map[string]interface{}{"delivery_status": status}
	if messageID != nil {
		updates["provider_message_id"] = *messageID
	}
	if status == entity.DeliverySent || status == entity.DeliveryResend {
		updates["sent_at"] = time.Now()
		// Clear any stale failure_reason left over from a previous
		// attempt - if it's now sent, the old error is misleading.
		updates["failure_reason"] = nil
	}
	if failureReason != nil {
		// Truncate defensively. text columns are unbounded in Postgres
		// but we don't want a 2KB AfroMessage stacktrace bloating the
		// admin UI.
		reason := *failureReason
		if len(reason) > 1024 {
			reason = reason[:1024]
		}
		updates["failure_reason"] = reason
	}
	if status == entity.DeliveryFailed {
		return r.db.WithContext(ctx).Model(&entity.Notification{}).
			Where("id = ?", id).
			Updates(updates).
			UpdateColumn("retry_count", gorm.Expr("retry_count + 1")).Error
	}
	return r.db.WithContext(ctx).Model(&entity.Notification{}).Where("id = ?", id).Updates(updates).Error
}

func (r *notificationRepository) ListWithFilter(ctx context.Context, filter irepository.NotificationListFilter) ([]entity.Notification, int64, error) {
	var ns []entity.Notification
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Notification{})

	if filter.HospitalID != nil || filter.DepartmentID != nil {
		query = query.Joins("JOIN referrals ON notifications.referral_id = referrals.id")
		if filter.HospitalID != nil {
			query = query.Where("referrals.target_hospital_id = ?", *filter.HospitalID)
		}
		if filter.DepartmentID != nil {
			query = query.Where("referrals.target_dept_id = ?", *filter.DepartmentID)
		}
	}

	if filter.ReferralID != nil {
		query = query.Where("notifications.referral_id = ?", *filter.ReferralID)
	}
	if filter.NotificationType != nil {
		query = query.Where("notifications.notification_type = ?", *filter.NotificationType)
	}
	if filter.DeliveryStatus != nil {
		query = query.Where("notifications.delivery_status = ?", *filter.DeliveryStatus)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.PageSize
	if limit <= 0 {
		limit = 20
	}
	offset := (filter.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	err := query.Limit(limit).Offset(offset).Order("notifications.created_at DESC").Find(&ns).Error
	return ns, total, err
}
