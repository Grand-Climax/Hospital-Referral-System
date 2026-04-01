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
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NationalID   string    `gorm:"type:varchar(20);unique"`
	Email        string    `gorm:"type:varchar(255);unique;not null"`
	FirstName    string    `gorm:"type:varchar(100);not null"`
	LastName     string    `gorm:"type:varchar(100);not null"`
	Role         UserRole  `gorm:"type:varchar(50);not null;index"`
	HospitalID   *uuid.UUID `gorm:"type:uuid;index"`
	DepartmentID *uuid.UUID `gorm:"type:uuid"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	IsActive     bool      `gorm:"default:true;index"`
	MFASecretEnc *string   `gorm:"type:text"`
	IsDeleted    bool      `gorm:"default:false"`
	DeletedAt    gorm.DeletedAt
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Relationships
	Hospital     *Hospital   `gorm:"foreignKey:HospitalID;constraint:OnDelete:SET NULL;"`
	Department   *Department `gorm:"foreignKey:DepartmentID;constraint:OnDelete:SET NULL;"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
