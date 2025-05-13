package repository

import (
	"Hospital-Referral-System/internal/domain/contract/repository_interface"
	"Hospital-Referral-System/internal/domain/entity"

	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) contract.RoleRepositoryInterface {
	return &RoleRepository{
		db: db,
	}
}

func (rolerepo *RoleRepository) GetRole() []*entity.Role {
	var roles []*entity.Role
	rolerepo.db.Find(&roles)
	return roles
}

func (rolerepo *RoleRepository) CreateRole(role *entity.Role) (*entity.Role, error) {
	result := rolerepo.db.Create(role)
	if result.Error != nil{
		return &entity.Role{}, result.Error
	}
	
	return role, nil
}

func (rolerepo *RoleRepository) UpdateRole(role *entity.Role) (*entity.Role, error) {
	var existingRole entity.Role
	if err := rolerepo.db.First(&existingRole, role.ID).Error; err != nil{
		return &entity.Role{}, err
	}

	err := rolerepo.db.Model(&existingRole).Updates(role).Error
	if err != nil {
		return &entity.Role{}, err
	}

	return &existingRole, nil
}

func(rolerepo *RoleRepository) DeleteRole(roleid uint) error {
	err := rolerepo.db.Delete(&entity.Role{}, roleid).Error
	if err != nil {
		return err
	}

	return nil
}