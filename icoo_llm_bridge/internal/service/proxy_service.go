package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"icoo_llm_bridge/internal/config"
	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/domain"
	"icoo_llm_bridge/internal/model/entity"
	"icoo_llm_bridge/internal/utils/ai_llm_proxy"
)

type proxyService struct {
	cfg          config.Config
	logger       *slog.Logger
	converter    ai_llm_proxy.Converter
	auth         proxyAuth
	routes       RouteResolver
	traffic      TrafficService
	tracker      RequestTracker
	client       *http.Client
	streamClient *http.Client

	// trafficQueue decouples traffic recording from the proxy hot path. Records
	// are handed to a background writer so a slow DB write (or "database is
	// locked" backoff) never blocks the client response. When nil (tests that
	// build the struct directly) recording falls back to synchronous writes.
	trafficQueue    chan entity.TrafficRecord
	trafficDone     chan struct{}
	trafficInflight sync.WaitGroup // tracks records enqueued but not yet written
}

type proxyAuth interface {
	Verify(ctx context.Context, secret string, scope string) bool
}

func NewProxyService(
	cfg config.Config,
	logger *slog.Logger,
	converter ai_llm_proxy.Converter,
	auth proxyAuth,
	routes RouteResolver,
	traffic TrafficService,
	tracker RequestTracker,
) ProxyService {
	return &proxyService{
		cfg:          cfg,
		logger:       logger,
		converter:    converter,
		auth:         auth,
		routes:       routes,
		traffic:      traffic,
		tracker:      tracker,
		client:       &http.Client{Timeout: cfg.WriteTimeout},
		streamClient: &http.Client{},
	}
}

// trafficQueueCapacity bounds how many records can be buffered while a slow DB
// write is in flight. It is deliberately generous for normal load while still
// capping memory under burst traffic (overflow is dropped, never blocks).
const trafficQueueCapacity = 1024

// StartTrafficWriter spins up the background worker that persists traffic
// records off the proxy hot path. It is opt-in (called by the container) so
// tests that build a proxyService directly keep deterministic synchronous
// recording. Safe to call at most once per service.
func (s *proxyService) StartTrafficWriter() {
	if s.trafficQueue != nil {
		return
	}
	s.trafficQueue = make(chan entity.TrafficRecord, trafficQueueCapacity)
	s.trafficDone = make(chan struct{})
	go s.runTrafficWriter()
}

// runTrafficWriter drains the queue, writing each record with an independent
// timeout so a stuck write cannot stall the whole queue indefinitely.
func (s *proxyService) runTrafficWriter() {
	defer close(s.trafficDone)
	for item := range s.trafficQueue {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.traffic.Record(ctx, item); err != nil && s.logger != nil {
			s.logger.Warn("failed to record traffic", "error", err)
		}
		cancel()
		s.trafficInflight.Done()
	}
}

// Close stops the background writer after draining buffered records. It blocks
// until the queue is empty or the timeout elapses, so records in flight are not
// lost on graceful shutdown.
func (s *proxyService) Close() error {
	if s.trafficQueue == nil {
		return nil
	}
	close(s.trafficQueue)
	// Drain with a bounded wait so a pathological write cannot hang shutdown.
	select {
	case <-s.trafficDone:
	case <-time.After(10 * time.Second):
	}
	s.trafficQueue = nil
	s.trafficDone = nil
	return nil
}

// FlushTraffic blocks until all currently-queued traffic records have been
// written. It is a no-op when no writer is running. Primarily intended to make
// traffic assertions deterministic in tests.
func (s *proxyService) FlushTraffic() {
	s.trafficInflight.Wait()
}

func (s *proxyService) Handle(w http.ResponseWriter, r *http.Request, downstream constants.Protocol) {
	start := time.Now()
	requestID := newProxyRequestID()
	w.Header().Set("X-ICOO-Request-ID", requestID)

	if r.Method != http.MethodPost {
		s.writeProxyError(w, downstream, http.StatusMethodNotAllowed, "method not allowed")
		s.recordTraffic(r, requestID, downstream, domain.Route{}, http.StatusMethodNotAllowed, start, "method not allowed", domain.TokenUsage{}, "", nil)
		return
	}
	if !s.authorize(r) {
		s.writeProxyError(w, downstream, http.StatusUnauthorized, "invalid proxy api key")
		s.recordTraffic(r, requestID, downstream, domain.Route{}, http.StatusUnauthorized, start, "invalid proxy api key", domain.TokenUsage{}, "", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadRequest, "failed to read request body")
		s.recordTraffic(r, requestID, downstream, domain.Route{}, http.StatusBadRequest, start, "failed to read request body", domain.TokenUsage{}, "", nil)
		return
	}
	requestedModel, err := extractRequestModel(body)
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadRequest, err.Error())
		s.recordTraffic(r, requestID, downstream, domain.Route{}, http.StatusBadRequest, start, err.Error(), domain.TokenUsage{}, "", body)
		return
	}
	route, err := s.routes.Resolve(r.Context(), downstream, requestedModel)
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadRequest, err.Error())
		s.recordTraffic(r, requestID, downstream, domain.Route{}, http.StatusBadRequest, start, err.Error(), domain.TokenUsage{}, requestedModel, body)
		return
	}

	ruleID := extractRuleID(route.Source)
	if ruleID != "" && s.tracker != nil {
		s.tracker.Acquire(ruleID)
		defer s.tracker.Release(ruleID)
	}
	upstreamBody, err := s.converter.ConvertRequest(ai_llm_proxy.RequestInput{
		Downstream: downstream,
		Upstream:   route.UpstreamProtocol,
		Model:      route.Model,
		Body:       body,
	})
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadRequest, err.Error())
		s.recordTraffic(r, requestID, downstream, route, http.StatusBadRequest, start, err.Error(), domain.TokenUsage{}, requestedModel, body)
		return
	}

	upstreamWantsStream := requestWantsStream(upstreamBody)
	resp, err := s.sendUpstream(r, route, upstreamBody, upstreamWantsStream)
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadGateway, err.Error())
		s.recordTraffic(r, requestID, downstream, route, http.StatusBadGateway, start, err.Error(), domain.TokenUsage{}, requestedModel, body)
		return
	}
	defer resp.Body.Close()

	if !isHTTPSuccess(resp.StatusCode) {
		s.handleUpstreamErrorResponse(w, r, resp, requestID, downstream, route, start, requestedModel, body)
		return
	}

	if isEventStream(resp.Header) {
		s.handleStreamResponse(w, r, resp, requestID, downstream, route, start, requestedModel, body)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadGateway, "read upstream response failed")
		s.recordTraffic(r, requestID, downstream, route, http.StatusBadGateway, start, err.Error(), domain.TokenUsage{}, requestedModel, body)
		return
	}
	wantsStream, includeUsage := downstreamStreamPreference(downstream, body)
	statusCode := resp.StatusCode

	converted, err := s.converter.ConvertResponse(ai_llm_proxy.ResponseInput{
		Downstream: downstream,
		Upstream:   route.UpstreamProtocol,
		Model:      route.Model,
		Body:       respBody,
	})
	if err != nil {
		s.writeProxyError(w, downstream, http.StatusBadGateway, err.Error())
		s.recordTraffic(r, requestID, downstream, route, http.StatusBadGateway, start, err.Error(), domain.TokenUsage{}, requestedModel, body)
		return
	}

	usage := s.converter.ExtractUsage(route.UpstreamProtocol, respBody).Normalize()
	if wantsStream && downstream == constants.ProtocolOpenAIChat && statusCode < http.StatusBadRequest {
		prepareStreamHeaders(w.Header(), resp.Header)
		w.WriteHeader(statusCode)
		if err := writeChatCompletionAsStream(converted, includeUsage, flushWriter{writer: w}); err != nil {
			s.recordTraffic(r, requestID, downstream, route, http.StatusBadGateway, start, err.Error(), usage, requestedModel, body)
			if s.logger != nil {
				s.logger.Warn("non-stream chat fallback conversion failed", "request_id", requestID, "error", err)
			}
			return
		}
		s.recordTraffic(r, requestID, downstream, route, statusCode, start, "", usage, requestedModel, body)
		return
	}

	copySafeHeaders(w.Header(), resp.Header)
	w.WriteHeader(statusCode)
	_, _ = w.Write(converted)
	s.recordTraffic(r, requestID, downstream, route, statusCode, start, "", usage, requestedModel, body)
}

func (s *proxyService) authorize(r *http.Request) bool {
	if s.cfg.AllowLocalWithoutAuth && isLoopbackRemote(r.RemoteAddr) {
		return true
	}
	key := extractRequestAPIKey(r)
	return key != "" && s.auth != nil && s.auth.Verify(r.Context(), key, "proxy")
}

func (s *proxyService) sendUpstream(r *http.Request, route domain.Route, body []byte, streaming bool) (*http.Response, error) {
	url := joinUpstreamURL(route.Provider.BaseURL, route.UpstreamProtocol)
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("upstream base_url is required")
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	applyUpstreamHeaders(req, r, route)
	client := s.client
	if streaming {
		client = s.streamClient
	}
	if strings.TrimSpace(route.Provider.ProxyURL) != "" {
		var err error
		timeout := s.cfg.WriteTimeout
		if streaming {
			timeout = 0
		}
		client, err = newProxiedHTTPClient(timeout, route.Provider.ProxyURL)
		if err != nil {
			return nil, err
		}
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	return resp, nil
}

func (s *proxyService) handleUpstreamErrorResponse(
	w http.ResponseWriter,
	r *http.Request,
	resp *http.Response,
	requestID string,
	downstream constants.Protocol,
	route domain.Route,
	start time.Time,
	requestedModel string,
	requestBody []byte,
) {
	respBody, err := readLimitedBody(resp.Body, maxUpstreamErrorBodyBytes)
	statusCode := downstreamErrorStatus(resp.StatusCode)
	message := upstreamErrorMessage(resp.StatusCode, respBody)
	if err != nil {
		message = "read upstream error response failed"
	}
	s.writeProxyError(w, downstream, statusCode, message)
	s.recordTraffic(r, requestID, downstream, route, statusCode, start, message, domain.TokenUsage{}, requestedModel, requestBody)
}

func (s *proxyService) handleStreamResponse(
	w http.ResponseWriter,
	r *http.Request,
	resp *http.Response,
	requestID string,
	downstream constants.Protocol,
	route domain.Route,
	start time.Time,
	requestedModel string,
	requestBody []byte,
) {
	reader, err := preflightStream(resp.Body, s.streamPreflightTimeout())
	if err != nil {
		statusCode := http.StatusBadGateway
		message := err.Error()
		s.writeProxyError(w, downstream, statusCode, message)
		s.recordTraffic(r, requestID, downstream, route, statusCode, start, message, domain.TokenUsage{}, requestedModel, requestBody)
		if s.logger != nil {
			s.logger.Warn("stream preflight failed", "request_id", requestID, "error", err)
		}
		return
	}

	prepareStreamHeaders(w.Header(), resp.Header)
	statusCode := resp.StatusCode
	w.WriteHeader(statusCode)
	writer := flushWriter{writer: w}
	result, err := s.converter.ConvertStream(ai_llm_proxy.StreamInput{
		Downstream: downstream,
		Upstream:   route.UpstreamProtocol,
		Model:      route.Model,
		Reader:     reader,
		Writer:     writer,
	})
	if err != nil {
		s.recordTraffic(r, requestID, downstream, route, http.StatusBadGateway, start, err.Error(), result.Usage, requestedModel, requestBody)
		if s.logger != nil {
			s.logger.Warn("stream conversion failed", "request_id", requestID, "error", err)
		}
		return
	}
	s.recordTraffic(r, requestID, downstream, route, statusCode, start, "", result.Usage, requestedModel, requestBody)
}

func (s *proxyService) recordTraffic(
	r *http.Request,
	requestID string,
	downstream constants.Protocol,
	route domain.Route,
	statusCode int,
	start time.Time,
	message string,
	usage domain.TokenUsage,
	requestedModel string,
	requestBody []byte,
) {
	if s.traffic == nil {
		return
	}
	bodyPreview, bodyBytes, bodyTruncated := s.requestBodyPreview(requestBody, r.ContentLength)
	matchedRuleID := extractRuleID(route.Source)
	item := entity.TrafficRecord{
		ID:                   requestID,
		RequestID:            requestID,
		Endpoint:             r.URL.Path,
		Method:               r.Method,
		ClientIP:             clientIP(r.RemoteAddr),
		UserAgent:            r.UserAgent(),
		ContentType:          r.Header.Get("Content-Type"),
		DownstreamProtocol:   downstream.String(),
		UpstreamProtocol:     route.UpstreamProtocol.String(),
		RouteName:            route.Name,
		RouteSource:          route.Source,
		MatchedRuleID:        matchedRuleID,
		MatchedRuleName:      matchedRuleName(matchedRuleID, route.Name),
		RequestedModel:       requestedModel,
		Model:                route.Model,
		RequestBody:          bodyPreview,
		RequestBodyBytes:     bodyBytes,
		RequestBodyTruncated: bodyTruncated,
		StatusCode:           statusCode,
		DurationMS:           time.Since(start).Milliseconds(),
		InputTokens:          usage.InputTokens,
		OutputTokens:         usage.OutputTokens,
		TotalTokens:          usage.TotalTokens,
		Error:                message,
		CreatedAt:            time.Now(),
	}
	s.enqueueTraffic(item)
}

// enqueueTraffic hands a record to the background writer when available.
// Falling behind is treated as a degraded-but-safe condition: if the queue is
// full we drop the record (and log once) rather than block the proxy response,
// because stalling the client to persist a log row is the wrong tradeoff.
func (s *proxyService) enqueueTraffic(item entity.TrafficRecord) {
	if s.trafficQueue == nil {
		// No worker configured: record synchronously (tests / unstarted service).
		s.recordTrafficSync(item)
		return
	}
	select {
	case s.trafficQueue <- item:
		s.trafficInflight.Add(1)
	default:
		if s.logger != nil {
			s.logger.Warn("traffic queue full; dropping traffic record", "request_id", item.RequestID)
		}
	}
}

// recordTrafficSync is the synchronous fallback used when no background worker
// is running. It keeps the same independent-timeout context semantics the tests
// rely on (the request's own context cancellation must not abort recording).
func (s *proxyService) recordTrafficSync(item entity.TrafficRecord) {
	if s.traffic == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.traffic.Record(ctx, item); err != nil && s.logger != nil {
		s.logger.Warn("failed to record traffic", "error", err)
	}
}

func (s *proxyService) requestBodyPreview(body []byte, contentLength int64) (string, int64, bool) {
	bodyBytes := int64(len(body))
	if bodyBytes == 0 && contentLength > 0 {
		bodyBytes = contentLength
	}
	if !s.cfg.Log.ChainLogBodies || len(body) == 0 {
		return "", bodyBytes, false
	}
	limit := s.cfg.Log.ChainLogMaxBodyBytes
	if limit <= 0 {
		return "", bodyBytes, bodyBytes > 0
	}
	if len(body) > limit {
		return string(body[:limit]), bodyBytes, true
	}
	return string(body), bodyBytes, false
}

func (s *proxyService) writeProxyError(w http.ResponseWriter, protocol constants.Protocol, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if protocol == constants.ProtocolAnthropic {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": message,
			},
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"type":    "invalid_request_error",
			"message": message,
		},
	})
}

func extractRequestModel(body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("invalid json body")
	}
	model, _ := payload["model"].(string)
	return strings.TrimSpace(model), nil
}

func extractRequestAPIKey(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("x-api-key")); key != "" {
		return key
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func joinUpstreamURL(baseURL string, protocol constants.Protocol) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}
	endpoint := upstreamEndpoint(protocol)
	if strings.HasSuffix(base, endpoint) {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + strings.TrimPrefix(endpoint, "/v1")
	}
	return base + endpoint
}

func upstreamEndpoint(protocol constants.Protocol) string {
	switch protocol {
	case constants.ProtocolAnthropic:
		return "/v1/messages"
	case constants.ProtocolOpenAIChat:
		return "/v1/chat/completions"
	case constants.ProtocolOpenAIResponses:
		return "/v1/responses"
	default:
		return ""
	}
}

func applyUpstreamHeaders(req *http.Request, source *http.Request, route domain.Route) {
	req.Header.Set("Content-Type", "application/json")
	if accept := strings.TrimSpace(source.Header.Get("Accept")); accept != "" {
		req.Header.Set("Accept", accept)
	}
	switch route.UpstreamProtocol {
	case constants.ProtocolAnthropic:
		req.Header.Set("x-api-key", route.Provider.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case constants.ProtocolOpenAIChat, constants.ProtocolOpenAIResponses:
		req.Header.Set("Authorization", "Bearer "+route.Provider.APIKey)
	}
	if route.Provider.UserAgent != "" {
		req.Header.Set("User-Agent", route.Provider.UserAgent)
	}
}

func copySafeHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		switch strings.ToLower(key) {
		case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "trailers", "transfer-encoding", "upgrade", "content-encoding", "content-length", "content-range":
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	if dst.Get("Content-Type") == "" {
		dst.Set("Content-Type", "application/json; charset=utf-8")
	}
}

func prepareStreamHeaders(dst http.Header, src http.Header) {
	copySafeHeaders(dst, src)
	dst.Set("Content-Type", "text/event-stream")
	dst.Set("Cache-Control", "no-cache")
	dst.Del("Content-Length")
}

func isEventStream(header http.Header) bool {
	return strings.Contains(strings.ToLower(header.Get("Content-Type")), "text/event-stream")
}

func downstreamStreamPreference(protocol constants.Protocol, body []byte) (bool, bool) {
	switch protocol {
	case constants.ProtocolOpenAIChat:
		var req ai_llm_proxy.ChatCompletionsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return false, false
		}
		return req.Stream, req.StreamOptions != nil && req.StreamOptions.IncludeUsage
	default:
		return false, false
	}
}

func writeChatCompletionAsStream(body []byte, includeUsage bool, writer io.Writer) error {
	var resp ai_llm_proxy.ChatCompletionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("chat completion response has no choices")
	}

	choice := resp.Choices[0]
	roleChunk := ai_llm_proxy.ChatCompletionsChunk{
		ID:      resp.ID,
		Object:  "chat.completion.chunk",
		Created: resp.Created,
		Model:   resp.Model,
		Choices: []ai_llm_proxy.ChatChunkChoice{{
			Index: 0,
			Delta: ai_llm_proxy.ChatDelta{Role: "assistant"},
		}},
	}
	if text, ok, err := extractChatMessageContent(choice.Message.Content); err != nil {
		return err
	} else if ok {
		contentChunk := ai_llm_proxy.ChatCompletionsChunk{
			ID:      resp.ID,
			Object:  "chat.completion.chunk",
			Created: resp.Created,
			Model:   resp.Model,
			Choices: []ai_llm_proxy.ChatChunkChoice{{
				Index: 0,
				Delta: ai_llm_proxy.ChatDelta{Content: &text},
			}},
		}
		if err := writeChatChunk(writer, roleChunk); err != nil {
			return err
		}
		if err := writeChatChunk(writer, contentChunk); err != nil {
			return err
		}
	} else {
		if err := writeChatChunk(writer, roleChunk); err != nil {
			return err
		}
	}

	finishReason := choice.FinishReason
	if finishReason == "" {
		finishReason = "stop"
	}
	empty := ""
	finishChunk := ai_llm_proxy.ChatCompletionsChunk{
		ID:      resp.ID,
		Object:  "chat.completion.chunk",
		Created: resp.Created,
		Model:   resp.Model,
		Choices: []ai_llm_proxy.ChatChunkChoice{{
			Index:        0,
			Delta:        ai_llm_proxy.ChatDelta{Content: &empty},
			FinishReason: &finishReason,
		}},
	}
	if err := writeChatChunk(writer, finishChunk); err != nil {
		return err
	}

	if includeUsage && resp.Usage != nil {
		usageChunk := ai_llm_proxy.ChatCompletionsChunk{
			ID:      resp.ID,
			Object:  "chat.completion.chunk",
			Created: resp.Created,
			Model:   resp.Model,
			Choices: []ai_llm_proxy.ChatChunkChoice{},
			Usage:   resp.Usage,
		}
		if err := writeChatChunk(writer, usageChunk); err != nil {
			return err
		}
	}

	_, err := io.WriteString(writer, "data: [DONE]\n\n")
	return err
}

func writeChatChunk(writer io.Writer, chunk ai_llm_proxy.ChatCompletionsChunk) error {
	text, err := ai_llm_proxy.ChatChunkToSSE(chunk)
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, text)
	return err
}

func extractChatMessageContent(raw json.RawMessage) (string, bool, error) {
	if len(raw) == 0 {
		return "", false, nil
	}
	var content string
	if err := json.Unmarshal(raw, &content); err == nil {
		return content, true, nil
	}
	return "", false, nil
}

type flushWriter struct {
	writer http.ResponseWriter
}

func (w flushWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	if flusher, ok := w.writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, err
}

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

func newProxyRequestID() string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return "req-" + hex.EncodeToString(data[:])
}

func extractRuleID(source string) string {
	const prefix = "routing_rule:"
	if strings.HasPrefix(source, prefix) {
		return source[len(prefix):]
	}
	return ""
}

func matchedRuleName(ruleID string, routeName string) string {
	if ruleID == "" {
		return ""
	}
	return routeName
}

const (
	maxUpstreamErrorBodyBytes     = 1 << 20
	streamPreflightMaxBytes       = 64 << 10
	streamPreflightMaxEvents      = 3
	defaultStreamPreflightTimeout = 30 * time.Second
)

func isHTTPSuccess(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

func downstreamErrorStatus(upstreamStatus int) int {
	if upstreamStatus >= http.StatusBadRequest {
		return upstreamStatus
	}
	return http.StatusBadGateway
}

func readLimitedBody(reader io.Reader, limit int64) ([]byte, error) {
	if reader == nil {
		return nil, nil
	}
	return io.ReadAll(io.LimitReader(reader, limit+1))
}

func upstreamErrorMessage(statusCode int, body []byte) string {
	fallback := fmt.Sprintf("upstream returned status %d", statusCode)
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return fallback
	}
	if len(body) > maxUpstreamErrorBodyBytes {
		body = body[:maxUpstreamErrorBodyBytes]
	}
	if message := extractErrorMessage(body); message != "" {
		return fallback + ": " + message
	}
	return fallback
}

func extractErrorMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if errValue, ok := payload["error"]; ok {
		if message := errorValueMessage(errValue); message != "" {
			return message
		}
	}
	if message, _ := payload["message"].(string); strings.TrimSpace(message) != "" {
		return strings.TrimSpace(message)
	}
	return ""
}

func errorValueMessage(value any) string {
	switch item := value.(type) {
	case string:
		return strings.TrimSpace(item)
	case map[string]any:
		if message, _ := item["message"].(string); strings.TrimSpace(message) != "" {
			return strings.TrimSpace(message)
		}
		if code, _ := item["code"].(string); strings.TrimSpace(code) != "" {
			return strings.TrimSpace(code)
		}
		if typ, _ := item["type"].(string); strings.TrimSpace(typ) != "" {
			return strings.TrimSpace(typ)
		}
	}
	return ""
}

func requestWantsStream(body []byte) bool {
	var payload struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &payload) == nil && payload.Stream
}

type streamPreflightResult struct {
	reader io.Reader
	err    error
}

func (s *proxyService) streamPreflightTimeout() time.Duration {
	if s.cfg.StreamPreflightTimeout > 0 {
		return s.cfg.StreamPreflightTimeout
	}
	return defaultStreamPreflightTimeout
}

func preflightStream(body io.ReadCloser, timeout time.Duration) (io.Reader, error) {
	if timeout <= 0 {
		timeout = defaultStreamPreflightTimeout
	}
	done := make(chan streamPreflightResult, 1)
	go func() {
		reader, err := scanStreamPreflight(body)
		done <- streamPreflightResult{reader: reader, err: err}
	}()
	select {
	case result := <-done:
		return result.reader, result.err
	case <-time.After(timeout):
		_ = body.Close()
		return nil, fmt.Errorf("upstream stream preflight timed out")
	}
}

func scanStreamPreflight(reader io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(reader)
	var prefix bytes.Buffer
	var dataLines []string
	eventName := ""
	events := 0
	seenBytes := false

	evaluateFrame := func() (bool, error) {
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = nil
		name := eventName
		eventName = ""
		if data == "" || data == "[DONE]" {
			return false, nil
		}
		if err := detectStreamError(name, []byte(data)); err != nil {
			return true, err
		}
		return true, nil
	}

	for {
		line, err := bufReader.ReadString('\n')
		if len(line) > 0 {
			seenBytes = true
			_, _ = prefix.WriteString(line)
			trimmed := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			switch {
			case trimmed == "":
				events++
				found, frameErr := evaluateFrame()
				if frameErr != nil {
					return nil, frameErr
				}
				if found || prefix.Len() >= streamPreflightMaxBytes || events >= streamPreflightMaxEvents {
					return io.MultiReader(bytes.NewReader(prefix.Bytes()), bufReader), nil
				}
			case strings.HasPrefix(trimmed, ":"):
			case strings.HasPrefix(trimmed, "event:"):
				eventName = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
			case strings.HasPrefix(trimmed, "data:"):
				value := strings.TrimPrefix(trimmed, "data:")
				dataLines = append(dataLines, strings.TrimPrefix(value, " "))
			}
			if prefix.Len() >= streamPreflightMaxBytes {
				return io.MultiReader(bytes.NewReader(prefix.Bytes()), bufReader), nil
			}
		}
		if err == io.EOF {
			if len(dataLines) > 0 {
				found, frameErr := evaluateFrame()
				if frameErr != nil {
					return nil, frameErr
				}
				if found {
					return io.MultiReader(bytes.NewReader(prefix.Bytes()), bufReader), nil
				}
			}
			if !seenBytes {
				return nil, fmt.Errorf("upstream stream was empty")
			}
			return nil, fmt.Errorf("upstream stream ended before first event")
		}
		if err != nil {
			return nil, fmt.Errorf("read upstream stream preflight failed: %w", err)
		}
	}
}

func detectStreamError(eventName string, data []byte) error {
	var payload map[string]any
	_ = json.Unmarshal(data, &payload)
	eventType, _ := payload["type"].(string)
	if eventType == "" {
		eventType = eventName
	}
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if eventType == "error" || strings.HasSuffix(eventType, ".failed") || strings.Contains(strings.ToLower(eventName), "error") {
		if message := errorValueMessage(payload["error"]); message != "" {
			return fmt.Errorf("upstream stream error: %s", message)
		}
		if message, _ := payload["message"].(string); strings.TrimSpace(message) != "" {
			return fmt.Errorf("upstream stream error: %s", strings.TrimSpace(message))
		}
		return fmt.Errorf("upstream stream error")
	}
	if message := errorValueMessage(payload["error"]); message != "" {
		return fmt.Errorf("upstream stream error: %s", message)
	}
	return nil
}
