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
		Where("hospital_id = ? AND department_id = ? AND appointment_date IS NOT NULL AND arrival_status = 'EXPECTED' AND appointment_date >= ? AND appointment_date <= ?", hospitalID, deptID, start, end).
		Order("appointment_date asc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) GetWaitingByDept(ctx context.Context, hospitalID, deptID uuid.UUID) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND appointment_date IS NULL AND arrival_status = 'EXPECTED'", hospitalID, deptID).
		Order("composite_score desc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) FindWaitingByHospitalAndDept(ctx context.Context, hospitalID, departmentID uuid.UUID) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND appointment_date IS NULL AND arrival_status = 'EXPECTED'", hospitalID, departmentID).
		Order("composite_score desc").
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) FindScheduledByHospitalAndDept(ctx context.Context, hospitalID, deptID uuid.UUID, startDate, endDate time.Time) ([]*entity.TriageQueue, error) {
	var queues []*entity.TriageQueue
	err := r.db.WithContext(ctx).
		Preload("Referral").
		Preload("Referral.Patient").
		Where("hospital_id = ? AND department_id = ? AND appointment_date IS NOT NULL AND arrival_status = 'EXPECTED' AND appointment_date BETWEEN ? AND ?", hospitalID, deptID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
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
	err := query.Order("composite_score desc").Limit(limit).Offset(offset).Find(&queues).Error
	return queues, count, err
}

func (r *triageRepository) FindAppointmentsForReminders(ctx context.Context, date time.Time) ([]*entity.TriageQueue, error) {
	var queues []*entity.TriageQueue
	err := r.db.WithContext(ctx).
		Preload("Referral").
		Preload("Referral.Patient").
		Where("appointment_date = ? AND arrival_status = 'EXPECTED'", date.Format("2006-01-02")).
		Find(&queues).Error
	return queues, err
}

func (r *triageRepository) IncrementWaitingWeights(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Where("appointment_date IS NULL AND arrival_status = ?", entity.ArrivalExpected).
		Update("waiting_hours_weight", gorm.Expr("waiting_hours_weight + 1"))
	return result.RowsAffected, result.Error
}

func (r *triageRepository) ListMissedByHospital(ctx context.Context, hospitalID uuid.UUID, limit, offset int) ([]*entity.TriageQueue, int64, error) {
	var queues []*entity.TriageQueue
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Preload("Referral").
		Preload("Referral.Patient").
		Where("hospital_id = ? AND arrival_status = ?", hospitalID, entity.ArrivalMissed)

	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("appointment_date desc").Limit(limit).Offset(offset).Find(&queues).Error
	return queues, count, err
}

func (r *triageRepository) FindMissedByDate(ctx context.Context, beforeDate time.Time) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Where("appointment_date IS NOT NULL AND appointment_date < ? AND arrival_status = ?", beforeDate.Format("2006-01-02"), entity.ArrivalExpected).
		Find(&queues).Error
	return queues, err
}

// CountByDeptAndDate returns the number of triage queue rows currently
// occupying capacity for (hospital, department, date). Rows whose
// ArrivalStatus is Expected, Arrived, or Admitted are counted; Missed
// and Completed rows are deliberately excluded so a no-show frees the
// slot for an in-day re-book (which is also why DailySchedule is now
// a snapshot rather than a counter).
func (r *triageRepository) CountByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int64, error) {
	var c int64
	err := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Where("hospital_id = ? AND department_id = ? AND appointment_date = ? AND arrival_status IN ?",
			hospitalID, deptID, date.Format("2006-01-02"),
			[]entity.ArrivalStatus{
				entity.ArrivalExpected,
				entity.ArrivalArrived,
				entity.ArrivalAdmitted,
			}).
		Count(&c).Error
	return c, err
}

// CountAssignedDoctorsByDeptAndDate returns the number of distinct
// assigned_doctor_ids on the triage queue for (hospital, department,
// date), restricted to rows whose patient has actually arrived
// (Arrived/Admitted) and whose referral is in SCHEDULED status. Used
// purely as a "staff_assigned" hint in capacity views; it does not
// gate any booking.
func (r *triageRepository) CountAssignedDoctorsByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (int64, error) {
	var c int64
	err := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Joins("JOIN referrals ON referrals.id = triage_queues.referral_id").
		Where("triage_queues.hospital_id = ? AND triage_queues.department_id = ? AND triage_queues.appointment_date = ?",
			hospitalID, deptID, date.Format("2006-01-02")).
		Where("triage_queues.arrival_status IN ?", []entity.ArrivalStatus{
			entity.ArrivalArrived,
			entity.ArrivalAdmitted,
		}).
		Where("referrals.status = ?", entity.StatusScheduled).
		Where("triage_queues.assigned_doctor_id IS NOT NULL").
		Distinct("triage_queues.assigned_doctor_id").
		Count(&c).Error
	return c, err
}

// CountMissedByDeptInRange returns the count of triage rows missed in the
// inclusive [start, end] window. Used by the dept-head dashboard to
// drive the "missed last 7 days" KPI.
func (r *triageRepository) CountMissedByDeptInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) (int64, error) {
	var c int64
	err := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Where("hospital_id = ? AND department_id = ?", hospitalID, deptID).
		Where("appointment_date BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Where("arrival_status = ?", entity.ArrivalMissed).
		Count(&c).Error
	return c, err
}

// CountScheduledByDeptInRange returns the count of triage rows scheduled
// in [start, end] with status Expected/Arrived/Admitted (i.e. still
// counts toward capacity).
func (r *triageRepository) CountScheduledByDeptInRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) (int64, error) {
	var c int64
	err := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Where("hospital_id = ? AND department_id = ?", hospitalID, deptID).
		Where("appointment_date BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02")).
		Where("arrival_status IN ?", []entity.ArrivalStatus{
			entity.ArrivalExpected,
			entity.ArrivalArrived,
			entity.ArrivalAdmitted,
		}).
		Count(&c).Error
	return c, err
}

// OldestWaitingDaysByDept returns the integer number of days since the
// earliest unscheduled EXPECTED triage row in the dept was created. Zero
// when the queue is empty or the query fails (caller should treat the
// metric as best-effort).
func (r *triageRepository) OldestWaitingDaysByDept(ctx context.Context, hospitalID, deptID uuid.UUID) (int, error) {
	var oldest *time.Time
	err := r.db.WithContext(ctx).Model(&entity.TriageQueue{}).
		Joins("JOIN referrals ON referrals.id = triage_queues.referral_id").
		Where("triage_queues.hospital_id = ? AND triage_queues.department_id = ?", hospitalID, deptID).
		Where("triage_queues.appointment_date IS NULL AND triage_queues.arrival_status = ?", entity.ArrivalExpected).
		Select("MIN(referrals.created_at)").Scan(&oldest).Error
	if err != nil || oldest == nil {
		return 0, err
	}
	days := int(time.Since(*oldest).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days, nil
}

// FindScheduledByDeptAndDate returns the triage rows scheduled for the
// given date in any "occupying-the-slot" arrival status (Expected,
// Arrived, Admitted). Referral + Patient are eagerly loaded so the
// dept-head schedule-patients view can render names without an extra
// round-trip.
func (r *triageRepository) FindScheduledByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) ([]entity.TriageQueue, error) {
	var queues []entity.TriageQueue
	err := r.db.WithContext(ctx).
		Preload("Referral").
		Preload("Referral.Patient").
		Where("hospital_id = ? AND department_id = ? AND appointment_date = ? AND arrival_status IN ?",
			hospitalID, deptID, date.Format("2006-01-02"),
			[]entity.ArrivalStatus{
				entity.ArrivalExpected,
				entity.ArrivalArrived,
				entity.ArrivalAdmitted,
			}).
		Order("composite_score desc").
		Find(&queues).Error
	return queues, err
}
