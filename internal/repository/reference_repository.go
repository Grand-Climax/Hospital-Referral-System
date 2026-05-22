package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type referenceRepository struct {
	db *gorm.DB
}

func NewReferenceRepository(db *gorm.DB) irepository.ReferenceRepository {
	return &referenceRepository{db: db}
}

func (r *referenceRepository) GetHospitals(ctx context.Context, tier string) ([]entity.Hospital, error) {
	var hospitals []entity.Hospital
	q := r.db.WithContext(ctx).Where("is_active = ? AND is_deleted = ?", true, false)
	if tier != "" {
		q = q.Where("tier_level = ?", tier)
	}
	err := q.Order("name asc").Find(&hospitals).Error
	return hospitals, err
}

func (r *referenceRepository) GetDepartments(ctx context.Context) ([]entity.Department, error) {
	var depts []entity.Department
	err := r.db.WithContext(ctx).Order("name asc").Find(&depts).Error
	return depts, err
}

func (r *referenceRepository) ListICDCodes(ctx context.Context, search string, category string, page int, pageSize int) ([]entity.ICDCode, int64, error) {
	var codes []entity.ICDCode
	var total int64

	q := r.db.WithContext(ctx).Model(&entity.ICDCode{})

	if category != "" {
		q = q.Where("category = ?", category)
	}

	if search != "" {
		// Check both code and description case-insensitively
		q = q.Where("LOWER(code) LIKE ? OR LOWER(description) LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize

	err := q.Order("code asc").Limit(pageSize).Offset(offset).Find(&codes).Error
	return codes, total, err
}

func (r *referenceRepository) ListICDCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).
		Model(&entity.ICDCode{}).
		Where("category IS NOT NULL AND category != ''").
		Order("category asc").
		Distinct("category").
		Pluck("category", &categories).Error
	return categories, err
}

func (r *referenceRepository) GetNetworkedHospitals(ctx context.Context, senderHospitalID uuid.UUID) ([]entity.Hospital, error) {
	var hospitals []entity.Hospital
	// Fetch hospitals joined through ReferralNetwork
	err := r.db.WithContext(ctx).
		Joins("JOIN referral_networks ON referral_networks.receiver_hospital_id = hospitals.id").
		Where("referral_networks.sender_hospital_id = ? AND hospitals.is_active = ? AND hospitals.is_deleted = ?", senderHospitalID, true, false).
		Find(&hospitals).Error
	return hospitals, err
}

func (r *referenceRepository) GetHospitalDepartments(ctx context.Context, hospitalID uuid.UUID) ([]entity.Department, error) {
	var depts []entity.Department
	// Fetch departments joined through HospitalDepartment mapping
	err := r.db.WithContext(ctx).
		Joins("JOIN hospital_departments ON hospital_departments.department_id = departments.id").
		Where("hospital_departments.hospital_id = ? AND hospital_departments.is_active = ?", hospitalID, true).
		Find(&depts).Error
	return depts, err
}

func (r *referenceRepository) GetLiaisonsByHospital(ctx context.Context, hospitalID uuid.UUID) ([]entity.User, error) {
	var liaisons []entity.User
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND role = ? AND is_active = ? AND is_deleted = ?",
			hospitalID, entity.RoleLiaisonOfficer, true, false).
		Order("first_name asc").
		Find(&liaisons).Error
	return liaisons, err
}
