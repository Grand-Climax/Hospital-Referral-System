package mockrepository

import (
	"Hospital-Referral-System/internal/domain/entity"

	"github.com/stretchr/testify/mock"
)

type MockDepartmentRepository struct {
	mock.Mock
}

func (m *MockDepartmentRepository) CreateDepartment(department *entity.Department) (*entity.Department, error) {
	args := m.Called(department)
	result := args.Get(0)

	if result == nil {
		return nil, args.Error(1)
	}

	return result.(*entity.Department), nil
}

func (m *MockDepartmentRepository) GetDepartments() ([]*entity.Department, error) {
	args := m.Called()
	result := args.Get(0)

	if result == nil {
		return nil, args.Error(1)
	}

	return result.([]*entity.Department), nil
}

func (m *MockDepartmentRepository) UpdateDepartment(department *entity.Department) (*entity.Department, error) {
	args := m.Called(department)
	result := args.Get(0)

	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Department), nil
}

func (m *MockDepartmentRepository) DeleteDepartment(department_id uint) error {
	args := m.Called(department_id)
	return args.Error(0)
}