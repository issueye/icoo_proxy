package pluginipc

import (
	"context"
	"encoding/json"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestRouteStreamNotifyDoesNotBlockWhenFull(t *testing.T) {
	c1, c2 := net.Pipe()
	t.Cleanup(func() {
		_ = c1.Close()
		_ = c2.Close()
	})
	// Host side demux only.
	host := NewConn(c1, ConnOptions{})
	t.Cleanup(func() { _ = host.Close() })

	// Fill the stream channel without a consumer.
	ch := host.registerStream("s1")
	for i := 0; i < 64; i++ {
		params, _ := json.Marshal(StreamChunkParams{StreamID: "s1", Seq: int64(i + 1), Data: []byte("x")})
		host.routeStreamNotify(&Message{Method: MethodStreamChunk, Params: params})
	}
	// One more event would previously block readLoop; must drop and return.
	params, _ := json.Marshal(StreamChunkParams{StreamID: "s1", Seq: 999, Data: []byte("y")})
	done := make(chan struct{})
	go func() {
		host.routeStreamNotify(&Message{Method: MethodStreamChunk, Params: params})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("routeStreamNotify blocked on full buffer")
	}
	if host.StreamDrops() == 0 {
		t.Fatal("expected stream drop counter to increase")
	}
	// Drain so unregister does not panic on closed/full channel.
	_ = ch
	host.unregisterStream("s1")
	_ = c2.Close()
}

func TestServerOnStreamCancelHook(t *testing.T) {
	c1, c2 := net.Pipe()
	t.Cleanup(func() {
		_ = c1.Close()
		_ = c2.Close()
	})
	var foundHits atomic.Int32
	var totalHits atomic.Int32
	srv := NewServer(c2, ServerOptions{
		HostToken: "tok",
		Handshake: HandshakeResult{PluginID: "p", PluginVersion: "1"},
		OnStreamCancel: func(streamID string, found bool) {
			totalHits.Add(1)
			if found {
				foundHits.Add(1)
			}
		},
	})
	// Open a stream that blocks until cancelled.
	srv.RegisterProxyStream(
		func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, error) {
			return &StreamOpenResult{StreamID: "cancel-me", Status: 200}, nil, nil
		},
		func(ctx context.Context, req ProxyRequest, w *StreamWriter) {
			<-ctx.Done()
			_ = w.End(nil)
		},
	)
	cli := NewClient(c1, ClientOptions{})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "test"); err != nil {
		t.Fatal(err)
	}
	stream, err := cli.OpenStream(ctx, ProxyRequest{Ingress: "anthropic", Path: "/v1/messages", Method: "POST", Body: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Cancel(ctx); err != nil {
		t.Fatal(err)
	}
	// Unknown / second cancel still invokes hook with found=false.
	if err := stream.Cancel(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if totalHits.Load() >= 2 && foundHits.Load() >= 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("hooks total=%d found=%d", totalHits.Load(), foundHits.Load())
}
