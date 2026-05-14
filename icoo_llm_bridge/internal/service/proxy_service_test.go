package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"icoo_llm_bridge/internal/config"
	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/domain"
	"icoo_llm_bridge/internal/model/entity"
	"icoo_llm_bridge/internal/utils/ai_llm_proxy"
)

type allowAuth struct{}

func (allowAuth) Verify(context.Context, string, string) bool { return true }

type fixedRouteResolver struct {
	route domain.Route
}

func (r fixedRouteResolver) Resolve(context.Context, constants.Protocol, string) (domain.Route, error) {
	return r.route, nil
}

type memoryTraffic struct {
	items []entity.TrafficRecord
}

func (m *memoryTraffic) Record(_ context.Context, item entity.TrafficRecord) error {
	m.items = append(m.items, item)
	return nil
}

func (m *memoryTraffic) List(context.Context, int) ([]entity.TrafficRecord, error) {
	return m.items, nil
}

func (m *memoryTraffic) Clear(context.Context) error {
	m.items = nil
	return nil
}

func TestProxyServiceForwardsJSONAndRewritesModel(t *testing.T) {
	var upstreamModel string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer upstream-key" {
			t.Fatalf("Authorization header = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		upstreamModel, _ = payload["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","usage":{"total_tokens":3}}`))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "test",
		UpstreamProtocol: constants.ProtocolOpenAIResponses,
		Model:            "target-model",
		Provider: domain.ProviderSnapshot{
			Name:    "openai",
			BaseURL: upstream.URL,
			APIKey:  "upstream-key",
		},
	}
	proxy := NewProxyService(
		config.Config{
			AllowLocalWithoutAuth: false,
			Log: config.LogConfig{
				ChainLogBodies:       true,
				ChainLogMaxBodyBytes: 12,
			},
		},
		slog.Default(),
		ai_llm_proxy.NewPassthroughConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
	)

	requestBody := `{"model":"requested-model","input":"hi"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
	req.Header.Set("x-api-key", "proxy-key")
	req.Header.Set("User-Agent", "test-client/1.0")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIResponses)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if upstreamModel != "target-model" {
		t.Fatalf("upstream model = %q", upstreamModel)
	}
	if len(traffic.items) != 1 {
		t.Fatalf("traffic records = %d", len(traffic.items))
	}
	if traffic.items[0].Model != "target-model" || traffic.items[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected traffic record: %+v", traffic.items[0])
	}
	if traffic.items[0].Method != http.MethodPost || traffic.items[0].ClientIP == "" {
		t.Fatalf("traffic request metadata not recorded: %+v", traffic.items[0])
	}
	if traffic.items[0].RequestedModel != "requested-model" || traffic.items[0].RequestBodyBytes != int64(len(requestBody)) {
		t.Fatalf("traffic request model/body size not recorded: %+v", traffic.items[0])
	}
	if traffic.items[0].RequestBody != `{"model":"re` || !traffic.items[0].RequestBodyTruncated {
		t.Fatalf("traffic request body preview not recorded: %+v", traffic.items[0])
	}
}

func TestProxyServiceConvertsEventStream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt","status":"in_progress"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hello"}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt","status":"completed"}}`,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "test",
		UpstreamProtocol: constants.ProtocolOpenAIResponses,
		Model:            "target-model",
		Provider: domain.ProviderSnapshot{
			Name:    "openai",
			BaseURL: upstream.URL,
			APIKey:  "upstream-key",
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: false},
		slog.Default(),
		ai_llm_proxy.NewProtocolConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("content-type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("expected chat stream body, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceNormalizesSuccessfulResponsesStatusCode(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"id":"resp_1","object":"response","model":"gpt-5.5","status":"completed","output":[]}`))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "test",
		UpstreamProtocol: constants.ProtocolOpenAIResponses,
		Model:            "gpt-5.5",
		Provider: domain.ProviderSnapshot{
			Name:    "openai",
			BaseURL: upstream.URL,
			APIKey:  "upstream-key",
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: false},
		slog.Default(),
		ai_llm_proxy.NewProtocolConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"object":"chat.completion"`) {
		t.Fatalf("expected chat completion body, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"content":""`) {
		t.Fatalf("expected explicit empty content, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceRoutesOpenAIChatToAnthropicStream(t *testing.T) {
	var upstreamPath string
	var upstreamModel string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key header = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		upstreamModel, _ = payload["model"].(string)
		if _, ok := payload["messages"]; !ok {
			t.Fatalf("expected anthropic messages payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[],"usage":{"input_tokens":2,"output_tokens":0}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			``,
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
			``,
			`event: content_block_stop`,
			`data: {"type":"content_block_stop","index":0}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
			``,
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "anthropic",
		UpstreamProtocol: constants.ProtocolAnthropic,
		Model:            "claude-3-5-sonnet",
		Provider: domain.ProviderSnapshot{
			Name:    "anthropic",
			BaseURL: upstream.URL,
			APIKey:  "anthropic-key",
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: false},
		slog.Default(),
		ai_llm_proxy.NewProtocolConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if upstreamPath != "/v1/messages" {
		t.Fatalf("upstream path = %q", upstreamPath)
	}
	if upstreamModel != "claude-3-5-sonnet" {
		t.Fatalf("upstream model = %q", upstreamModel)
	}
	if !strings.Contains(rec.Body.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("expected chat stream body, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"content":"hello"`) {
		t.Fatalf("expected converted text, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].InputTokens != 2 || traffic.items[0].OutputTokens != 3 {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}
