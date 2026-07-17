package pluginipc

import "encoding/json"

// ProtocolVersion is the IPC protocol version negotiated at handshake.
const ProtocolVersion = 1

// Body encoding values for proxy payloads.
const (
	BodyEncodingInline      = "inline"
	BodyEncodingRawFollowup = "raw-followup"
)

// Default limits (bytes).
const (
	DefaultMaxFrameBytes       = 64 << 20 // 64 MiB
	DefaultInlineBodyLimit     = 256 << 10 // 256 KiB
	DefaultMaxStreamChunkBytes = 64 << 10  // 64 KiB
	DefaultMaxConcurrentStreams = 32
)

// JSON-RPC envelope fields.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`

	// Body is populated by demux when body_encoding=raw-followup.
	// Not part of the JSON wire format.
	Body []byte `json:"-"`
}

func (m *Message) IsRequest() bool {
	return m != nil && m.Method != "" && len(m.ID) > 0 && m.Error == nil && len(m.Result) == 0
}

func (m *Message) IsNotification() bool {
	return m != nil && m.Method != "" && len(m.ID) == 0
}

func (m *Message) IsResponse() bool {
	return m != nil && m.Method == "" && len(m.ID) > 0
}

// HandshakeParams is sent with plugin.handshake.
type HandshakeParams struct {
	IPCProtocolVersion int    `json:"ipc_protocol_version"`
	HostToken          string `json:"host_token"`
	HostVersion        string `json:"host_version,omitempty"`
}

// HandshakeResult is returned by plugin.handshake.
type HandshakeResult struct {
	IPCProtocolVersion int      `json:"ipc_protocol_version"`
	PluginID           string   `json:"plugin_id"`
	PluginVersion      string   `json:"plugin_version"`
	Capabilities       []string `json:"capabilities"`
	SupportedIngress   []string `json:"supported_ingress"`
	UpstreamKind       string   `json:"upstream_kind,omitempty"`
	// AdminBaseURL is an optional loopback HTTP base for plugin-provided UI
	// (e.g. http://127.0.0.1:19283). Host reverse-proxies under /api/v1/plugins/:id/ui/.
	AdminBaseURL string `json:"admin_base_url,omitempty"`
	// AdminToken is an ephemeral secret for the plugin admin HTTP surface.
	// The host injects it as HeaderPluginAdminToken when reverse-proxying UI.
	// Must NOT be exposed in public management REST DTOs (PluginView / ui-pages).
	AdminToken string `json:"admin_token,omitempty"`
	// UIPages lists extension pages the desktop shell can mount in the sidebar.
	UIPages []UIPage `json:"ui_pages,omitempty"`
}

// UIPage describes a desktop extension page contributed by a plugin.
type UIPage struct {
	// ID is stable within the plugin (e.g. "credentials").
	ID string `json:"id"`
	// Title is the nav label.
	Title string `json:"title"`
	// Path is relative to the plugin admin UI root (e.g. "/" or "/credentials").
	Path string `json:"path"`
	// Icon is a desktop icon key (optional; e.g. "key", "plugin").
	Icon string `json:"icon,omitempty"`
	// Group is the sidebar group name (e.g. "插件").
	Group string `json:"group,omitempty"`
	// Description is optional nav tooltip/subtitle.
	Description string `json:"description,omitempty"`
}

// ProxyRequest is shared by proxy.complete and proxy.stream.open params.
type ProxyRequest struct {
	Ingress      string            `json:"ingress"`
	Path         string            `json:"path,omitempty"`
	Method       string            `json:"method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyEncoding string            `json:"body_encoding"`
	BodyLen      int               `json:"body_len,omitempty"`
	Body         []byte            `json:"body,omitempty"` // inline only; omit with raw-followup
	Model        string            `json:"model,omitempty"`
	Stream       bool              `json:"stream,omitempty"`
}

// ProxyResponse is the result of proxy.complete or a non-stream open error path.
type ProxyResponse struct {
	Status       int               `json:"status"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyEncoding string            `json:"body_encoding,omitempty"`
	BodyLen      int               `json:"body_len,omitempty"`
	Body         []byte            `json:"body,omitempty"`
	Usage        *Usage            `json:"usage,omitempty"`
}

// StreamOpenResult is the successful result of proxy.stream.open.
type StreamOpenResult struct {
	StreamID string            `json:"stream_id"`
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// StreamChunkParams is a stream.chunk notification.
type StreamChunkParams struct {
	StreamID string `json:"stream_id"`
	Seq      int64  `json:"seq"`
	Data     []byte `json:"data"` // JSON base64
}

// StreamEndParams is a stream.end notification.
type StreamEndParams struct {
	StreamID string `json:"stream_id"`
	Seq      int64  `json:"seq"`
	Usage    *Usage `json:"usage,omitempty"`
}

// StreamErrorParams is a stream.error notification.
type StreamErrorParams struct {
	StreamID string `json:"stream_id"`
	Seq      int64  `json:"seq"`
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message"`
}

// StreamCancelParams is stream.cancel request/notification.
type StreamCancelParams struct {
	StreamID string `json:"stream_id"`
}

// Usage mirrors token accounting carried on stream end / complete.
type Usage struct {
	InputTokens  int64 `json:"input_tokens,omitempty"`
	OutputTokens int64 `json:"output_tokens,omitempty"`
	TotalTokens  int64 `json:"total_tokens,omitempty"`
}

// HealthResult is returned by plugin.health / plugin.get_info.
type HealthResult struct {
	OK      bool           `json:"ok"`
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}

// ModelInfo is one entry from models.list.
type ModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

// ModelsListResult is returned by models.list.
type ModelsListResult struct {
	Models []ModelInfo `json:"models"`
}

// Methods
const (
	MethodHandshake    = "plugin.handshake"
	MethodPing         = "plugin.ping"
	MethodGetInfo      = "plugin.get_info"
	MethodShutdown     = "plugin.shutdown"
	MethodHealth       = "plugin.health"
	MethodModelsList   = "models.list"
	MethodProxyComplete = "proxy.complete"
	MethodStreamOpen   = "proxy.stream.open"
	MethodStreamChunk  = "stream.chunk"
	MethodStreamEnd    = "stream.end"
	MethodStreamError  = "stream.error"
	MethodStreamCancel = "stream.cancel"
)
