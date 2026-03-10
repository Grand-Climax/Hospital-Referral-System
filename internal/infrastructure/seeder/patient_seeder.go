package seeder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

func seedPatients(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding patients...")

	natID1 := "NAT-SEED-001"
	hash1 := sha256.Sum256([]byte(natID1))
	hashStr1 := hex.EncodeToString(hash1[:])

	phone1 := "+251911000001"
	dob1, _ := time.Parse("2006-01-02", "1980-01-01")

	p1 := entity.Patient{
		NationalIDEnc:  &natID1,
		NationalIDHash: &hashStr1,
		PhoneNumber:    &phone1,
		FirstName:      "Abebe",
		MiddleName:     "K",
		LastName:       "Kebede",
		Sex:            "male",
		DateOfBirth:    &dob1,
	}

	phone2 := "+251911000002"
	dob2, _ := time.Parse("2006-01-02", "1990-05-15")
	p2 := entity.Patient{
		PhoneNumber: &phone2,
		FirstName:   "Liya",
		MiddleName:  "H",
		LastName:    "Haile",
		Sex:         "female",
		DateOfBirth: &dob2,
	}

	patients := []entity.Patient{p1, p2}
	for _, p := range patients {
		if err := db.Where("phone_number = ?", p.PhoneNumber).FirstOrCreate(&p).Error; err != nil {
			return err
		}
	}
	return nil
}
