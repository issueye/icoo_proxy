//go:build windows

package pluginipc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// DefaultWindowsPipeSDDL grants full access only to the pipe owner (creating user).
// OW = Owner Rights SID.
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
	// Retry briefly: plugin may still be starting the listener.
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	var last error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		conn, err := winio.DialPipeContext(ctx, cfg.Endpoint)
		if err == nil {
			return conn, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("pluginipc dial pipe: %w", last)
}
