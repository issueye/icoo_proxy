package entity

import "time"

// UIPreference UI偏好
type UIPreference struct {
	Key       string    `gorm:"primaryKey;column:key;comment:偏好键"`
	ValueJSON string    `gorm:"column:value_json;comment:偏好值JSON"`
	CreatedAt time.Time `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;comment:更新时间"`
}

func (UIPreference) TableName() string {
	return "ui_preferences"
}
