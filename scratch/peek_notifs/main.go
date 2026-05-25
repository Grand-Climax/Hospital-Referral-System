package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load(".env.local")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	type col struct {
		Column   string
		Type     string
		Nullable string
	}
	var cols []col
	db.Raw(`select column_name as column, data_type as type, is_nullable as nullable
	        from information_schema.columns
	        where table_name = 'triage_queues' and column_name in ('appointment_date','created_at')`).Scan(&cols)
	fmt.Println("triage_queues key columns:")
	for _, c := range cols {
		fmt.Printf("  %s -> %s null=%s\n", c.Column, c.Type, c.Nullable)
	}

	type row struct {
		ReferralID      string
		AppointmentDate string
	}
	var rows []row
	db.Raw(`select referral_id::text, appointment_date::text from triage_queues
	        where referral_id::text in ('2b8766d8-984e-42a4-a443-6ea8dc551ae5','6cf3ec11-75bd-419e-a83b-7513d10f5600')`).Scan(&rows)
	fmt.Println("\nActual stored appointment_date values:")
	for _, r := range rows {
		fmt.Printf("  %s -> %s\n", r.ReferralID, r.AppointmentDate)
	}
}
