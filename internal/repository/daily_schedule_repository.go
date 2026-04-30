package repository

import (
	"context"
	"errors"
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

func (r *dailyScheduleRepository) GetByDeptAndDate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time) (*entity.DailySchedule, error) {
	var schedule entity.DailySchedule
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND schedule_date = ?", hospitalID, deptID, date.Format("2006-01-02")).
		First(&schedule).Error
	return &schedule, err
}

func (r *dailyScheduleRepository) GetOrCreate(ctx context.Context, hospitalID, deptID uuid.UUID, date time.Time, defaultMaxSlots int) (*entity.DailySchedule, error) {
	var schedule entity.DailySchedule
	dateStr := date.Format("2006-01-02")
	
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND schedule_date = ?", hospitalID, deptID, dateStr).
		First(&schedule).Error
	
	if err == nil {
		return &schedule, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var hospDept entity.HospitalDepartment
	r.db.WithContext(ctx).Where("hospital_id = ? AND department_id = ?", hospitalID, deptID).First(&hospDept)

	schedule = entity.DailySchedule{
		HospitalID:    hospitalID,
		DepartmentID:  deptID,
		DeptID:        hospDept.ID,
		ScheduleDate:  date,
		MaxSlots:      defaultMaxSlots,
		OverbookLimit: 2,
		Version:       1,
	}

	if err := r.db.WithContext(ctx).Create(&schedule).Error; err != nil {
		return r.GetByDeptAndDate(ctx, hospitalID, deptID, date)
	}

	return &schedule, nil
}

func (r *dailyScheduleRepository) IncrementBookedSlots(ctx context.Context, id uuid.UUID, version int) error {
	result := r.db.WithContext(ctx).Model(&entity.DailySchedule{}).
		Where("id = ? AND version = ?", id, version).
		Updates(map[string]interface{}{
			"booked_slots": gorm.Expr("booked_slots + 1"),
			"version":      gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("concurrent_modification: schedule was modified by another request")
	}
	return nil
}

func (r *dailyScheduleRepository) FindByDeptAndDateRange(ctx context.Context, hospitalID, deptID uuid.UUID, start, end time.Time) ([]entity.DailySchedule, error) {
	var schedules []entity.DailySchedule
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND department_id = ? AND schedule_date BETWEEN ? AND ?", hospitalID, deptID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Order("schedule_date ASC").
		Find(&schedules).Error
	return schedules, err
}
