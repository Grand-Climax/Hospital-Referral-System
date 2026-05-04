package seeds

import (
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/infrastructure/crypto"
)

// LegacyPatient is used to extract data from the old schema before migration.
type LegacyPatient struct {
	ID          uuid.UUID `gorm:"column:id"`
	NationalID  *string   `gorm:"column:national_id_enc"` // was previously NationalIDEnc
	PhoneNumber *string   `gorm:"column:phone_number"`
	FirstName   string    `gorm:"column:first_name"`
	MiddleName  string    `gorm:"column:middle_name"`
	LastName    string    `gorm:"column:last_name"`
	Sex         string    `gorm:"column:sex"`
}

// TableName overrides the table name
func (LegacyPatient) TableName() string {
	return "patients"
}

// MigratePatientData encrypts existing patient plaintext data and stores it in the new columns.
func MigratePatientData(db *gorm.DB) error {
	log.Println("Starting patient data encryption migration...")

	// 1. AutoMigrate to create the new columns (first_name_enc, etc.)
	err := db.AutoMigrate(&entity.Patient{})
	if err != nil {
		return err
	}

	// 2. Initialize Crypto Service
	aesKey := os.Getenv("PATIENT_AES_KEY")
	hmacKey := os.Getenv("PATIENT_HMAC_KEY")

	// If missing, use a deterministic dev key for seeding (matches full_seeder if needed)
	if aesKey == "" {
		aesKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" // 32 bytes base64 (all zeros essentially)
	}
	if hmacKey == "" {
		hmacKey = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
	}

	cryptoSvc, err := crypto.NewPatientCryptoService(aesKey, hmacKey)
	if err != nil {
		return err
	}

	// 3. Fetch all legacy patients
	var legacyPatients []LegacyPatient
	if err := db.Find(&legacyPatients).Error; err != nil {
		return err
	}

	for _, lp := range legacyPatients {
		// Only encrypt if they aren't already encrypted (first_name is plain)
		// Assuming we run this once, we will just update the new columns.

		// Normalize phone
		var phoneEnc *string
		var phoneHash *string
		if lp.PhoneNumber != nil && *lp.PhoneNumber != "" {
			normalized, err := crypto.NormalizePhone(*lp.PhoneNumber)
			if err == nil {
				hash := cryptoSvc.GenerateHMAC(normalized)
				enc, err := cryptoSvc.Encrypt([]byte(normalized))
				if err == nil {
					phoneEnc = &enc
					phoneHash = &hash
				}
			}
		}

		// Encrypt Names
		encFirst, _ := cryptoSvc.Encrypt([]byte(lp.FirstName))
		encMiddle, _ := cryptoSvc.Encrypt([]byte(lp.MiddleName))
		encLast, _ := cryptoSvc.Encrypt([]byte(lp.LastName))

		// Encrypt National ID
		var natIDEnc *string
		var natIDHash *string
		if lp.NationalID != nil && *lp.NationalID != "" {
			hash := cryptoSvc.GenerateHMAC(*lp.NationalID)
			enc, err := cryptoSvc.Encrypt([]byte(*lp.NationalID))
			if err == nil {
				natIDEnc = &enc
				natIDHash = &hash
			}
		}

		// Update database with raw query to avoid gorm struct mismatches during migration
		updateQuery := `
			UPDATE patients 
			SET first_name_enc = ?, middle_name_enc = ?, last_name_enc = ?, 
			    phone_number_enc = ?, phone_hash = ?, 
			    national_id_enc = ?, national_id_hash = ?
			WHERE id = ?
		`
		if err := db.Exec(updateQuery, encFirst, encMiddle, encLast, phoneEnc, phoneHash, natIDEnc, natIDHash, lp.ID).Error; err != nil {
			log.Printf("Failed to update patient %s: %v", lp.ID, err)
		}
	}

	log.Println("Patient data encryption migration completed.")
	return nil
}
