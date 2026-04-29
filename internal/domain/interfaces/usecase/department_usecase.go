package interfaces

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type DepartmentUseCase interface {
	CreateDepartment(ctx context.Context, dept *entity.Department) error
	GetDepartmentByID(ctx context.Context, id uuid.UUID) (*entity.Department, error)
	UpdateDepartment(ctx context.Context, dept *entity.Department) error
	DeleteDepartment(ctx context.Context, id uuid.UUID) error
	ListDepartments(ctx context.Context, filter irepository.DepartmentListFilter) ([]entity.Department, int64, error)
	LinkDepartmentToHospital(ctx context.Context, hospitalID, departmentID uuid.UUID, dailyLimit int) error
	UnlinkDepartmentFromHospital(ctx context.Context, hospitalID, departmentID uuid.UUID) error
	ListHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.HospitalDepartment, error)
	SetHospitalDepartmentActive(ctx context.Context, hospitalID, departmentID uuid.UUID, isActive bool) error
}
