package config

// GatewayConfig holds gateway server configuration.
type GatewayConfig struct {
	ListenPort      int    `json:"listenPort" toml:"listen_port"`
	DefaultProvider string `json:"defaultProvider" toml:"default_provider"`
	LogLevel        string `json:"logLevel" toml:"log_level"`
	RetryCount      int    `json:"retryCount" toml:"retry_count"`
	RetryIntervalMs int    `json:"retryIntervalMs" toml:"retry_interval_ms"`
	AuthKey         string `json:"authKey" toml:"auth_key"`
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

// ModelEntry represents a model with optional target mapping.
type ModelEntry struct {
	Model  string `json:"model" toml:"model"`   // 对外暴露的模型名称
	Target string `json:"target" toml:"target"` // 实际调用的目标模型
	// Deprecated: Alias is kept for backward compatibility, use Target instead.
	Alias string `json:"alias,omitempty" toml:"alias,omitempty"` // 模型别名（已废弃）
}

// RouteRuleConfig holds a route rule configuration.
type RouteRuleConfig struct {
	Name        string `json:"name" toml:"name"`
	MatchType   string `json:"matchType" toml:"match_type"`
	Pattern     string `json:"pattern" toml:"pattern"`
	ProviderID  string `json:"providerId" toml:"provider_id"`
	TargetModel string `json:"targetModel" toml:"target_model"`
	Priority    int    `json:"priority" toml:"priority"`
	Enabled     bool   `json:"enabled" toml:"enabled"`
}

// ConfigProvider is an interface for accessing configuration.
// This decouples the provider/gateway packages from the services package.
type ConfigProvider interface {
	GetProviders() []ProviderConfig
	GetGatewayConfig() GatewayConfig
	GetRouteRules() []RouteRuleConfig
	AddProvider(p ProviderConfig) error
	UpdateProvider(p ProviderConfig) error
	DeleteProvider(id string) error
	SetGatewayConfig(cfg GatewayConfig) error
	SetRouteRules(rules []RouteRuleConfig) error
}
