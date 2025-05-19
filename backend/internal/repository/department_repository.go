package repository

import (
	repositoryinterface "Hospital-Referral-System/internal/domain/contract/repository_interface"
	"Hospital-Referral-System/internal/domain/entity"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) repositoryinterface.DepartmentRepositoryInterface{
	return &DepartmentRepository{db:db}
}

func(repo *DepartmentRepository) CreateDepartment(department *entity.Department) (*entity.Department, error){
	result := repo.db.Create(department)
	if result.Error != nil {
		return &entity.Department{}, result.Error
	}

	return department, nil
}

func (repo *DepartmentRepository) GetDepartments() ([]*entity.Department, error){
	var departments []*entity.Department

	if result := repo.db.Find(&departments); result.Error != nil{
		return []*entity.Department{}, result.Error
	}
	return departments, nil
}

func (repo *DepartmentRepository) UpdateDepartment(department *entity.Department) (*entity.Department, error){
	department_id := department.ID
	var existingRole entity.Department

	if err := repo.db.First(&existingRole, department_id).Error; err != nil{
		return &entity.Department{}, err
	}

	if err := repo.db.Model(&existingRole).Updates(department).Error; err != nil {
		return &entity.Department{}, err
	}

	return department, nil
}

func (repo *DepartmentRepository) DeleteDepartment(departmentid uint) error{

	if err := repo.db.Delete(&entity.Department{}, departmentid).Error; err != nil {
		return err
	}
	return nil
}