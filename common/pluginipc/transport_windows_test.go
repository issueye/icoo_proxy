//go:build windows

package pluginipc

import (
	"context"
	"testing"
	"time"
)

func TestDefaultWindowsPipeSDDLIsOwnerOnly(t *testing.T) {
	const want = "D:P(A;;GA;;;OW)"
	if DefaultWindowsPipeSDDL != want {
		t.Fatalf("DefaultWindowsPipeSDDL=%q want %q", DefaultWindowsPipeSDDL, want)
	}
}

func TestWindowsPipeListenDialRoundTrip(t *testing.T) {
	endpoint, err := NewEndpoint("sddltest", "")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln, err := Listen(ctx, ListenConfig{Endpoint: endpoint})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			accepted <- err
			return
		}
		_ = conn.Close()
		accepted <- nil
	}()

	conn, err := Dial(ctx, DialConfig{Endpoint: endpoint})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	_ = conn.Close()

	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("Accept: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for Accept")
	}
}
