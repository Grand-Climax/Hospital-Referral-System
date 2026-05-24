package repository

import (
	"context"
	"encoding/json"
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

func (r *auditLogRepository) LogWithContext(ctx context.Context, userID uuid.UUID, action entity.ActionType, referralID *uuid.UUID, oldValue, newValue interface{}) error {
	var oldJSON, newJSON *string

	if oldValue != nil {
		ov, err := json.Marshal(oldValue)
		if err == nil {
			s := string(ov)
			oldJSON = &s
		}
	}

	if newValue != nil {
		nv, err := json.Marshal(newValue)
		if err == nil {
			s := string(nv)
			newJSON = &s
		}
	}

	ip, _ := ctx.Value("ip_address").(string)
	ua, _ := ctx.Value("user_agent").(string)

	logEntry := &entity.AuditLog{
		UserID:     userID,
		ReferralID: referralID,
		ActionType: action,
		OldValue:   oldJSON,
		NewValue:   newJSON,
		IPAddress:  &ip,
		UserAgent:  &ua,
	}

	return r.db.WithContext(ctx).Create(logEntry).Error
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

	if len(filter.ActionTypes) > 0 {
		query = query.Where("audit_logs.action_type IN ?", filter.ActionTypes)
	} else if filter.ActionType != nil {
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

// ListByReferralAndActions returns audit_log rows for a referral filtered
// to the requested action types in chronological order. Preloads the
// User association so the triage-detail endpoint can render actor names
// in the arrival_history timeline without an extra round trip.
func (r *auditLogRepository) ListByReferralAndActions(ctx context.Context, referralID uuid.UUID, actions []entity.ActionType, limit int) ([]entity.AuditLog, error) {
	var logs []entity.AuditLog
	q := r.db.WithContext(ctx).Model(&entity.AuditLog{}).
		Preload("User").
		Where("referral_id = ?", referralID)
	if len(actions) > 0 {
		q = q.Where("action_type IN ?", actions)
	}
	q = q.Order("timestamp ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
