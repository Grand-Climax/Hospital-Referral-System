package repositoryinterface

import "Hospital-Referral-System/internal/domain/entity"

type HospitalTypeRepositoryInterface interface {
	CreateHospitalType(hospitaltype *entity.HospitalType) (*entity.HospitalType, error)
	GetHospitalTypes() ([]*entity.HospitalType, error)
	GetHospitalTypeByName(name string) (*entity.HospitalType, error)
	GetHospitalTypeByID(id uint) (*entity.HospitalType, error)
	UpdateHospitalType(hospitaltype *entity.HospitalType) (*entity.HospitalType, error)
	DeleteHospitalType(hospitaltypeid uint)  error
}