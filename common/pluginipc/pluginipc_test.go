package pluginipc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func startPair(t *testing.T, token string) (*Client, *Server) {
	t.Helper()
	c1, c2 := net.Pipe()
	srv := NewServer(c2, ServerOptions{
		HostToken: token,
		Handshake: HandshakeResult{
			PluginID:         "mock",
			PluginVersion:    "0.0.1",
			Capabilities:     []string{"proxy.complete", "proxy.stream", "models.list", "health"},
			SupportedIngress: []string{"anthropic", "openai-chat"},
		},
		InlineBodyLimit: 256 << 10,
	})
	cli := NewClient(c1, ClientOptions{InlineBodyLimit: 256 << 10})
	t.Cleanup(func() {
		_ = cli.Close()
		_ = srv.Close()
	})
	return cli, srv
}

func TestHandshakeAndPing(t *testing.T) {
	cli, _ := startPair(t, "secret-token")
	ctx := context.Background()
	hs, err := cli.Handshake(ctx, "secret-token", "test-host")
	if err != nil {
		t.Fatal(err)
	}
	if hs.PluginID != "mock" {
		t.Fatalf("plugin id: %s", hs.PluginID)
	}
	if err := cli.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Handshake(ctx, "wrong", "x"); err == nil {
		t.Fatal("expected unauthorized")
	}
}

func TestCompleteInlineAndRawFollowup(t *testing.T) {
	cli, srv := startPair(t, "tok")
	srv.RegisterComplete(func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
		return &ProxyResponse{
			Status: 200,
			Headers: map[string]string{"content-type": "application/json"},
			Body:    append([]byte("echo:"), req.Body...),
		}, nil
	})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}

	// Inline small body
	small := bytes.Repeat([]byte("a"), 100)
	resp, err := cli.Complete(ctx, ProxyRequest{Ingress: "anthropic", Body: small})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Body, append([]byte("echo:"), small...)) {
		t.Fatalf("inline body mismatch: got %d bytes", len(resp.Body))
	}

	// Raw-followup large body (> 256 KiB)
	large := bytes.Repeat([]byte("b"), 300*1024)
	resp, err = cli.Complete(ctx, ProxyRequest{Ingress: "anthropic", Body: large})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resp.Body, append([]byte("echo:"), large...)) {
		t.Fatalf("raw body mismatch: got %d want %d", len(resp.Body), len(large)+5)
	}
}

func TestConcurrentRawFollowupAndPing(t *testing.T) {
	cli, srv := startPair(t, "tok")
	srv.RegisterComplete(func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
		// Echo body with marker so we can verify no cross-talk.
		return &ProxyResponse{Status: 200, Body: req.Body}, nil
	})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}

	const n = 8
	var wg sync.WaitGroup
	errCh := make(chan error, n*2)
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := bytes.Repeat([]byte{byte(i)}, 300*1024)
			resp, err := cli.Complete(ctx, ProxyRequest{Ingress: "anthropic", Body: body})
			if err != nil {
				errCh <- err
				return
			}
			if !bytes.Equal(resp.Body, body) {
				errCh <- fmt.Errorf("body cross-talk for worker %d", i)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cli.Ping(ctx); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func TestStreamOpenBeforeChunk(t *testing.T) {
	cli, srv := startPair(t, "tok")
	srv.RegisterProxyStream(
		func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, error) {
			return &StreamOpenResult{
				StreamID: "s1",
				Status:   200,
				Headers:  map[string]string{"content-type": "text/event-stream"},
			}, nil, nil
		},
		func(ctx context.Context, req ProxyRequest, w *StreamWriter) {
			_ = w.WriteChunk([]byte("data: hello\n\n"))
			_ = w.End(&Usage{OutputTokens: 1})
		},
	)
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}
	st, err := cli.OpenStream(ctx, ProxyRequest{Ingress: "anthropic", Body: []byte(`{}`), Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Status() != 200 || st.StreamID() != "s1" {
		t.Fatalf("open: status=%d id=%s", st.Status(), st.StreamID())
	}
	// First event must be chunk (open already completed).
	ev, err := st.Recv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != "chunk" || string(ev.Chunk.Data) != "data: hello\n\n" {
		t.Fatalf("first event: %+v", ev)
	}
	ev, err = st.Recv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != "end" {
		t.Fatalf("expected end, got %s", ev.Kind)
	}
}

func TestStreamOpenNon2xxNoStreamID(t *testing.T) {
	cli, srv := startPair(t, "tok")
	srv.RegisterProxyStream(
		func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, error) {
			return nil, &ProxyResponse{
				Status: 401,
				Body:   []byte(`{"error":"no creds"}`),
			}, nil
		},
		func(ctx context.Context, req ProxyRequest, w *StreamWriter) {
			t.Error("run must not be called on non-2xx open")
		},
	)
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}
	st, err := cli.OpenStream(ctx, ProxyRequest{Ingress: "anthropic", Body: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if st.Status() != 401 {
		t.Fatalf("status=%d", st.Status())
	}
	if st.StreamID() != "" {
		t.Fatalf("stream id should be empty on error open")
	}
}

func TestFrameTooLarge(t *testing.T) {
	err := WriteFrame(&bytes.Buffer{}, make([]byte, 100), 50)
	if err == nil {
		t.Fatal("expected frame too large")
	}
}

func TestFilterHeaders(t *testing.T) {
	in := map[string]string{
		"Authorization":            "Bearer x",
		"Content-Type":             "application/json",
		"anthropic-version":        "2023-06-01",
		"x-claude-code-session-id": "abc",
		"X-Api-Key":                "secret",
		"Random":                   "nope",
	}
	out := FilterHeaders(in)
	if _, ok := out["authorization"]; ok {
		t.Fatal("auth should be stripped")
	}
	if out["content-type"] != "application/json" {
		t.Fatal("content-type")
	}
	if out["x-claude-code-session-id"] != "abc" {
		t.Fatal("session")
	}
	out = EnsureAnthropicVersion("anthropic", out)
	if out["anthropic-version"] != "2023-06-01" {
		t.Fatal("version")
	}
}

func TestChooseBodyEncoding(t *testing.T) {
	enc, raw, inline := ChooseBodyEncoding([]byte("hi"), 10)
	if enc != BodyEncodingInline || raw != nil || string(inline) != "hi" {
		t.Fatalf("inline: %s %v %v", enc, raw, inline)
	}
	big := make([]byte, 20)
	enc, raw, inline = ChooseBodyEncoding(big, 10)
	if enc != BodyEncodingRawFollowup || len(raw) != 20 || inline != nil {
		t.Fatalf("raw: %s %d %v", enc, len(raw), inline)
	}
}

func TestWriteMessageAtomicWithNotify(t *testing.T) {
	// Demux integrity: concurrent Complete(raw) + Ping must not protocol-error.
	cli, srv := startPair(t, "tok")
	srv.RegisterComplete(func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
		time.Sleep(5 * time.Millisecond)
		return &ProxyResponse{Status: 200, Body: req.Body}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}
	body := bytes.Repeat([]byte("z"), 400*1024)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			resp, err := cli.Complete(ctx, ProxyRequest{Body: body})
			if err != nil {
				t.Error(err)
				return
			}
			if !bytes.Equal(resp.Body, body) {
				t.Error("mismatch")
			}
		}()
		go func() {
			defer wg.Done()
			if err := cli.Ping(ctx); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestNewEndpointAndToken(t *testing.T) {
	ep, err := NewEndpoint("grokbuild", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ep == "" {
		t.Fatal("empty endpoint")
	}
	tok, err := NewHostToken()
	if err != nil || len(tok) != 64 {
		t.Fatalf("token: %v %s", err, tok)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/plugin.manifest.json"
	m := Manifest{
		PluginID:         "grokbuild",
		Name:             "GrokBuild",
		Version:          "0.1.0",
		Capabilities:     []string{"proxy.stream"},
		SupportedIngress: []string{"anthropic"},
	}
	b, _ := json.Marshal(m)
	if err := writeFile(path, b); err != nil {
		t.Fatal(err)
	}
	got, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.PluginID != "grokbuild" {
		t.Fatal(got.PluginID)
	}
}

func writeFile(path string, b []byte) error {
	return writeFileOS(path, b)
}
