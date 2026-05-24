// Safe one-shot runner for schema_update_v16.sql.
//
// Usage:
//
//	go run scripts/run_db_migration_v16.go --confirm
//
// Behavior:
//   - Refuses to run without the --confirm flag (the SQL TRUNCATEs
//     daily_schedules, so it must be an explicit human decision).
//   - Checks system_configs.schema_version BEFORE running. If it already
//     equals "v16", prints "already applied" and exits 0.
//   - Otherwise reads scripts/db_migration/schema_update_v16.sql and
//     executes it inside a single transaction.
//   - On success, UPSERTs system_configs.schema_version = "v16" so the
//     next invocation no-ops.
//   - Does NOT self-delete. Re-running is safe (it just no-ops once the
//     guard row is in place).
//
// Connection: reads DB_* env vars from .env / .env.local (same as the
// main server), or honors a DATABASE_URL override if present.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"Hospital-Referral-System/config"
)

const (
	schemaVersionKey   = "schema_version"
	schemaVersionValue = "v16"
	migrationFile      = "scripts/db_migration/schema_update_v16.sql"
)

func main() {
	confirm := flag.Bool("confirm", false, "Required acknowledgement that you understand this migration TRUNCATEs daily_schedules.")
	flag.Parse()

	if !*confirm {
		log.Fatal("refusing to run without --confirm; this migration TRUNCATEs daily_schedules. " +
			"Pass --confirm to acknowledge and proceed.")
	}

	db, err := openDB()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if alreadyApplied(db) {
		log.Println("schema_update_v16 is already applied (system_configs.schema_version = 'v16'). No-op.")
		return
	}

	sql, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("failed to read %s: %v", migrationFile, err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("migration SQL failed: %w", err)
		}
		return upsertSchemaVersion(tx)
	})
	if err != nil {
		log.Fatalf("migration v16 failed (rolled back): %v", err)
	}

	log.Println("schema_update_v16 applied successfully; system_configs.schema_version = 'v16'.")
}

func openDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		cfg := config.LoadConfig().DB
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.DB_Name, cfg.Port, cfg.SSLMode)
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func alreadyApplied(db *gorm.DB) bool {
	var value string
	err := db.Raw(
		"SELECT value FROM system_configs WHERE key = ?",
		schemaVersionKey,
	).Scan(&value).Error
	if err != nil {
		return false
	}
	return value == schemaVersionValue
}

func upsertSchemaVersion(tx *gorm.DB) error {
	return tx.Exec(`
		INSERT INTO system_configs (key, value, updated_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (key) DO UPDATE
		    SET value = EXCLUDED.value,
		        updated_at = EXCLUDED.updated_at
	`, schemaVersionKey, schemaVersionValue).Error
}
