package repository

import (
	repositoryinterface "Hospital-Referral-System/internal/domain/contract/repository_interface"
	"Hospital-Referral-System/internal/domain/entity"
	customerrors "Hospital-Referral-System/internal/errors"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) repositoryinterface.DepartmentRepositoryInterface{
	return &DepartmentRepository{db:db}
}

func(repo *DepartmentRepository) CreateDepartment(department *entity.Department) (*entity.Department, error){
	err := repo.db.Create(department).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key"){
			return nil , customerrors.ErrConflict
		}
		return nil, customerrors.ErrInternal
	}

	return department, nil
}

func (repo *DepartmentRepository) GetDepartmentByID(department_id uint) (*entity.Department, error) {
	var department entity.Department
	err := repo.db.Where("id = ?", department_id).First(&department).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customerrors.ErrNotFound
		}

		return nil, customerrors.ErrInternal
	}

	return &department, nil
}

func (repo *DepartmentRepository) GetDepartmentByName(name string) (*entity.Department, error) {
	var department entity.Department
	err := repo.db.Where("name ILIKE ?",name).First(&department).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			return nil, customerrors.ErrNotFound
		}

		return nil, customerrors.ErrInternal
	}

	return &department, nil 
}

func (repo *DepartmentRepository) GetDepartments() ([]*entity.Department, error){
	var departments []*entity.Department

	if result := repo.db.Find(&departments); result.Error != nil{
		return nil, customerrors.ErrInternal
	}
	return departments, nil
}

func (repo *DepartmentRepository) UpdateDepartment(department *entity.Department) (*entity.Department, error){

	if err := repo.db.Model(&entity.Department{}).Where("id = ?", department.ID).Updates(department).Error; err != nil {
		return nil, customerrors.ErrInternal
	}

	return department, nil
}

func (repo *DepartmentRepository) DeleteDepartment(departmentid uint) error{

	if err := repo.db.Delete(&entity.Department{}, departmentid).Error; err != nil {
		return err
	}
	return nil
}