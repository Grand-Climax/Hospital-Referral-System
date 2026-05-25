package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Idempotent: ALTER TYPE ... ADD VALUE IF NOT EXISTS is safe to repeat.
func main() {
	_ = godotenv.Load(".env.local")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set in env (.env.local)")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	for _, lbl := range []string{"MISSED", "MISSED_RESCHEDULE"} {
		stmt := fmt.Sprintf(`ALTER TYPE notificationtype ADD VALUE IF NOT EXISTS '%s'`, lbl)
		fmt.Printf("→ %s\n", stmt)
		if err := db.Exec(stmt).Error; err != nil {
			log.Fatalf("  ✗ %v", err)
		}
		fmt.Println("  ✓ ok")
	}

	type lbl struct{ EnumLabel string }
	var labels []lbl
	db.Raw(`select unnest(enum_range(null::notificationtype))::text as enum_label`).Scan(&labels)
	fmt.Println("\nFinal notificationtype values:")
	for _, l := range labels {
		fmt.Printf("  %s\n", l.EnumLabel)
	}
}
