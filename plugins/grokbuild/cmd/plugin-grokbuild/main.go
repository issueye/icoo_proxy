// plugin-grokbuild is an icoo process plugin that proxies SuperGrok / Grok Build
// Responses traffic and contributes a desktop extension UI for credentials.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/issueye/icoo_proxy/common/pluginipc"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/admin"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/oauth"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/proxyhandler"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/upstream"
)

func main() {
	// Ensure plugin.log (host redirects stdout/stderr) gets timestamps early.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stderr)

	endpoint := flag.String("endpoint", os.Getenv("ICOO_PLUGIN_ENDPOINT"), "IPC endpoint")
	dataDir := flag.String("data-dir", ".", "plugin data directory")
	pluginID := flag.String("plugin-id", "grokbuild", "plugin id")
	flag.Parse()

	token := os.Getenv("ICOO_PLUGIN_TOKEN")
	if *endpoint == "" || token == "" {
		fmt.Fprintln(os.Stderr, "endpoint and ICOO_PLUGIN_TOKEN are required")
		os.Exit(2)
	}
	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// CRITICAL: open the IPC listener BEFORE any network/admin init.
	// The host dials with a short timeout; OAuth discovery / admin bind must not
	// delay Listen/Accept or dial fails with "open \\.\pipe\icoo-plugin-...".
	log.Printf("plugin-grokbuild starting ipc=%s data_dir=%s", *endpoint, *dataDir)
	ln, err := pluginipc.Listen(ctx, pluginipc.ListenConfig{Endpoint: *endpoint})
	if err != nil {
		log.Fatal("listen:", err)
	}
	defer ln.Close()
	log.Printf("plugin-grokbuild listening on %s", *endpoint)

	st := store.New(*dataDir)
	up := upstream.New(os.Getenv("GROK_UPSTREAM_BASE"))
	oauthClient := oauth.NewClient()
	// Bound discovery so a blocked network cannot stall Accept/handshake.
	discCtx, discCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := oauthClient.Discover(discCtx); err != nil {
		log.Printf("oauth discover skipped/failed (using defaults): %v", err)
	}
	discCancel()

	refresher := oauth.NewRefresher(oauthClient)
	sessions := oauth.NewSessionManager(oauthClient, func(sessionID string, ts oauth.TokenSet) (string, error) {
		id := "oauth-" + sessionID
		cred := store.Credential{
			ID:           id,
			Label:        "device-login",
			AccessToken:  ts.AccessToken,
			RefreshToken: ts.RefreshToken,
			ExpiresAt:    ts.ExpiresAt,
			Enabled:      true,
			Priority:     10,
		}
		if err := st.Upsert(cred); err != nil {
			return "", err
		}
		return id, nil
	})
	handler := proxyhandler.New(st, up, refresher)

	// Background pre-refresh keeps access tokens warm before expiry.
	go oauth.StartPreRefresh(ctx, st, refresher, oauth.PreRefreshConfig{})

	adminSrv, adminURL, err := admin.Start(st, sessions, up, refresher)
	if err != nil {
		log.Fatal("admin ui:", err)
	}
	defer adminSrv.Close()

	log.Printf("plugin-grokbuild admin=%s; waiting for host dial", adminURL)
	conn, err := ln.Accept()
	if err != nil {
		log.Fatal("accept:", err)
	}
	log.Printf("plugin-grokbuild host connected")

	srv := pluginipc.NewServer(conn, pluginipc.ServerOptions{
		HostToken: token,
		Handshake: pluginipc.HandshakeResult{
			PluginID:         *pluginID,
			PluginVersion:    "0.3.1",
			Capabilities:     []string{"proxy.complete", "proxy.stream", "models.list", "health", "ui", "oauth.device", "billing", "oauth.prerefresh"},
			SupportedIngress: []string{"anthropic", "openai-responses", "openai-chat"},
			UpstreamKind:     "grok-build-responses",
			AdminBaseURL:     adminURL,
			UIPages: []pluginipc.UIPage{
				{
					ID:          "credentials",
					Title:       "Grok 凭据",
					Path:        "/",
					Icon:        "key",
					Group:       "插件",
					Description: "Device Login / 凭据池 / 额度",
				},
			},
		},
		OnShutdown: func() {
			stop()
			_ = conn.Close()
			_ = adminSrv.Close()
		},
	})

	srv.RegisterComplete(handler.Complete)

	var pendingRuns sync.Map
	srv.RegisterProxyStream(
		func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.StreamOpenResult, *pluginipc.ProxyResponse, error) {
			open, errResp, run, err := handler.PrepareStream(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			if open != nil && run != nil {
				pendingRuns.Store(open.StreamID, run)
			}
			return open, errResp, nil
		},
		func(ctx context.Context, req pluginipc.ProxyRequest, w *pluginipc.StreamWriter) {
			v, ok := pendingRuns.LoadAndDelete(w.StreamID())
			if !ok {
				_ = w.Error(pluginipc.CodeInternalError, "missing stream runner")
				return
			}
			v.(func(*pluginipc.StreamWriter))(w)
		},
	)

	srv.RegisterModelsList(func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
		models := []pluginipc.ModelInfo{
			{ID: "grok-4", DisplayName: "Grok 4"},
			{ID: "grok-4.5", DisplayName: "Grok 4.5"},
			{ID: "grok-build-0.1", DisplayName: "Grok Build"},
		}
		if cred, err := st.Pick(); err == nil {
			token := cred.AccessToken
			if ts, err := refresher.EnsureAccess(ctx, cred.ID, cred.AccessToken, cred.RefreshToken, cred.ExpiresAt); err == nil {
				token = ts.AccessToken
				_ = st.ApplyTokens(cred.ID, ts.AccessToken, ts.RefreshToken, ts.ExpiresAt)
			}
			status, raw, listErr := up.ListModels(ctx, token)
			if listErr == nil && status == 200 {
				var payload struct {
					Data []struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if json.Unmarshal(raw, &payload) == nil && len(payload.Data) > 0 {
					models = models[:0]
					for _, d := range payload.Data {
						if d.ID != "" {
							models = append(models, pluginipc.ModelInfo{ID: d.ID})
						}
					}
				}
			}
		}
		return &pluginipc.ModelsListResult{Models: models}, nil
	})

	log.Printf("plugin-grokbuild v0.3.1 ready ipc=%s admin=%s", *endpoint, adminURL)
	<-ctx.Done()
	_ = srv.Close()
	time.Sleep(50 * time.Millisecond)
}
