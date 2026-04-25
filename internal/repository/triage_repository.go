package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

// triageRepository handles all TriageQueue persistence.
// It embeds BaseRepository for standard CRUD and adds domain-specific queries.
type triageRepository struct {
	*BaseRepository[entity.TriageQueue]
	db *gorm.DB
}

func NewTriageRepository(db *gorm.DB) irepository.TriageQueueRepository {
	return &triageRepository{
		BaseRepository: NewBaseRepository[entity.TriageQueue](db),
		db:             db,
	}
}

func (r *triageRepository) Create(ctx context.Context, queue *entity.TriageQueue) error {
	return r.db.WithContext(ctx).Create(queue).Error
}

func (r *triageRepository) GetByReferralID(ctx context.Context, referralID uuid.UUID) (*entity.TriageQueue, error) {
	var queue entity.TriageQueue
	err := r.db.WithContext(ctx).Where("referral_id = ?", referralID).First(&queue).Error
	return &queue, err
}

func (r *triageRepository) GetByHospitalAndStatus(ctx context.Context, hospitalID uuid.UUID, status entity.QueueStatus) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("dept_id IN (SELECT id FROM hospital_departments WHERE hospital_id = ?) AND queue_status = ?", hospitalID, status).
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) Update(ctx context.Context, queue *entity.TriageQueue) error {
	return r.db.WithContext(ctx).Save(queue).Error
}

func (r *triageRepository) DeleteByReferralID(ctx context.Context, referralID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("referral_id = ?", referralID).Delete(&entity.TriageQueue{}).Error
}

func (r *triageRepository) ListForTriage(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]entity.TriageQueue, int64, error) {
	var queues []entity.TriageQueue
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Where("hospital_id = ?", hospitalID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("composite_score desc").Limit(limit).Offset(offset).Find(&queues).Error
	return queues, count, err
}

func (r *triageRepository) ListScheduledInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Preload("Referral").
		Preload("Referral.Patient").
		Where("hospital_id = ? AND department_id = ? AND queue_status = 'SCHEDULED' AND appointment_date >= ? AND appointment_date <= ?", hospitalID, deptID, start, end).
		Order("appointment_date asc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) GetWaitingByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND queue_status = 'WAITING'", hospitalID, deptID).
		Order("composite_score desc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) FindWaitingByHospitalAndDept(ctx context.Context, hospitalID, departmentID uuid.UUID) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND queue_status = 'WAITING'", hospitalID, departmentID).
		Order("composite_score desc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) FindScheduledByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]*entity.TriageQueue, error) {
	var queues []*entity.TriageQueue
	err := r.db.WithContext(ctx).
		Preload("Referral").
		Preload("Referral.Patient").
		Where("hospital_id = ? AND department_id = ? AND queue_status = 'SCHEDULED' AND appointment_date BETWEEN ? AND ?", hospitalID, deptID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Order("appointment_date asc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) FindByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error) {
	var queues []*entity.TriageQueue
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Where("hospital_id = ? AND department_id = ?", hospitalID, deptID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := query.Limit(limit).Offset(offset).Find(&queues).Error
	return queues, count, err
}
