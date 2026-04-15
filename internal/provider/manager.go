package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"icoo_proxy/internal/config"
	"icoo_proxy/internal/protocol"
)

// ProviderRuntime holds the runtime state of a provider.
type ProviderRuntime struct {
	Config  config.ProviderConfig
	Adapter protocol.ProtocolAdapter
	Healthy bool
	Models  []protocol.ModelInfo
}

type RouteDecision struct {
	Provider    *ProviderRuntime
	TargetModel string
	Rule        *config.RouteRuleConfig
}

// Manager manages AI providers.
type Manager struct {
	mu             sync.RWMutex
	providers      map[string]*ProviderRuntime
	client         *http.Client
	configProvider config.ConfigProvider
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager returns the singleton Manager instance.
func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			providers: make(map[string]*ProviderRuntime),
			client: &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
				},
			},
		}
	})
	return instance
}

// SetConfigProvider sets the config provider (called during startup).
func (m *Manager) SetConfigProvider(cp config.ConfigProvider) {
	m.configProvider = cp
}

// GetModels returns the model list and default model for a provider.
func (m *Manager) GetModels(providerID string) ([]config.ModelEntry, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, exists := m.providers[providerID]
	if !exists {
		return nil, "", fmt.Errorf("provider not found: %s", providerID)
	}
	return p.Config.LLMs, p.Config.DefaultModel, nil
}

// SetModels updates the model list and default model for a provider.
func (m *Manager) SetModels(providerID string, llms []config.ModelEntry, defaultModel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.providers[providerID]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerID)
	}

	// Update config
	p.Config.LLMs = llms
	p.Config.DefaultModel = defaultModel

	// Persist to config provider
	if m.configProvider != nil {
		if err := m.configProvider.UpdateProvider(p.Config); err != nil {
			return err
		}
	}

	return nil
}

// ResolveModel maps a requested model name to the actual target model based on provider's LLMs config.
func (m *Manager) ResolveModel(providerID, requestedModel string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.providers[providerID]
	if !exists {
		return requestedModel // 供应商不存在，返回原始名称
	}

	// 遍历 LLMs 查找匹配
	for _, entry := range p.Config.LLMs {
		if entry.Model == requestedModel {
			// 如果配置了 target，返回 target；否则返回 model
			if entry.Target != "" {
				return entry.Target
			}
			return entry.Model
		}
	}

	// 未找到映射，返回原始请求
	return requestedModel
}

func (m *Manager) getConfigProvider() config.ConfigProvider {
	return m.configProvider
}

func (m *Manager) GetGatewayConfig() config.GatewayConfig {
	if m.configProvider == nil {
		return config.GatewayConfig{}
	}
	return m.configProvider.GetGatewayConfig()
}

func adapterForConfig(cfg config.ProviderConfig) (protocol.ProtocolAdapter, error) {
	switch cfg.Type {
	case "openai":
		if config.NormalizeProviderEndpointMode(cfg.Type, cfg.EndpointMode) == config.ProviderEndpointModeResponses {
			return &protocol.OpenAIResponsesAdapter{}, nil
		}
		return &protocol.OpenAIAdapter{}, nil
	default:
		return protocol.GetAdapter(cfg.Type)
	}
}

// LoadFromConfig loads providers from the configuration service.
func (m *Manager) LoadFromConfig() {
	if m.configProvider == nil {
		return
	}
	cfg := m.configProvider.GetProviders()
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers = make(map[string]*ProviderRuntime)
	for _, p := range cfg {
		adapter, err := adapterForConfig(p)
		if err != nil {
			continue
		}

		// 向后兼容：迁移 alias 字段到 target
		for i := range p.LLMs {
			if p.LLMs[i].Target == "" && p.LLMs[i].Alias != "" {
				p.LLMs[i].Target = p.LLMs[i].Alias
				p.LLMs[i].Alias = ""
			}
		}

		m.providers[p.ID] = &ProviderRuntime{
			Config:  p,
			Adapter: adapter,
			Healthy: false,
		}
	}
}

// GetAll returns all provider runtimes.
func (m *Manager) GetAll() []*ProviderRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*ProviderRuntime, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result
}

// Get returns a provider runtime by ID.
func (m *Manager) Get(id string) *ProviderRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[id]
}

// Add adds a new provider from config.
func (m *Manager) Add(cfg config.ProviderConfig) error {
	cfg.EndpointMode = config.NormalizeProviderEndpointMode(cfg.Type, cfg.EndpointMode)
	adapter, err := adapterForConfig(cfg)
	if err != nil {
		return fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
	if strings.TrimSpace(cfg.ID) == "" {
		cfg.ID = fmt.Sprintf("provider-%d", time.Now().UnixMilli())
	}
	if m.configProvider != nil {
		if err := m.configProvider.AddProvider(cfg); err != nil {
			return err
		}
	}
	m.mu.Lock()
	m.providers[cfg.ID] = &ProviderRuntime{
		Config:  cfg,
		Adapter: adapter,
		Healthy: false,
	}
	m.mu.Unlock()
	return nil
}

// Update updates an existing provider.
func (m *Manager) Update(cfg config.ProviderConfig) error {
	// Get existing provider to preserve API key if not provided
	m.mu.RLock()
	existing, exists := m.providers[cfg.ID]
	m.mu.RUnlock()

	if exists && cfg.APIKey == "" {
		cfg.APIKey = existing.Config.APIKey
	}
	cfg.EndpointMode = config.NormalizeProviderEndpointMode(cfg.Type, cfg.EndpointMode)

	adapter, err := adapterForConfig(cfg)
	if err != nil {
		return fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
	if m.configProvider != nil {
		if err := m.configProvider.UpdateProvider(cfg); err != nil {
			return err
		}
	}
	m.mu.Lock()
	m.providers[cfg.ID] = &ProviderRuntime{
		Config:  cfg,
		Adapter: adapter,
		Healthy: false,
	}
	m.mu.Unlock()
	return nil
}

// Delete removes a provider.
func (m *Manager) Delete(id string) error {
	if m.configProvider != nil {
		if err := m.configProvider.DeleteProvider(id); err != nil {
			return err
		}
	}
	m.mu.Lock()
	delete(m.providers, id)
	m.mu.Unlock()
	return nil
}

// TestConnection tests connectivity to a provider.
func (m *Manager) TestConnection(ctx context.Context, cfg config.ProviderConfig) error {
	m.mu.RLock()
	existing, exists := m.providers[cfg.ID]
	m.mu.RUnlock()

	if exists && strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = existing.Config.APIKey
	}

	cfg.EndpointMode = config.NormalizeProviderEndpointMode(cfg.Type, cfg.EndpointMode)
	adapter, err := adapterForConfig(cfg)
	if err != nil {
		return fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
	req, err := adapter.ListModelsRequest(ctx, cfg.APIBase, cfg.APIKey)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// RefreshModels fetches model lists from all enabled providers.
func (m *Manager) RefreshModels(ctx context.Context) {
	m.mu.RLock()
	providers := make([]*ProviderRuntime, 0)
	for _, p := range m.providers {
		if p.Config.Enabled {
			providers = append(providers, p)
		}
	}
	m.mu.RUnlock()

	for _, p := range providers {
		models, err := m.fetchModels(ctx, p)
		m.mu.Lock()
		if err == nil {
			p.Models = models
			p.Healthy = true
		} else {
			p.Healthy = false
		}
		m.mu.Unlock()
	}
}

func (m *Manager) fetchModels(ctx context.Context, p *ProviderRuntime) ([]protocol.ModelInfo, error) {
	req, err := p.Adapter.ListModelsRequest(ctx, p.Config.APIBase, p.Config.APIKey)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return p.Adapter.ParseModelsResponse(body)
}

// GetAllModels returns aggregated model list from all enabled providers.
func (m *Manager) GetAllModels() []protocol.ModelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var models []protocol.ModelInfo
	for _, p := range m.providers {
		if p.Config.Enabled {
			models = append(models, p.Models...)
		}
	}
	return models
}

// ResolveProvider finds the best provider for a given model name.
func (m *Manager) ResolveProvider(model string) *ProviderRuntime {
	decision := m.ResolveRequest(&protocol.InternalRequest{Model: model})
	if decision == nil {
		return nil
	}
	return decision.Provider
}

func (m *Manager) ResolveRequest(req *protocol.InternalRequest) *RouteDecision {
	model := req.Model
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. Check route rules
	if decision := m.resolveByRouteRules(req); decision != nil {
		return decision
	}
	// 2. Try exact model match
	for _, p := range m.providers {
		if !p.Config.Enabled {
			continue
		}
		for _, mi := range p.Models {
			if mi.ID == model {
				return &RouteDecision{
					Provider:    p,
					TargetModel: m.resolveTargetModelLocked(p.Config.ID, model, ""),
				}
			}
		}
	}
	// 3. Prefix matching
	if p := m.resolveByPrefix(model); p != nil {
		return &RouteDecision{
			Provider:    p,
			TargetModel: m.resolveTargetModelLocked(p.Config.ID, model, ""),
		}
	}
	// 4. Default provider
	if m.configProvider != nil {
		gwCfg := m.configProvider.GetGatewayConfig()
		if gwCfg.DefaultProvider != "" {
			if p, ok := m.providers[gwCfg.DefaultProvider]; ok && p.Config.Enabled {
				return &RouteDecision{
					Provider:    p,
					TargetModel: m.resolveTargetModelLocked(p.Config.ID, model, ""),
				}
			}
		}
	}
	// 5. First enabled provider
	for _, p := range m.providers {
		if p.Config.Enabled {
			return &RouteDecision{
				Provider:    p,
				TargetModel: m.resolveTargetModelLocked(p.Config.ID, model, ""),
			}
		}
	}
	return nil
}

func (m *Manager) resolveByRouteRules(req *protocol.InternalRequest) *RouteDecision {
	if m.configProvider == nil {
		return nil
	}
	rules := m.configProvider.GetRouteRules()
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if m.matchRouteRule(rule, req) {
			if p, ok := m.providers[rule.ProviderID]; ok && p.Config.Enabled {
				return &RouteDecision{
					Provider:    p,
					TargetModel: m.resolveTargetModelLocked(p.Config.ID, req.Model, rule.TargetModel),
					Rule:        &rule,
				}
			}
		}
	}
	return nil
}

func (m *Manager) resolveByPrefix(model string) *ProviderRuntime {
	prefixMap := map[string]string{
		"gpt-":    "openai",
		"o1-":     "openai",
		"o3-":     "openai",
		"dall-e":  "openai",
		"text-":   "openai",
		"claude-": "anthropic",
		"gemini-": "gemini",
		"models/": "gemini",
	}
	for prefix, providerType := range prefixMap {
		if strings.HasPrefix(model, prefix) {
			for _, p := range m.providers {
				if p.Config.Enabled && p.Config.Type == providerType {
					return p
				}
			}
		}
	}
	return nil
}

func (m *Manager) matchPattern(pattern, model string) bool {
	if pattern == model {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(model, prefix)
	}
	return false
}

func (m *Manager) matchRouteRule(rule config.RouteRuleConfig, req *protocol.InternalRequest) bool {
	pattern := strings.TrimSpace(rule.Pattern)
	if pattern == "" {
		return false
	}

	switch strings.TrimSpace(rule.MatchType) {
	case "", "model":
		return m.matchPattern(pattern, req.Model)
	case "system_contains":
		return strings.Contains(strings.ToLower(req.System), strings.ToLower(pattern))
	case "message_contains":
		return strings.Contains(strings.ToLower(m.requestText(req, "")), strings.ToLower(pattern))
	case "user_contains":
		return strings.Contains(strings.ToLower(m.requestText(req, "user")), strings.ToLower(pattern))
	case "assistant_contains":
		return strings.Contains(strings.ToLower(m.requestText(req, "assistant")), strings.ToLower(pattern))
	default:
		return false
	}
}

func (m *Manager) requestText(req *protocol.InternalRequest, role string) string {
	var parts []string
	for _, msg := range req.Messages {
		if role != "" && msg.Role != role {
			continue
		}
		for _, block := range msg.Content {
			if block.Type == "text" && block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func (m *Manager) resolveTargetModelLocked(providerID, requestedModel, override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	p, exists := m.providers[providerID]
	if !exists {
		return requestedModel
	}
	for _, entry := range p.Config.LLMs {
		if entry.Model == requestedModel {
			if entry.Target != "" {
				return entry.Target
			}
			return entry.Model
		}
	}
	return requestedModel
}

// DoRequest sends a request through the provider's adapter.
func (m *Manager) DoRequest(ctx context.Context, p *ProviderRuntime, req *protocol.InternalRequest) (*http.Response, error) {
	body, path, err := p.Adapter.BuildRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	httpReq, err := p.Adapter.BuildHTTPRequest(ctx, p.Config.APIBase, p.Config.APIKey, "POST", path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	return m.client.Do(httpReq)
}

// DoRequestRaw forwards a raw HTTP request to a provider.
func (m *Manager) DoRequestRaw(ctx context.Context, p *ProviderRuntime, method, path string, body []byte) (*http.Response, error) {
	httpReq, err := p.Adapter.BuildHTTPRequest(ctx, p.Config.APIBase, p.Config.APIKey, method, path, body)
	if err != nil {
		return nil, err
	}
	return m.client.Do(httpReq)
}

// DoRequestWithRetry sends a request with retry logic.
func (m *Manager) DoRequestWithRetry(ctx context.Context, p *ProviderRuntime, req *protocol.InternalRequest) (*http.Response, error) {
	maxRetries := 2
	retryInterval := 500 * time.Millisecond
	if m.configProvider != nil {
		gwCfg := m.configProvider.GetGatewayConfig()
		maxRetries = gwCfg.RetryCount
		retryInterval = time.Duration(gwCfg.RetryIntervalMs) * time.Millisecond
	}

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		resp, err := m.DoRequest(ctx, p, req)
		if err != nil {
			lastErr = err
			if i < maxRetries {
				time.Sleep(retryInterval)
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
			if i < maxRetries {
				time.Sleep(retryInterval)
				continue
			}
			return nil, lastErr
		}
		return resp, nil
	}
	return nil, lastErr
}

// ProviderListJSON returns the provider list as JSON for the frontend.
func ProviderListJSON(providers []*ProviderRuntime) string {
	type providerInfo struct {
		ID           string              `json:"id"`
		Name         string              `json:"name"`
		Type         string              `json:"type"`
		APIBase      string              `json:"apiBase"`
		EndpointMode string              `json:"endpointMode,omitempty"`
		Enabled      bool                `json:"enabled"`
		Healthy      bool                `json:"healthy"`
		Priority     int                 `json:"priority"`
		ModelCount   int                 `json:"modelCount"`
		LLMs         []config.ModelEntry `json:"llms,omitempty"`
		DefaultModel string              `json:"defaultModel,omitempty"`
	}
	list := make([]providerInfo, 0, len(providers))
	for _, p := range providers {
		list = append(list, providerInfo{
			ID:           p.Config.ID,
			Name:         p.Config.Name,
			Type:         p.Config.Type,
			APIBase:      p.Config.APIBase,
			EndpointMode: config.NormalizeProviderEndpointMode(p.Config.Type, p.Config.EndpointMode),
			Enabled:      p.Config.Enabled,
			Healthy:      p.Healthy,
			Priority:     p.Config.Priority,
			ModelCount:   len(p.Models),
			LLMs:         p.Config.LLMs,
			DefaultModel: p.Config.DefaultModel,
		})
	}
	data, _ := json.Marshal(list)
	return string(data)
}
