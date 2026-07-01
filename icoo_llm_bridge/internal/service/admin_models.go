package service

import "icoo_llm_bridge/internal/constants"

type ProviderUpsertInput struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Protocol    constants.Protocol `json:"protocol"`
	Vendor      constants.Vendor   `json:"vendor"`
	BaseURL     string             `json:"base_url"`
	ModelsURL   string             `json:"models_url"`
	APIKey      string             `json:"api_key"`
	OnlyStream  bool               `json:"only_stream"`
	UserAgent   string             `json:"user_agent"`
	Enabled     bool               `json:"enabled"`
	Description string             `json:"description"`
}

type ProviderModelUpsertInput struct {
	ID         string `json:"id"`
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	MaxTokens  int    `json:"max_tokens"`
	Enabled    bool   `json:"enabled"`
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
