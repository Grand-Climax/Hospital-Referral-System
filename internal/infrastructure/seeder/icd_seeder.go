package seeder

import (
	"context"
	"log"

	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

var sampleICDCodes = []entity.ICDCode{
	{Code: "J18.9", Description: "Pneumonia, unspecified organism", Category: "Diseases of the respiratory system"},
	{Code: "I10", Description: "Essential (primary) hypertension", Category: "Diseases of the circulatory system"},
	{Code: "E11.9", Description: "Type 2 diabetes mellitus without complications", Category: "Endocrine, nutritional and metabolic diseases"},
	{Code: "A09", Description: "Infectious gastroenteritis and colitis, unspecified", Category: "Certain infectious and parasitic diseases"},
	{Code: "O80", Description: "Encounter for full-term uncomplicated delivery", Category: "Pregnancy, childbirth and the puerperium"},
	{Code: "S06.9X9A", Description: "Unspecified intracranial injury with loss of consciousness of unspecified duration, initial encounter", Category: "Injury, poisoning and certain other consequences of external causes"},
	{Code: "D50.9", Description: "Iron deficiency anemia, unspecified", Category: "Diseases of the blood and blood-forming organs and certain disorders involving the immune mechanism"},
	{Code: "R50.9", Description: "Fever, unspecified", Category: "Symptoms, signs and abnormal clinical and laboratory findings, not elsewhere classified"},
	{Code: "Z00.00", Description: "Encounter for general adult medical examination without abnormal findings", Category: "Factors influencing health status and contact with health services"},
	{Code: "K35.80", Description: "Unspecified acute appendicitis", Category: "Diseases of the digestive system"},
}

func seedICDCodes(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding ICD-10 codes...")
	for _, raw := range sampleICDCodes {
		var existing entity.ICDCode
		if err := db.WithContext(ctx).Where("code = ?", raw.Code).First(&existing).Error; err != nil {
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
