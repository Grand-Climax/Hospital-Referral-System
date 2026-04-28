package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dbURL := "postgresql://neondb_owner:npg_0j7tnACSzsPJ@ep-still-scene-altar03k-pooler.c-3.eu-central-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	migration, err := os.ReadFile("scripts/db_migration/schema_update_v6.sql")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(string(migration)).Error
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migration schema_update_v6.sql executed successfully.")
}
