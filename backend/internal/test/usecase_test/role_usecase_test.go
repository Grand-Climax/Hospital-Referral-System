package usecasetest

import (
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/usecase"
	mockrepository "Hospital-Referral-System/internal/test/mocks/mock_repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateRole(t *testing.T) {
	mockrepo := new(mockrepository.MockRoleRepository)
	roleusecase := usecase.NewRoleUsecase(mockrepo)

	inputRole := &entity.Role{Name: "Admin"}
	expectedRole := &entity.Role{ID: 1, Name: "Admin"}

	mockrepo.On("CreateRole",inputRole).Return(expectedRole,nil)

	result, err := roleusecase.CreateRole(inputRole)

	assert.NoError(t, err)
	assert.Equal(t,result,expectedRole)
	mockrepo.AssertExpectations(t)

}

func TestGetRole(t *testing.T) {
	mockrepo := new(mockrepository.MockRoleRepository)
	roleusecase := usecase.NewRoleUsecase(mockrepo)

	expectedRole := &entity.Role{ID: 1, Name: "Admin"}
	roles := []*entity.Role{expectedRole}
	
	mockrepo.On("GetRole").Return(roles)

	result := roleusecase.GetRole()

	assert.Equal(t, roles, result)
	mockrepo.AssertExpectations(t)
}

func TestUpdateRole(t *testing.T) {
	mockrepo := new(mockrepository.MockRoleRepository)
	roleusecase := usecase.NewRoleUsecase(mockrepo)

	inputRole := &entity.Role{ID: 1, Name: "Admin"}
	UpdatedRole := &entity.Role{ID: 1, Name: "Admin"}

	mockrepo.On("UpdateRole", inputRole).Return(UpdatedRole,nil)

	result, err := roleusecase.UpdateRole(inputRole)

	assert.NoError(t, err)
	assert.Equal(t, UpdatedRole, result)

	mockrepo.AssertExpectations(t)
}

func TestDeleteRole(t *testing.T) {
	mockrepo := new(mockrepository.MockRoleRepository)
	roleusecase := usecase.NewRoleUsecase(mockrepo)

	role_id := 1
	mockrepo.On("DeleteRole", uint(role_id)).Return(nil)

	err := roleusecase.DeleteRole(uint(role_id))

	assert.NoError(t, err)
	mockrepo.AssertExpectations(t)
}