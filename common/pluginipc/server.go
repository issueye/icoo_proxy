package pluginipc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net"
)

// Server is the plugin-side JSON-RPC server.
type Server struct {
	conn            *Conn
	hostToken       string
	handshakeResult HandshakeResult
	inlineBodyLimit int
	onShutdown      func()
}

// ServerOptions configures the plugin server.
type ServerOptions struct {
	MaxFrameBytes   int
	InlineBodyLimit int
	HostToken       string
	Handshake       HandshakeResult
	OnShutdown      func()
}

// NewServer wraps an accepted connection as a plugin server.
// Default handlers for handshake/ping/shutdown/health/get_info are installed.
func NewServer(raw net.Conn, opts ServerOptions) *Server {
	s := &Server{
		hostToken:       opts.HostToken,
		handshakeResult: opts.Handshake,
		inlineBodyLimit: opts.InlineBodyLimit,
		onShutdown:      opts.OnShutdown,
	}
	if s.inlineBodyLimit <= 0 {
		s.inlineBodyLimit = DefaultInlineBodyLimit
	}
	if s.handshakeResult.IPCProtocolVersion == 0 {
		s.handshakeResult.IPCProtocolVersion = ProtocolVersion
	}
	s.conn = NewConn(raw, ConnOptions{MaxFrameBytes: opts.MaxFrameBytes})
	s.installDefaults()
	return s
}

// Conn returns the underlying connection.
func (s *Server) Conn() *Conn { return s.conn }

// RegisterHandler registers a custom method handler.
func (s *Server) RegisterHandler(method string, h Handler) {
	s.conn.RegisterHandler(method, h)
}

// Close closes the server connection.
func (s *Server) Close() error { return s.conn.Close() }

// Wait blocks until the connection is closed.
func (s *Server) Wait() { <-s.conn.Done() }

func (s *Server) installDefaults() {
	s.conn.RegisterHandler(MethodHandshake, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		var p HandshakeParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, NewRPCError(CodeInvalidParams, err.Error(), nil)
		}
		if subtle.ConstantTimeCompare([]byte(p.HostToken), []byte(s.hostToken)) != 1 {
			return nil, nil, ErrUnauthorized
		}
		return s.handshakeResult, nil, nil
	})
	s.conn.RegisterHandler(MethodPing, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		return map[string]string{"status": "pong"}, nil, nil
	})
	s.conn.RegisterHandler(MethodShutdown, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		if s.onShutdown != nil {
			go s.onShutdown()
		}
		return map[string]string{"status": "ok"}, nil, nil
	})
	s.conn.RegisterHandler(MethodHealth, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		return HealthResult{OK: true, Status: "healthy"}, nil, nil
	})
	s.conn.RegisterHandler(MethodGetInfo, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		return HealthResult{OK: true, Status: "ok", Details: map[string]any{
			"plugin_id": s.handshakeResult.PluginID,
			"version":   s.handshakeResult.PluginVersion,
		}}, nil, nil
	})
}

// RegisterComplete installs a proxy.complete handler with body resolution.
func (s *Server) RegisterComplete(fn func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error)) {
	s.conn.RegisterHandler(MethodProxyComplete, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		var req ProxyRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, nil, NewRPCError(CodeInvalidParams, err.Error(), nil)
		}
		req.Body = ResolveBody(req.BodyEncoding, req.Body, body)
		resp, err := fn(ctx, req)
		if err != nil {
			return nil, nil, err
		}
		raw := PrepareProxyResponse(resp, s.inlineBodyLimit)
		return resp, raw, nil
	})
}

// RegisterModelsList installs models.list.
func (s *Server) RegisterModelsList(fn func(ctx context.Context) (*ModelsListResult, error)) {
	s.conn.RegisterHandler(MethodModelsList, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		res, err := fn(ctx)
		if err != nil {
			return nil, nil, err
		}
		return res, nil, nil
	})
}

// StreamWriter sends stream notifications after open result is on the wire.
type StreamWriter struct {
	conn     *Conn
	streamID string
	seq      int64
	maxChunk int
}

// WriteChunk sends stream.chunk (auto-splits over max chunk size).
func (w *StreamWriter) WriteChunk(data []byte) error {
	max := w.maxChunk
	if max <= 0 {
		max = DefaultMaxStreamChunkBytes
	}
	for len(data) > 0 {
		n := len(data)
		if n > max {
			n = max
		}
		part := data[:n]
		data = data[n:]
		w.seq++
		if err := w.conn.Notify(MethodStreamChunk, StreamChunkParams{
			StreamID: w.streamID,
			Seq:      w.seq,
			Data:     part,
		}); err != nil {
			return err
		}
	}
	return nil
}

// End sends stream.end.
func (w *StreamWriter) End(usage *Usage) error {
	w.seq++
	return w.conn.Notify(MethodStreamEnd, StreamEndParams{
		StreamID: w.streamID,
		Seq:      w.seq,
		Usage:    usage,
	})
}

// Error sends stream.error.
func (w *StreamWriter) Error(code int, message string) error {
	w.seq++
	return w.conn.Notify(MethodStreamError, StreamErrorParams{
		StreamID: w.streamID,
		Seq:      w.seq,
		Code:     code,
		Message:  message,
	})
}

// streamOpenWireResult is intercepted by Conn.handleRequest so that AfterWrite
// runs only after the open result frame is fully written (open-before-chunk).
type streamOpenWireResult struct {
	Open       *StreamOpenResult
	AfterWrite func()
}

// RegisterProxyStream installs proxy.stream.open with guaranteed open-before-chunk.
//
// prepare returns a 2xx StreamOpenResult with stream_id, or a non-2xx errResp.
// run is invoked only after the open result is written; it must not be nil for
// successful opens if chunks are expected.
func (s *Server) RegisterProxyStream(
	prepare func(ctx context.Context, req ProxyRequest) (open *StreamOpenResult, errResp *ProxyResponse, err error),
	run func(ctx context.Context, req ProxyRequest, w *StreamWriter),
) {
	s.conn.RegisterHandler(MethodStreamOpen, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		var req ProxyRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, nil, NewRPCError(CodeInvalidParams, err.Error(), nil)
		}
		req.Body = ResolveBody(req.BodyEncoding, req.Body, body)

		open, errResp, err := prepare(ctx, req)
		if err != nil {
			return nil, nil, err
		}
		if errResp != nil {
			raw := PrepareProxyResponse(errResp, s.inlineBodyLimit)
			return errResp, raw, nil
		}
		if open == nil || open.StreamID == "" {
			return nil, nil, NewRPCError(CodeInternalError, "missing stream_id", nil)
		}
		if open.Status == 0 {
			open.Status = 200
		}
		if open.Status < 200 || open.Status >= 300 {
			pr := &ProxyResponse{Status: open.Status, Headers: open.Headers}
			raw := PrepareProxyResponse(pr, s.inlineBodyLimit)
			return pr, raw, nil
		}

		return &streamOpenWireResult{
			Open: open,
			AfterWrite: func() {
				if run == nil {
					return
				}
				w := &StreamWriter{
					conn:     s.conn,
					streamID: open.StreamID,
					maxChunk: DefaultMaxStreamChunkBytes,
				}
				run(ctx, req, w)
			},
		}, nil, nil
	})
}
