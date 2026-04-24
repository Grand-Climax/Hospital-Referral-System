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
		Where("dept_id IN (SELECT id FROM hospital_departments WHERE hospital_id = ?)", hospitalID)
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
		Where("dept_id = ? AND queue_status = 'SCHEDULED' AND appointment_date >= ? AND appointment_date <= ?", deptID, start, end).
		Order("appointment_date asc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) GetWaitingByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("dept_id = ? AND queue_status = 'WAITING'", deptID).
		Order("composite_score desc").
		Find(&queues).Error
	return queues, err
}
