package usecase

import (
	repositoryinterface "Hospital-Referral-System/internal/domain/contract/repository_interface"
	usecaseinterface "Hospital-Referral-System/internal/domain/contract/usecase_interface"
	"Hospital-Referral-System/internal/domain/entity"
)

type DepartmentUsecase struct {
	repo repositoryinterface.DepartmentRepositoryInterface
}

func NewDepartmentUsecase(repo repositoryinterface.DepartmentRepositoryInterface) usecaseinterface.DepartmentUsecaseInterface{
	return &DepartmentUsecase{repo:repo}
}

func (usecase *DepartmentUsecase) CreateDepartment(department *entity.Department) (*entity.Department, error) {
	return usecase.repo.CreateDepartment(department)
}

func (usecase *DepartmentUsecase) GetDepartments() ([]*entity.Department, error){
	return usecase.repo.GetDepartments()
}

func (usecase *DepartmentUsecase) UpdateDepartment(department *entity.Department) (*entity.Department, error){
	return usecase.repo.UpdateDepartment(department)
}

func (usecase *DepartmentUsecase) DeleteDepartment(department_id uint) error{
	return usecase.repo.DeleteDepartment(department_id)
}