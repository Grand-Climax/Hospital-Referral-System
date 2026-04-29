package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) irepository.AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditLogRepository) ListByHospital(ctx context.Context, hospitalID uuid.UUID, filter irepository.AuditLogFilter) ([]entity.AuditLog, int64, error) {
	var logs []entity.AuditLog
	var total int64

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&entity.AuditLog{}).
		Joins("JOIN users ON users.id = audit_logs.user_id").
		Where("users.hospital_id = ?", hospitalID)

	if filter.ActionType != nil {
		query = query.Where("audit_logs.action_type = ?", *filter.ActionType)
	}
	if filter.StartDate != nil && *filter.StartDate != "" {
		if t, err := time.Parse("2006-01-02", *filter.StartDate); err == nil {
			query = query.Where("audit_logs.timestamp >= ?", t)
		}
	}
	if filter.EndDate != nil && *filter.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *filter.EndDate); err == nil {
			// inclusive end day
			query = query.Where("audit_logs.timestamp < ?", t.Add(24*time.Hour))
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("audit_logs.timestamp DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
