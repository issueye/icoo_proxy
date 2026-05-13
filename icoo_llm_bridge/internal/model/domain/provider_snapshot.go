package domain

import "icoo_llm_bridge/internal/constants"

type ProviderSnapshot struct {
	ID          string
	Name        string
	Protocol    constants.Protocol
	Vendor      constants.Vendor
	BaseURL     string
	APIKey      string
	OnlyStream  bool
	UserAgent   string
	Enabled     bool
	Description string
	Models      []ProviderModelSnapshot
}

type ProviderModelSnapshot struct {
	Name      string
	MaxTokens int
	IsDefault bool
	Enabled   bool
}
