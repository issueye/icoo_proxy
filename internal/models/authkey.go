package models

import "time"

type AuthKeyRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SecretMasked string `json:"secret_masked"`
	Enabled      bool   `json:"enabled"`
	Description  string `json:"description"`
	UpdatedAt    string `json:"updated_at"`
	CreatedAt    string `json:"created_at"`
}

type AuthKeyModel struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"index"`
	Secret      string `gorm:"uniqueIndex"`
	Enabled     bool
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (AuthKeyModel) TableName() string {
	return "auth_keys"
}

type AuthKeyUpsertInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Secret      string `json:"secret"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}
