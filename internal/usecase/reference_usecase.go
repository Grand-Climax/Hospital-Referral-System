package usecase

import (
	"context"

	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type referenceUseCase struct {
	referenceRepo irepository.ReferenceRepository
}

func NewReferenceUseCase(repo irepository.ReferenceRepository) iusecase.ReferenceUseCase {
	return &referenceUseCase{referenceRepo: repo}
}

func (u *referenceUseCase) GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error) {
	return u.referenceRepo.GetHospitals(ctx, tier)
}

func (u *referenceUseCase) GetDepartments(ctx context.Context) ([]entity.Department, error) {
	return u.referenceRepo.GetDepartments(ctx)
}

func (u *referenceUseCase) ListICDCodes(ctx context.Context, search string, category string, page int, pageSize int) ([]entity.ICDCode, int64, error) {
	return u.referenceRepo.ListICDCodes(ctx, search, category, page, pageSize)
}

func (u *referenceUseCase) ListICDCategories(ctx context.Context) ([]string, error) {
	categories, err := u.referenceRepo.ListICDCategories(ctx)
	if err != nil {
		return nil, err
	}
	if len(categories) == 0 {
		return []string{
			"Blood & Immune Disorders",
			"Cancers & Tumors",
			"Circulatory System Diseases",
			"Conditions Originating in Perinatal Period",
			"Congenital Malformations & Chromosomal Abnormalities",
			"Digestive System Diseases",
			"Ear & Mastoid Diseases",
			"Endocrine, Nutritional & Metabolic Diseases",
			"External Causes of Morbidity & Mortality",
			"Eye & Adnexa Diseases",
			"Genitourinary System Diseases",
			"Infectious & Parasitic Diseases",
			"Injury, Poisoning & External Causes",
			"Mental & Behavioral Disorders",
			"Musculoskeletal & Connective Tissue Diseases",
			"Nervous System Diseases",
			"Pregnancy, Childbirth & Puerperium",
			"Respiratory System Diseases",
			"Skin & Subcutaneous Tissue Diseases",
			"Symptoms, Signs & Abnormal Findings",
			"Unknown",
		}, nil
	}
	return categories, nil
}

func (u *referenceUseCase) GetNetworkedHospitals(ctx context.Context, senderHospitalID uuid.UUID) ([]entity.Hospital, error) {
	hospitals, err := u.referenceRepo.GetNetworkedHospitals(ctx, senderHospitalID)
	if err != nil {
		return nil, err
	}
	return hospitals, nil
}

func (u *referenceUseCase) GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error) {
	return u.referenceRepo.GetHospitalDepartments(ctx, hospitalID)
}

func (u *referenceUseCase) GetLiaisonsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error) {
	return u.referenceRepo.GetLiaisonsByHospital(ctx, hospitalID)
}
