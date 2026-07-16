package service

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/domain"
	"github.com/issueye/icoo_proxy/common/ai_llm_proxy"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

type stubPluginRuntime struct {
	client *pluginipc.Client
	err    error
}

func (s *stubPluginRuntime) List() []PluginRuntimeInstance { return nil }
func (s *stubPluginRuntime) Start(ctx context.Context, id string) error {
	return nil
}
func (s *stubPluginRuntime) Stop(ctx context.Context, id string) error    { return nil }
func (s *stubPluginRuntime) Restart(ctx context.Context, id string) error { return nil }
func (s *stubPluginRuntime) Health(ctx context.Context, id string) (*pluginipc.HealthResult, error) {
	return &pluginipc.HealthResult{OK: true, Status: "healthy"}, nil
}
func (s *stubPluginRuntime) ListModels(ctx context.Context, id string) (*pluginipc.ModelsListResult, error) {
	return &pluginipc.ModelsListResult{Models: []pluginipc.ModelInfo{{ID: "mock-model"}}}, nil
}
func (s *stubPluginRuntime) Client(id string) (*pluginipc.Client, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}
func (s *stubPluginRuntime) Discover() []PluginDiscoverCandidate { return nil }
func (s *stubPluginRuntime) Register(ctx context.Context, id string, in PluginRegisterInput, autoStart bool) error {
	return nil
}
func (s *stubPluginRuntime) Unregister(ctx context.Context, id string) error { return nil }
func (s *stubPluginRuntime) SetEnabled(ctx context.Context, id string, enabled bool) error {
	return nil
}
func (s *stubPluginRuntime) InstallCandidate(ctx context.Context, id string, enabled bool) error {
	return nil
}

type pluginRouteResolver struct {
	route domain.Route
}

func (f pluginRouteResolver) Resolve(ctx context.Context, downstream constants.Protocol, requestedModel string) (domain.Route, error) {
	return f.route, nil
}

type pluginAllowAuth struct{}

func (pluginAllowAuth) Verify(ctx context.Context, secret string, scope string) bool { return true }

func TestProxyPluginCompletePath(t *testing.T) {
	c1, c2 := net.Pipe()
	t.Cleanup(func() {
		_ = c1.Close()
		_ = c2.Close()
	})
	srv := pluginipc.NewServer(c2, pluginipc.ServerOptions{
		HostToken: "tok",
		Handshake: pluginipc.HandshakeResult{
			PluginID:         "mock",
			PluginVersion:    "0.0.1",
			Capabilities:     []string{"proxy.complete"},
			SupportedIngress: []string{"anthropic"},
		},
	})
	srv.RegisterComplete(func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
		return &pluginipc.ProxyResponse{
			Status:  200,
			Headers: map[string]string{"content-type": "application/json"},
			Body:    []byte(`{"type":"message","content":[{"type":"text","text":"hi"}]}`),
		}, nil
	})
	cli := pluginipc.NewClient(c1, pluginipc.ClientOptions{})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "test"); err != nil {
		t.Fatal(err)
	}

	route := domain.Route{
		Name:             "plugin-route",
		UpstreamProtocol: constants.ProtocolAnthropic,
		Model:            "mock-model",
		Source:           "provider_model",
		Provider: domain.ProviderSnapshot{
			ID:       "p1",
			Name:     "Mock Plugin",
			Protocol: constants.ProtocolAnthropic,
			Vendor:   constants.VendorPlugin,
			PluginID: "mock",
			BaseURL:  "plugin://mock",
			Enabled:  true,
		},
	}

	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: true, MaxRequestBodyBytes: 1 << 20},
		nil,
		ai_llm_proxy.NewProtocolConverter(),
		pluginAllowAuth{},
		pluginRouteResolver{route: route},
		nil,
		nil,
		&stubPluginRuntime{client: cli},
	)

	body := `{"model":"mock-model","messages":[{"role":"user","content":"hi"}],"max_tokens":16}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	proxy.Handle(rr, req, constants.ProtocolAnthropic)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["type"] != "message" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestProxyPluginMissingRuntime(t *testing.T) {
	route := domain.Route{
		Name:  "plugin-route",
		Model: "m",
		Provider: domain.ProviderSnapshot{
			Vendor:   constants.VendorPlugin,
			PluginID: "missing",
			Enabled:  true,
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: true},
		nil,
		ai_llm_proxy.NewProtocolConverter(),
		pluginAllowAuth{},
		pluginRouteResolver{route: route},
		nil,
		nil,
	)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"m"}`))
	req.RemoteAddr = "127.0.0.1:1"
	rr := httptest.NewRecorder()
	proxy.Handle(rr, req, constants.ProtocolAnthropic)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestIsPluginProvider(t *testing.T) {
	if !isPluginProvider(domain.Route{Provider: domain.ProviderSnapshot{Vendor: constants.VendorPlugin}}) {
		t.Fatal("expected plugin")
	}
	if isPluginProvider(domain.Route{Provider: domain.ProviderSnapshot{Vendor: constants.VendorOpenAI}}) {
		t.Fatal("expected non-plugin")
	}
}

func TestResolveProviderPluginID(t *testing.T) {
	if got := ResolveProviderPluginID(constants.VendorPlugin, "g", ""); got != "g" {
		t.Fatal(got)
	}
	if got := ResolveProviderPluginID(constants.VendorPlugin, "", "plugin://x"); got != "x" {
		t.Fatal(got)
	}
	if got := ResolveProviderPluginID(constants.VendorOpenAI, "g", "plugin://x"); got != "" {
		t.Fatal(got)
	}
}
