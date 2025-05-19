package usecaseinterface

import "Hospital-Referral-System/internal/domain/entity"

type DepartmentUsecaseInterface interface {
	CreateDepartment(department *entity.Department) (*entity.Department, error)
	GetDepartments() ([]*entity.Department, error)
	UpdateDepartment(updatedepartment *entity.Department) (*entity.Department, error)
	DeleteDepartment(departmentid uint) error
}