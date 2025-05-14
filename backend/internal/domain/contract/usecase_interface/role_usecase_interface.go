package usecaseinterface

import "Hospital-Referral-System/internal/domain/entity"

type RoleUsecaseInterface interface {
	GetRole() []*entity.Role
	CreateRole(role *entity.Role) (*entity.Role, error)
	UpdateRole(role *entity.Role) (*entity.Role, error)
	DeleteRole(role_id uint) error
}