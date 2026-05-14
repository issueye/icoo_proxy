package entity

import "time"

type APIKey struct {
	ID            string `gorm:"primaryKey"`
	Name          string `gorm:"index;not null"`
	SecretHash    string `gorm:"uniqueIndex;not null"`
	SecretPreview string
	SecretCipher  string
	Scopes        string
	Enabled       bool
	ExpiresAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (APIKey) TableName() string {
	return "api_keys"
}
