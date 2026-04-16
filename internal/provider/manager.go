package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

type HTTPError struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if len(e.Body) == 0 {
		return fmt.Sprintf("status %d", e.StatusCode)
	}
	return fmt.Sprintf("status %d: %s", e.StatusCode, string(e.Body))
}

// Manager manages AI providers.
type Manager struct {
	mu             sync.RWMutex
	providers      map[string]*ProviderRuntime
	client         *http.Client
	streamClient   *http.Client
	configProvider config.ConfigProvider
}

var (
	instance *Manager
	once     sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		transport := &http.Transport{MaxIdleConns: 100, MaxIdleConnsPerHost: 10}
		instance = &Manager{
			providers: make(map[string]*ProviderRuntime),
			client: &http.Client{Timeout: 30 * time.Second, Transport: transport},
			streamClient: &http.Client{Transport: transport},
		}
	})
	return instance
}

func (m *Manager) SetConfigProvider(cp config.ConfigProvider) { m.configProvider = cp }

func (m *Manager) GetModels(providerID string) ([]config.ModelEntry, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, exists := m.providers[providerID]
	if !exists {
		return nil, "", fmt.Errorf("provider not found: %s", providerID)
	}
	return p.Config.LLMs, p.Config.DefaultModel, nil
}

func (m *Manager) SetModels(providerID string, llms []config.ModelEntry, defaultModel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, exists := m.providers[providerID]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerID)
	}
	p.Config.LLMs = llms
	p.Config.DefaultModel = defaultModel
	if m.configProvider != nil {
		if err := m.configProvider.UpdateProvider(p.Config); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ResolveModel(providerID, requestedModel string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
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

func (m *Manager) getConfigProvider() config.ConfigProvider { return m.configProvider }

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
		for i := range p.LLMs {
			if p.LLMs[i].Target == "" && p.LLMs[i].Alias != "" {
				p.LLMs[i].Target = p.LLMs[i].Alias
				p.LLMs[i].Alias = ""
			}
		}
		m.providers[p.ID] = &ProviderRuntime{Config: p, Adapter: adapter, Healthy: false}
	}
}

func (m *Manager) GetAll() []*ProviderRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*ProviderRuntime, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result
}

func (m *Manager) Get(id string) *ProviderRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[id]
}

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
	m.providers[cfg.ID] = &ProviderRuntime{Config: cfg, Adapter: adapter, Healthy: false}
	m.mu.Unlock()
	return nil
}

func (m *Manager) Update(cfg config.ProviderConfig) error {
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
	m.providers[cfg.ID] = &ProviderRuntime{Config: cfg, Adapter: adapter, Healthy: false}
	m.mu.Unlock()
	return nil
}

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

	for _, p := range m.providers {
		if !p.Config.Enabled {
			continue
		}
		for _, mi := range p.Models {
			if mi.ID == model {
				return &RouteDecision{Provider: p, TargetModel: m.resolveTargetModelLocked(p.Config.ID, model)}
			}
		}
	}
	if p := m.resolveByPrefix(model); p != nil {
		return &RouteDecision{Provider: p, TargetModel: m.resolveTargetModelLocked(p.Config.ID, model)}
	}
	if m.configProvider != nil {
		gwCfg := m.configProvider.GetGatewayConfig()
		if gwCfg.DefaultProvider != "" {
			if p, ok := m.providers[gwCfg.DefaultProvider]; ok && p.Config.Enabled {
				return &RouteDecision{Provider: p, TargetModel: m.resolveTargetModelLocked(p.Config.ID, model)}
			}
		}
	}
	for _, p := range m.providers {
		if p.Config.Enabled {
			return &RouteDecision{Provider: p, TargetModel: m.resolveTargetModelLocked(p.Config.ID, model)}
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

func (m *Manager) resolveTargetModelLocked(providerID, requestedModel string) string {
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

func (m *Manager) DoRequest(ctx context.Context, p *ProviderRuntime, req *protocol.InternalRequest) (*http.Response, error) {
	body, path, err := p.Adapter.BuildRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	httpReq, err := p.Adapter.BuildHTTPRequest(ctx, p.Config.APIBase, p.Config.APIKey, "POST", path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	return m.httpClient(req.Stream).Do(httpReq)
}

func (m *Manager) DoRequestRaw(ctx context.Context, p *ProviderRuntime, method, path string, body []byte, stream bool) (*http.Response, error) {
	httpReq, err := p.Adapter.BuildHTTPRequest(ctx, p.Config.APIBase, p.Config.APIKey, method, path, body)
	if err != nil {
		return nil, err
	}
	return m.httpClient(stream).Do(httpReq)
}

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
			lastErr = &HTTPError{StatusCode: resp.StatusCode, Body: body, Header: resp.Header.Clone()}
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

func (m *Manager) httpClient(stream bool) *http.Client {
	if stream && m.streamClient != nil {
		return m.streamClient
	}
	if m.client != nil {
		return m.client
	}
	return http.DefaultClient
}

func ProviderListJSON(providers []*ProviderRuntime) string {
	type providerInfo struct {
		ID           string           `json:"id"`
		Name         string           `json:"name"`
		Type         string           `json:"type"`
		APIBase      string           `json:"apiBase"`
		EndpointMode string           `json:"endpointMode,omitempty"`
		Enabled      bool             `json:"enabled"`
		Healthy      bool             `json:"healthy"`
		Priority     int              `json:"priority"`
		LLMs         []config.ModelEntry `json:"llms"`
		DefaultModel string           `json:"defaultModel"`
	}
	list := make([]providerInfo, 0, len(providers))
	for _, p := range providers {
		list = append(list, providerInfo{
			ID:           p.Config.ID,
			Name:         p.Config.Name,
			Type:         p.Config.Type,
			APIBase:      p.Config.APIBase,
			EndpointMode: p.Config.EndpointMode,
			Enabled:      p.Config.Enabled,
			Healthy:      p.Healthy,
			Priority:     p.Config.Priority,
			LLMs:         p.Config.LLMs,
			DefaultModel: p.Config.DefaultModel,
		})
	}
	data, _ := json.Marshal(list)
	return string(data)
}