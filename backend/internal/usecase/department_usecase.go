package usecase

import (
	repositoryinterface "Hospital-Referral-System/internal/domain/contract/repository_interface"
	usecaseinterface "Hospital-Referral-System/internal/domain/contract/usecase_interface"
	"Hospital-Referral-System/internal/domain/entity"
	customerrors "Hospital-Referral-System/internal/errors"
	"errors"
)

type DepartmentUsecase struct {
	repo repositoryinterface.DepartmentRepositoryInterface
}

func NewDepartmentUsecase(repo repositoryinterface.DepartmentRepositoryInterface) usecaseinterface.DepartmentUsecaseInterface{
	return &DepartmentUsecase{repo:repo}
}

func (usecase *DepartmentUsecase) CreateDepartment(department *entity.Department) (*entity.Department, error) {
	name := department.Name
	_, err := usecase.repo.GetDepartmentByName(name)
	if err == nil {
		return nil, customerrors.ErrDuplicate
	}

	if !errors.Is(err, customerrors.ErrNotFound) {
		return nil, customerrors.ErrInternal
	}
	return usecase.repo.CreateDepartment(department)
}

func (usecase *DepartmentUsecase) GetDepartmentByName(name string) (*entity.Department, error) {
	return usecase.repo.GetDepartmentByName(name)
}

func (usecase *DepartmentUsecase) GetDepartments() ([]*entity.Department, error){
	return usecase.repo.GetDepartments()
}

func (usecase *DepartmentUsecase) UpdateDepartment(department *entity.Department) (*entity.Department, error){
	_, err := usecase.repo.GetDepartmentByID(department.ID)
	if err != nil {
		return nil, err
	}
	return usecase.repo.UpdateDepartment(department)
}

func (usecase *DepartmentUsecase) DeleteDepartment(department_id uint) error{
	_, err := usecase.repo.GetDepartmentByID(department_id)
	if err != nil {
		return err
	}
	return usecase.repo.DeleteDepartment(department_id)
}