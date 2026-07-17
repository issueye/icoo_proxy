package pluginipc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// Common capability strings declared in HandshakeResult.Capabilities.
const (
	CapProxyComplete = "proxy.complete"
	CapProxyStream   = "proxy.stream"
	CapModelsList    = "models.list"
	CapHealth        = "health"
	CapUI            = "ui"
)

// HeaderPluginAdminToken is injected by the bridge UI reverse-proxy and
// required by plugin admin HTTP handlers when AdminToken is set.
const HeaderPluginAdminToken = "X-ICOO-Plugin-Admin-Token"

// DefaultHostVersion is used by Connect when HostVersion is empty.
const DefaultHostVersion = "icoo_llm_bridge"

// HandshakeFrom builds a HandshakeResult from PluginMeta (fills protocol version).
func HandshakeFrom(meta PluginMeta) HandshakeResult {
	return HandshakeResult{
		IPCProtocolVersion: ProtocolVersion,
		PluginID:           meta.ID,
		PluginVersion:      meta.Version,
		Capabilities:       append([]string(nil), meta.Capabilities...),
		SupportedIngress:   append([]string(nil), meta.SupportedIngress...),
		UpstreamKind:       meta.UpstreamKind,
		AdminBaseURL:       meta.AdminBaseURL,
		AdminToken:         meta.AdminToken,
		UIPages:            append([]UIPage(nil), meta.UIPages...),
	}
}

// NewStreamID returns a random stream id with an optional prefix
// (e.g. prefix "gb" → "gb-<16hex>").
func NewStreamID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; fall back so callers still get a non-empty id.
		return strings.Trim(prefix+"-fallback", "-")
	}
	id := hex.EncodeToString(b[:])
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return id
	}
	return prefix + "-" + id
}

// OKJSON builds a 200 application/json ProxyResponse.
func OKJSON(body []byte, usage *Usage) *ProxyResponse {
	return &ProxyResponse{
		Status:  http.StatusOK,
		Headers: map[string]string{"content-type": "application/json"},
		Body:    body,
		Usage:   usage,
	}
}

// JSONError builds a JSON error ProxyResponse with the given HTTP status.
func JSONError(status int, message string) *ProxyResponse {
	if status <= 0 {
		status = http.StatusBadGateway
	}
	if message == "" {
		message = http.StatusText(status)
	}
	payload, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "plugin_error",
			"code":    status,
		},
	})
	return &ProxyResponse{
		Status:  status,
		Headers: map[string]string{"content-type": "application/json"},
		Body:    payload,
	}
}

// Unauthorized is JSONError(401, message).
func Unauthorized(message string) *ProxyResponse {
	if message == "" {
		message = "unauthorized"
	}
	return JSONError(http.StatusUnauthorized, message)
}

// BadGateway is JSONError(502, message).
func BadGateway(message string) *ProxyResponse {
	if message == "" {
		message = "bad gateway"
	}
	return JSONError(http.StatusBadGateway, message)
}

// SSEOpen builds a successful stream open result for text/event-stream.
func SSEOpen(streamID string) *StreamOpenResult {
	if streamID == "" {
		streamID = NewStreamID("stream")
	}
	return &StreamOpenResult{
		StreamID: streamID,
		Status:   http.StatusOK,
		Headers:  map[string]string{"content-type": "text/event-stream"},
	}
}

// RPCUnsupportedIngress wraps err as CodeUnsupportedIngress.
func RPCUnsupportedIngress(err error) error {
	if err == nil {
		return ErrUnsupportedIngress
	}
	return NewRPCError(CodeUnsupportedIngress, err.Error(), nil)
}

// UpstreamRPCError builds CodeUpstreamError (-32003) with data.status for MapCallError.
// status should be a 4xx/5xx; zero/invalid status falls back to 502 on the host side.
func UpstreamRPCError(status int, message string) error {
	if message == "" {
		message = "upstream error"
	}
	var data any
	if status >= 400 && status <= 599 {
		data = map[string]any{"status": status}
	}
	return NewRPCError(CodeUpstreamError, message, data)
}

// JSONStatus builds an application/json ProxyResponse with an arbitrary HTTP status.
// Prefer OKJSON for 200 success paths.
func JSONStatus(status int, body []byte, usage *Usage) *ProxyResponse {
	if status <= 0 {
		status = http.StatusOK
	}
	return &ProxyResponse{
		Status:  status,
		Headers: map[string]string{"content-type": "application/json"},
		Body:    body,
		Usage:   usage,
	}
}

// StatusOrOK returns r.Status, defaulting 0 to 200.
func (r *ProxyResponse) StatusOrOK() int {
	if r == nil || r.Status == 0 {
		return http.StatusOK
	}
	return r.Status
}

// Success reports whether the proxy response is a 2xx (treating 0 as 200).
func (r *ProxyResponse) Success() bool {
	st := r.StatusOrOK()
	return st >= 200 && st < 300
}
