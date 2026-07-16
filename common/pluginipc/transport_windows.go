//go:build windows

package pluginipc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// DefaultWindowsPipeSDDL allows the creating owner, interactive users, admins,
// and SYSTEM. OW-only descriptors have been observed to reject same-user dials
// from a parent process (desktop → bridge → plugin) on some Windows builds.
// Pipes remain local-only (named pipe namespace); not exposed on the network.
const DefaultWindowsPipeSDDL = "D:P(A;;GA;;;OW)(A;;GA;;;IU)(A;;GA;;;BA)(A;;GA;;;SY)"

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
