package repository

import (
	"context"
	"encoding/json"

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
