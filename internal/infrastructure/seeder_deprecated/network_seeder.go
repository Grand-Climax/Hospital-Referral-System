package seeder

import (
	"context"
	"log"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

func seedNetworks(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding Referral Networks...")

	// Fetch all hospitals to map their IDs
	var hospitals []entity.Hospital
	if err := db.WithContext(ctx).Find(&hospitals).Error; err != nil {
		return err
	}

	var primaryOromia, generalOromia, specializedOromia *entity.Hospital
	var primaryAmhara, generalAmhara, specializedAmhara *entity.Hospital

	// Identify hospitals by mapping names/regions/tiers
	for i, h := range hospitals {
		if h.Name == "Bishoftu Primary Hospital" {
			primaryOromia = &hospitals[i]
		}
		if h.Name == "Adama General Hospital" {
			generalOromia = &hospitals[i]
		}
		if h.Name == "Jimma University Medical Center" {
			specializedOromia = &hospitals[i]
		}

		if h.Name == "Kobo Primary Hospital" {
			primaryAmhara = &hospitals[i]
		}
		if h.Name == "Dessie Referral Hospital" {
			generalAmhara = &hospitals[i]
		}
		if h.Name == "Felege Hiwot Comprehensive Specialized Hospital" {
			specializedAmhara = &hospitals[i]
		}
	}

	var networks []entity.ReferralNetwork

	// Build Network Routes: Primary -> General -> Specialized
	if primaryOromia != nil && generalOromia != nil {
		networks = append(networks, entity.ReferralNetwork{
			SenderHospitalID:      primaryOromia.ID,
			ReceiverHospitalID:    generalOromia.ID,
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		})
	}
	if generalOromia != nil && specializedOromia != nil {
		networks = append(networks, entity.ReferralNetwork{
			SenderHospitalID:      generalOromia.ID,
			ReceiverHospitalID:    specializedOromia.ID,
			ReferralType:          "routine",
			RequiresAdminApproval: true,
		})
	}

	if primaryAmhara != nil && generalAmhara != nil {
		networks = append(networks, entity.ReferralNetwork{
			SenderHospitalID:      primaryAmhara.ID,
			ReceiverHospitalID:    generalAmhara.ID,
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		})
	}
	if generalAmhara != nil && specializedAmhara != nil {
		networks = append(networks, entity.ReferralNetwork{
			SenderHospitalID:      generalAmhara.ID,
			ReceiverHospitalID:    specializedAmhara.ID,
			ReferralType:          "routine",
			RequiresAdminApproval: true,
		})
	}

	// ── Critical dev/test routes ─────────────────────────────────────────────
	// Bishoftu Primary → Adama General  (matches test JWT hosp_id cd323204...)
	// Adama General → Jimma Specialized
	var bishoftu, adama, jimma *entity.Hospital
	for i, h := range hospitals {
		switch h.Name {
		case "Bishoftu Primary Hospital":
			bishoftu = &hospitals[i]
		case "Adama General Hospital":
			adama = &hospitals[i]
		case "Jimma University Medical Center":
			jimma = &hospitals[i]
		}
	}

	if bishoftu != nil && adama != nil {
		networks = append(networks, entity.ReferralNetwork{
			SenderHospitalID:      bishoftu.ID,
			ReceiverHospitalID:    adama.ID,
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		})
	}
	if adama != nil && jimma != nil {
		networks = append(networks, entity.ReferralNetwork{
			SenderHospitalID:      adama.ID,
			ReceiverHospitalID:    jimma.ID,
			ReferralType:          "routine",
			RequiresAdminApproval: false,
		})
	}

	// Persist
	for _, raw := range networks {
		var existing entity.ReferralNetwork
		if err := db.WithContext(ctx).Where("sender_hospital_id = ? AND receiver_hospital_id = ?", raw.SenderHospitalID, raw.ReceiverHospitalID).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.WithContext(ctx).Create(&raw).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}
