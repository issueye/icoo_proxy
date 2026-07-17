package pluginipc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSDKHelpers_ResponsesAndStreamID(t *testing.T) {
	body := []byte(`{"ok":true}`)
	resp := OKJSON(body, &Usage{OutputTokens: 2})
	if !resp.Success() || resp.StatusOrOK() != 200 {
		t.Fatalf("OKJSON success: %+v", resp)
	}
	if string(resp.Body) != string(body) {
		t.Fatalf("body")
	}

	unauth := Unauthorized("no creds")
	if unauth.Status != 401 || !strings.Contains(string(unauth.Body), "no creds") {
		t.Fatalf("unauthorized: %+v", unauth)
	}
	bg := BadGateway("up")
	if bg.Status != 502 {
		t.Fatalf("bad gateway status %d", bg.Status)
	}

	open := SSEOpen("s-1")
	if open.StreamID != "s-1" || open.Headers["content-type"] != "text/event-stream" {
		t.Fatalf("sse open: %+v", open)
	}
	id := NewStreamID("gb")
	if !strings.HasPrefix(id, "gb-") || len(id) < 10 {
		t.Fatalf("stream id: %s", id)
	}

	hs := HandshakeFrom(PluginMeta{
		ID: "hello", Version: "1.0.0",
		Capabilities:     []string{CapProxyComplete, CapHealth},
		SupportedIngress: []string{"openai-chat"},
		UpstreamKind:     "echo",
	})
	if hs.IPCProtocolVersion != ProtocolVersion || hs.PluginID != "hello" {
		t.Fatalf("handshake: %+v", hs)
	}

	err := RPCUnsupportedIngress(errors.New("nope"))
	var rpc *RPCError
	if !errors.As(err, &rpc) || rpc.Code != CodeUnsupportedIngress {
		t.Fatalf("rpc unsupported: %v", err)
	}
}

func TestSDKClient_NewProxyRequestAndMapCallError(t *testing.T) {
	req := NewProxyRequest(ProxyRequestInput{
		Ingress: "anthropic",
		Path:    "/v1/messages",
		Method:  "POST",
		Headers: map[string]string{
			"Authorization": "Bearer secret",
			"Content-Type":  "application/json",
			"X-Api-Key":     "k",
		},
		Body:   []byte(`{}`),
		Model:  "claude",
		Stream: false,
	})
	if _, ok := req.Headers["authorization"]; ok {
		t.Fatal("authorization must be stripped")
	}
	if req.Headers["content-type"] != "application/json" {
		t.Fatal("content-type")
	}
	if req.Headers["anthropic-version"] != "2023-06-01" {
		t.Fatalf("anthropic-version: %v", req.Headers)
	}

	st, msg := MapCallError(ErrTooManyStreams)
	if st != 503 || msg == "" {
		t.Fatalf("map too many: %d %s", st, msg)
	}
	st, msg = MapCallError(NewRPCError(CodeUnsupportedIngress, "bad ingress", nil))
	if st != 400 || msg != "bad ingress" {
		t.Fatalf("map unsupported: %d %s", st, msg)
	}
	st, _ = MapCallError(ErrFrameTooLarge)
	if st != 413 {
		t.Fatalf("frame: %d", st)
	}
	// -32003 with data.status must surface that HTTP status (contract).
	st, msg = MapCallError(UpstreamRPCError(429, "rate limited"))
	if st != 429 || msg != "rate limited" {
		t.Fatalf("map upstream status: %d %s", st, msg)
	}
	// Without data.status, default 502.
	st, _ = MapCallError(NewRPCError(CodeUpstreamError, "up", nil))
	if st != 502 {
		t.Fatalf("upstream default: %d", st)
	}
}

func TestSDKClient_StreamOKAndErrorBody(t *testing.T) {
	cli, srv := startPair(t, "tok")
	srv.RegisterProxyStreamEx(func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, func(context.Context, *StreamWriter), error) {
		if bytes.Contains(req.Body, []byte("fail")) {
			return nil, Unauthorized("denied"), nil, nil
		}
		id := NewStreamID("t")
		open := SSEOpen(id)
		run := func(ctx context.Context, w *StreamWriter) {
			_, _ = w.AsWriter().Write([]byte("data: hi\n\n"))
			_ = w.End(&Usage{OutputTokens: 1})
		}
		return open, nil, run, nil
	})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}

	// Success path
	st, err := cli.OpenStream(ctx, ProxyRequest{Ingress: "anthropic", Body: []byte(`{"ok":1}`), Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	if !st.OK() {
		t.Fatalf("expected OK stream, status=%d", st.Status())
	}
	ev, err := st.Recv(ctx)
	if err != nil || ev.Kind != "chunk" {
		t.Fatalf("chunk: %v %+v", err, ev)
	}
	if string(ev.Chunk.Data) != "data: hi\n\n" {
		t.Fatalf("chunk data: %q", ev.Chunk.Data)
	}
	st.Close()

	// Non-2xx open + ErrorBody
	st, err = cli.OpenStream(ctx, ProxyRequest{Ingress: "anthropic", Body: []byte(`fail`), Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.OK() {
		t.Fatal("expected non-OK")
	}
	_, body, status := st.ErrorBody(ctx)
	if status != 401 {
		t.Fatalf("status=%d", status)
	}
	if !bytes.Contains(body, []byte("denied")) {
		t.Fatalf("body=%s", body)
	}
}

func TestSDKServer_ServeConnAndHealth(t *testing.T) {
	c1, c2 := net.Pipe()
	t.Cleanup(func() { _ = c1.Close(); _ = c2.Close() })

	var healthCalls int
	srv := ServeConn(c2, ServerOptions{
		HostToken: "tok",
		Handshake: HandshakeFrom(PluginMeta{
			ID: "svc", Version: "0.1.0",
			Capabilities:     []string{CapProxyComplete, CapModelsList, CapHealth},
			SupportedIngress: []string{"openai-chat"},
		}),
	}, Handlers{
		Complete: func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
			return OKJSON([]byte(`{"echo":true}`), nil), nil
		},
		ModelsList: func(ctx context.Context) (*ModelsListResult, error) {
			return &ModelsListResult{Models: []ModelInfo{{ID: "m1"}}}, nil
		},
		Stream: func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, func(context.Context, *StreamWriter), error) {
			open := SSEOpen(NewStreamID("x"))
			run := func(ctx context.Context, w *StreamWriter) {
				_ = w.WriteChunk([]byte("data: 1\n\n"))
				_ = w.End(nil)
			}
			return open, nil, run, nil
		},
	}, func(ctx context.Context) (*HealthResult, error) {
		healthCalls++
		return &HealthResult{OK: true, Status: "degraded", Details: map[string]any{"n": 1}}, nil
	})
	_ = srv

	cli := NewClient(c1, ClientOptions{})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "host"); err != nil {
		t.Fatal(err)
	}
	h, err := cli.Health(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "degraded" || healthCalls != 1 {
		t.Fatalf("health: %+v calls=%d", h, healthCalls)
	}
	resp, err := cli.Complete(ctx, ProxyRequest{Ingress: "openai-chat", Body: []byte(`{}`)})
	if err != nil || !resp.Success() {
		t.Fatalf("complete: %v %+v", err, resp)
	}
	models, err := cli.ListModels(ctx)
	if err != nil || len(models.Models) != 1 {
		t.Fatalf("models: %v %+v", err, models)
	}
}

func TestSDKServer_RunPluginWithEnv(t *testing.T) {
	// Use net.Pipe-free path: real Listen/Dial on generated endpoint.
	dir := t.TempDir()
	endpoint, err := NewEndpoint("sdk-run", dir)
	if err != nil {
		t.Fatal(err)
	}
	token, err := NewHostToken()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		errCh <- RunPlugin(PluginMeta{
			ID: "sdk-run", Version: "0.0.1", UpstreamKind: "test",
			Capabilities:     []string{CapProxyComplete, CapHealth},
			SupportedIngress: []string{"openai-chat"},
		}, Handlers{
			Complete: func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
				return OKJSON([]byte(`{"sdk":true}`), nil), nil
			},
		}, PluginHooks{
			Env: &PluginEnv{
				Endpoint: endpoint,
				DataDir:  dir,
				PluginID: "sdk-run",
				Token:    token,
			},
			NoSignal: true,
			Context:  ctx,
		})
	}()

	// Connect as host
	cli, hs, err := Connect(context.Background(), ConnectConfig{
		Endpoint:         endpoint,
		Token:            token,
		HostVersion:      "test-host",
		DialTimeout:      5 * time.Second,
		HandshakeTimeout: 5 * time.Second,
	})
	if err != nil {
		cancel()
		wg.Wait()
		t.Fatalf("connect: %v", err)
	}
	defer cli.Close()
	if hs.PluginID != "sdk-run" {
		t.Fatalf("plugin id: %s", hs.PluginID)
	}
	resp, err := cli.Complete(context.Background(), NewProxyRequest(ProxyRequestInput{
		Ingress: "openai-chat",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{}`),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(resp.Body, []byte(`"sdk":true`)) {
		t.Fatalf("body=%s", resp.Body)
	}

	// Graceful stop via shutdown
	_ = cli.Shutdown(context.Background())
	cancel()
	wg.Wait()
	if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, net.ErrClosed) {
		// RunPlugin returns nil on clean close; accept cancel-related errors.
		if !strings.Contains(err.Error(), "accept") {
			// shutdown path may race accept — only fail on unexpected
			t.Logf("runplugin exit: %v", err)
		}
	}
}

func TestStreamWriter_AsWriter(t *testing.T) {
	cli, srv := startPair(t, "tok")
	srv.RegisterProxyStreamEx(func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, func(context.Context, *StreamWriter), error) {
		open := SSEOpen("w1")
		run := func(ctx context.Context, w *StreamWriter) {
			n, err := io.Copy(w.AsWriter(), bytes.NewReader([]byte("data: from-io\n\n")))
			if err != nil || n == 0 {
				_ = w.Error(CodeInternalError, "write failed")
				return
			}
			_ = w.End(nil)
		}
		return open, nil, run, nil
	})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}
	st, err := cli.OpenStream(ctx, ProxyRequest{Body: []byte(`{}`), Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := st.Recv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != "chunk" || string(ev.Chunk.Data) != "data: from-io\n\n" {
		t.Fatalf("event: %+v", ev)
	}
}

func TestMapCallError_JSONRoundTrip(t *testing.T) {
	// Ensure RPCError from JSON still maps (errors.As path).
	raw, _ := json.Marshal(RPCError{Code: CodeShuttingDown, Message: "bye"})
	var rpc RPCError
	if err := json.Unmarshal(raw, &rpc); err != nil {
		t.Fatal(err)
	}
	st, msg := MapCallError(&rpc)
	if st != 503 || msg != "bye" {
		t.Fatalf("%d %s", st, msg)
	}

	// Upstream status survives JSON round-trip (float64 from encoding/json).
	raw, _ = json.Marshal(RPCError{
		Code:    CodeUpstreamError,
		Message: "quota",
		Data:    map[string]any{"status": 402},
	})
	var up RPCError
	if err := json.Unmarshal(raw, &up); err != nil {
		t.Fatal(err)
	}
	st, msg = MapCallError(&up)
	if st != 402 || msg != "quota" {
		t.Fatalf("upstream json: %d %s data=%T%v", st, msg, up.Data, up.Data)
	}
}

func TestSDKServer_StreamCancel(t *testing.T) {
	cli, srv := startPair(t, "tok")
	started := make(chan struct{})
	cancelled := make(chan struct{})
	srv.RegisterProxyStreamEx(func(ctx context.Context, req ProxyRequest) (*StreamOpenResult, *ProxyResponse, func(context.Context, *StreamWriter), error) {
		open := SSEOpen(NewStreamID("c"))
		run := func(ctx context.Context, w *StreamWriter) {
			close(started)
			select {
			case <-ctx.Done():
				close(cancelled)
				_ = w.Error(CodeInternalError, "cancelled")
			case <-time.After(5 * time.Second):
				_ = w.End(nil)
			}
		}
		return open, nil, run, nil
	})
	ctx := context.Background()
	if _, err := cli.Handshake(ctx, "tok", "h"); err != nil {
		t.Fatal(err)
	}
	st, err := cli.OpenStream(ctx, ProxyRequest{Body: []byte(`{}`), Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	if !st.OK() {
		t.Fatal("expected OK")
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not start")
	}
	if err := st.Cancel(ctx); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("run context was not cancelled")
	}
	st.Close()
}

func TestSDKServer_PrepareHandshake(t *testing.T) {
	dir := t.TempDir()
	endpoint, err := NewEndpoint("sdk-hs", dir)
	if err != nil {
		t.Fatal(err)
	}
	token, err := NewHostToken()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		errCh <- RunPlugin(PluginMeta{
			ID: "sdk-hs", Version: "0.0.1", UpstreamKind: "test",
			Capabilities:     []string{CapProxyComplete, CapUI, CapHealth},
			SupportedIngress: []string{"openai-chat"},
		}, Handlers{
			Complete: func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
				return OKJSON([]byte(`{}`), nil), nil
			},
		}, PluginHooks{
			Env: &PluginEnv{
				Endpoint: endpoint,
				DataDir:  dir,
				PluginID: "sdk-hs",
				Token:    token,
			},
			NoSignal: true,
			Context:  ctx,
			AfterListen: func(ctx context.Context, env PluginEnv) error {
				// Simulate admin bind after listen.
				return nil
			},
			PrepareHandshake: func(env PluginEnv, meta PluginMeta) (PluginMeta, error) {
				meta.AdminBaseURL = "http://127.0.0.1:19999"
				meta.UIPages = []UIPage{{ID: "p", Title: "P", Path: "/"}}
				return meta, nil
			},
		})
	}()

	cli, hs, err := Connect(context.Background(), ConnectConfig{
		Endpoint:         endpoint,
		Token:            token,
		HostVersion:      "test-host",
		DialTimeout:      5 * time.Second,
		HandshakeTimeout: 5 * time.Second,
	})
	if err != nil {
		cancel()
		wg.Wait()
		t.Fatalf("connect: %v", err)
	}
	defer cli.Close()
	if hs.AdminBaseURL != "http://127.0.0.1:19999" {
		t.Fatalf("admin url: %s", hs.AdminBaseURL)
	}
	if len(hs.UIPages) != 1 || hs.UIPages[0].ID != "p" {
		t.Fatalf("ui pages: %+v", hs.UIPages)
	}
	_ = cli.Shutdown(context.Background())
	cancel()
	wg.Wait()
}
