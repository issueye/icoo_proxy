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
}

func (g *gatewayTestConfigProvider) GetProviders() []config.ProviderConfig        { return nil }
func (g *gatewayTestConfigProvider) GetGatewayConfig() config.GatewayConfig       { return g.gateway }
func (g *gatewayTestConfigProvider) GetRouteRules() []config.RouteRuleConfig      { return nil }
func (g *gatewayTestConfigProvider) AddProvider(p config.ProviderConfig) error    { return nil }
func (g *gatewayTestConfigProvider) UpdateProvider(p config.ProviderConfig) error { return nil }
func (g *gatewayTestConfigProvider) DeleteProvider(id string) error               { return nil }
func (g *gatewayTestConfigProvider) SetGatewayConfig(cfg config.GatewayConfig) error {
	g.gateway = cfg
	return nil
}
func (g *gatewayTestConfigProvider) SetRouteRules(rules []config.RouteRuleConfig) error { return nil }

func TestAuthMiddlewareRejectsMissingKey(t *testing.T) {
	provider.GetManager().SetConfigProvider(&gatewayTestConfigProvider{
		gateway: config.GatewayConfig{AuthKey: "secret-key"},
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
		gateway: config.GatewayConfig{AuthKey: "secret-key"},
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
