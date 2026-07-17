package pluginipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ConnectConfig configures Dial + NewClient + Handshake.
// It does NOT spawn plugin processes; the host still owns exec/env lifecycle.
type ConnectConfig struct {
	Endpoint             string
	Token                string
	HostVersion          string        // default DefaultHostVersion
	DialTimeout          time.Duration // default 30s
	HandshakeTimeout     time.Duration // default 15s
	MaxFrameBytes        int
	MaxConcurrentStreams int
	InlineBodyLimit      int
	MaxStreamChunkBytes  int
}

// Connect dials a plugin endpoint and completes handshake.
// On handshake failure the connection is closed.
func Connect(ctx context.Context, cfg ConnectConfig) (*Client, *HandshakeResult, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, nil, fmt.Errorf("pluginipc: endpoint is required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, nil, fmt.Errorf("pluginipc: token is required")
	}
	if cfg.HostVersion == "" {
		cfg.HostVersion = DefaultHostVersion
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 30 * time.Second
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 15 * time.Second
	}

	// Avoid inheriting a short caller deadline that would cancel a long-lived conn.
	base := context.WithoutCancel(ctx)

	dialCtx, dialCancel := context.WithTimeout(base, cfg.DialTimeout)
	defer dialCancel()
	raw, err := Dial(dialCtx, DialConfig{Endpoint: cfg.Endpoint})
	if err != nil {
		return nil, nil, err
	}

	cli := NewClient(raw, ClientOptions{
		MaxFrameBytes:        cfg.MaxFrameBytes,
		InlineBodyLimit:      cfg.InlineBodyLimit,
		MaxStreamChunkBytes:  cfg.MaxStreamChunkBytes,
		MaxConcurrentStreams: cfg.MaxConcurrentStreams,
	})

	hsCtx, hsCancel := context.WithTimeout(base, cfg.HandshakeTimeout)
	defer hsCancel()
	hs, err := cli.Handshake(hsCtx, cfg.Token, cfg.HostVersion)
	if err != nil {
		_ = cli.Close()
		return nil, nil, err
	}
	return cli, hs, nil
}

// ProxyRequestInput is the host-facing request builder input.
// Headers are filtered and Anthropic version is injected when needed.
type ProxyRequestInput struct {
	Ingress string
	Path    string
	Method  string
	Headers map[string]string // raw; will be filtered
	Body    []byte
	Model   string
	Stream  bool
}

// NewProxyRequest applies FilterHeaders + EnsureAnthropicVersion.
func NewProxyRequest(in ProxyRequestInput) ProxyRequest {
	headers := FilterHeaders(in.Headers)
	headers = EnsureAnthropicVersion(in.Ingress, headers)
	return ProxyRequest{
		Ingress: in.Ingress,
		Path:    in.Path,
		Method:  in.Method,
		Headers: headers,
		Body:    in.Body,
		Model:   in.Model,
		Stream:  in.Stream,
	}
}

// HeadersFromHTTP flattens http.Header to map[string]string (first value wins).
func HeadersFromHTTP(h http.Header) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, vals := range h {
		if len(vals) == 0 {
			continue
		}
		out[k] = vals[0]
	}
	return out
}

// MapCallError maps a Client RPC/transport error to a suggested HTTP status + message.
// For CodeUpstreamError (-32003), prefers data.status when it is a valid 4xx/5xx.
func MapCallError(err error) (status int, message string) {
	if err == nil {
		return http.StatusBadGateway, "plugin error"
	}
	var rpc *RPCError
	if errors.As(err, &rpc) && rpc != nil {
		return mapRPCError(rpc)
	}
	// Also accept direct pointer equality style returns.
	if rpc, ok := err.(*RPCError); ok && rpc != nil {
		return mapRPCError(rpc)
	}
	msg := err.Error()
	switch {
	case errors.Is(err, ErrTooManyStreams) || strings.Contains(msg, "too many streams"):
		return http.StatusServiceUnavailable, msg
	case errors.Is(err, ErrFrameTooLarge) || strings.Contains(msg, "frame too large"):
		return http.StatusRequestEntityTooLarge, msg
	case errors.Is(err, ErrClosed):
		return http.StatusBadGateway, msg
	default:
		return http.StatusBadGateway, msg
	}
}

func mapRPCError(rpc *RPCError) (int, string) {
	if rpc == nil {
		return http.StatusBadGateway, "plugin error"
	}
	if rpc.Code == CodeUpstreamError {
		if st := statusFromRPCData(rpc.Data); st > 0 {
			return st, rpc.Message
		}
	}
	return HTTPStatus(rpc.Code), rpc.Message
}

// statusFromRPCData extracts a 4xx/5xx HTTP status from RPCError.Data.
// Accepts map[string]any{"status": N} (JSON decode) and similar shapes.
func statusFromRPCData(data any) int {
	if data == nil {
		return 0
	}
	m, ok := data.(map[string]any)
	if !ok {
		return 0
	}
	return asHTTPStatus(m["status"])
}

func asHTTPStatus(v any) int {
	var n int
	switch x := v.(type) {
	case int:
		n = x
	case int32:
		n = int(x)
	case int64:
		n = int(x)
	case float64:
		n = int(x)
	case float32:
		n = int(x)
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0
		}
		n = int(i)
	default:
		return 0
	}
	if n >= 400 && n <= 599 {
		return n
	}
	return 0
}

// OK reports whether stream open returned a 2xx status (0 treated as 200).
// Hosts MUST NOT start SSE when OK is false (contract Issue 18).
func (s *Stream) OK() bool {
	if s == nil {
		return false
	}
	st := s.status
	if st == 0 {
		st = http.StatusOK
	}
	return st >= 200 && st < 300
}

// ErrorBody drains a synthetic non-2xx open payload (chunk + end).
// On successful opens it returns immediately with empty body.
// Safe to call once; subsequent Recv may see EOF.
func (s *Stream) ErrorBody(ctx context.Context) (headers map[string]string, body []byte, status int) {
	if s == nil {
		return nil, nil, http.StatusBadGateway
	}
	if ctx == nil {
		ctx = context.Background()
	}
	status = s.status
	if status == 0 {
		status = http.StatusOK
	}
	headers = s.headers
	if s.OK() {
		return headers, nil, status
	}
	for {
		ev, err := s.Recv(ctx)
		if err != nil {
			break
		}
		if ev == nil {
			continue
		}
		if ev.Kind == "chunk" && ev.Chunk != nil {
			body = append(body, ev.Chunk.Data...)
		}
		if ev.Kind == "end" || ev.Kind == "error" {
			break
		}
	}
	return headers, body, status
}
