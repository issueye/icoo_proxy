package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"icoo_proxy/internal/config"
	"icoo_proxy/internal/protocol"
)

type routeTestConfigProvider struct {
	providers []config.ProviderConfig
	gateway   config.GatewayConfig
	rules     []config.RouteRuleConfig
}

func (r *routeTestConfigProvider) GetProviders() []config.ProviderConfig        { return r.providers }
func (r *routeTestConfigProvider) GetGatewayConfig() config.GatewayConfig       { return r.gateway }
func (r *routeTestConfigProvider) GetRouteRules() []config.RouteRuleConfig      { return r.rules }
func (r *routeTestConfigProvider) AddProvider(p config.ProviderConfig) error    { return nil }
func (r *routeTestConfigProvider) UpdateProvider(p config.ProviderConfig) error { return nil }
func (r *routeTestConfigProvider) DeleteProvider(id string) error               { return nil }
func (r *routeTestConfigProvider) SetGatewayConfig(cfg config.GatewayConfig) error {
	r.gateway = cfg
	return nil
}
func (r *routeTestConfigProvider) SetRouteRules(rules []config.RouteRuleConfig) error {
	r.rules = rules
	return nil
}

func TestResolveRequestMatchesUserContainsRule(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.mu.Lock()
	m.providers = map[string]*ProviderRuntime{
		"gemini-main": {
			Config: config.ProviderConfig{
				ID:      "gemini-main",
				Name:    "Gemini",
				Type:    "gemini",
				Enabled: true,
			},
			Healthy: true,
		},
	}
	m.mu.Unlock()

	m.SetConfigProvider(&routeTestConfigProvider{
		rules: []config.RouteRuleConfig{
			{
				Name:        "translate",
				MatchType:   "user_contains",
				Pattern:     "翻译",
				ProviderID:  "gemini-main",
				TargetModel: "gemini-2.5-flash",
				Priority:    100,
				Enabled:     true,
			},
		},
	})

	decision := m.ResolveRequest(&protocol.InternalRequest{
		Model: "gpt-4o",
		Messages: []protocol.InternalMessage{
			{
				Role: "user",
				Content: []protocol.ContentBlock{
					{Type: "text", Text: "请帮我翻译这段英文"},
				},
			},
		},
	})
	if decision == nil || decision.Provider == nil {
		t.Fatalf("expected a route decision")
	}
	if decision.Provider.Config.ID != "gemini-main" {
		t.Fatalf("ProviderID = %q", decision.Provider.Config.ID)
	}
	if decision.TargetModel != "gemini-2.5-flash" {
		t.Fatalf("TargetModel = %q", decision.TargetModel)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTestConnectionUsesStoredAPIKeyWhenRequestKeyMissing(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.mu.Lock()
	m.providers = map[string]*ProviderRuntime{
		"openai-main": {
			Config: config.ProviderConfig{
				ID:      "openai-main",
				Name:    "OpenAI",
				Type:    "openai",
				APIBase: "https://api.openai.com/v1",
				APIKey:  "stored-secret",
				Enabled: true,
			},
			Adapter: &protocol.OpenAIAdapter{},
			Healthy: true,
		},
	}
	m.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Authorization"); got != "Bearer stored-secret" {
				t.Fatalf("Authorization = %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"object":"list","data":[]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	m.mu.Unlock()

	err := m.TestConnection(context.Background(), config.ProviderConfig{
		ID:      "openai-main",
		Name:    "OpenAI",
		Type:    "openai",
		APIBase: "https://api.openai.com/v1",
		APIKey:  "",
	})
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
}
