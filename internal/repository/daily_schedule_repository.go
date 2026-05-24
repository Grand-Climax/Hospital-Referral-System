package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
)

type dailyScheduleRepository struct {
	*BaseRepository[entity.DailySchedule]
	db *gorm.DB
}

func NewDailyScheduleRepository(db *gorm.DB) irepository.DailyScheduleRepository {
	return &dailyScheduleRepository{
		BaseRepository: NewBaseRepository[entity.DailySchedule](db),
		db:             db,
	}
}

// FindByDeptAndDate returns the existing log row for the given
// (hospital, department, date) or gorm.ErrRecordNotFound. The schedule_date
// column is a DATE in Postgres so we compare formatted as YYYY-MM-DD.
func (r *dailyScheduleRepository) FindByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.DailySchedule, error) {
	var schedule entity.DailySchedule
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND schedule_date = ?",
			hospitalID, deptID, date.Format("2006-01-02")).
		First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// FindByDeptAndDateRange returns log rows for the inclusive date range,
// ordered ascending by schedule_date. Used by calendar views.
func (r *dailyScheduleRepository) FindByDeptAndDateRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.DailySchedule, error) {
	var schedules []entity.DailySchedule
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND schedule_date BETWEEN ? AND ?",
			hospitalID, deptID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Order("schedule_date ASC").
		Find(&schedules).Error
	return schedules, err
}

// CreateLog inserts the immutable history row for the first booking
// of a (hospital, department, date) tuple. MaxSlots / OverbookLimit are
// captured by the caller from the effective capacity at booking time.
func (r *dailyScheduleRepository) CreateLog(ctx context.Context, log *entity.DailySchedule) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// UpdateBookedSlots updates the BookedSlots snapshot. MaxSlots is never
// updated after creation.
func (r *dailyScheduleRepository) UpdateBookedSlots(ctx context.Context, id uuid.UUID, count int) error {
	return r.db.WithContext(ctx).Model(&entity.DailySchedule{}).
		Where("id = ?", id).
		Update("booked_slots", count).Error
}
