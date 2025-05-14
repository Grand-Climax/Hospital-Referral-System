package mockusecase

import (
	"Hospital-Referral-System/internal/domain/entity"

	"github.com/stretchr/testify/mock"
)

type MockRoleUsecase struct {
	mock.Mock
}

func (m *MockRoleUsecase) CreateRole(role *entity.Role) (*entity.Role, error) {
	args := m.Called(role)
	result := args.Get(0)

	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Role), args.Error(1)
}

func (m *MockRoleUsecase) GetRole() []*entity.Role {
	args := m.Called()
	result := args.Get(0)

	if result == nil {
		return nil
	}
	return result.([]*entity.Role)
}

func (m *MockRoleUsecase) UpdateRole(role *entity.Role) (*entity.Role, error) {
	args := m.Called(role)
	result := args.Get(0)

	if result == nil {
		return nil, args.Error(1)
	}

	return result.(*entity.Role), args.Error(1)
}

func (m *MockRoleUsecase) DeleteRole(roleid uint) error {
	args := m.Called(roleid)
	return args.Error(0)
}