package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccessType string

const (
	AccessTreatingDoctor AccessType = "TREATING_DOCTOR"
	AccessConsultedDoctor AccessType = "CONSULTED_DOCTOR"
)

type ReferralAccess struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ReferralID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_referral_user" json:"referral_id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_referral_user;index" json:"user_id"`
	AccessType AccessType `gorm:"type:accesstype;not null" json:"access_type"`
	GrantedBy  uuid.UUID  `gorm:"type:uuid;not null" json:"granted_by"`
	GrantedAt  time.Time  `gorm:"default:now()" json:"granted_at"`
	RevokedAt  *time.Time `gorm:"index:idx_active_access" json:"revoked_at,omitempty"`
	RevokeReason *string   `gorm:"type:text" json:"revoke_reason,omitempty"`
}

func (ra *ReferralAccess) BeforeCreate(tx *gorm.DB) (err error) {
	if ra.ID == uuid.Nil {
		ra.ID = uuid.New()
	}
	return
}
