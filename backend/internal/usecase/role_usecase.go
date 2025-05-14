package usecase

import (
	repository "Hospital-Referral-System/internal/domain/contract/repository_interface"
	usecase "Hospital-Referral-System/internal/domain/contract/usecase_interface"
	"Hospital-Referral-System/internal/domain/entity"
)

type RoleUsecase struct {
	roleRepo repository.RoleRepositoryInterface
}

func NewRoleUsecase(rolerepo repository.RoleRepositoryInterface) usecase.RoleUsecaseInterface {
	return &RoleUsecase{roleRepo: rolerepo}
}

func (usecase *RoleUsecase) CreateRole(role *entity.Role) (*entity.Role, error) {
	// search for the name of the given role using elastic search to be done later but for now simply assume that the given role doesn't exist in the database
	return usecase.roleRepo.CreateRole(role)
}

func (usecase *RoleUsecase) GetRole() []*entity.Role{
	return usecase.roleRepo.GetRole()
}

func (usecase *RoleUsecase) UpdateRole(role *entity.Role) (*entity.Role, error) {
	return usecase.roleRepo.UpdateRole(role)
}

func (usecase *RoleUsecase) DeleteRole(roleid uint) error {
	return usecase.roleRepo.DeleteRole(roleid)
}