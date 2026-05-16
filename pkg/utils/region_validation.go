package utils

import "Hospital-Referral-System/internal/domain/entity"

// IsValidEthiopianRegion checks if a given region string is a valid Ethiopian region as defined in entities.
func IsValidEthiopianRegion(region string) bool {
	return entity.EthiopianRegion(region).IsValid()
}
