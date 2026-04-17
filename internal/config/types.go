package config

import "time"

// GatewayConfig holds gateway server configuration.
type GatewayConfig struct {
	ListenHost      string `json:"listenHost" toml:"listen_host"`
	ListenPort      int    `json:"listenPort" toml:"listen_port"`
	DefaultProvider string `json:"defaultProvider" toml:"default_provider"`
	LogLevel        string `json:"logLevel" toml:"log_level"`
	RetryCount      int    `json:"retryCount" toml:"retry_count"`
	RetryIntervalMs int    `json:"retryIntervalMs" toml:"retry_interval_ms"`
	// Deprecated: retained only for one-way migration from legacy config.
	AuthKey string `json:"authKey,omitempty" toml:"auth_key,omitempty"`
}

// ApiKeyConfig holds gateway access credentials and scope.
type ApiKeyConfig struct {
	ID          string    `json:"id" toml:"id"`
	Name        string    `json:"name" toml:"name"`
	Key         string    `json:"key" toml:"key"`
	Description string    `json:"description" toml:"description"`
	Enabled     bool      `json:"enabled" toml:"enabled"`
	ScopeMode   string    `json:"scopeMode" toml:"scope_mode"`
	ProviderIDs []string  `json:"providerIds" toml:"provider_ids"`
	EndpointIDs []string  `json:"endpointIds" toml:"endpoint_ids"`
	LastUsedAt  time.Time `json:"lastUsedAt" toml:"last_used_at"`
	CreatedAt   time.Time `json:"createdAt" toml:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" toml:"updated_at"`
}

// EndpointConfig describes an upstream endpoint.
type EndpointConfig struct {
	ID               string `json:"id" toml:"id"`
	Name             string `json:"name" toml:"name"`
	ProviderID       string `json:"providerId" toml:"provider_id"`
	Path             string `json:"path" toml:"path"`
	Method           string `json:"method" toml:"method"`
	Capability       string `json:"capability" toml:"capability"`
	RequestProtocol  string `json:"requestProtocol" toml:"request_protocol"`
	ResponseProtocol string `json:"responseProtocol" toml:"response_protocol"`
	Enabled          bool   `json:"enabled" toml:"enabled"`
	Priority         int    `json:"priority" toml:"priority"`
	IsDefault        bool   `json:"isDefault" toml:"is_default"`
	Remark           string `json:"remark" toml:"remark"`
}

// ProviderConfig holds a single AI provider configuration.
type ProviderConfig struct {
	ID           string            `json:"id" toml:"id"`
	Name         string            `json:"name" toml:"name"`
	Type         string            `json:"type" toml:"type"`
	APIBase      string            `json:"apiBase" toml:"api_base"`
	APIKey       string            `json:"apiKey" toml:"api_key"`
	EndpointMode string            `json:"endpointMode,omitempty" toml:"endpoint_mode,omitempty"`
	Enabled      bool              `json:"enabled" toml:"enabled"`
	Priority     int               `json:"priority" toml:"priority"`
	ExtraConfig  map[string]string `json:"extraConfig" toml:"extra_config"`
	LLMs         []ModelEntry      `json:"llms" toml:"llms"`                  // 模型列表
	DefaultModel string            `json:"defaultModel" toml:"default_model"` // 默认模型
}

const (
	ProviderEndpointModeChatCompletions   = "chat_completions"
	ProviderEndpointModeResponses         = "responses"
	ProviderEndpointModeAnthropicMessages = "anthropic_messages"
	ProviderEndpointModeGeminiGenerate    = "gemini_generate_content"

	ApiKeyScopeAll        = "all"
	ApiKeyScopeRestricted = "restricted"
)

func NormalizeProviderEndpointMode(providerType, endpointMode string) string {
	if endpointMode != "" {
		return endpointMode
	}
	switch providerType {
	case "anthropic":
		return ProviderEndpointModeAnthropicMessages
	case "gemini":
		return ProviderEndpointModeGeminiGenerate
	default:
		return ProviderEndpointModeChatCompletions
	}
}

func NormalizeAPIKeyScopeMode(scopeMode string) string {
	switch scopeMode {
	case ApiKeyScopeRestricted:
		return ApiKeyScopeRestricted
	default:
		return ApiKeyScopeAll
	}
}

// ModelEntry represents a model with optional target mapping.
type ModelEntry struct {
	Model  string `json:"model" toml:"model"`   // 对外暴露的模型名称
	Target string `json:"target" toml:"target"` // 实际调用的目标模型
	// Deprecated: Alias is kept for backward compatibility, use Target instead.
	Alias string `json:"alias,omitempty" toml:"alias,omitempty"` // 模型别名（已废弃）
}

// ConfigProvider is an interface for accessing configuration.
// This decouples the provider/gateway packages from the services package.
type ConfigProvider interface {
	GetProviders() []ProviderConfig
	GetAPIKeys() []ApiKeyConfig
	GetEndpoints() []EndpointConfig
	GetGatewayConfig() GatewayConfig
	AddProvider(p ProviderConfig) error
	UpdateProvider(p ProviderConfig) error
	DeleteProvider(id string) error
	AddAPIKey(k ApiKeyConfig) error
	UpdateAPIKey(k ApiKeyConfig) error
	DeleteAPIKey(id string) error
	AddEndpoint(e EndpointConfig) error
	UpdateEndpoint(e EndpointConfig) error
	DeleteEndpoint(id string) error
	SetGatewayConfig(cfg GatewayConfig) error
}
