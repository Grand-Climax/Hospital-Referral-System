package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/domain/entity"
)

func main() {
	cfg := config.LoadConfig()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.DB_Name, cfg.DB.Port, cfg.DB.SSLMode)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	ctx := context.Background()
	
	var links []entity.HospitalDepartment
	if err := db.WithContext(ctx).Find(&links).Error; err != nil {
		log.Fatalf("Failed to fetch hospital departments: %v", err)
	}

	fmt.Printf("Found %d hospital-department links. Seeding 30 days...\n", len(links))

	for _, link := range links {
		for i := 0; i < 30; i++ {
			date := time.Now().AddDate(0, 0, i)
			dateStr := date.Format("2006-01-02")
			
			schedule := entity.DailySchedule{
				HospitalID:    link.HospitalID,
				DeptID:        link.DepartmentID,
				ScheduleDate:  date,
				MaxSlots:      link.StandardDailyLimit,
				OverbookLimit: 2,
				Version:       1,
			}

			// Use ON CONFLICT DO NOTHING (Postgres specific syntax via GORM)
			err := db.WithContext(ctx).
				Where("hospital_id = ? AND dept_id = ? AND schedule_date = ?", link.HospitalID, link.DepartmentID, dateStr).
				FirstOrCreate(&schedule).Error
			
			if err != nil {
				fmt.Printf("Error seeding for %s/%s on %s: %v\n", link.HospitalID, link.DepartmentID, dateStr, err)
			}
		}
	}

	fmt.Println("Seeding complete.")
}
