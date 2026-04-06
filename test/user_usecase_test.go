package test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	"Hospital-Referral-System/internal/usecase"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hospitalID() *uuid.UUID {
	id := uuid.New()
	return &id
}

func newUser(role entity.UserRole, hospID *uuid.UUID) *entity.User {
	return &entity.User{
		ID:         uuid.New(),
		Role:       role,
		HospitalID: hospID,
		IsActive:   true,
		IsDeleted:  false,
	}
}

// ---------------------------------------------------------------------------
// Tests: canSeeTarget visibility matrix (exercised via GetUserByID)
// ---------------------------------------------------------------------------

func TestGetUserByID_MohAnalyst_CannotViewOthers(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	requester := newUser(entity.RoleMohAnalyst, nil)
	target := newUser(entity.RoleReferringDoctor, hospitalID())

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	_, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGetUserByID_MohAnalyst_CanViewSelf(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	requester := newUser(entity.RoleMohAnalyst, nil)

	// ID lookup returns the same user twice (self lookup)
	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)

	result, err := uc.GetUserByID(context.Background(), requester.ID, requester.ID)
	assert.NoError(t, err)
	assert.Equal(t, requester.ID, result.ID)
}

func TestGetUserByID_Receptionist_CannotViewOtherHospital(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	hospA := hospitalID()
	hospB := hospitalID()
	requester := newUser(entity.RoleReceptionist, hospA)
	target := newUser(entity.RoleReferringDoctor, hospB)

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	_, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGetUserByID_Receptionist_CanViewSameHospital(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	hosp := hospitalID()
	requester := newUser(entity.RoleReceptionist, hosp)
	target := newUser(entity.RoleLiaisonOfficer, hosp)

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	result, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.NoError(t, err)
	assert.Equal(t, target.ID, result.ID)
}

func TestGetUserByID_Doctor_CannotViewSystemAdmin(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	requester := newUser(entity.RoleReferringDoctor, hospitalID())
	target := newUser(entity.RoleSystemSuperAdmin, nil)

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	_, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGetUserByID_Doctor_CanViewOtherHospitalSpecialist(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	hospA := hospitalID()
	hospB := hospitalID()
	requester := newUser(entity.RoleReferringDoctor, hospA)
	target := newUser(entity.RoleReceivingSpecialist, hospB)

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	result, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.NoError(t, err)
	assert.Equal(t, target.ID, result.ID)
}

func TestGetUserByID_SystemAdmin_CanViewAnyone(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	requester := newUser(entity.RoleSystemSuperAdmin, nil)
	target := newUser(entity.RoleMohAnalyst, nil)

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	result, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.NoError(t, err)
	assert.Equal(t, target.ID, result.ID)
}

func TestGetUserByID_Doctor_CannotViewReceptionistFromOtherHospital(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	hospA := hospitalID()
	hospB := hospitalID()
	requester := newUser(entity.RoleReferringDoctor, hospA)
	target := newUser(entity.RoleReceptionist, hospB) // different hospital receptionist

	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	_, err := uc.GetUserByID(context.Background(), target.ID, requester.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

// ---------------------------------------------------------------------------
// Tests: ListUsers – role-based hospital isolation
// ---------------------------------------------------------------------------

func TestListUsers_MohAnalyst_ReturnsEmpty(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	requester := newUser(entity.RoleMohAnalyst, nil)
	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)

	users, count, err := uc.ListUsers(context.Background(), irepository.UserListFilter{}, requester.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Empty(t, users)
	// Repo ListUsers should NOT have been called
	repo.AssertNotCalled(t, "ListUsers")
}

func TestListUsers_Receptionist_FiltersToOwnHospital(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	hosp := hospitalID()
	requester := newUser(entity.RoleReceptionist, hosp)
	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)

	hospStr := hosp.String()
	expectedFilter := irepository.UserListFilter{HospitalID: &hospStr}
	repo.On("ListUsers", mock.Anything, mock.MatchedBy(func(f irepository.UserListFilter) bool {
		return f.HospitalID != nil && *f.HospitalID == hosp.String()
	})).Return([]entity.User{}, int64(0), nil)

	_, _, err := uc.ListUsers(context.Background(), expectedFilter, requester.ID)
	assert.NoError(t, err)
	repo.AssertNumberOfCalls(t, "ListUsers", 1)
}

func TestListUsers_Doctor_ExcludesSystemAdminRole(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	hosp := hospitalID()
	requester := newUser(entity.RoleReferringDoctor, hosp)
	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("ListUsers", mock.Anything, mock.MatchedBy(func(f irepository.UserListFilter) bool {
		for _, excluded := range f.ExcludeRoles {
			if excluded == entity.RoleSystemSuperAdmin {
				return true
			}
		}
		return false
	})).Return([]entity.User{}, int64(0), nil)

	_, _, err := uc.ListUsers(context.Background(), irepository.UserListFilter{}, requester.ID)
	assert.NoError(t, err)
	repo.AssertNumberOfCalls(t, "ListUsers", 1)
}

func TestListUsers_SystemAdmin_NoHospitalFilter(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	requester := newUser(entity.RoleSystemSuperAdmin, nil)
	repo.On("FindByID", mock.Anything, requester.ID).Return(requester, nil)
	repo.On("ListUsers", mock.Anything, mock.MatchedBy(func(f irepository.UserListFilter) bool {
		return f.HospitalID == nil // no hospital restriction
	})).Return([]entity.User{}, int64(0), nil)

	_, _, err := uc.ListUsers(context.Background(), irepository.UserListFilter{}, requester.ID)
	assert.NoError(t, err)
	repo.AssertNumberOfCalls(t, "ListUsers", 1)
}

// ---------------------------------------------------------------------------
// Tests: CreateUser
// ---------------------------------------------------------------------------

func TestCreateUser_DuplicateEmail_ReturnsError(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	existing := newUser(entity.RoleReferringDoctor, nil)
	repo.On("FindByEmail", mock.Anything, "test@example.com").Return(existing, nil)

	newU := &entity.User{Email: "test@example.com", Role: entity.RoleReferringDoctor}
	err := uc.CreateUser(context.Background(), newU, "password123")
	assert.ErrorIs(t, err, usecase.ErrEmailExists)
}

func TestCreateUser_InvalidRole_ReturnsError(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	newU := &entity.User{Email: "test@example.com", Role: "INVALID_ROLE"}
	err := uc.CreateUser(context.Background(), newU, "password123")
	assert.ErrorIs(t, err, usecase.ErrInvalidRole)
}

func TestCreateUser_RequesterNotFound_OnList(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := usecase.NewUserUseCase(repo, svc)

	repo.On("FindByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	_, _, err := uc.ListUsers(context.Background(), irepository.UserListFilter{}, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}
