package models

import (
	"icoo_proxy/internal/consts"
	"strings"
	"time"
)

const DefaultSupplierModelMaxTokens = 32768

type SupplierModelItem struct {
	Name      string `json:"name"`
	MaxTokens int    `json:"max_tokens"`
}

type SupplierRecord struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Protocol     consts.Protocol     `json:"protocol"`
	Vendor       consts.Vendor       `json:"vendor"`
	BaseURL      string              `json:"base_url"`
	APIKeyMasked string              `json:"api_key_masked"`
	OnlyStream   bool                `json:"only_stream"`
	UserAgent    string              `json:"user_agent"`
	Enabled      bool                `json:"enabled"`
	Description  string              `json:"description"`
	Models       []SupplierModelItem `json:"models"`
	DefaultModel string              `json:"default_model"`
	UpdatedAt    string              `json:"updated_at"`
	CreatedAt    string              `json:"created_at"`
}

type SupplierModel struct {
	ID           string `gorm:"primaryKey"`
	Name         string `gorm:"index"`
	Protocol     consts.Protocol
	Vendor       consts.Vendor
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
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Protocol     string              `json:"protocol"`
	Vendor       string              `json:"vendor"`
	BaseURL      string              `json:"base_url"`
	APIKey       string              `json:"api_key"`
	OnlyStream   bool                `json:"only_stream"`
	UserAgent    string              `json:"user_agent"`
	Enabled      bool                `json:"enabled"`
	Description  string              `json:"description"`
	Models       []SupplierModelItem `json:"models"`
	DefaultModel string              `json:"default_model"`
}

func NormalizeSupplierModelItems(items []SupplierModelItem) []SupplierModelItem {
	normalized := make([]SupplierModelItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		maxTokens := item.MaxTokens
		if maxTokens <= 0 {
			maxTokens = DefaultSupplierModelMaxTokens
		}
		normalized = append(normalized, SupplierModelItem{
			Name:      name,
			MaxTokens: maxTokens,
		})
	}
	return normalized
}

func FindSupplierModel(items []SupplierModelItem, target string) (SupplierModelItem, bool) {
	name := strings.TrimSpace(target)
	if name == "" {
		return SupplierModelItem{}, false
	}
	for _, item := range NormalizeSupplierModelItems(items) {
		if strings.EqualFold(item.Name, name) {
			return item, true
		}
	}
	return SupplierModelItem{}, false
}
