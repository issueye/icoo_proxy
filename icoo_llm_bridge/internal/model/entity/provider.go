package entity

import (
	"time"

	"icoo_llm_bridge/internal/constants"
)

type Provider struct {
	ID           string `gorm:"primaryKey"`
	Name         string `gorm:"uniqueIndex;not null"`
	Protocol     constants.Protocol
	Vendor       constants.Vendor
	BaseURL      string
	APIKeyCipher string
	OnlyStream   bool
	UserAgent    string
	Enabled      bool
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Provider) TableName() string {
	return "providers"
}

type ProviderModel struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	ProviderID string    `json:"provider_id" gorm:"index:idx_provider_model,unique;not null"`
	Name       string    `json:"name" gorm:"index:idx_provider_model,unique;not null"`
	MaxTokens  int       `json:"max_tokens"`
	IsDefault  bool      `json:"is_default"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (ProviderModel) TableName() string {
	return "provider_models"
}
