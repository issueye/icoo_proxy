package domain

import "github.com/issueye/icoo_proxy/common/constants"

type ProviderSnapshot struct {
	ID          string
	Name        string
	Protocol    constants.Protocol
	Vendor      constants.Vendor
	PluginID    string
	BaseURL     string
	ProxyURL    string
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
	Enabled   bool
}
