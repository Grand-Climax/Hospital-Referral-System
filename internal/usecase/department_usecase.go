package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
)

var (
	ErrDepartmentNotFound       = errors.New("department not found")
	ErrHospitalDeptLinkExists   = errors.New("department is already linked to this hospital")
	ErrHospitalDeptLinkNotFound = errors.New("department is not linked to this hospital")
)

type DepartmentUseCase interface {
	CreateDepartment(ctx context.Context, dept *entity.Department) error
	GetDepartmentByID(ctx context.Context, id uuid.UUID) (*entity.Department, error)
	UpdateDepartment(ctx context.Context, dept *entity.Department) error
	DeleteDepartment(ctx context.Context, id uuid.UUID) error
	ListDepartments(ctx context.Context, filter repository.DepartmentListFilter) ([]entity.Department, int64, error)
	LinkDepartmentToHospital(ctx context.Context, hospitalID, departmentID uuid.UUID, dailyLimit int) error
	UnlinkDepartmentFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error
	ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error)
}

type departmentUseCase struct {
	repo     repository.DepartmentRepository
	hospRepo repository.HospitalRepository
}

func NewDepartmentUseCase(repo repository.DepartmentRepository, hospRepo repository.HospitalRepository) DepartmentUseCase {
	return &departmentUseCase{repo: repo, hospRepo: hospRepo}
}

func (u *departmentUseCase) CreateDepartment(ctx context.Context, dept *entity.Department) error {
	return u.repo.Create(ctx, dept)
}

func (u *departmentUseCase) GetDepartmentByID(ctx context.Context, id uuid.UUID) (*entity.Department, error) {
	dept, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrDepartmentNotFound
	}
	return dept, nil
}

func (u *departmentUseCase) UpdateDepartment(ctx context.Context, dept *entity.Department) error {
	existing, err := u.repo.FindByID(ctx, dept.ID)
	if err != nil {
		return ErrDepartmentNotFound
	}
	dept.CreatedAt = existing.CreatedAt
	return u.repo.Update(ctx, dept)
}

func (u *departmentUseCase) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	_, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return ErrDepartmentNotFound
	}
	return u.repo.Delete(ctx, id)
}

func (u *departmentUseCase) ListDepartments(ctx context.Context, filter repository.DepartmentListFilter) ([]entity.Department, int64, error) {
	return u.repo.ListDepartments(ctx, filter)
}

func (u *departmentUseCase) LinkDepartmentToHospital(ctx context.Context, hospitalID, departmentID uuid.UUID, dailyLimit int) error {
	// Verify hospital exists
	hosp, err := u.hospRepo.FindByID(ctx, hospitalID)
	if err != nil || hosp.IsDeleted {
		return ErrHospitalNotFound
	}

	// Verify department exists
	_, err = u.repo.FindByID(ctx, departmentID)
	if err != nil {
		return ErrDepartmentNotFound
	}

	// Check if link already exists
	existing, _ := u.repo.FindHospitalDepartment(ctx, hospitalID, departmentID)
	if existing != nil {
		return ErrHospitalDeptLinkExists
	}

	if dailyLimit <= 0 {
		dailyLimit = 20
	}

	link := &entity.HospitalDepartment{
		HospitalID:         hospitalID,
		DepartmentID:       departmentID,
		StandardDailyLimit: dailyLimit,
		IsActive:           true,
	}
	return u.repo.LinkToHospital(ctx, link)
}

func (u *departmentUseCase) UnlinkDepartmentFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error {
	existing, _ := u.repo.FindHospitalDepartment(ctx, hospitalID, departmentID)
	if existing == nil {
		return ErrHospitalDeptLinkNotFound
	}
	return u.repo.UnlinkFromHospital(ctx, hospitalID, departmentID)
}

func (u *departmentUseCase) ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error) {
	// Verify hospital exists
	hosp, err := u.hospRepo.FindByID(ctx, hospitalID)
	if err != nil || hosp.IsDeleted {
		return nil, ErrHospitalNotFound
	}
	return u.repo.ListHospitalDepartments(ctx, hospitalID)
}
