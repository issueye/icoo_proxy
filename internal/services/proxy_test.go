package services

import (
	"net/http/httptest"
	"testing"
	"time"

	"icoo_proxy/internal/config"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
	"icoo_proxy/internal/services/translation"
)

func TestProxyRequestContextViewIncludesEndpoint(t *testing.T) {
	ctx := proxyRequestContext{
		requestID:      "req-test",
		start:          time.Now().Add(-25 * time.Millisecond),
		endpointPath:   "/v1/chat/completions",
		downstream:     consts.ProtocolOpenAIChat,
		requestedModel: "gpt-4.1-mini",
		route: models.Route{
			Upstream: consts.ProtocolOpenAIResponses,
			Model:    "gpt-4.1-mini",
		},
	}

	item := ctx.view(200, "", translation.TokenUsage{
		InputTokens:  12,
		OutputTokens: 8,
		TotalTokens:  20,
	})

	if item.Endpoint != "/v1/chat/completions" {
		t.Fatalf("expected endpoint recorded, got %#v", item)
	}
	if item.Downstream != consts.ProtocolOpenAIChat.ToString() {
		t.Fatalf("expected downstream protocol preserved, got %#v", item)
	}
	if item.Upstream != consts.ProtocolOpenAIResponses.ToString() {
		t.Fatalf("expected upstream protocol preserved, got %#v", item)
	}
	if item.TotalTokens != 20 {
		t.Fatalf("expected token usage preserved, got %#v", item)
	}
}

func TestProxyUsesRouteScopedSupplierForUpstreamRequest(t *testing.T) {
	service := New(config.Config{
		OpenAIRResponsesConfig: &config.OpenAIRResponsesConfig{
			BaseURL: "https://yybb.codes",
			APIKey:  "global-key",
		},
	}, nil)
	route := models.Route{
		Upstream: consts.ProtocolOpenAIResponses,
		Supplier: models.Snapshot{
			ID:         "supplier-1",
			Name:       "daw111.asia",
			Protocol:   consts.ProtocolOpenAIResponses,
			Vendor:     consts.VendorOpenAI,
			BaseURL:    "https://sub2api.daw111.asia/v1",
			APIKey:     "route-key",
			OnlyStream: true,
			UserAgent:  "route-agent",
			IsEnabled:  true,
		},
	}

	upstreamURL, err := service.upstreamURL(route)
	if err != nil {
		t.Fatalf("upstreamURL: %v", err)
	}
	if upstreamURL != "https://sub2api.daw111.asia/v1/responses" {
		t.Fatalf("upstreamURL = %q, want %q", upstreamURL, "https://sub2api.daw111.asia/v1/responses")
	}
	if !service.shouldForceUpstreamStream(route) {
		t.Fatal("expected route-scoped only_stream to be honored")
	}

	target := httptest.NewRequest("POST", "https://example.com", nil)
	source := httptest.NewRequest("POST", "http://localhost/v1/messages", nil)
	source.Header.Set("Accept", "application/json")
	source.Header.Set("OpenAI-Beta", "responses=v1")
	service.applyRequestHeaders(target, source, route)

	if got := target.Header.Get("Authorization"); got != "Bearer route-key" {
		t.Fatalf("authorization = %q, want %q", got, "Bearer route-key")
	}
	if got := target.Header.Get("User-Agent"); got != "route-agent" {
		t.Fatalf("user-agent = %q, want %q", got, "route-agent")
	}
}
