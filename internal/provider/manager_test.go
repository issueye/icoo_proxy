package provider

import (
	"context"
	"errors"
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
}

func (r *routeTestConfigProvider) GetProviders() []config.ProviderConfig { return r.providers }
func (r *routeTestConfigProvider) GetGatewayConfig() config.GatewayConfig { return r.gateway }
func (r *routeTestConfigProvider) AddProvider(p config.ProviderConfig) error {
	r.providers = append(r.providers, p)
	return nil
}
func (r *routeTestConfigProvider) UpdateProvider(p config.ProviderConfig) error {
	for i := range r.providers {
		if r.providers[i].ID == p.ID {
			r.providers[i] = p
			return nil
		}
	}
	r.providers = append(r.providers, p)
	return nil
}
func (r *routeTestConfigProvider) DeleteProvider(id string) error { return nil }
func (r *routeTestConfigProvider) SetGatewayConfig(cfg config.GatewayConfig) error {
	r.gateway = cfg
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTestConnectionUsesStoredAPIKeyWhenRequestKeyMissing(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.mu.Lock()
	m.providers = map[string]*ProviderRuntime{
		"openai-main": {
			Config: config.ProviderConfig{ID: "openai-main", Name: "OpenAI", Type: "openai", APIBase: "https://api.openai.com/v1", APIKey: "stored-secret", Enabled: true},
			Adapter: &protocol.OpenAIAdapter{},
			Healthy: true,
		},
	}
	m.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer stored-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"object":"list","data":[]}`)), Header: make(http.Header)}, nil
	})}
	m.mu.Unlock()

	err := m.TestConnection(context.Background(), config.ProviderConfig{ID: "openai-main", Name: "OpenAI", Type: "openai", APIBase: "https://api.openai.com/v1"})
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
}

func TestAddProviderWithoutIDAllowsSetModels(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.mu.Lock()
	m.providers = map[string]*ProviderRuntime{}
	m.mu.Unlock()

	cfgProvider := &routeTestConfigProvider{}
	m.SetConfigProvider(cfgProvider)

	err := m.Add(config.ProviderConfig{Name: "OpenAI", Type: "openai", APIBase: "https://api.openai.com/v1", Enabled: true})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	providers := cfgProvider.GetProviders()
	if len(providers) != 1 {
		t.Fatalf("providers len = %d", len(providers))
	}
	if providers[0].ID == "" {
		t.Fatalf("expected generated provider id")
	}

	err = m.SetModels(providers[0].ID, []config.ModelEntry{{Model: "gpt-4o", Target: "gpt-4o"}}, "gpt-4o")
	if err != nil {
		t.Fatalf("SetModels() error = %v", err)
	}

	got := m.Get(providers[0].ID)
	if got == nil {
		t.Fatalf("expected provider runtime for %q", providers[0].ID)
	}
	if len(got.Config.LLMs) != 1 || got.Config.LLMs[0].Model != "gpt-4o" {
		t.Fatalf("unexpected llms: %+v", got.Config.LLMs)
	}
	if got.Config.DefaultModel != "gpt-4o" {
		t.Fatalf("DefaultModel = %q", got.Config.DefaultModel)
	}
}

func TestAddOpenAIResponsesProviderUsesResponsesAdapter(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.mu.Lock()
	m.providers = map[string]*ProviderRuntime{}
	m.mu.Unlock()

	cfgProvider := &routeTestConfigProvider{}
	m.SetConfigProvider(cfgProvider)

	err := m.Add(config.ProviderConfig{ID: "openai-responses", Name: "OpenAI Responses", Type: "openai", APIBase: "https://api.openai.com/v1", EndpointMode: config.ProviderEndpointModeResponses, Enabled: true})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got := m.Get("openai-responses")
	if got == nil {
		t.Fatalf("expected provider runtime")
	}
	if _, ok := got.Adapter.(*protocol.OpenAIResponsesAdapter); !ok {
		t.Fatalf("expected OpenAIResponsesAdapter, got %T", got.Adapter)
	}
}

func TestDoRequestWithRetryReturnsStructuredHTTPError(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Upstream request failed","type":"upstream_error"}}`)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})}
	m.SetConfigProvider(&routeTestConfigProvider{gateway: config.GatewayConfig{RetryCount: 0, RetryIntervalMs: 1}})

	resp, err := m.DoRequestWithRetry(context.Background(), &ProviderRuntime{Config: config.ProviderConfig{ID: "openai-main", Type: "openai", APIBase: "https://api.openai.com/v1"}, Adapter: &protocol.OpenAIAdapter{}}, &protocol.InternalRequest{Model: "gpt-4o-mini", Messages: []protocol.InternalMessage{{Role: "user", Content: []protocol.ContentBlock{{Type: "text", Text: "hello"}}}}})
	if resp != nil {
		t.Fatalf("expected nil response")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T (%v)", err, err)
	}
	if httpErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("StatusCode = %d", httpErr.StatusCode)
	}
}

func TestDoRequestUsesStreamClientForStreamingRequests(t *testing.T) {
	protocol.RegisterDefaults()
	m := GetManager()
	m.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("non-stream client should not be used for streaming requests")
		return nil, nil
	})}
	m.streamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"ok"}`)), Header: make(http.Header)}, nil
	})}

	resp, err := m.DoRequest(context.Background(), &ProviderRuntime{Config: config.ProviderConfig{ID: "openai-main", Type: "openai", APIBase: "https://api.openai.com/v1"}, Adapter: &protocol.OpenAIAdapter{}}, &protocol.InternalRequest{Model: "gpt-4o-mini", Stream: true, Messages: []protocol.InternalMessage{{Role: "user", Content: []protocol.ContentBlock{{Type: "text", Text: "hello"}}}}})
	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d", resp.StatusCode)
	}
}