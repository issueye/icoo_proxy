// plugin-grokbuild is an icoo process plugin that proxies SuperGrok / Grok Build
// Responses traffic and contributes a desktop extension UI for credentials.
//
// Bootstrap uses the pluginipc Server SDK:
//
//	custom flags → RunPlugin(Listen → AfterListen → Accept → PrepareHandshake → Serve)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/issueye/icoo_proxy/common/pluginipc"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/admin"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/netx"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/oauth"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/proxyhandler"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/upstream"
)

const pluginVersion = "0.3.2"

func main() {
	// Ensure plugin.log (host redirects stdout/stderr) gets timestamps early.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stderr)

	// Custom flags must be registered before RunPlugin → ParsePluginFlags.
	httpProxyFlag := flag.String("http-proxy", "", "outbound HTTP/SOCKS5 proxy (overrides settings.json when set)")

	// Shared deps written in AfterListen, read by handlers / handshake / health.
	var (
		adminSrv   *admin.Server
		adminURL   string
		adminToken string
		st         *store.Store
		refresher  *oauth.Refresher
		up         *upstream.Client
		handler    *proxyhandler.Handler
	)

	meta := pluginipc.PluginMeta{
		ID:           "grokbuild",
		Version:      pluginVersion,
		UpstreamKind: "grok-build-responses",
		Capabilities: []string{
			pluginipc.CapProxyComplete,
			pluginipc.CapProxyStream,
			pluginipc.CapModelsList,
			pluginipc.CapHealth,
			pluginipc.CapUI,
			"oauth.device",
			"billing",
			"oauth.prerefresh",
			"settings.proxy",
		},
		SupportedIngress: []string{"anthropic", "openai-responses", "openai-chat"},
		// AdminBaseURL / UIPages filled by PrepareHandshake after admin.Start.
	}

	err := pluginipc.RunPlugin(meta, pluginipc.Handlers{
		Complete: func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
			if handler == nil {
				return pluginipc.BadGateway("plugin not ready"), nil
			}
			return handler.Complete(ctx, req)
		},
		Stream: func(ctx context.Context, req pluginipc.ProxyRequest) (
			*pluginipc.StreamOpenResult, *pluginipc.ProxyResponse,
			func(context.Context, *pluginipc.StreamWriter), error,
		) {
			if handler == nil {
				return nil, pluginipc.BadGateway("plugin not ready"), nil, nil
			}
			return handler.PrepareStream(ctx, req)
		},
		ModelsList: func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
			if st == nil || refresher == nil || up == nil {
				return &pluginipc.ModelsListResult{Models: defaultModels()}, nil
			}
			return makeModelsList(st, refresher, up)(ctx)
		},
	}, pluginipc.PluginHooks{
		AfterListen: func(ctx context.Context, env pluginipc.PluginEnv) error {
			// Listen already open; keep this path under host dial timeout (~30s).
			if env.PluginID == "" {
				env.PluginID = "grokbuild"
			}
			if err := os.MkdirAll(env.DataDir, 0o700); err != nil {
				return err
			}
			log.Printf("plugin-grokbuild after_listen ipc=%s data_dir=%s", env.Endpoint, env.DataDir)

			st = store.New(env.DataDir)
			settingsStore := store.NewSettingsStore(env.DataDir)
			settings, _ := settingsStore.Load()

			// Priority: --http-proxy flag > ICOO_PLUGIN_HTTP_PROXY env > settings.json > process env proxies.
			httpProxy := firstNonEmpty(
				*httpProxyFlag,
				os.Getenv("ICOO_PLUGIN_HTTP_PROXY"),
				settings.HTTPProxy,
			)

			up = upstream.New(os.Getenv("GROK_UPSTREAM_BASE"))
			oauthClient := oauth.NewClient()

			applyProxy := func(proxyURL string) error {
				oauthHTTP, err := netx.NewHTTPClient(proxyURL, oauth.DefaultHTTPTimeout)
				if err != nil {
					return err
				}
				oauthClient.SetHTTPClient(oauthHTTP)
				// Upstream streams need no overall client timeout.
				upHTTP, err := netx.NewHTTPClient(proxyURL, 0)
				if err != nil {
					return err
				}
				up.SetHTTPClient(upHTTP)
				log.Printf("outbound proxy applied: %s", netx.EffectiveProxyDescription(proxyURL))
				return nil
			}
			if err := applyProxy(httpProxy); err != nil {
				log.Printf("proxy init warning: %v (falling back to env/direct)", err)
				_ = applyProxy("")
				httpProxy = ""
			}

			// Bound discovery so a blocked network cannot stall Accept/handshake.
			discCtx, discCancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := oauthClient.Discover(discCtx); err != nil {
				log.Printf("oauth discover skipped/failed (using defaults): %v", err)
			}
			discCancel()

			refresher = oauth.NewRefresher(oauthClient)
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
			handler = proxyhandler.New(st, up, refresher)

			// Background pre-refresh keeps access tokens warm before expiry.
			// ctx is cancelled on host shutdown / signal via RunPlugin.
			go oauth.StartPreRefresh(ctx, st, refresher, oauth.PreRefreshConfig{})

			token, err := pluginipc.NewHostToken()
			if err != nil {
				return fmt.Errorf("admin token: %w", err)
			}
			adminToken = token
			adminSrv, adminURL, err = admin.Start(admin.StartOpts{
				Store:      st,
				Settings:   settingsStore,
				Sessions:   sessions,
				Upstream:   up,
				Refresh:    refresher,
				ApplyProxy: applyProxy,
				HTTPProxy:  httpProxy,
				AdminToken: adminToken,
			})
			if err != nil {
				return fmt.Errorf("admin ui: %w", err)
			}
			log.Printf("plugin-grokbuild admin=%s proxy=%s; waiting for host dial", adminURL, netx.EffectiveProxyDescription(httpProxy))
			return nil
		},
		PrepareHandshake: func(env pluginipc.PluginEnv, m pluginipc.PluginMeta) (pluginipc.PluginMeta, error) {
			if adminURL == "" {
				return m, fmt.Errorf("admin UI not started")
			}
			m.AdminBaseURL = adminURL
			m.AdminToken = adminToken
			m.UIPages = []pluginipc.UIPage{
				{
					ID:          "credentials",
					Title:       "Grok 凭据",
					Path:        "/",
					Icon:        "key",
					Group:       "插件",
					Description: "Device Login / 凭据池 / 代理 / 额度",
				},
			}
			return m, nil
		},
		OnShutdown: func() {
			// RunPlugin already closes the IPC conn; only tear down admin here.
			if adminSrv != nil {
				_ = adminSrv.Close()
			}
		},
		Health: func(ctx context.Context) (*pluginipc.HealthResult, error) {
			if st == nil {
				return &pluginipc.HealthResult{OK: true, Status: "starting"}, nil
			}
			return makeHealth(st)(ctx)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func defaultModels() []pluginipc.ModelInfo {
	return []pluginipc.ModelInfo{
		{ID: "grok-4", DisplayName: "Grok 4"},
		{ID: "grok-4.5", DisplayName: "Grok 4.5"},
		{ID: "grok-build-0.1", DisplayName: "Grok Build"},
	}
}

func makeModelsList(st *store.Store, refresher *oauth.Refresher, up *upstream.Client) func(context.Context) (*pluginipc.ModelsListResult, error) {
	return func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
		models := defaultModels()
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
	}
}

func makeHealth(st *store.Store) func(context.Context) (*pluginipc.HealthResult, error) {
	return func(ctx context.Context) (*pluginipc.HealthResult, error) {
		list, err := st.List()
		if err != nil {
			return &pluginipc.HealthResult{
				OK:     false,
				Status: "unhealthy",
				Details: map[string]any{
					"error": err.Error(),
				},
			}, nil
		}
		enabled := 0
		withToken := 0
		for _, c := range list {
			if c.Enabled {
				enabled++
				if strings.TrimSpace(c.AccessToken) != "" || strings.TrimSpace(c.RefreshToken) != "" {
					withToken++
				}
			}
		}
		status := "healthy"
		ok := true
		if enabled == 0 || withToken == 0 {
			status = "degraded"
		}
		return &pluginipc.HealthResult{
			OK:     ok,
			Status: status,
			Details: map[string]any{
				"credentials_total":   len(list),
				"credentials_enabled": enabled,
				"credentials_ready":   withToken,
			},
		}, nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
