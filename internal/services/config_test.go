package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"icoo_proxy/internal/config"
)

func TestConfigServiceSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_proxy.db")
	service := &ConfigService{config: defaultConfig(), configPath: path, keyPath: filepath.Join(filepath.Dir(path), "icoo_proxy.key")}
	defer service.Close()

	err := service.SetGatewayConfig(GatewayConfig{ListenPort: 26790, DefaultProvider: "openai-main", LogLevel: "debug", RetryCount: 3, RetryIntervalMs: 900})
	if err != nil {
		t.Fatalf("SetGatewayConfig() error = %v", err)
	}
	if err := service.AddAPIKey(APIKeyConfig{ID: "key-1", Name: "Default Key", Key: "gateway-secret", Enabled: true}); err != nil {
		t.Fatalf("AddAPIKey() error = %v", err)
	}
	if err := service.AddEndpoint(EndpointConfig{ID: "endpoint-1", Name: "Default Endpoint", ProviderID: "openai-main", Path: "/v1/chat/completions", Method: "POST", Capability: "chat", RequestProtocol: "openai_chat", ResponseProtocol: "openai_chat", Enabled: true, Priority: 10, IsDefault: true}); err != nil {
		t.Fatalf("AddEndpoint() error = %v", err)
	}

	loaded := &ConfigService{config: defaultConfig(), configPath: path, keyPath: filepath.Join(filepath.Dir(path), "icoo_proxy.key")}
	defer loaded.Close()
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := loaded.GetGatewayConfig()
	if cfg.ListenPort != 26790 {
		t.Fatalf("ListenPort = %d", cfg.ListenPort)
	}
	if cfg.DefaultProvider != "openai-main" {
		t.Fatalf("DefaultProvider = %q", cfg.DefaultProvider)
	}
	if cfg.AuthKey != "" {
		t.Fatalf("AuthKey = %q", cfg.AuthKey)
	}

	apiKeys := loaded.GetAPIKeys()
	if len(apiKeys) != 1 {
		t.Fatalf("apiKeys len = %d", len(apiKeys))
	}
	if apiKeys[0].Key != "gateway-secret" {
		t.Fatalf("apiKeys[0].Key = %q", apiKeys[0].Key)
	}

	endpoints := loaded.GetEndpoints()
	if len(endpoints) != 1 {
		t.Fatalf("endpoints len = %d", len(endpoints))
	}
	if endpoints[0].Path != "/v1/chat/completions" {
		t.Fatalf("endpoints[0].Path = %q", endpoints[0].Path)
	}
}

func TestConfigServiceSetGatewayConfigMigratesLegacyAuthKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_proxy.db")
	service := &ConfigService{config: defaultConfig(), configPath: path, keyPath: filepath.Join(filepath.Dir(path), "icoo_proxy.key")}
	defer service.Close()

	err := service.SetGatewayConfig(GatewayConfig{
		ListenHost:      "127.0.0.1",
		ListenPort:      26790,
		DefaultProvider: "openai-main",
		LogLevel:        "debug",
		RetryCount:      3,
		RetryIntervalMs: 900,
		AuthKey:         "legacy-gateway-secret",
	})
	if err != nil {
		t.Fatalf("SetGatewayConfig() error = %v", err)
	}

	cfg := service.GetGatewayConfig()
	if cfg.AuthKey != "" {
		t.Fatalf("AuthKey = %q", cfg.AuthKey)
	}

	apiKeys := service.GetAPIKeys()
	if len(apiKeys) != 1 {
		t.Fatalf("apiKeys len = %d", len(apiKeys))
	}
	if apiKeys[0].Key != "legacy-gateway-secret" {
		t.Fatalf("apiKeys[0].Key = %q", apiKeys[0].Key)
	}
	if apiKeys[0].ScopeMode != config.ApiKeyScopeAll {
		t.Fatalf("apiKeys[0].ScopeMode = %q", apiKeys[0].ScopeMode)
	}
}

func TestConfigServiceLoadAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_proxy.db")
	service := &ConfigService{config: defaultConfig(), configPath: path, keyPath: filepath.Join(filepath.Dir(path), "icoo_proxy.key")}
	defer service.Close()

	if err := service.SetGatewayConfig(GatewayConfig{DefaultProvider: "custom-provider"}); err != nil {
		t.Fatalf("SetGatewayConfig() error = %v", err)
	}
	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	gwCfg := service.GetGatewayConfig()
	if gwCfg.DefaultProvider != "custom-provider" {
		t.Fatalf("DefaultProvider = %q, want custom-provider", gwCfg.DefaultProvider)
	}
	if gwCfg.ListenPort != 16790 {
		t.Fatalf("ListenPort = %d, want default", gwCfg.ListenPort)
	}
	if gwCfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want default", gwCfg.LogLevel)
	}
}

func TestConfigServiceEnsureDatabaseCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_proxy.db")
	service := &ConfigService{config: defaultConfig(), configPath: path, keyPath: filepath.Join(filepath.Dir(path), "icoo_proxy.key")}
	defer service.Close()

	if err := service.ensureDatabase(); err != nil {
		t.Fatalf("ensureDatabase() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected database file to exist: %v", err)
	}
}

func TestConfigServiceMigratesLegacyTOML(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "icoo_proxy.db")
	legacyPath := filepath.Join(dir, "icoo_proxy.toml")
	content := []byte(`
[gateway]
listen_port = 26790
default_provider = "openai-main"
log_level = "debug"
retry_count = 3
retry_interval_ms = 900
auth_key = "gateway-secret"

[[providers]]
id = "openai-main"
name = "OpenAI"
type = "openai"
api_base = "https://api.openai.com/v1"
api_key = "secret"
enabled = true
priority = 10
default_model = "gpt-4o"
endpoint_mode = "responses"

[[providers.llms]]
model = "chat-default"
target = "gpt-4o"
`)
	if err := os.WriteFile(legacyPath, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := &ConfigService{config: defaultConfig(), configPath: dbPath, legacyConfigPath: legacyPath, keyPath: filepath.Join(dir, "icoo_proxy.key")}
	defer service.Close()
	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	gwCfg := service.GetGatewayConfig()
	if gwCfg.ListenPort != 26790 {
		t.Fatalf("ListenPort = %d", gwCfg.ListenPort)
	}
	if gwCfg.DefaultProvider != "openai-main" {
		t.Fatalf("DefaultProvider = %q", gwCfg.DefaultProvider)
	}
	if gwCfg.AuthKey != "" {
		t.Fatalf("AuthKey = %q", gwCfg.AuthKey)
	}

	providers := service.GetProviders()
	if len(providers) != 1 {
		t.Fatalf("providers len = %d", len(providers))
	}
	if providers[0].DefaultModel != "gpt-4o" {
		t.Fatalf("DefaultModel = %q", providers[0].DefaultModel)
	}
	if len(providers[0].LLMs) != 1 || providers[0].LLMs[0].Target != "gpt-4o" {
		t.Fatalf("unexpected provider llms: %+v", providers[0].LLMs)
	}

	apiKeys := service.GetAPIKeys()
	if len(apiKeys) != 1 {
		t.Fatalf("apiKeys len = %d", len(apiKeys))
	}
	if apiKeys[0].Key != "gateway-secret" {
		t.Fatalf("apiKeys[0].Key = %q", apiKeys[0].Key)
	}
	if apiKeys[0].ScopeMode != config.ApiKeyScopeAll {
		t.Fatalf("apiKeys[0].ScopeMode = %q", apiKeys[0].ScopeMode)
	}

	endpoints := service.GetEndpoints()
	if len(endpoints) != 1 {
		t.Fatalf("endpoints len = %d", len(endpoints))
	}
	if endpoints[0].ProviderID != "openai-main" {
		t.Fatalf("endpoints[0].ProviderID = %q", endpoints[0].ProviderID)
	}
	if endpoints[0].Path != "/v1/responses" {
		t.Fatalf("endpoints[0].Path = %q", endpoints[0].Path)
	}
	if !endpoints[0].IsDefault {
		t.Fatalf("expected migrated endpoint to be default")
	}
}

func TestConfigServiceEncryptsProviderAPIKeyAtRest(t *testing.T) {
	dir := t.TempDir()
	service := &ConfigService{config: defaultConfig(), configPath: filepath.Join(dir, "icoo_proxy.db"), keyPath: filepath.Join(dir, "icoo_proxy.key")}
	defer service.Close()

	err := service.AddProvider(ProviderConfig{ID: "openai-main", Name: "OpenAI", Type: "openai", APIBase: "https://api.openai.com/v1", APIKey: "super-secret-key", EndpointMode: "responses", Enabled: true, Priority: 10})
	if err != nil {
		t.Fatalf("AddProvider() error = %v", err)
	}

	providers := service.GetProviders()
	if len(providers) != 1 {
		t.Fatalf("providers len = %d", len(providers))
	}
	if providers[0].APIKey != "super-secret-key" {
		t.Fatalf("APIKey = %q", providers[0].APIKey)
	}

	var stored providerRecord
	if err := service.db.Where("id = ?", "openai-main").Take(&stored).Error; err != nil {
		t.Fatalf("query stored provider error = %v", err)
	}
	if stored.APIKey == "super-secret-key" {
		t.Fatalf("API key was stored in plaintext")
	}
	if !strings.HasPrefix(stored.APIKey, "enc:v1:") {
		t.Fatalf("API key = %q, want encrypted prefix", stored.APIKey)
	}

	loaded := &ConfigService{config: defaultConfig(), configPath: filepath.Join(dir, "icoo_proxy.db"), keyPath: filepath.Join(dir, "icoo_proxy.key")}
	defer loaded.Close()
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	reloaded := loaded.GetProviders()
	if len(reloaded) != 1 {
		t.Fatalf("reloaded providers len = %d", len(reloaded))
	}
	if reloaded[0].APIKey != "super-secret-key" {
		t.Fatalf("reloaded APIKey = %q", reloaded[0].APIKey)
	}
	if reloaded[0].EndpointMode != "responses" {
		t.Fatalf("reloaded EndpointMode = %q", reloaded[0].EndpointMode)
	}
}
