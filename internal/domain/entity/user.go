package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleReferringDoctor     UserRole = "REFERRING_DOCTOR"
	RoleLiaisonOfficer      UserRole = "LIAISON_OFFICER"
	RoleReceivingSpecialist UserRole = "RECEIVING_SPECIALIST"
	RoleReceptionist        UserRole = "RECEPTIONIST"
	RoleMohAnalyst          UserRole = "MOH_ANALYST"
	RoleDeptHead            UserRole = "DEPT_HEAD"
	RoleHospitalAdmin       UserRole = "HOSPITAL_ADMIN"
	RoleSystemSuperAdmin    UserRole = "SYSTEM_SUPER_ADMIN"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	NationalID   string         `gorm:"type:varchar(20);unique" json:"national_id"`
	Email        string         `gorm:"type:varchar(255);unique;not null" json:"email"`
	FirstName    string         `gorm:"type:varchar(100);not null" json:"first_name"`
	MiddleName   string         `gorm:"type:varchar(100);not null;default:''" json:"middle_name"`
	LastName     string         `gorm:"type:varchar(100);not null" json:"last_name"`
	Role         UserRole       `gorm:"type:varchar(50);not null;index" json:"role"`
	HospitalID   *uuid.UUID     `gorm:"type:uuid;index" json:"hospital_id,omitempty"`
	DepartmentID *uuid.UUID     `gorm:"type:uuid" json:"department_id,omitempty"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	ProfileImageURL string      `gorm:"type:text" json:"profile_image_url"`
	ProfileImagePublicID string `gorm:"type:varchar(255)" json:"-"`
	IsActive     bool           `gorm:"default:true;index" json:"is_active"`
	MFASecretEnc *string        `gorm:"type:text" json:"-"`
	IsDeleted    bool           `gorm:"default:false" json:"-"`
	DeletedAt    gorm.DeletedAt `json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// Relationships
	Hospital     *Hospital   `gorm:"foreignKey:HospitalID;constraint:OnDelete:SET NULL;" json:"hospital,omitempty"`
	Department   *Department `gorm:"foreignKey:DepartmentID;constraint:OnDelete:SET NULL;" json:"department,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
