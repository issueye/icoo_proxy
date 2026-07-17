package pluginipc

import (
	"context"
	"encoding/json"
	"io"
	"sync"
)

// StreamWriterAdapter adapts StreamWriter to io.Writer.
// Each Write is sent as one or more stream.chunk notifications (auto-split).
type StreamWriterAdapter struct {
	W *StreamWriter
}

// Write implements io.Writer.
func (a StreamWriterAdapter) Write(p []byte) (int, error) {
	if a.W == nil {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}
	// Copy so callers can reuse the buffer after Write returns.
	cp := append([]byte(nil), p...)
	if err := a.W.WriteChunk(cp); err != nil {
		return 0, err
	}
	return len(p), nil
}

// AsWriter returns an io.Writer that forwards to WriteChunk.
func (w *StreamWriter) AsWriter() io.Writer {
	return StreamWriterAdapter{W: w}
}

// RegisterProxyStreamEx installs a unified stream handler.
// The prepare phase may return a run closure that captures local state,
// eliminating the need for a plugin-side pendingRuns map.
func (s *Server) RegisterProxyStreamEx(fn StreamHandler) {
	if fn == nil {
		return
	}
	var pending sync.Map // streamID → func(ctx, *StreamWriter)
	s.RegisterProxyStream(
		func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, error) {
			open, errResp, run, err := fn(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			if open != nil && run != nil && open.StreamID != "" {
				pending.Store(open.StreamID, run)
			}
			return open, errResp, nil
		},
		func(ctx context.Context, req ProxyRequest, w *StreamWriter) {
			if w == nil {
				return
			}
			v, ok := pending.LoadAndDelete(w.StreamID())
			if !ok {
				_ = w.Error(CodeInternalError, "missing stream runner")
				return
			}
			run, _ := v.(func(context.Context, *StreamWriter))
			if run == nil {
				_ = w.Error(CodeInternalError, "invalid stream runner")
				return
			}
			run(ctx, w)
		},
	)
}

// RegisterHealth overrides the default plugin.health handler.
func (s *Server) RegisterHealth(fn func(ctx context.Context) (*HealthResult, error)) {
	if fn == nil {
		return
	}
	s.RegisterHandler(MethodHealth, func(ctx context.Context, params json.RawMessage, body []byte) (any, []byte, error) {
		res, err := fn(ctx)
		if err != nil {
			return nil, nil, err
		}
		if res == nil {
			return HealthResult{OK: true, Status: "healthy"}, nil, nil
		}
		return res, nil, nil
	})
}
