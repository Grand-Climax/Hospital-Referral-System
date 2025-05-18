package repositoryinterface

import "Hospital-Referral-System/internal/domain/entity"

type DepartmentRepositoryInterface interface {
	CreateDepartment(department *entity.Department) (*entity.Department, error)
	GetDepartments() ([]*entity.Department, error)
	UpdateDepartment(updatedepartment *entity.Department) (*entity.Department, error) 
	DeleteDepartment(departmentid uint) error
}