package pluginipc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Conn is a multiplexed JSON-RPC connection with raw-followup body support.
type Conn struct {
	raw           net.Conn
	maxFrameBytes int
	writeMu       sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan *Message

	streamMu sync.Mutex
	streams  map[string]chan streamEvent

	handlers   map[string]Handler
	handlersMu sync.RWMutex

	onNotify func(method string, params json.RawMessage, body []byte)

	closed   atomic.Bool
	closeOnce sync.Once
	done     chan struct{}
	readErr  atomic.Value // error

	nextID atomic.Uint64
}

// Handler handles an incoming request (not notification).
type Handler func(ctx context.Context, params json.RawMessage, body []byte) (result any, respBody []byte, err error)

type streamEvent struct {
	kind   string // chunk | end | error
	params json.RawMessage
}

// ConnOptions configures a Conn.
type ConnOptions struct {
	MaxFrameBytes int
	// OnNotification is called for notifications that are not stream.*
	// (stream events are demuxed into OpenStream channels).
	OnNotification func(method string, params json.RawMessage, body []byte)
}

// NewConn wraps a net.Conn as a pluginipc connection and starts the demux loop.
func NewConn(raw net.Conn, opts ConnOptions) *Conn {
	max := opts.MaxFrameBytes
	if max <= 0 {
		max = DefaultMaxFrameBytes
	}
	c := &Conn{
		raw:           raw,
		maxFrameBytes: max,
		pending:       make(map[string]chan *Message),
		streams:       make(map[string]chan streamEvent),
		handlers:      make(map[string]Handler),
		onNotify:      opts.OnNotification,
		done:          make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// RegisterHandler registers a method handler (plugin server side).
func (c *Conn) RegisterHandler(method string, h Handler) {
	c.handlersMu.Lock()
	c.handlers[method] = h
	c.handlersMu.Unlock()
}

// Close closes the underlying connection and unblocks waiters.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		err = c.raw.Close()
		close(c.done)
		c.failAll(ErrClosed)
	})
	return err
}

// Done returns a channel closed when the connection is closed.
func (c *Conn) Done() <-chan struct{} { return c.done }

// MaxFrameBytes returns the frame size limit.
func (c *Conn) MaxFrameBytes() int { return c.maxFrameBytes }

// WriteMessage atomically writes a JSON control frame and optional raw body frame.
func (c *Conn) WriteMessage(ctrlJSON []byte, rawBody []byte) error {
	if c.closed.Load() {
		return ErrClosed
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := WriteFrame(c.raw, ctrlJSON, c.maxFrameBytes); err != nil {
		return err
	}
	if rawBody == nil {
		return nil
	}
	return WriteFrame(c.raw, rawBody, c.maxFrameBytes)
}

// writeFrameJSON encodes v as JSON and writes a single frame under writeMu.
func (c *Conn) writeFrameJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return WriteFrame(c.raw, b, c.maxFrameBytes)
}

// Call sends a request and waits for the response. If body is non-nil and
// params already encode body_encoding=raw-followup, body is attached as raw frame.
func (c *Conn) Call(ctx context.Context, method string, params any, body []byte) (*Message, error) {
	if c.closed.Load() {
		return nil, ErrClosed
	}
	idNum := c.nextID.Add(1)
	idRaw, _ := json.Marshal(fmt.Sprintf("%d", idNum))

	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	// Inject body into params when using raw-followup: params is re-encoded
	// by callers who already set BodyEncoding / BodyLen. We only attach the raw frame.
	msg := Message{
		JSONRPC: "2.0",
		ID:      idRaw,
		Method:  method,
		Params:  paramsRaw,
	}
	ctrl, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Message, 1)
	key := string(idRaw)
	c.pendingMu.Lock()
	c.pending[key] = ch
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, key)
		c.pendingMu.Unlock()
	}()

	if err := c.WriteMessage(ctrl, body); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClosed
	case resp := <-ch:
		if resp == nil {
			return nil, ErrClosed
		}
		if resp.Error != nil {
			return resp, resp.Error
		}
		return resp, nil
	}
}

// Notify sends a JSON-RPC notification (no id).
func (c *Conn) Notify(method string, params any) error {
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	msg := Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsRaw,
	}
	return c.writeFrameJSON(msg)
}

func (c *Conn) failAll(err error) {
	c.pendingMu.Lock()
	for k, ch := range c.pending {
		select {
		case ch <- &Message{Error: NewRPCError(CodeInternalError, err.Error(), nil)}:
		default:
		}
		delete(c.pending, k)
	}
	c.pendingMu.Unlock()

	c.streamMu.Lock()
	for id, ch := range c.streams {
		close(ch)
		delete(c.streams, id)
	}
	c.streamMu.Unlock()
}

func (c *Conn) readLoop() {
	defer c.Close()

	type expectState struct {
		active bool
		msg    *Message
		len    int
	}
	var expect expectState

	for {
		frame, err := ReadFrame(c.raw, c.maxFrameBytes)
		if err != nil {
			if err != io.EOF && !c.closed.Load() {
				c.readErr.Store(err)
			}
			return
		}

		if expect.active {
			if len(frame) != expect.len {
				c.readErr.Store(fmt.Errorf("%w: raw body length mismatch want %d got %d", ErrProtocol, expect.len, len(frame)))
				return
			}
			expect.msg.Body = frame
			msg := expect.msg
			expect = expectState{}
			c.dispatch(msg)
			continue
		}

		var msg Message
		if err := json.Unmarshal(frame, &msg); err != nil {
			c.readErr.Store(fmt.Errorf("%w: json: %v", ErrProtocol, err))
			return
		}

		// Detect raw-followup on params or result envelopes.
		if need, n := rawFollowupLen(msg); need {
			expect = expectState{active: true, msg: &msg, len: n}
			continue
		}
		c.dispatch(&msg)
	}
}

type bodyMeta struct {
	BodyEncoding string `json:"body_encoding"`
	BodyLen      int    `json:"body_len"`
}

func rawFollowupLen(msg Message) (bool, int) {
	var meta bodyMeta
	// Prefer params (request), then result (response).
	src := msg.Params
	if len(src) == 0 {
		src = msg.Result
	}
	if len(src) == 0 {
		return false, 0
	}
	if err := json.Unmarshal(src, &meta); err != nil {
		return false, 0
	}
	if meta.BodyEncoding == BodyEncodingRawFollowup && meta.BodyLen > 0 {
		return true, meta.BodyLen
	}
	return false, 0
}

func (c *Conn) dispatch(msg *Message) {
	switch {
	case msg.IsResponse():
		key := string(msg.ID)
		c.pendingMu.Lock()
		ch, ok := c.pending[key]
		if ok {
			delete(c.pending, key)
		}
		c.pendingMu.Unlock()
		if ok {
			ch <- msg
		}
	case msg.IsNotification():
		switch msg.Method {
		case MethodStreamChunk, MethodStreamEnd, MethodStreamError:
			c.routeStreamNotify(msg)
		default:
			if c.onNotify != nil {
				c.onNotify(msg.Method, msg.Params, msg.Body)
			}
		}
	case msg.IsRequest():
		go c.handleRequest(msg)
	}
}

func (c *Conn) routeStreamNotify(msg *Message) {
	var sid struct {
		StreamID string `json:"stream_id"`
	}
	_ = json.Unmarshal(msg.Params, &sid)
	if sid.StreamID == "" {
		return
	}
	c.streamMu.Lock()
	ch, ok := c.streams[sid.StreamID]
	c.streamMu.Unlock()
	if !ok {
		return
	}
	kind := "chunk"
	switch msg.Method {
	case MethodStreamEnd:
		kind = "end"
	case MethodStreamError:
		kind = "error"
	}
	select {
	case ch <- streamEvent{kind: kind, params: msg.Params}:
	case <-c.done:
	}
}

func (c *Conn) handleRequest(msg *Message) {
	c.handlersMu.RLock()
	h, ok := c.handlers[msg.Method]
	c.handlersMu.RUnlock()

	var result any
	var respBody []byte
	var rpcErr *RPCError

	if !ok {
		rpcErr = NewRPCError(CodeMethodNotFound, "method not found: "+msg.Method, nil)
	} else {
		res, body, err := h(context.Background(), msg.Params, msg.Body)
		if err != nil {
			if e, ok := err.(*RPCError); ok {
				rpcErr = e
			} else {
				rpcErr = NewRPCError(CodeInternalError, err.Error(), nil)
			}
		} else {
			result = res
			respBody = body
		}
	}

	resp := Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}
	if rpcErr != nil {
		resp.Error = rpcErr
		_ = c.writeFrameJSON(resp)
		return
	}

	var afterWrite func()
	if wire, ok := result.(*streamOpenWireResult); ok {
		result = wire.Open
		afterWrite = wire.AfterWrite
	}

	resultRaw, err := json.Marshal(result)
	if err != nil {
		resp.Error = NewRPCError(CodeInternalError, err.Error(), nil)
		_ = c.writeFrameJSON(resp)
		return
	}
	resp.Result = resultRaw
	ctrl, err := json.Marshal(resp)
	if err != nil {
		return
	}
	if err := c.WriteMessage(ctrl, respBody); err != nil {
		return
	}
	// open-before-chunk: only after open result is fully written.
	if afterWrite != nil {
		go afterWrite()
	}
}

// registerStream creates a receive channel for stream notifications.
func (c *Conn) registerStream(streamID string) chan streamEvent {
	ch := make(chan streamEvent, 64)
	c.streamMu.Lock()
	c.streams[streamID] = ch
	c.streamMu.Unlock()
	return ch
}

// unregisterStream removes a stream channel.
func (c *Conn) unregisterStream(streamID string) {
	c.streamMu.Lock()
	if ch, ok := c.streams[streamID]; ok {
		delete(c.streams, streamID)
		close(ch)
	}
	c.streamMu.Unlock()
}
