package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"icoo_proxy/internal/config"
	"icoo_proxy/internal/provider"
)

type gatewayTestConfigProvider struct {
	gateway config.GatewayConfig
	apiKeys []config.ApiKeyConfig
}

func (g *gatewayTestConfigProvider) GetProviders() []config.ProviderConfig              { return nil }
func (g *gatewayTestConfigProvider) GetAPIKeys() []config.ApiKeyConfig                  { return g.apiKeys }
func (g *gatewayTestConfigProvider) GetEndpoints() []config.EndpointConfig              { return nil }
func (g *gatewayTestConfigProvider) GetGatewayConfig() config.GatewayConfig             { return g.gateway }
func (g *gatewayTestConfigProvider) AddProvider(p config.ProviderConfig) error          { return nil }
func (g *gatewayTestConfigProvider) UpdateProvider(p config.ProviderConfig) error       { return nil }
func (g *gatewayTestConfigProvider) DeleteProvider(id string) error                     { return nil }
func (g *gatewayTestConfigProvider) AddAPIKey(k config.ApiKeyConfig) error              { return nil }
func (g *gatewayTestConfigProvider) UpdateAPIKey(k config.ApiKeyConfig) error           { return nil }
func (g *gatewayTestConfigProvider) DeleteAPIKey(id string) error                       { return nil }
func (g *gatewayTestConfigProvider) AddEndpoint(e config.EndpointConfig) error          { return nil }
func (g *gatewayTestConfigProvider) UpdateEndpoint(e config.EndpointConfig) error       { return nil }
func (g *gatewayTestConfigProvider) DeleteEndpoint(id string) error                     { return nil }
func (g *gatewayTestConfigProvider) SetGatewayConfig(cfg config.GatewayConfig) error {
	g.gateway = cfg
	return nil
}

func TestAuthMiddlewareRejectsMissingKey(t *testing.T) {
	provider.GetManager().SetConfigProvider(&gatewayTestConfigProvider{
		apiKeys: []config.ApiKeyConfig{{ID: "k1", Key: "secret-key", Enabled: true}},
	})

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareAcceptsBearerKey(t *testing.T) {
	provider.GetManager().SetConfigProvider(&gatewayTestConfigProvider{
		apiKeys: []config.ApiKeyConfig{{ID: "k1", Key: "secret-key", Enabled: true}},
	})

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestAuthMiddlewareStoresAPIKeyInRequestContext(t *testing.T) {
	provider.GetManager().SetConfigProvider(&gatewayTestConfigProvider{
		apiKeys: []config.ApiKeyConfig{{ID: "k1", Key: "secret-key", Enabled: true}},
	})

	var got string
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = requestAPIKey(r)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("x-api-key", "secret-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if got != "secret-key" {
		t.Fatalf("request api key = %q", got)
	}
}
