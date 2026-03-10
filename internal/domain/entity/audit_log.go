package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActionType string

const (
	ActionCreateReferral      ActionType = "CREATE_REFERRAL"
	ActionApproveReferral     ActionType = "APPROVE_REFERRAL"
	ActionRejectReferral      ActionType = "REJECT_REFERRAL"
	ActionAcceptReferral      ActionType = "ACCEPT_REFERRAL"
	ActionRedirectReferral    ActionType = "REDIRECT_REFERRAL"
	ActionOverrideMLScore     ActionType = "OVERRIDE_ML_SCORE"
	ActionViewPatientData     ActionType = "VIEW_PATIENT_DATA"
	ActionLogin               ActionType = "LOGIN"
	ActionLogout              ActionType = "LOGOUT"
	ActionExportData          ActionType = "EXPORT_DATA"
	ActionManageUsers         ActionType = "MANAGE_USERS"
	ActionAssignRoles         ActionType = "ASSIGN_ROLES"
	ActionManageCapacity      ActionType = "MANAGE_CAPACITY"
	ActionViewAuditLog        ActionType = "VIEW_AUDIT_LOG"
	ActionResetMFA            ActionType = "RESET_MFA"
	ActionManageHospitals     ActionType = "MANAGE_HOSPITALS"
	ActionManageDepts         ActionType = "MANAGE_DEPTS"
	ActionAPICall             ActionType = "API_CALL"
	ActionOverrideQueue       ActionType = "OVERRIDE_QUEUE"
	ActionGenerateReports     ActionType = "GENERATE_REPORTS"
	ActionUpdatePatientStatus ActionType = "UPDATE_PATIENT_STATUS"
)

type AuditLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index;index:idx_audit_user_time"`
	ReferralID *uuid.UUID `gorm:"type:uuid;index"`
	ActionType ActionType `gorm:"type:varchar(50);not null;index"`
	Resource   *string    `gorm:"type:varchar(100);index"`
	ResourceID *string    `gorm:"type:varchar(100)"`
	OldValue   *string    `gorm:"type:jsonb"`
	NewValue   *string    `gorm:"type:jsonb"`
	IPAddress  *string    `gorm:"type:varchar(45)"`
	UserAgent  *string    `gorm:"type:text"`
	Timestamp  time.Time  `gorm:"default:now();index;index:idx_audit_user_time"`
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return
}
