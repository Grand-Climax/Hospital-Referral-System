package mockusecase

import (
	"Hospital-Referral-System/internal/domain/entity"

	"github.com/stretchr/testify/mock"
)

type MockDepartmentUsecase struct {
	mock.Mock
}

func NewDepartmentUsecaseMock() *MockDepartmentUsecase {
	return &MockDepartmentUsecase{}
}

func (m *MockDepartmentUsecase) CreateDepartment(department *entity.Department) (*entity.Department, error) {
	args := m.Called(department)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Department), args.Error(1)
}

func (m *MockDepartmentUsecase) GetDepartmentByName(name string) (*entity.Department, error) {
	args := m.Called(name)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Department), args.Error(1)
}

func (m *MockDepartmentUsecase) GetDepartments() ([]*entity.Department, error) {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.([]*entity.Department), args.Error(1)
}

func (m *MockDepartmentUsecase) UpdateDepartment(department *entity.Department) (*entity.Department, error) {
	args := m.Called(department)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Department), args.Error(1)
}

func (m *MockDepartmentUsecase) DeleteDepartment(departmentID uint) error {
	args := m.Called(departmentID)
	return args.Error(0)
}