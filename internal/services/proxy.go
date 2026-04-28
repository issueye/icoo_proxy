package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"icoo_proxy/internal/api"
	"icoo_proxy/internal/config"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
	"icoo_proxy/internal/pkg/utils"
	"icoo_proxy/internal/services/translation"
)

// ProxyService 负责代理入口的鉴权、路由解析、协议转换、上游转发和请求记录。
type ProxyService struct {
	cfg      config.Config
	client   *http.Client
	logger   *slog.Logger
	recorder RequestRecorder
	mu       sync.RWMutex
	recent   []api.RequestView
	catalog  *CatalogService
}

// RequestRecorder 定义代理请求记录器，用于将请求概览写入持久化存储。
type RequestRecorder interface {
	RecordRequest(api.RequestView) error
}

// New 创建代理服务实例，并注入运行配置和模型路由目录。
func New(cfg config.Config, catalog *CatalogService) *ProxyService {
	return &ProxyService{
		cfg:     cfg,
		catalog: catalog,
		client:  &http.Client{},
	}
}

// SetChainLogger 设置链路日志记录器，用于记录请求和响应的关键阶段。
func (s *ProxyService) SetChainLogger(logger *slog.Logger) {
	s.logger = logger
}

// SetRequestRecorder 设置请求记录器，用于保存最近请求和流量历史。
func (s *ProxyService) SetRequestRecorder(recorder RequestRecorder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder = recorder
}

// proxyRequestContext 保存单次代理请求在各处理阶段共享的上下文数据。
type proxyRequestContext struct {
	requestID             string
	start                 time.Time
	downstream            consts.Protocol
	requestedModel        string
	body                  []byte
	downstreamWantsStream bool
	route                 models.Route
	routeSource           string
}

// newProxyRequestContext 初始化请求上下文，并生成本次请求的追踪 ID。
func newProxyRequestContext(downstream consts.Protocol) proxyRequestContext {
	return proxyRequestContext{
		requestID:  newRequestID(),
		start:      time.Now(),
		downstream: downstream,
	}
}

// view 将当前请求上下文转换为前端和流量记录使用的请求视图。
func (ctx *proxyRequestContext) view(statusCode int, message string) api.RequestView {
	item := api.RequestView{
		RequestID:  ctx.requestID,
		Downstream: ctx.downstream.ToString(),
		Model:      ctx.requestedModel,
		StatusCode: statusCode,
		DurationMS: time.Since(ctx.start).Milliseconds(),
		Error:      message,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	if ctx.route.Upstream != "" {
		item.Upstream = ctx.route.Upstream.ToString()
		item.Model = ctx.route.Model
	}
	return item
}

// Handle 是代理请求入口，按校验、读取、路由、转换、转发和响应处理顺序编排各阶段。
func (s *ProxyService) Handle(w http.ResponseWriter, r *http.Request, downstream consts.Protocol) {
	ctx := newProxyRequestContext(downstream)
	w.Header().Set("X-ICOO-Request-ID", ctx.requestID)

	if !s.validateDownstreamRequest(w, r, &ctx) {
		return
	}
	if !s.readDownstreamRequest(w, r, &ctx) {
		return
	}
	if !s.resolveRequestRoute(w, &ctx) {
		return
	}
	preparedBody, ok := s.prepareUpstreamRequestBody(w, &ctx)
	if !ok {
		return
	}
	resp, ok := s.sendUpstreamRequest(w, r, &ctx, preparedBody)
	if !ok {
		return
	}
	defer resp.Body.Close()

	s.handleUpstreamResponse(w, resp, &ctx)
}

// validateDownstreamRequest 校验下游请求方法和代理鉴权信息。
func (s *ProxyService) validateDownstreamRequest(w http.ResponseWriter, r *http.Request, ctx *proxyRequestContext) bool {
	if r.Method != http.MethodPost {
		s.logRejectedDownstreamRequest(r, ctx, "method not allowed")
		s.fail(w, ctx.downstream, ctx.view(http.StatusMethodNotAllowed, "method not allowed"))
		return false
	}
	if err := s.authorize(r); err != nil {
		s.logRejectedDownstreamRequest(r, ctx, err.Error())
		s.fail(w, ctx.downstream, ctx.view(http.StatusUnauthorized, err.Error()))
		return false
	}
	return true
}

// logRejectedDownstreamRequest 记录被拒绝的下游请求，便于排查鉴权或方法错误。
func (s *ProxyService) logRejectedDownstreamRequest(r *http.Request, ctx *proxyRequestContext, message string) {
	s.logChain("downstream.request.rejected",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"method", r.Method,
		"path", r.URL.Path,
		"headers", sanitizedHeaders(r.Header),
		"error", message,
	)
}

// readDownstreamRequest 读取下游请求体，并记录原始请求链路日志。
func (s *ProxyService) readDownstreamRequest(w http.ResponseWriter, r *http.Request, ctx *proxyRequestContext) bool {
	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadRequest, "failed to read request body"))
		return false
	}
	ctx.body = body
	s.logChain("downstream.request",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"headers", sanitizedHeaders(r.Header),
		"body", s.logBody(body),
	)
	return true
}

// resolveRequestRoute 提取请求模型，并根据下游协议和模型目录解析目标上游路由。
func (s *ProxyService) resolveRequestRoute(w http.ResponseWriter, ctx *proxyRequestContext) bool {
	requestModel, err := extractModel(ctx.body)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadRequest, err.Error()))
		return false
	}
	ctx.requestedModel = requestModel

	route, err := s.catalog.Resolve(ctx.downstream, requestModel)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadRequest, err.Error()))
		return false
	}
	ctx.route = route
	ctx.routeSource = route.Source
	s.logChain("route.resolved",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"requested_model", requestModel,
		"upstream", route.Upstream.ToString(),
		"target_model", route.Model,
		"route_name", route.Name,
		"route_source", ctx.routeSource,
	)
	return true
}

// prepareUpstreamRequestBody 按路由结果改写或转换请求体，并按上游配置强制开启流式请求。
func (s *ProxyService) prepareUpstreamRequestBody(w http.ResponseWriter, ctx *proxyRequestContext) ([]byte, bool) {
	preparedBody, err := s.prepareRequestBody(ctx.downstream, ctx.route, ctx.body)
	if err != nil {
		status := mapPrepareErrorStatus(err)
		s.fail(w, ctx.downstream, ctx.view(status, err.Error()))
		return nil, false
	}

	ctx.downstreamWantsStream = requestUsesStreaming(ctx.body)
	forcedUpstreamStream := false
	if s.shouldForceUpstreamStream(ctx.route.Upstream) && !requestUsesStreaming(preparedBody) {
		preparedBody, err = forceStreamRequest(preparedBody)
		if err != nil {
			s.fail(w, ctx.downstream, ctx.view(http.StatusBadRequest, err.Error()))
			return nil, false
		}
		forcedUpstreamStream = true
	}
	if ctx.route.Upstream == consts.ProtocolAnthropic {
		s.logAnthropicThinkingBlocks(ctx, preparedBody)
	}
	s.logChain("conversion.request",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"target_model", ctx.route.Model,
		"translated", ctx.route.Upstream != ctx.downstream,
		"forced_stream", forcedUpstreamStream,
		"input_body", s.logBody(ctx.body),
		"output_body", s.logBody(preparedBody),
	)
	return preparedBody, true
}

// sendUpstreamRequest 构建上游 HTTP 请求、应用协议请求头并发送到目标供应商。
func (s *ProxyService) sendUpstreamRequest(w http.ResponseWriter, r *http.Request, ctx *proxyRequestContext, preparedBody []byte) (*http.Response, bool) {
	upstreamURL, err := s.upstreamURL(ctx.route.Upstream)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, err.Error()))
		return nil, false
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, strings.NewReader(string(preparedBody)))
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, "failed to build upstream request"))
		return nil, false
	}

	s.applyRequestHeaders(req, r, ctx.route.Upstream)
	s.logChain("upstream.request",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"method", req.Method,
		"url", upstreamURL,
		"headers", sanitizedHeaders(req.Header),
		"body", s.logBody(preparedBody),
	)

	resp, err := s.client.Do(req)
	if err != nil {
		message := fmt.Sprintf("upstream request failed: %v", err)
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, message))
		return nil, false
	}
	return resp, true
}

// handleUpstreamResponse 分发上游响应处理流程，区分同协议、跨协议和事件流响应。
func (s *ProxyService) handleUpstreamResponse(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	copyResponseHeaders(w.Header(), resp.Header)
	w.Header().Set("X-ICOO-Request-ID", ctx.requestID)
	w.Header().Set("X-ICOO-Upstream-Protocol", ctx.route.Upstream.ToString())

	if ctx.route.Upstream == ctx.downstream {
		s.handleSameProtocolResponse(w, resp, ctx)
		return
	}
	if isEventStream(resp.Header) {
		s.handleCrossProtocolEventStream(w, resp, ctx)
		return
	}
	s.handleCrossProtocolJSON(w, resp, ctx)
}

// handleSameProtocolResponse 处理上下游协议相同的响应，直接透传或按需聚合流式结果。
func (s *ProxyService) handleSameProtocolResponse(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	var ok bool
	if isEventStream(resp.Header) {
		ok = s.handleSameProtocolEventStream(w, resp, ctx)
	} else {
		ok = s.handleSameProtocolJSON(w, resp, ctx)
	}
	if ok {
		s.logSuccessfulRequest(resp.StatusCode, ctx)
	}
}

// handleSameProtocolEventStream 处理同协议事件流响应；下游非流式时支持聚合 OpenAI Responses 流。
func (s *ProxyService) handleSameProtocolEventStream(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) bool {
	s.logEventStreamUpstreamResponse(resp, ctx)
	if ctx.downstreamWantsStream {
		w.WriteHeader(resp.StatusCode)
		s.logDownstreamResponse(w, resp.StatusCode, "<event-stream body not captured>", ctx)
		copyStream(w, resp.Body)
		return true
	}
	if ctx.route.Upstream != consts.ProtocolOpenAIResponses {
		s.fail(w, ctx.downstream, ctx.view(http.StatusNotImplemented, "stream-only upstream aggregation is not implemented for this protocol"))
		return false
	}

	aggregatedBody, err := translation.AggregateResponsesStreamToJSON(resp.Body)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, err.Error()))
		return false
	}
	s.logStreamAggregation(aggregatedBody, ctx)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(aggregatedBody)
	s.logDownstreamResponse(w, resp.StatusCode, s.logBody(aggregatedBody), ctx)
	return true
}

// handleSameProtocolJSON 处理同协议 JSON 响应，读取上游响应体后原样返回给下游。
func (s *ProxyService) handleSameProtocolJSON(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) bool {
	upstreamBody, ok := s.readUpstreamResponse(w, resp, ctx)
	if !ok {
		return false
	}
	s.logJSONUpstreamResponse(resp, upstreamBody, ctx)
	s.logDownstreamResponse(w, resp.StatusCode, s.logBody(upstreamBody), ctx)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(upstreamBody)
	return true
}

// handleCrossProtocolEventStream 处理跨协议事件流响应，目前支持 OpenAI Responses 到 Anthropic/OpenAI Chat 的流式转换或聚合转换。
func (s *ProxyService) handleCrossProtocolEventStream(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	s.logEventStreamUpstreamResponse(resp, ctx)
	switch {
	case ctx.downstream == consts.ProtocolAnthropic && ctx.route.Upstream == consts.ProtocolOpenAIResponses && ctx.downstreamWantsStream:
		s.translateStreamingResponsesToAnthropic(w, resp, ctx)
	case ctx.downstream == consts.ProtocolOpenAIChat && ctx.route.Upstream == consts.ProtocolOpenAIResponses && ctx.downstreamWantsStream:
		s.translateStreamingResponsesToChat(w, resp, ctx)
	case !ctx.downstreamWantsStream && ctx.route.Upstream == consts.ProtocolOpenAIResponses:
		s.aggregateAndTranslateResponsesStream(w, resp, ctx)
	default:
		s.fail(w, ctx.downstream, ctx.view(http.StatusNotImplemented, "streaming cross protocol translation is not implemented yet"))
	}
}

// translateStreamingResponsesToAnthropic 将 OpenAI Responses 事件流转换为 Anthropic 事件流。
func (s *ProxyService) translateStreamingResponsesToAnthropic(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Del("Content-Length")
	w.WriteHeader(resp.StatusCode)
	err := translation.TranslateResponsesStreamToAnthropic(w, resp.Body, ctx.route.Model, ctx.requestID, s.logger)
	s.logDownstreamResponse(w, resp.StatusCode, "<translated event-stream body not captured>", ctx)
	item := ctx.view(resp.StatusCode, "")
	if err != nil {
		item.Error = err.Error()
		s.logChain("conversion.stream.error",
			"request_id", ctx.requestID,
			"downstream", ctx.downstream.ToString(),
			"upstream", ctx.route.Upstream.ToString(),
			"error", err.Error(),
		)
	}
	s.logRequest(item)
}

// translateStreamingResponsesToChat 将 OpenAI Responses 事件流转换为 OpenAI Chat Completions 事件流。
func (s *ProxyService) translateStreamingResponsesToChat(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Del("Content-Length")
	w.WriteHeader(resp.StatusCode)
	err := translation.TranslateResponsesStreamToChat(w, resp.Body, ctx.route.Model, ctx.requestID, s.logger)
	s.logDownstreamResponse(w, resp.StatusCode, "<translated event-stream body not captured>", ctx)
	item := ctx.view(resp.StatusCode, "")
	if err != nil {
		item.Error = err.Error()
		s.logChain("conversion.stream.error",
			"request_id", ctx.requestID,
			"downstream", ctx.downstream.ToString(),
			"upstream", ctx.route.Upstream.ToString(),
			"error", err.Error(),
		)
	}
	s.logRequest(item)
}

// aggregateAndTranslateResponsesStream 将 OpenAI Responses 流式响应聚合为 JSON，再转换为下游协议响应。
func (s *ProxyService) aggregateAndTranslateResponsesStream(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	aggregatedBody, err := translation.AggregateResponsesStreamToJSON(resp.Body)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, err.Error()))
		return
	}
	s.logStreamAggregation(aggregatedBody, ctx)
	translated, err := translation.ConvertResponse(ctx.downstream, ctx.route.Upstream, ctx.route.Model, aggregatedBody)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, err.Error()))
		return
	}
	s.logResponseConversion(aggregatedBody, translated, ctx)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(translated)
	s.logDownstreamResponse(w, resp.StatusCode, s.logBody(translated), ctx)
	s.logSuccessfulRequest(resp.StatusCode, ctx)
}

// handleCrossProtocolJSON 处理跨协议 JSON 响应；上游错误直接透传，成功响应执行协议转换。
func (s *ProxyService) handleCrossProtocolJSON(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) {
	upstreamBody, ok := s.readUpstreamResponse(w, resp, ctx)
	if !ok {
		return
	}
	s.logJSONUpstreamResponse(resp, upstreamBody, ctx)
	if resp.StatusCode >= 400 {
		s.writeUpstreamError(w, resp, upstreamBody, ctx)
		return
	}

	translated, err := translation.ConvertResponse(ctx.downstream, ctx.route.Upstream, ctx.route.Model, upstreamBody)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, err.Error()))
		return
	}
	s.logResponseConversion(upstreamBody, translated, ctx)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(translated)
	s.logDownstreamResponse(w, resp.StatusCode, s.logBody(translated), ctx)
	s.logSuccessfulRequest(resp.StatusCode, ctx)
}

// readUpstreamResponse 读取上游响应体，读取失败时写入网关错误。
func (s *ProxyService) readUpstreamResponse(w http.ResponseWriter, resp *http.Response, ctx *proxyRequestContext) ([]byte, bool) {
	upstreamBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.fail(w, ctx.downstream, ctx.view(http.StatusBadGateway, "failed to read upstream response body"))
		return nil, false
	}
	return upstreamBody, true
}

// writeUpstreamError 将上游错误响应体按原状态码透传给下游，并记录请求失败原因。
func (s *ProxyService) writeUpstreamError(w http.ResponseWriter, resp *http.Response, body []byte, ctx *proxyRequestContext) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
	s.logDownstreamResponse(w, resp.StatusCode, s.logBody(body), ctx)
	s.logRequest(ctx.view(resp.StatusCode, "upstream returned error"))
}

// logEventStreamUpstreamResponse 记录上游事件流响应元信息，避免捕获大体积流式内容。
func (s *ProxyService) logEventStreamUpstreamResponse(resp *http.Response, ctx *proxyRequestContext) {
	s.logChain("upstream.response",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"status_code", resp.StatusCode,
		"headers", sanitizedHeaders(resp.Header),
		"body", "<event-stream body not captured>",
	)
}

// logJSONUpstreamResponse 记录上游 JSON 响应头、状态码和脱敏后的响应体。
func (s *ProxyService) logJSONUpstreamResponse(resp *http.Response, body []byte, ctx *proxyRequestContext) {
	s.logChain("upstream.response",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"status_code", resp.StatusCode,
		"headers", sanitizedHeaders(resp.Header),
		"body", s.logBody(body),
	)
}

// logDownstreamResponse 记录返回给下游客户端的响应信息。
func (s *ProxyService) logDownstreamResponse(w http.ResponseWriter, statusCode int, body string, ctx *proxyRequestContext) {
	s.logChain("downstream.response",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"status_code", statusCode,
		"headers", sanitizedHeaders(w.Header()),
		"body", body,
	)
}

// logStreamAggregation 记录流式响应聚合为 JSON 后的转换结果。
func (s *ProxyService) logStreamAggregation(body []byte, ctx *proxyRequestContext) {
	s.logChain("conversion.stream.aggregate",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"target_model", ctx.route.Model,
		"output_body", s.logBody(body),
	)
}

// logResponseConversion 记录跨协议响应转换前后的内容，响应体会按配置脱敏和截断。
func (s *ProxyService) logResponseConversion(inputBody, outputBody []byte, ctx *proxyRequestContext) {
	s.logChain("conversion.response",
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"target_model", ctx.route.Model,
		"translated", ctx.route.Upstream != ctx.downstream,
		"input_body", s.logBody(inputBody),
		"output_body", s.logBody(outputBody),
	)
}

// logSuccessfulRequest 记录一次成功完成的代理请求。
func (s *ProxyService) logSuccessfulRequest(statusCode int, ctx *proxyRequestContext) {
	s.logRequest(ctx.view(statusCode, ""))
}

// shouldForceUpstreamStream 判断指定上游协议是否要求强制使用流式请求。
func (s *ProxyService) shouldForceUpstreamStream(protocol consts.Protocol) bool {
	switch protocol {
	case consts.ProtocolOpenAIResponses:
		return s.cfg.OpenAIRResponsesConfig.OnlyStream
	default:
		return false
	}
}

// authorize 校验代理访问密钥，支持 x-api-key 和 Authorization Bearer 两种方式。
func (s *ProxyService) authorize(r *http.Request) error {
	expected := s.cfg.AuthKeys()
	if len(expected) == 0 && s.cfg.AllowUnauthenticatedLocal {
		if isLocalRequest(r) {
			return nil
		}
		return fmt.Errorf("proxy api key is required")
	}
	if len(expected) == 0 {
		return fmt.Errorf("proxy api key is required")
	}
	if slices.Contains(expected, strings.TrimSpace(r.Header.Get("x-api-key"))) {
		return nil
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") && slices.Contains(expected, strings.TrimSpace(auth[7:])) {
		return nil
	}
	return fmt.Errorf("invalid proxy api key")
}

// isLocalRequest 判断请求来源是否为本机回环地址。
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// upstreamURL 获取指定协议的上游URL
func (s *ProxyService) upstreamURL(protocol consts.Protocol) (string, error) {
	switch protocol {
	case consts.ProtocolAnthropic:
		if strings.TrimSpace(s.cfg.AnthropicConfig.APIKey) == "" {
			return "", fmt.Errorf("anthropic upstream is not configured")
		}
		return strings.TrimRight(s.cfg.AnthropicConfig.BaseURL, "/") + "/v1/messages", nil
	case consts.ProtocolOpenAIChat:
		if strings.TrimSpace(s.cfg.OpenAIChatConfig.APIKey) == "" {
			return "", fmt.Errorf("openai chat upstream is not configured")
		}
		return strings.TrimRight(s.cfg.OpenAIChatConfig.BaseURL, "/") + "/v1/chat/completions", nil
	case consts.ProtocolOpenAIResponses:
		if strings.TrimSpace(s.cfg.OpenAIRResponsesConfig.APIKey) == "" {
			return "", fmt.Errorf("openai responses upstream is not configured")
		}
		return strings.TrimRight(s.cfg.OpenAIRResponsesConfig.BaseURL, "/") + "/v1/responses", nil
	default:
		return "", fmt.Errorf("unsupported upstream protocol %q", protocol)
	}
}

// applyRequestHeaders 应用请求头到目标请求
func (s *ProxyService) applyRequestHeaders(target *http.Request, source *http.Request, protocol consts.Protocol) {
	target.Header.Set("Content-Type", "application/json")
	if accept := strings.TrimSpace(source.Header.Get("Accept")); accept != "" {
		target.Header.Set("Accept", accept)
	}
	switch protocol {
	case consts.ProtocolAnthropic:
		target.Header.Set("x-api-key", s.cfg.AnthropicConfig.APIKey)
		target.Header.Set("anthropic-version", s.cfg.AnthropicConfig.Version)
		if userAgent := strings.TrimSpace(s.cfg.AnthropicConfig.UserAgent); userAgent != "" {
			target.Header.Set("User-Agent", userAgent)
		}
		if beta := strings.TrimSpace(source.Header.Get("anthropic-beta")); beta != "" {
			beta = sanitizeAnthropicBetaForVendor(beta, s.cfg.AnthropicConfig.Vendor)
			if beta != "" {
				target.Header.Set("anthropic-beta", beta)
			}
		}
	case consts.ProtocolOpenAIChat:
		target.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIChatConfig.APIKey)
		if userAgent := strings.TrimSpace(s.cfg.OpenAIChatConfig.UserAgent); userAgent != "" {
			target.Header.Set("User-Agent", userAgent)
		}
		if value := strings.TrimSpace(source.Header.Get("OpenAI-Beta")); value != "" {
			target.Header.Set("OpenAI-Beta", value)
		}
	case consts.ProtocolOpenAIResponses:
		target.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIRResponsesConfig.APIKey)
		if userAgent := strings.TrimSpace(s.cfg.OpenAIRResponsesConfig.UserAgent); userAgent != "" {
			target.Header.Set("User-Agent", userAgent)
		}
		if value := strings.TrimSpace(source.Header.Get("OpenAI-Beta")); value != "" {
			target.Header.Set("OpenAI-Beta", value)
		}
	}
}

func sanitizeAnthropicBetaForVendor(raw string, vendor consts.Vendor) string {
	if vendor != consts.VendorDeepSeek {
		return strings.TrimSpace(raw)
	}
	blocked := map[string]struct{}{
		"claude-code-20250219":            {},
		"interleaved-thinking-2025-05-14": {},
		"prompt-caching-scope-2026-01-05": {},
	}
	parts := strings.Split(raw, ",")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if _, found := blocked[strings.ToLower(value)]; found {
			continue
		}
		kept = append(kept, value)
	}
	return strings.Join(kept, ",")
}

// extractModel 从请求 JSON 中提取 model 字段。
func extractModel(body []byte) (string, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("invalid json body")
	}
	model, _ := payload["model"].(string)
	return strings.TrimSpace(model), nil
}

// requestUsesStreaming 判断请求体是否显式声明需要流式响应。
func requestUsesStreaming(body []byte) bool {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	stream, _ := payload["stream"].(bool)
	return stream
}

type anthropicThinkingLogSummary struct {
	TotalMessages             int      `json:"total_messages"`
	AssistantMessages         int      `json:"assistant_messages"`
	ThinkingBlocks            int      `json:"thinking_blocks"`
	IncompleteThinkingBlocks  int      `json:"incomplete_thinking_blocks"`
	ThinkingEnabled           bool     `json:"thinking_enabled"`
	HasPotentialThinkingIssue bool     `json:"has_potential_thinking_issue"`
	Issues                    []string `json:"issues,omitempty"`
}

// logAnthropicThinkingBlocks 检查发送到 Anthropic 上游的消息体中是否存在结构异常的 thinking block。
func (s *ProxyService) logAnthropicThinkingBlocks(ctx *proxyRequestContext, body []byte) {
	if s == nil || ctx == nil {
		return
	}
	summary, ok := inspectAnthropicThinkingBlocks(body)
	if !ok {
		s.logChain("anthropic.request.thinking.inspect_failed",
			"request_id", ctx.requestID,
			"downstream", ctx.downstream.ToString(),
			"upstream", ctx.route.Upstream.ToString(),
		)
		return
	}
	attrs := []interface{}{
		"request_id", ctx.requestID,
		"downstream", ctx.downstream.ToString(),
		"upstream", ctx.route.Upstream.ToString(),
		"thinking_enabled", summary.ThinkingEnabled,
		"total_messages", summary.TotalMessages,
		"assistant_messages", summary.AssistantMessages,
		"thinking_blocks", summary.ThinkingBlocks,
		"incomplete_thinking_blocks", summary.IncompleteThinkingBlocks,
		"has_potential_thinking_issue", summary.HasPotentialThinkingIssue,
	}
	if len(summary.Issues) > 0 {
		attrs = append(attrs, "issues", summary.Issues)
	}
	event := "anthropic.request.thinking.summary"
	if summary.HasPotentialThinkingIssue {
		event = "anthropic.request.thinking.warning"
	}
	s.logChain(event, attrs...)
}

func inspectAnthropicThinkingBlocks(body []byte) (anthropicThinkingLogSummary, bool) {
	summary := anthropicThinkingLogSummary{}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return summary, false
	}
	summary.ThinkingEnabled = anthropicThinkingEnabled(payload)
	messages, _ := payload["messages"].([]interface{})
	summary.TotalMessages = len(messages)
	for msgIndex, rawMessage := range messages {
		message, ok := rawMessage.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := message["role"].(string)
		if role != "assistant" {
			continue
		}
		summary.AssistantMessages++
		content, ok := message["content"].([]interface{})
		if !ok {
			continue
		}
		for blockIndex, rawBlock := range content {
			block, ok := rawBlock.(map[string]interface{})
			if !ok {
				continue
			}
			if strings.TrimSpace(translation.StringValue(block["type"], "")) != "thinking" {
				continue
			}
			summary.ThinkingBlocks++
			if !anthropicThinkingBlockComplete(block) {
				summary.IncompleteThinkingBlocks++
				summary.HasPotentialThinkingIssue = true
				summary.Issues = append(summary.Issues, fmt.Sprintf("messages[%d].content[%d] has incomplete thinking block", msgIndex, blockIndex))
			}
		}
	}
	return summary, true
}

func anthropicThinkingEnabled(payload map[string]interface{}) bool {
	if payload == nil {
		return false
	}
	raw, ok := payload["thinking"]
	if !ok || raw == nil {
		return false
	}
	thinking, ok := raw.(map[string]interface{})
	if !ok {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(translation.StringValue(thinking["type"], "")), "disabled") {
		return false
	}
	return true
}

func anthropicThinkingBlockComplete(block map[string]interface{}) bool {
	if block == nil {
		return false
	}
	_, exists := block["thinking"]
	if !exists {
		return false
	}
	_, ok := block["thinking"].(string)
	return ok
}

// forceStreamRequest 将请求体改写为 stream=true，用于只支持流式的上游。
func forceStreamRequest(body []byte) ([]byte, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid json body")
	}
	payload["stream"] = true
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite request body")
	}
	return rewritten, nil
}

// prepareRequestBody 根据上下游协议关系准备上游请求体：同协议改写模型，跨协议执行请求转换。
func (s *ProxyService) prepareRequestBody(downstream consts.Protocol, route models.Route, body []byte) ([]byte, error) {
	return translation.ConvertRequest(downstream, route, body)
}

// writeProtocolError 按下游协议格式写出统一错误响应。
func writeProtocolError(w http.ResponseWriter, protocol consts.Protocol, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	switch protocol {
	case consts.ProtocolAnthropic:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": message,
			},
		})
	default:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": message,
			},
		})
	}
}

// fail 统一处理代理失败场景：记录链路、保存请求记录并写出协议错误响应。
func (s *ProxyService) fail(w http.ResponseWriter, protocol consts.Protocol, item api.RequestView) {
	s.logChain("downstream.response.error",
		"request_id", item.RequestID,
		"downstream", item.Downstream,
		"upstream", item.Upstream,
		"model", item.Model,
		"status_code", item.StatusCode,
		"duration_ms", item.DurationMS,
		"error", item.Error,
	)
	s.recordRequest(item)
	writeProtocolError(w, protocol, item.StatusCode, item.Error)
}

// mapPrepareErrorStatus 将请求准备阶段的错误映射为合适的 HTTP 状态码。
func mapPrepareErrorStatus(err error) int {
	if strings.Contains(strings.ToLower(err.Error()), "not implemented") {
		return http.StatusNotImplemented
	}
	return http.StatusBadRequest
}

// copyResponseHeaders 复制上游响应头，同时过滤不应转发的逐跳头和 Content-Length。
func copyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		switch strings.ToLower(key) {
		case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailers", "transfer-encoding", "upgrade", "content-length":
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// isEventStream 判断响应头是否表示 Server-Sent Events 流。
func isEventStream(header http.Header) bool {
	return strings.Contains(strings.ToLower(header.Get("Content-Type")), "text/event-stream")
}

// copyStream 将上游流式响应持续转发给下游，并在支持时主动刷新。
func copyStream(w http.ResponseWriter, body io.Reader) {
	flusher, _ := w.(http.Flusher)
	buffer := make([]byte, 4096)
	for {
		n, err := body.Read(buffer)
		if n > 0 {
			_, _ = w.Write(buffer[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("icoo_proxy stream relay error: %v", err)
			}
			return
		}
	}
}

// logChain 写入结构化链路日志；未配置日志器时直接忽略。
func (s *ProxyService) logChain(event string, attrs ...interface{}) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Info(event, attrs...)
}

// logBody 根据链路日志配置返回响应体日志内容，并执行敏感字段脱敏和长度截断。
func (s *ProxyService) logBody(body []byte) string {
	if s == nil || !s.cfg.ChainLogBodies {
		return "<body logging disabled>"
	}
	if body == nil {
		return ""
	}
	result := utils.RedactJSONBody(body)
	if max := s.cfg.ChainLogMaxBodyBytes; max > 0 && len([]byte(result)) > max {
		return string([]byte(result)[:max]) + "...<truncated>"
	}
	return result
}

// sanitizedHeaders 复制请求或响应头，并对敏感头字段进行脱敏。
func sanitizedHeaders(headers http.Header) map[string][]string {
	result := make(map[string][]string, len(headers))
	for key, values := range headers {
		if utils.IsSensitiveName(key) {
			result[key] = []string{"<redacted>"}
			continue
		}
		result[key] = slices.Clone(values)
	}
	return result
}

// newRequestID 生成短请求 ID，用于链路日志、响应头和请求记录关联。
func newRequestID() string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return "req-" + hex.EncodeToString(data[:])
}

// RecentRequests 返回内存中最近的代理请求记录快照。
func (s *ProxyService) RecentRequests() []api.RequestView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.recent)
}

// logRequest 写入请求完成日志，并保存到最近请求和可选的持久化记录器。
func (s *ProxyService) logRequest(item api.RequestView) {
	s.recordRequest(item)
	log.Printf("icoo_proxy request_id=%s downstream=%s upstream=%s model=%s status=%d duration_ms=%d", item.RequestID, item.Downstream, item.Upstream, item.Model, item.StatusCode, item.DurationMS)
	s.logChain("request.completed",
		"request_id", item.RequestID,
		"downstream", item.Downstream,
		"upstream", item.Upstream,
		"model", item.Model,
		"status_code", item.StatusCode,
		"duration_ms", item.DurationMS,
		"error", item.Error,
	)
}

// recordRequest 更新内存最近请求列表，并在配置了记录器时写入持久化流量记录。
func (s *ProxyService) recordRequest(item api.RequestView) {
	s.mu.Lock()
	s.recent = append([]api.RequestView{item}, s.recent...)
	if len(s.recent) > 12 {
		s.recent = s.recent[:12]
	}
	recorder := s.recorder
	s.mu.Unlock()

	if recorder != nil {
		if err := recorder.RecordRequest(item); err != nil {
			log.Printf("icoo_proxy traffic record error: %v", err)
		}
	}
}
