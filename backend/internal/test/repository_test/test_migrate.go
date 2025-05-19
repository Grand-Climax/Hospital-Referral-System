package repositorytest

import (
	"Hospital-Referral-System/internal/domain/entity"

	"gorm.io/gorm"
)

var testDB *gorm.DB

func Migrate() {
	testDB.AutoMigrate(&entity.Role{}, &entity.Department{})
}