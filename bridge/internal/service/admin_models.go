package service

import (
	"time"

	"github.com/issueye/icoo_proxy/common/constants"
)

type ProviderUpsertInput struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Protocol    constants.Protocol `json:"protocol"`
	Vendor      constants.Vendor   `json:"vendor"`
	PluginID    string             `json:"plugin_id"`
	BaseURL     string             `json:"base_url"`
	ModelsURL   string             `json:"models_url"`
	ProxyURL    string             `json:"proxy_url"`
	APIKey      string             `json:"api_key"`
	OnlyStream  bool               `json:"only_stream"`
	UserAgent   string             `json:"user_agent"`
	Enabled     bool               `json:"enabled"`
	Description string             `json:"description"`
}

type ProviderView struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Protocol     constants.Protocol `json:"protocol"`
	Vendor       constants.Vendor   `json:"vendor"`
	PluginID     string             `json:"plugin_id,omitempty"`
	BaseURL      string             `json:"base_url"`
	ModelsURL    string             `json:"models_url"`
	ProxyURL     string             `json:"proxy_url"`
	APIKeyMasked string             `json:"api_key_masked"`
	HasAPIKey    bool               `json:"has_api_key"`
	OnlyStream   bool               `json:"only_stream"`
	UserAgent    string             `json:"user_agent"`
	Enabled      bool               `json:"enabled"`
	Description  string             `json:"description"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type ProviderModelUpsertInput struct {
	ID         string `json:"id"`
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	MaxTokens  int    `json:"max_tokens"`
	Enabled    bool   `json:"enabled"`
}

type ModelCatalogUpsertInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Family      string `json:"family"`
	Icon        string `json:"icon"`
	MaxTokens   int    `json:"max_tokens"`
	Description string `json:"description"`
}

type ProviderChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ProviderChatInput struct {
	Model       string                `json:"model"`
	Messages    []ProviderChatMessage `json:"messages"`
	MaxTokens   int                   `json:"max_tokens"`
	Temperature *float64              `json:"temperature,omitempty"`
}

type ProviderChatResult struct {
	SupplierID string              `json:"supplier_id"`
	Model      string              `json:"model"`
	Message    ProviderChatMessage `json:"message"`
	StatusCode int                 `json:"status_code"`
	DurationMS int64               `json:"duration_ms"`
}

type ProviderHealthResult struct {
	SupplierID string `json:"supplier_id"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	DurationMS int64  `json:"duration_ms"`
	Message    string `json:"message"`
	CheckedAt  string `json:"checked_at"`
}

type EndpointUpsertInput struct {
	ID                 string             `json:"id"`
	Path               string             `json:"path"`
	DownstreamProtocol constants.Protocol `json:"downstream_protocol"`
	Enabled            bool               `json:"enabled"`
	Protected          bool               `json:"protected"`
	Description        string             `json:"description"`
}

type RoutingRuleUpsertInput struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Priority          int                `json:"priority"`
	MatchProtocol     constants.Protocol `json:"match_protocol"`
	MatchModelPattern string             `json:"match_model_pattern"`
	UpstreamProtocol  constants.Protocol `json:"upstream_protocol"`
	TargetProviderID  string             `json:"target_provider_id"`
	TargetModel       string             `json:"target_model"`
	Enabled           bool               `json:"enabled"`
	Force             bool               `json:"force"`
}

type APIKeyCreateInput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Secret  string `json:"secret"`
	Scopes  string `json:"scopes"`
	Enabled bool   `json:"enabled"`
}

type APIKeyView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SecretPreview string `json:"secret_preview"`
	CanReveal     bool   `json:"can_reveal"`
	Scopes        string `json:"scopes"`
	Enabled       bool   `json:"enabled"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type APIKeySecretView struct {
	Secret string `json:"secret"`
}

type UIPrefsInput struct {
	Theme      string `json:"theme"`
	ButtonSize string `json:"buttonSize"`
}

type UIPrefsView struct {
	Theme      string `json:"theme"`
	ButtonSize string `json:"buttonSize"`
}
