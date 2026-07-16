package pluginipc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// Client is the host-side high-level API over a Conn.
type Client struct {
	conn            *Conn
	inlineBodyLimit int
	maxStreamChunks int
	maxConcurrent   int

	streamCountMu sync.Mutex
	activeStreams int
}

// ClientOptions configures a Client.
type ClientOptions struct {
	MaxFrameBytes       int
	InlineBodyLimit     int
	MaxStreamChunkBytes int
	MaxConcurrentStreams int
}

// NewClient wraps a net.Conn as a host client.
func NewClient(raw net.Conn, opts ClientOptions) *Client {
	c := NewConn(raw, ConnOptions{MaxFrameBytes: opts.MaxFrameBytes})
	inline := opts.InlineBodyLimit
	if inline <= 0 {
		inline = DefaultInlineBodyLimit
	}
	maxConc := opts.MaxConcurrentStreams
	if maxConc <= 0 {
		maxConc = DefaultMaxConcurrentStreams
	}
	return &Client{
		conn:            c,
		inlineBodyLimit: inline,
		maxStreamChunks: opts.MaxStreamChunkBytes,
		maxConcurrent:   maxConc,
	}
}

// Conn exposes the underlying multiplexed connection.
func (c *Client) Conn() *Conn { return c.conn }

// Close closes the client connection.
func (c *Client) Close() error { return c.conn.Close() }

// Handshake performs plugin.handshake.
func (c *Client) Handshake(ctx context.Context, token, hostVersion string) (*HandshakeResult, error) {
	params := HandshakeParams{
		IPCProtocolVersion: ProtocolVersion,
		HostToken:          token,
		HostVersion:        hostVersion,
	}
	resp, err := c.conn.Call(ctx, MethodHandshake, params, nil)
	if err != nil {
		return nil, err
	}
	var out HandshakeResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Ping calls plugin.ping.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.conn.Call(ctx, MethodPing, map[string]any{}, nil)
	return err
}

// Health calls plugin.health.
func (c *Client) Health(ctx context.Context) (*HealthResult, error) {
	resp, err := c.conn.Call(ctx, MethodHealth, map[string]any{}, nil)
	if err != nil {
		return nil, err
	}
	var out HealthResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetInfo calls plugin.get_info.
func (c *Client) GetInfo(ctx context.Context) (*HealthResult, error) {
	resp, err := c.conn.Call(ctx, MethodGetInfo, map[string]any{}, nil)
	if err != nil {
		return nil, err
	}
	var out HealthResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Shutdown calls plugin.shutdown.
func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.conn.Call(ctx, MethodShutdown, map[string]any{}, nil)
	return err
}

// ListModels calls models.list.
func (c *Client) ListModels(ctx context.Context) (*ModelsListResult, error) {
	resp, err := c.conn.Call(ctx, MethodModelsList, map[string]any{}, nil)
	if err != nil {
		return nil, err
	}
	var out ModelsListResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Complete calls proxy.complete with optional large-body transfer.
func (c *Client) Complete(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	raw := PrepareProxyRequest(&req, c.inlineBodyLimit)
	resp, err := c.conn.Call(ctx, MethodProxyComplete, req, raw)
	if err != nil {
		return nil, err
	}
	var out ProxyResponse
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	out.Body = ResolveBody(out.BodyEncoding, out.Body, resp.Body)
	return &out, nil
}

// Stream is a host-side streaming proxy session.
type Stream struct {
	client   *Client
	streamID string
	events   chan streamEvent
	status   int
	headers  map[string]string
	closed   bool
}

// OpenStream calls proxy.stream.open and registers for stream notifications.
// On non-2xx open result, returns a Stream with Status set and no body reader
// (caller must treat as complete-style error — never start SSE).
func (c *Client) OpenStream(ctx context.Context, req ProxyRequest) (*Stream, error) {
	c.streamCountMu.Lock()
	if c.activeStreams >= c.maxConcurrent {
		c.streamCountMu.Unlock()
		return nil, ErrTooManyStreams
	}
	c.activeStreams++
	c.streamCountMu.Unlock()

	releaseSlot := true
	defer func() {
		if releaseSlot {
			c.streamCountMu.Lock()
			c.activeStreams--
			c.streamCountMu.Unlock()
		}
	}()

	req.Stream = true
	raw := PrepareProxyRequest(&req, c.inlineBodyLimit)
	resp, err := c.conn.Call(ctx, MethodStreamOpen, req, raw)
	if err != nil {
		return nil, err
	}

	// Prefer StreamOpenResult; also accept ProxyResponse-shaped errors.
	var open StreamOpenResult
	if err := json.Unmarshal(resp.Result, &open); err != nil {
		return nil, err
	}
	if open.Status == 0 {
		open.Status = 200
	}

	// Non-2xx: may include body via raw-followup on the same result envelope.
	if open.Status < 200 || open.Status >= 300 {
		// Re-parse as ProxyResponse for body if present.
		var pr ProxyResponse
		_ = json.Unmarshal(resp.Result, &pr)
		pr.Body = ResolveBody(pr.BodyEncoding, pr.Body, resp.Body)
		if pr.Status == 0 {
			pr.Status = open.Status
		}
		// Return a closed stream carrying error payload via headers/status only;
		// body is exposed through a one-shot Read on synthetic stream.
		s := &Stream{
			client:  c,
			status:  pr.Status,
			headers: pr.Headers,
			// no streamID registration — caller must not start SSE
		}
		// Buffer both optional error body chunk and terminal end.
		s.events = make(chan streamEvent, 2)
		if len(pr.Body) > 0 {
			chunk, _ := json.Marshal(StreamChunkParams{Data: pr.Body, Seq: 1})
			s.events <- streamEvent{kind: "chunk", params: chunk}
		}
		end, _ := json.Marshal(StreamEndParams{Seq: 2})
		s.events <- streamEvent{kind: "end", params: end}
		close(s.events)
		// keep releaseSlot true
		return s, nil
	}

	if open.StreamID == "" {
		return nil, fmt.Errorf("pluginipc: empty stream_id")
	}

	ch := c.conn.registerStream(open.StreamID)
	releaseSlot = false
	return &Stream{
		client:   c,
		streamID: open.StreamID,
		events:   ch,
		status:   open.Status,
		headers:  open.Headers,
	}, nil
}

// StreamID returns the plugin-assigned stream id (empty on non-2xx open).
func (s *Stream) StreamID() string { return s.streamID }

// Status is the open response HTTP status.
func (s *Stream) Status() int { return s.status }

// Headers are response headers from open.
func (s *Stream) Headers() map[string]string { return s.headers }

// StreamEvent is one demuxed stream notification.
type StreamEvent struct {
	Kind   string // chunk | end | error
	Chunk  *StreamChunkParams
	End    *StreamEndParams
	Error  *StreamErrorParams
}

// Recv blocks until the next stream event or context cancellation.
func (s *Stream) Recv(ctx context.Context) (*StreamEvent, error) {
	if s.closed {
		return nil, io.EOF
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.client.conn.done:
		return nil, ErrClosed
	case ev, ok := <-s.events:
		if !ok {
			s.closed = true
			return nil, io.EOF
		}
		out := &StreamEvent{Kind: ev.kind}
		switch ev.kind {
		case "chunk":
			var p StreamChunkParams
			if err := json.Unmarshal(ev.params, &p); err != nil {
				return nil, err
			}
			out.Chunk = &p
		case "end":
			var p StreamEndParams
			if err := json.Unmarshal(ev.params, &p); err != nil {
				return nil, err
			}
			out.End = &p
			s.finish()
		case "error":
			var p StreamErrorParams
			if err := json.Unmarshal(ev.params, &p); err != nil {
				return nil, err
			}
			out.Error = &p
			s.finish()
		}
		return out, nil
	}
}

// Cancel asks the plugin to stop the stream.
func (s *Stream) Cancel(ctx context.Context) error {
	if s.streamID == "" {
		return nil
	}
	_, err := s.client.conn.Call(ctx, MethodStreamCancel, StreamCancelParams{StreamID: s.streamID}, nil)
	return err
}

// Close unregisters the stream without cancel (use Cancel for graceful stop).
func (s *Stream) Close() {
	s.finish()
}

func (s *Stream) finish() {
	if s.closed {
		return
	}
	s.closed = true
	if s.streamID != "" {
		s.client.conn.unregisterStream(s.streamID)
		s.client.streamCountMu.Lock()
		s.client.activeStreams--
		s.client.streamCountMu.Unlock()
	}
}
