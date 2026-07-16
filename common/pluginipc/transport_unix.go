//go:build unix

package pluginipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Listen creates a pathname Unix domain socket listener.
func Listen(ctx context.Context, cfg ListenConfig) (net.Listener, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("pluginipc: empty endpoint")
	}
	dir := filepath.Dir(cfg.Endpoint)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("pluginipc mkdir: %w", err)
	}
	// Remove stale socket.
	_ = os.Remove(cfg.Endpoint)

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "unix", cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("pluginipc listen unix: %w", err)
	}
	if err := os.Chmod(cfg.Endpoint, 0o600); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("pluginipc chmod socket: %w", err)
	}
	return ln, nil
}

// Dial connects to a pathname Unix domain socket.
func Dial(ctx context.Context, cfg DialConfig) (net.Conn, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("pluginipc: empty endpoint")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var d net.Dialer
	deadline := time.Now().Add(10 * time.Second)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	var last error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		conn, err := d.DialContext(ctx, "unix", cfg.Endpoint)
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
	return nil, fmt.Errorf("pluginipc dial unix: %w", last)
}
