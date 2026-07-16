// mockplugin is a process plugin used for pluginhost integration tests.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/issueye/icoo_proxy/common/pluginipc"
)

func main() {
	endpoint := flag.String("endpoint", os.Getenv("ICOO_PLUGIN_ENDPOINT"), "IPC endpoint")
	dataDir := flag.String("data-dir", "", "plugin data dir")
	pluginID := flag.String("plugin-id", "mock", "plugin id")
	flag.Parse()
	_ = dataDir

	token := os.Getenv("ICOO_PLUGIN_TOKEN")
	if *endpoint == "" || token == "" {
		fmt.Fprintln(os.Stderr, "endpoint and ICOO_PLUGIN_TOKEN required")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := pluginipc.Listen(ctx, pluginipc.ListenConfig{Endpoint: *endpoint})
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		os.Exit(1)
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		fmt.Fprintln(os.Stderr, "accept:", err)
		os.Exit(1)
	}

	srv := pluginipc.NewServer(conn, pluginipc.ServerOptions{
		HostToken: token,
		Handshake: pluginipc.HandshakeResult{
			PluginID:         *pluginID,
			PluginVersion:    "0.0.1-mock",
			Capabilities:     []string{"proxy.complete", "proxy.stream", "models.list", "health"},
			SupportedIngress: []string{"anthropic", "openai-chat", "openai-responses"},
			UpstreamKind:     "mock",
		},
		OnShutdown: func() {
			stop()
			_ = conn.Close()
		},
	})
	srv.RegisterComplete(func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
		return &pluginipc.ProxyResponse{
			Status:  200,
			Headers: map[string]string{"content-type": "application/json"},
			Body:    append([]byte(`{"echo":true,"bytes":`), append([]byte(fmt.Sprintf("%d", len(req.Body))), '}')...),
		}, nil
	})
	srv.RegisterModelsList(func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
		return &pluginipc.ModelsListResult{Models: []pluginipc.ModelInfo{{ID: "mock-model"}}}, nil
	})
	srv.RegisterProxyStream(
		func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.StreamOpenResult, *pluginipc.ProxyResponse, error) {
			return &pluginipc.StreamOpenResult{
				StreamID: "mock-stream-1",
				Status:   200,
				Headers:  map[string]string{"content-type": "text/event-stream"},
			}, nil, nil
		},
		func(ctx context.Context, req pluginipc.ProxyRequest, w *pluginipc.StreamWriter) {
			_ = w.WriteChunk([]byte("data: {\"type\":\"message\"}\n\n"))
			_ = w.End(&pluginipc.Usage{OutputTokens: 1})
		},
	)

	<-ctx.Done()
	_ = srv.Close()
}
