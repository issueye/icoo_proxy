package models

import (
	"icoo_proxy/internal/consts"
	"time"
)

type SupplierRecord struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Protocol     consts.Protocol `json:"protocol"`
	BaseURL      string          `json:"base_url"`
	APIKeyMasked string          `json:"api_key_masked"`
	OnlyStream   bool            `json:"only_stream"`
	UserAgent    string          `json:"user_agent"`
	Enabled      bool            `json:"enabled"`
	Description  string          `json:"description"`
	Models       []string        `json:"models"`
	DefaultModel string          `json:"default_model"`
	UpdatedAt    string          `json:"updated_at"`
	CreatedAt    string          `json:"created_at"`
}

type SupplierModel struct {
	ID           string `gorm:"primaryKey"`
	Name         string `gorm:"index"`
	Protocol     consts.Protocol
	BaseURL      string
	APIKey       string
	OnlyStream   bool
	UserAgent    string
	Enabled      bool
	Description  string
	Models       string
	DefaultModel string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (SupplierModel) TableName() string {
	return "suppliers"
}

type SupplierUpsertInput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	OnlyStream   bool   `json:"only_stream"`
	UserAgent    string `json:"user_agent"`
	Enabled      bool   `json:"enabled"`
	Description  string `json:"description"`
	Models       string `json:"models"`
	DefaultModel string `json:"default_model"`
}
