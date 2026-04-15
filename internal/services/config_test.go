package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigServiceSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_proxy.db")
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
		keyPath:    filepath.Join(filepath.Dir(path), "icoo_proxy.key"),
	}
	defer service.Close()

	err := service.SetGatewayConfig(GatewayConfig{
		ListenPort:      26790,
		DefaultProvider: "openai-main",
		LogLevel:        "debug",
		RetryCount:      3,
		RetryIntervalMs: 900,
		AuthKey:         "gateway-secret",
	})
	if err != nil {
		t.Fatalf("SetGatewayConfig() error = %v", err)
	}

	loaded := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
		keyPath:    filepath.Join(filepath.Dir(path), "icoo_proxy.key"),
	}
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
	if cfg.AuthKey != "gateway-secret" {
		t.Fatalf("AuthKey = %q", cfg.AuthKey)
	}
}

func TestConfigServiceLoadAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_proxy.db")
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
		keyPath:    filepath.Join(filepath.Dir(path), "icoo_proxy.key"),
	}
	defer service.Close()

	if err := service.SetGatewayConfig(GatewayConfig{
		DefaultProvider: "custom-provider",
	}); err != nil {
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
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
		keyPath:    filepath.Join(filepath.Dir(path), "icoo_proxy.key"),
	}
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

[[providers]]
id = "openai-main"
name = "OpenAI"
type = "openai"
api_base = "https://api.openai.com/v1"
api_key = "secret"
enabled = true
priority = 10
default_model = "gpt-4o"

[[providers.llms]]
model = "chat-default"
target = "gpt-4o"

[[route_rules]]
pattern = "gpt-*"
provider_id = "openai-main"
priority = 100
enabled = true
`)
	if err := os.WriteFile(legacyPath, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := &ConfigService{
		config:           defaultConfig(),
		configPath:       dbPath,
		legacyConfigPath: legacyPath,
		keyPath:          filepath.Join(dir, "icoo_proxy.key"),
	}
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

	rules := service.GetRouteRules()
	if len(rules) != 1 {
		t.Fatalf("route rules len = %d", len(rules))
	}
	if rules[0].Pattern != "gpt-*" {
		t.Fatalf("Pattern = %q", rules[0].Pattern)
	}
	if rules[0].MatchType != "model" {
		t.Fatalf("MatchType = %q", rules[0].MatchType)
	}
}

func TestConfigServiceEncryptsProviderAPIKeyAtRest(t *testing.T) {
	dir := t.TempDir()
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: filepath.Join(dir, "icoo_proxy.db"),
		keyPath:    filepath.Join(dir, "icoo_proxy.key"),
	}
	defer service.Close()

	err := service.AddProvider(ProviderConfig{
		ID:           "openai-main",
		Name:         "OpenAI",
		Type:         "openai",
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "super-secret-key",
		EndpointMode: "responses",
		Enabled:      true,
		Priority:     10,
	})
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

	loaded := &ConfigService{
		config:     defaultConfig(),
		configPath: filepath.Join(dir, "icoo_proxy.db"),
		keyPath:    filepath.Join(dir, "icoo_proxy.key"),
	}
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

func TestConfigServicePersistsGatewayAuthKey(t *testing.T) {
	dir := t.TempDir()
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: filepath.Join(dir, "icoo_proxy.db"),
		keyPath:    filepath.Join(dir, "icoo_proxy.key"),
	}
	defer service.Close()

	err := service.SetGatewayConfig(GatewayConfig{
		ListenPort:      16790,
		DefaultProvider: "openai-main",
		LogLevel:        "info",
		RetryCount:      2,
		RetryIntervalMs: 500,
		AuthKey:         "gateway-secret",
	})
	if err != nil {
		t.Fatalf("SetGatewayConfig() error = %v", err)
	}

	loaded := &ConfigService{
		config:     defaultConfig(),
		configPath: filepath.Join(dir, "icoo_proxy.db"),
		keyPath:    filepath.Join(dir, "icoo_proxy.key"),
	}
	defer loaded.Close()
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.GetGatewayConfig().AuthKey != "gateway-secret" {
		t.Fatalf("AuthKey = %q", loaded.GetGatewayConfig().AuthKey)
	}
}

func TestConfigServicePersistsExtendedRouteRules(t *testing.T) {
	dir := t.TempDir()
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: filepath.Join(dir, "icoo_proxy.db"),
		keyPath:    filepath.Join(dir, "icoo_proxy.key"),
	}
	defer service.Close()

	rules := []RouteRuleConfig{
		{
			Name:        "translate-to-gemini",
			MatchType:   "user_contains",
			Pattern:     "翻译",
			ProviderID:  "gemini-main",
			TargetModel: "gemini-2.5-flash",
			Priority:    120,
			Enabled:     true,
		},
	}
	if err := service.SetRouteRules(rules); err != nil {
		t.Fatalf("SetRouteRules() error = %v", err)
	}

	loaded := &ConfigService{
		config:     defaultConfig(),
		configPath: filepath.Join(dir, "icoo_proxy.db"),
		keyPath:    filepath.Join(dir, "icoo_proxy.key"),
	}
	defer loaded.Close()
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := loaded.GetRouteRules()
	if len(got) != 1 {
		t.Fatalf("len(got) = %d", len(got))
	}
	if got[0].Name != "translate-to-gemini" {
		t.Fatalf("Name = %q", got[0].Name)
	}
	if got[0].MatchType != "user_contains" {
		t.Fatalf("MatchType = %q", got[0].MatchType)
	}
	if got[0].TargetModel != "gemini-2.5-flash" {
		t.Fatalf("TargetModel = %q", got[0].TargetModel)
	}
}
