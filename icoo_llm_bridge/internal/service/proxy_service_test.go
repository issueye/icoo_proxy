package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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

type contextCheckingTraffic struct {
	seenErr error
}

func (m *contextCheckingTraffic) Record(ctx context.Context, item entity.TrafficRecord) error {
	m.seenErr = ctx.Err()
	return m.seenErr
}

func (m *contextCheckingTraffic) List(context.Context, int) ([]entity.TrafficRecord, error) {
	return nil, nil
}

func (m *contextCheckingTraffic) Clear(context.Context) error {
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
		Name:             "openai-responses default route",
		Source:           "routing_rule:rule-default",
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
		nil,
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
	if traffic.items[0].MatchedRuleID != "rule-default" || traffic.items[0].MatchedRuleName != "openai-responses default route" {
		t.Fatalf("traffic matched rule metadata not recorded: %+v", traffic.items[0])
	}
	if traffic.items[0].RouteSource != "routing_rule:rule-default" || traffic.items[0].RouteName != "openai-responses default route" {
		t.Fatalf("traffic route metadata not recorded: %+v", traffic.items[0])
	}
}

func TestProxyServiceUsesProviderProxyURL(t *testing.T) {
	var proxiedURL string
	var proxiedAuth string
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedURL = r.URL.String()
		proxiedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_proxy","usage":{"total_tokens":5}}`))
	}))
	defer proxyServer.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "proxied route",
		UpstreamProtocol: constants.ProtocolOpenAIResponses,
		Model:            "target-model",
		Provider: domain.ProviderSnapshot{
			Name:     "openai",
			BaseURL:  "http://upstream.example",
			ProxyURL: proxyServer.URL,
			APIKey:   "upstream-key",
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: false},
		slog.Default(),
		ai_llm_proxy.NewPassthroughConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"source-model"}`))
	req.Header.Set("Authorization", "Bearer local-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIResponses)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if proxiedURL != "http://upstream.example/v1/responses" {
		t.Fatalf("proxied URL = %q", proxiedURL)
	}
	if proxiedAuth != "Bearer upstream-key" {
		t.Fatalf("proxied Authorization = %q", proxiedAuth)
	}
}

func TestProxyServiceRecordsTrafficWithIndependentContext(t *testing.T) {
	traffic := &contextCheckingTraffic{}
	proxy := &proxyService{traffic: traffic}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	proxy.recordTraffic(req, "req-test", constants.ProtocolOpenAIResponses, domain.Route{}, http.StatusOK, time.Now(), "", domain.TokenUsage{}, "model", nil)

	if traffic.seenErr != nil {
		t.Fatalf("traffic record context error = %v", traffic.seenErr)
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
		nil,
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

func TestProxyServiceReturnsErrorForNon2xxStreamResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusServiceUnavailable)
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content-type = %q", got)
	}
	if strings.Contains(rec.Body.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("non-2xx stream was converted as success: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusServiceUnavailable || traffic.items[0].Error == "" {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceReturnsErrorForNon2xxJSONResponse(t *testing.T) {
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "upstream returned status 503") {
		t.Fatalf("expected upstream error body, got: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"object":"chat.completion"`) {
		t.Fatalf("non-2xx response was converted as success: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusServiceUnavailable || traffic.items[0].Error == "" {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceFallsBackToStreamForSuccessfulChatJSON(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","created":1778735097,"model":"gpt-5.5","choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "test",
		UpstreamProtocol: constants.ProtocolOpenAIChat,
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hi"}]}`))
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
	if !strings.Contains(rec.Body.String(), `"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}`) {
		t.Fatalf("expected usage chunk, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "data: [DONE]") {
		t.Fatalf("expected done marker, got: %s", rec.Body.String())
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
		nil,
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

func TestProxyServiceUpstreamNon2xxReturnsDownstreamErrorAndRecordsTraffic(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "chat",
		UpstreamProtocol: constants.ProtocolOpenAIChat,
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"error"`) || !strings.Contains(rec.Body.String(), "slow down") {
		t.Fatalf("expected downstream error body, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
	if !strings.Contains(traffic.items[0].Error, "slow down") {
		t.Fatalf("traffic error = %q", traffic.items[0].Error)
	}
}

func TestProxyServiceDropsUnsafeHeadersAfterResponseRewrite(t *testing.T) {
	body := []byte(`{"id":"resp_1","object":"response","model":"gpt","status":"completed","output":[]}`)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("X-Upstream-Test", "kept")
		_, _ = w.Write(body)
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "responses",
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q", got)
	}
	if got := rec.Header().Get("Content-Length"); got != "" {
		t.Fatalf("Content-Length = %q", got)
	}
	if got := rec.Header().Get("X-Upstream-Test"); got != "kept" {
		t.Fatalf("X-Upstream-Test = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"object":"chat.completion"`) {
		t.Fatalf("expected rewritten chat body, got: %s", rec.Body.String())
	}
}

func TestProxyServiceStreamingRequestDoesNotUseGlobalClientTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(50 * time.Millisecond)
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
		Name:             "responses",
		UpstreamProtocol: constants.ProtocolOpenAIResponses,
		Model:            "gpt-5.5",
		Provider: domain.ProviderSnapshot{
			Name:    "openai",
			BaseURL: upstream.URL,
			APIKey:  "upstream-key",
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: false, WriteTimeout: 10 * time.Millisecond},
		slog.Default(),
		ai_llm_proxy.NewProtocolConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"content":"hello"`) {
		t.Fatalf("expected delayed stream body, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceStreamPreflightUsesConfiguredTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(100 * time.Millisecond)
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
		Name:             "responses",
		UpstreamProtocol: constants.ProtocolOpenAIResponses,
		Model:            "gpt-5.5",
		Provider: domain.ProviderSnapshot{
			Name:    "openai",
			BaseURL: upstream.URL,
			APIKey:  "upstream-key",
		},
	}
	proxy := NewProxyService(
		config.Config{AllowLocalWithoutAuth: false, StreamPreflightTimeout: 500 * time.Millisecond},
		slog.Default(),
		ai_llm_proxy.NewProtocolConverter(),
		allowAuth{},
		fixedRouteResolver{route: route},
		traffic,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"content":"hello"`) {
		t.Fatalf("expected delayed stream body, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceStreamPreflightRejectsEmptyStream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "responses",
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content-type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "empty") {
		t.Fatalf("expected empty stream error, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}

func TestProxyServiceStreamPreflightRejectsErrorEvent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: error`,
			`data: {"type":"error","error":{"type":"invalid_request_error","message":"bad stream"}}`,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	traffic := &memoryTraffic{}
	route := domain.Route{
		Name:             "responses",
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
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"requested-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("x-api-key", "proxy-key")
	rec := httptest.NewRecorder()

	proxy.Handle(rec, req, constants.ProtocolOpenAIChat)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("error stream was converted as success: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "bad stream") {
		t.Fatalf("expected stream error message, got: %s", rec.Body.String())
	}
	if len(traffic.items) != 1 || traffic.items[0].StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected traffic records: %+v", traffic.items)
	}
}
