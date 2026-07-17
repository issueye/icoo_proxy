//go:build windows

package pluginipc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// DefaultWindowsPipeSDDL grants the creating owner full access only.
// Matches Plugin IPC Contract v1 (owner-only). Host and plugin run as the
// same local user; the host token remains the second auth factor.
// Pipes stay in the local named-pipe namespace (not network-exposed).
// Override via ListenConfig.SecurityDescriptor when a broader ACL is required.
const DefaultWindowsPipeSDDL = "D:P(A;;GA;;;OW)"

// Listen creates a Windows named pipe listener.
func Listen(ctx context.Context, cfg ListenConfig) (net.Listener, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("pluginipc: empty endpoint")
	}
	sddl := cfg.SecurityDescriptor
	if sddl == "" {
		sddl = DefaultWindowsPipeSDDL
	}
	pc := &winio.PipeConfig{
		SecurityDescriptor: sddl,
		MessageMode:        false, // byte mode for length-prefix framing
		InputBufferSize:    1 << 20,
		OutputBufferSize:   1 << 20,
	}
	ln, err := winio.ListenPipe(cfg.Endpoint, pc)
	if err != nil {
		return nil, fmt.Errorf("pluginipc listen pipe: %w", err)
	}
	// Best-effort cancel: close listener when ctx done.
	if ctx != nil {
		go func() {
			<-ctx.Done()
			_ = ln.Close()
		}()
	}
	return ln, nil
}

// Dial connects to a Windows named pipe.
func Dial(ctx context.Context, cfg DialConfig) (net.Conn, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("pluginipc: empty endpoint")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Retry until ctx deadline (or a 30s default). Plugin may still be starting.
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	var last error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			if last != nil {
				return nil, fmt.Errorf("pluginipc dial pipe: %w", last)
			}
			return nil, err
		}
		// Per-attempt timeout so a single hung DialPipe does not eat the whole budget.
		attemptCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		conn, err := winio.DialPipeContext(attemptCtx, cfg.Endpoint)
		cancel()
		if err == nil {
			return conn, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("pluginipc dial pipe: %w", last)
		case <-time.After(50 * time.Millisecond):
		}
	}
	if last == nil {
		last = context.DeadlineExceeded
	}
	return nil, fmt.Errorf("pluginipc dial pipe: %w", last)
}
