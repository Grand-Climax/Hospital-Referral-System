package mockrepository

import (
	"Hospital-Referral-System/internal/domain/entity"

	"github.com/stretchr/testify/mock"
)

type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) CreateRole(role *entity.Role) (*entity.Role, error) {
	args := m.Called(role)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Role), args.Error(1)
}

func (m *MockRoleRepository) GetRole() []*entity.Role {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}

	return result.([]*entity.Role)
}

func (m *MockRoleRepository) UpdateRole(role *entity.Role) (*entity.Role, error) {
	args := m.Called(role)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*entity.Role), args.Error(1)
}

func (m *MockRoleRepository) DeleteRole(roleid uint) error {
	args := m.Called(roleid)
	return args.Error(0)
}