package entity

type ICDCode struct {
	Code        string `gorm:"type:varchar(10);primaryKey" json:"code"`
	Description string `gorm:"type:varchar(255);not null;index" json:"description"`
	Category    string `gorm:"type:varchar(100);not null;index" json:"category"`
}
