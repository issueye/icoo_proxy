package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/domain"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

// handlePluginRoute proxies a downstream request through a process plugin.
// It must be invoked BEFORE ConvertRequest. Raw body is forwarded; plugins
// own protocol adaptation for their upstream.
func (s *proxyService) handlePluginRoute(
	w http.ResponseWriter,
	r *http.Request,
	requestID string,
	downstream constants.Protocol,
	route domain.Route,
	start time.Time,
	requestedModel string,
	body []byte,
) {
	pluginID := ResolveProviderPluginID(route.Provider.Vendor, route.Provider.PluginID, route.Provider.BaseURL)
	if pluginID == "" {
		msg := "plugin provider missing plugin_id"
		s.writeProxyError(w, downstream, http.StatusBadRequest, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, http.StatusBadRequest, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}
	if s.plugins == nil {
		msg := "plugin runtime is not configured"
		s.writeProxyError(w, downstream, http.StatusBadGateway, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, http.StatusBadGateway, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}

	cli, err := s.plugins.Client(pluginID)
	if err != nil {
		msg := fmt.Sprintf("plugin %q unavailable: %v", pluginID, err)
		s.writeProxyError(w, downstream, http.StatusBadGateway, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, http.StatusBadGateway, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}

	wantsStream := requestWantsStream(body)
	if route.Provider.OnlyStream && !wantsStream {
		msg := "provider only_stream is enabled; non-stream requests are not allowed"
		s.writeProxyError(w, downstream, http.StatusBadRequest, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, http.StatusBadRequest, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}

	req := pluginipc.NewProxyRequest(pluginipc.ProxyRequestInput{
		Ingress: downstream.String(),
		Path:    r.URL.Path,
		Method:  r.Method,
		Headers: pluginipc.HeadersFromHTTP(r.Header),
		Body:    body,
		Model:   route.Model,
		Stream:  wantsStream,
	})

	if wantsStream {
		s.handlePluginStream(w, r, cli, req, requestID, downstream, route, pluginID, start, requestedModel, body)
		return
	}

	resp, err := cli.Complete(r.Context(), req)
	if err != nil {
		status, msg := pluginipc.MapCallError(err)
		s.writeProxyError(w, downstream, status, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}
	if resp == nil {
		msg := "empty plugin response"
		s.writeProxyError(w, downstream, http.StatusBadGateway, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, http.StatusBadGateway, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}

	status := resp.StatusOrOK()
	usage := usageFromPlugin(resp.Usage)
	if !resp.Success() {
		// Prefer plugin error body when present.
		if len(resp.Body) > 0 {
			writePluginHTTPResponse(w, status, resp.Headers, resp.Body)
		} else {
			s.writeProxyError(w, downstream, status, "plugin upstream error")
		}
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, "plugin upstream error", usage, requestedModel, body)
		return
	}

	writePluginHTTPResponse(w, status, resp.Headers, resp.Body)
	s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, "", usage, requestedModel, body)
}

func (s *proxyService) handlePluginStream(
	w http.ResponseWriter,
	r *http.Request,
	cli *pluginipc.Client,
	req pluginipc.ProxyRequest,
	requestID string,
	downstream constants.Protocol,
	route domain.Route,
	pluginID string,
	start time.Time,
	requestedModel string,
	body []byte,
) {
	stream, err := cli.OpenStream(r.Context(), req)
	if err != nil {
		status, msg := pluginipc.MapCallError(err)
		s.writeProxyError(w, downstream, status, msg)
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, msg, domain.TokenUsage{}, requestedModel, body)
		return
	}
	defer stream.Close()

	// Non-2xx open: never commit SSE (design Issue 18).
	if !stream.OK() {
		headers, errBody, status := stream.ErrorBody(r.Context())
		if len(errBody) > 0 {
			writePluginHTTPResponse(w, status, headers, errBody)
		} else {
			s.writeProxyError(w, downstream, status, "plugin stream open failed")
		}
		s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, "plugin stream open failed", domain.TokenUsage{}, requestedModel, body)
		return
	}

	status := stream.Status()
	if status == 0 {
		status = http.StatusOK
	}

	// Cancel upstream stream when client disconnects.
	ctx := r.Context()
	go func() {
		<-ctx.Done()
		_ = stream.Cancel(context.WithoutCancel(ctx))
	}()

	hdr := http.Header{}
	for k, v := range stream.Headers() {
		hdr.Set(k, v)
	}
	prepareStreamHeaders(w.Header(), hdr)
	w.WriteHeader(status)
	writer := flushWriter{writer: w}

	var usage domain.TokenUsage
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				statusCode, message := proxyOperationErrorStatus(ctx, err, status, "")
				if ctx.Err() != nil {
					// Client cancelled — 499-class classification via helper.
					s.recordPluginTraffic(r, requestID, downstream, route, pluginID, statusCode, start, message, usage, requestedModel, body)
					return
				}
				if err == io.EOF {
					s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, "", usage, requestedModel, body)
					return
				}
			}
			errorStatus, message := proxyOperationErrorStatus(ctx, err, http.StatusBadGateway, "")
			s.recordPluginTraffic(r, requestID, downstream, route, pluginID, errorStatus, start, message, usage, requestedModel, body)
			if s.logger != nil {
				s.logger.Warn("plugin stream recv failed", "request_id", requestID, "error", err)
			}
			return
		}
		switch ev.Kind {
		case "chunk":
			if ev.Chunk == nil {
				continue
			}
			if _, err := writer.Write(ev.Chunk.Data); err != nil {
				// Downstream gone — cancel plugin stream so upstream work stops.
				_ = stream.Cancel(context.WithoutCancel(ctx))
				errorStatus, message := proxyOperationErrorStatus(ctx, err, http.StatusBadGateway, "write downstream response failed")
				s.recordPluginTraffic(r, requestID, downstream, route, pluginID, errorStatus, start, message, usage, requestedModel, body)
				return
			}
		case "end":
			if ev.End != nil {
				usage = usageFromPlugin(ev.End.Usage)
			}
			s.recordPluginTraffic(r, requestID, downstream, route, pluginID, status, start, "", usage, requestedModel, body)
			return
		case "error":
			msg := "plugin stream error"
			if ev.Error != nil && ev.Error.Message != "" {
				msg = ev.Error.Message
			}
			s.recordPluginTraffic(r, requestID, downstream, route, pluginID, http.StatusBadGateway, start, msg, usage, requestedModel, body)
			if s.logger != nil {
				s.logger.Warn("plugin stream error", "request_id", requestID, "error", msg)
			}
			return
		}
	}
}

func (s *proxyService) recordPluginTraffic(
	r *http.Request,
	requestID string,
	downstream constants.Protocol,
	route domain.Route,
	pluginID string,
	statusCode int,
	start time.Time,
	message string,
	usage domain.TokenUsage,
	requestedModel string,
	requestBody []byte,
) {
	// Override upstream protocol label for traffic analytics.
	patched := route
	if pluginID != "" {
		patched.UpstreamProtocol = constants.Protocol("plugin:" + pluginID)
	}
	s.recordTraffic(r, requestID, downstream, patched, statusCode, start, message, usage, requestedModel, requestBody)
}

func writePluginHTTPResponse(w http.ResponseWriter, status int, headers map[string]string, body []byte) {
	if headers != nil {
		src := http.Header{}
		for k, v := range headers {
			src.Set(k, v)
		}
		copySafeHeaders(w.Header(), src)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(status)
	if len(body) > 0 {
		_, _ = w.Write(body)
	}
}

func usageFromPlugin(u *pluginipc.Usage) domain.TokenUsage {
	if u == nil {
		return domain.TokenUsage{}
	}
	return domain.TokenUsage{
		InputTokens:  int(u.InputTokens),
		OutputTokens: int(u.OutputTokens),
		TotalTokens:  int(u.TotalTokens),
	}.Normalize()
}

// isPluginProvider reports whether the resolved route should use IPC plugins.
func isPluginProvider(route domain.Route) bool {
	return route.Provider.Vendor == constants.VendorPlugin
}
