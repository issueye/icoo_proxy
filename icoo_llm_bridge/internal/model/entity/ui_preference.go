package entity

import "time"

type UIPreference struct {
	Key       string `gorm:"primaryKey"`
	ValueJSON string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UIPreference) TableName() string {
	return "ui_preferences"
}
