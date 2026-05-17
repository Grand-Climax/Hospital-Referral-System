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
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
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

func newTestUserUC(repo *MockUserRepo, svc *MockStorageService) iusecase.UserUseCase {
	mnotif := new(MockInAppNotificationUseCase)
	mnotif.On("CreateForEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	return usecase.NewUserUseCase(repo, svc, mnotif)
}

// ---------------------------------------------------------------------------
// Tests: canSeeTarget visibility matrix (exercised via GetUserByID)
// ---------------------------------------------------------------------------

func TestGetUserByID_MohAnalyst_CannotViewOthers(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

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
	uc := newTestUserUC(repo, svc)

	existing := newUser(entity.RoleReferringDoctor, nil)
	repo.On("FindByEmail", mock.Anything, "test@example.com").Return(existing, nil)

	newU := &entity.User{Email: "test@example.com", Role: entity.RoleReferringDoctor}
	err := uc.CreateUser(context.Background(), newU, "password123")
	assert.ErrorIs(t, err, usecase.ErrEmailExists)
}

func TestCreateUser_InvalidRole_ReturnsError(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	newU := &entity.User{Email: "test@example.com", Role: "INVALID_ROLE"}
	err := uc.CreateUser(context.Background(), newU, "password123")
	assert.ErrorIs(t, err, usecase.ErrInvalidRole)
}

func TestCreateUser_RequesterNotFound_OnList(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	repo.On("FindByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	_, _, err := uc.ListUsers(context.Background(), irepository.UserListFilter{}, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestHospitalAdminCreateStaff_ScopesToAdminHospital(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	adminHosp := hospitalID()
	admin := newUser(entity.RoleHospitalAdmin, adminHosp)
	newStaff := &entity.User{
		Email:      "new.staff@hospital.et",
		FirstName:  "New",
		MiddleName: "Staff",
		LastName:   "User",
		Role:       entity.RoleLiaisonOfficer,
	}

	repo.On("FindByID", mock.Anything, admin.ID).Return(admin, nil)
	repo.On("FindByEmail", mock.Anything, newStaff.Email).Return(nil, errors.New("not found"))
	repo.On("Create", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.HospitalID != nil && *u.HospitalID == *adminHosp && u.Role == entity.RoleLiaisonOfficer
	})).Return(nil)

	err := uc.HospitalAdminCreateStaff(context.Background(), admin.ID, newStaff, "password123")
	assert.NoError(t, err)
}

func TestHospitalAdminChangeRole_DeniesCrossHospital(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	adminHosp := hospitalID()
	otherHosp := hospitalID()
	admin := newUser(entity.RoleHospitalAdmin, adminHosp)
	target := newUser(entity.RoleReceptionist, otherHosp)

	repo.On("FindByID", mock.Anything, admin.ID).Return(admin, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)

	err := uc.HospitalAdminChangeStaffRole(context.Background(), admin.ID, target.ID, entity.RoleLiaisonOfficer)
	assert.ErrorIs(t, err, usecase.ErrForbiddenStaffScope)
}

func TestHospitalAdminSoftDeleteStaff_SetsSoftDeleteFlags(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	adminHosp := hospitalID()
	admin := newUser(entity.RoleHospitalAdmin, adminHosp)
	target := newUser(entity.RoleLiaisonOfficer, adminHosp)

	repo.On("FindByID", mock.Anything, admin.ID).Return(admin, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == target.ID && u.IsDeleted && !u.IsActive
	})).Return(nil)

	err := uc.HospitalAdminSoftDeleteStaff(context.Background(), admin.ID, target.ID)
	assert.NoError(t, err)
}

func TestHospitalAdminReplaceStaff_InPlaceAndLogged(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	adminHosp := hospitalID()
	admin := newUser(entity.RoleHospitalAdmin, adminHosp)
	target := newUser(entity.RoleLiaisonOfficer, adminHosp)
	target.Email = "old@hospital.et"

	repo.On("FindByID", mock.Anything, admin.ID).Return(admin, nil)
	repo.On("FindByID", mock.Anything, target.ID).Return(target, nil)
	repo.On("FindByEmail", mock.Anything, "new@hospital.et").Return(nil, errors.New("not found"))
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == target.ID && u.Email == "new@hospital.et" && u.PasswordHash != ""
	})).Return(nil)
	repo.On("CreateStaffReplacementLog", mock.Anything, mock.MatchedBy(func(l *entity.StaffReplacementLog) bool {
		return l.UserID == target.ID &&
			l.HospitalID == *adminHosp &&
			l.OldEmail == "old@hospital.et" &&
			l.NewEmail == "new@hospital.et" &&
			l.Reason == "staff transition"
	})).Return(nil)

	err := uc.HospitalAdminReplaceStaff(context.Background(), admin.ID, target.ID, iusecase.HospitalAdminReplacementInput{
		FirstName:  "New",
		MiddleName: "Middle",
		LastName:   "Name",
		Email:      "new@hospital.et",
		Password:   "newStrongPass123",
		Reason:     "staff transition",
	})
	assert.NoError(t, err)
}

func TestUpdateUser_PasswordPreservation_WhenPasswordIsEmpty(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	existing := newUser(entity.RoleReferringDoctor, nil)
	existing.PasswordHash = "old_hashed_password"

	repo.On("FindByID", mock.Anything, existing.ID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == existing.ID && u.PasswordHash == "old_hashed_password"
	})).Return(nil)

	// Update user with an empty PasswordHash
	userToUpdate := &entity.User{
		ID:           existing.ID,
		PasswordHash: "",
	}

	err := uc.UpdateUser(context.Background(), userToUpdate)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateUser_PasswordUpdate_WhenPasswordIsNotEmpty(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	existing := newUser(entity.RoleReferringDoctor, nil)
	existing.PasswordHash = "old_hashed_password"

	repo.On("FindByID", mock.Anything, existing.ID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == existing.ID && u.PasswordHash == "new_hashed_password"
	})).Return(nil)

	// Update user with a non-empty PasswordHash
	userToUpdate := &entity.User{
		ID:           existing.ID,
		PasswordHash: "new_hashed_password",
	}

	err := uc.UpdateUser(context.Background(), userToUpdate)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateUser_UniquenessValidation(t *testing.T) {
	repo := new(MockUserRepo)
	svc := new(MockStorageService)
	uc := newTestUserUC(repo, svc)

	uID := uuid.New()
	existing := &entity.User{
		ID:         uID,
		Email:      "old@test.com",
		NationalID: "NAT-OLD",
	}

	t.Run("Fails if updating to an already taken email", func(t *testing.T) {
		repo.On("FindByID", mock.Anything, uID).Return(existing, nil).Once()
		
		takenUser := &entity.User{
			ID:    uuid.New(),
			Email: "taken@test.com",
		}
		repo.On("FindByEmail", mock.Anything, "taken@test.com").Return(takenUser, nil).Once()

		updatedUser := &entity.User{
			ID:    uID,
			Email: "taken@test.com",
		}

		err := uc.UpdateUser(context.Background(), updatedUser)
		assert.Equal(t, usecase.ErrEmailExists, err)
	})

	t.Run("Fails if updating to an already taken national ID", func(t *testing.T) {
		repo.On("FindByID", mock.Anything, uID).Return(existing, nil).Once()
		
		takenUser := &entity.User{
			ID:         uuid.New(),
			NationalID: "NAT-TAKEN",
		}
		repo.On("FindByNationalID", mock.Anything, "NAT-TAKEN").Return(takenUser, nil).Once()

		updatedUser := &entity.User{
			ID:         uID,
			Email:      "old@test.com",
			NationalID: "NAT-TAKEN",
		}

		err := uc.UpdateUser(context.Background(), updatedUser)
		assert.Equal(t, usecase.ErrNationalIDExists, err)
	})
}


