package admin

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/netx"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/oauth"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/upstream"
)

// ProxyApplier updates live HTTP clients when the user changes proxy settings.
type ProxyApplier func(proxyURL string) error

// Server is a loopback-only admin UI + JSON API for credentials / OAuth / billing / proxy.
type Server struct {
	store      *store.Store
	settings   *store.SettingsStore
	sessions   *oauth.SessionManager
	upstream   *upstream.Client
	refresh    *oauth.Refresher
	applyProxy ProxyApplier
	mu         sync.RWMutex
	httpProxy  string // last applied explicit proxy (may be empty = env/direct)
	ln         net.Listener
	srv        *http.Server
}

type StartOpts struct {
	Store      *store.Store
	Settings   *store.SettingsStore
	Sessions   *oauth.SessionManager
	Upstream   *upstream.Client
	Refresh    *oauth.Refresher
	ApplyProxy ProxyApplier
	HTTPProxy  string // initial proxy already applied by main
}

func Start(opts StartOpts) (*Server, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	s := &Server{
		store:      opts.Store,
		settings:   opts.Settings,
		sessions:   opts.Sessions,
		upstream:   opts.Upstream,
		refresh:    opts.Refresh,
		applyProxy: opts.ApplyProxy,
		httpProxy:  strings.TrimSpace(opts.HTTPProxy),
		ln:         ln,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/credentials", s.handleCredentials)
	mux.HandleFunc("/api/credentials/import", s.handleImport)
	mux.HandleFunc("/api/oauth/device/start", s.handleDeviceStart)
	mux.HandleFunc("/api/oauth/device/status", s.handleDeviceStatus)
	mux.HandleFunc("/api/billing", s.handleBilling)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/settings/proxy-test", s.handleProxyTest)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		list, _ := s.store.List()
		enabled := 0
		for _, c := range list {
			if c.Enabled && strings.TrimSpace(c.AccessToken) != "" {
				enabled++
			}
		}
		s.mu.RLock()
		proxy := s.httpProxy
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                 true,
			"plugin":             "grokbuild",
			"credentials_total":  len(list),
			"credentials_active": enabled,
			"http_proxy":         proxy,
			"http_proxy_effective": netx.EffectiveProxyDescription(proxy),
		})
	})
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return s, "http://" + ln.Addr().String(), nil
}

func (s *Server) Close() error {
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var cfg store.Settings
		if s.settings != nil {
			if loaded, err := s.settings.Load(); err == nil {
				cfg = loaded
			}
		}
		s.mu.RLock()
		live := s.httpProxy
		s.mu.RUnlock()
		// Prefer live applied value if settings file empty (env/flag seed).
		if strings.TrimSpace(cfg.HTTPProxy) == "" {
			cfg.HTTPProxy = live
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"http_proxy":           cfg.HTTPProxy,
			"http_proxy_effective": netx.EffectiveProxyDescription(cfg.HTTPProxy),
			"examples": []string{
				"http://127.0.0.1:7890",
				"socks5://127.0.0.1:7891",
				"http://user:pass@127.0.0.1:7890",
			},
		})
	case http.MethodPut, http.MethodPost:
		var body struct {
			HTTPProxy string `json:"http_proxy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		proxyURL := strings.TrimSpace(body.HTTPProxy)
		// Validate by building transport (empty is allowed = env/direct).
		if proxyURL != "" {
			if _, err := netx.NewTransport(proxyURL); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if s.applyProxy != nil {
			if err := s.applyProxy(proxyURL); err != nil {
				http.Error(w, "apply proxy failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if s.settings != nil {
			if err := s.settings.Save(store.Settings{HTTPProxy: proxyURL}); err != nil {
				http.Error(w, "save settings failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		s.mu.Lock()
		s.httpProxy = proxyURL
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"http_proxy":           proxyURL,
			"http_proxy_effective": netx.EffectiveProxyDescription(proxyURL),
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProxyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		HTTPProxy string `json:"http_proxy"`
		URL       string `json:"url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	proxyURL := strings.TrimSpace(body.HTTPProxy)
	if proxyURL == "" {
		s.mu.RLock()
		proxyURL = s.httpProxy
		s.mu.RUnlock()
	}
	target := strings.TrimSpace(body.URL)
	if target == "" {
		target = "https://auth.x.ai/.well-known/openid-configuration"
	}
	cli, err := netx.NewHTTPClient(proxyURL, 12*time.Second)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      false,
			"error":   err.Error(),
			"proxy":   netx.EffectiveProxyDescription(proxyURL),
			"url":     target,
			"elapsed": time.Since(start).String(),
		})
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      resp.StatusCode >= 200 && resp.StatusCode < 500,
		"status":  resp.StatusCode,
		"proxy":   netx.EffectiveProxyDescription(proxyURL),
		"url":     target,
		"elapsed": time.Since(start).String(),
	})
}

func (s *Server) handleDeviceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.sessions == nil {
		http.Error(w, "oauth sessions unavailable", http.StatusServiceUnavailable)
		return
	}
	sess, err := s.sessions.StartDeviceLogin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleDeviceStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	sess, ok := s.sessions.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleBilling(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cred, err := s.store.Pick()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	token := cred.AccessToken
	if s.refresh != nil {
		if ts, err := s.refresh.EnsureAccess(r.Context(), cred.ID, cred.AccessToken, cred.RefreshToken, cred.ExpiresAt); err == nil {
			token = ts.AccessToken
			_ = s.store.ApplyTokens(cred.ID, ts.AccessToken, ts.RefreshToken, ts.ExpiresAt)
		}
	}
	out := map[string]any{"credential_id": cred.ID, "label": cred.Label}
	if s.upstream != nil {
		if st, raw, err := s.upstream.GetBilling(r.Context(), token); err != nil {
			out["monthly_error"] = err.Error()
		} else {
			out["monthly_status"] = st
			var v any
			_ = json.Unmarshal(raw, &v)
			out["monthly"] = v
		}
		if st, raw, err := s.upstream.GetBillingCredits(r.Context(), token); err != nil {
			out["weekly_error"] = err.Error()
		} else {
			out["weekly_status"] = st
			var v any
			_ = json.Unmarshal(raw, &v)
			out["weekly"] = v
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		views := make([]map[string]any, 0, len(list))
		now := time.Now().UTC()
		for _, c := range list {
			cooling := c.CooldownUntil != nil && c.CooldownUntil.After(now)
			views = append(views, map[string]any{
				"id":             c.ID,
				"label":          c.Label,
				"email":          c.Email,
				"enabled":        c.Enabled,
				"priority":       c.Priority,
				"has_token":      strings.TrimSpace(c.AccessToken) != "",
				"has_refresh":    strings.TrimSpace(c.RefreshToken) != "",
				"expires_at":     c.ExpiresAt,
				"failure_count":  c.FailureCount,
				"cooling":        cooling,
				"cooldown_until": c.CooldownUntil,
				"last_error":     c.LastError,
				"updated_at":     c.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"credentials": views})
	case http.MethodPost:
		var body struct {
			ID           string `json:"id"`
			Label        string `json:"label"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			Priority     int    `json:"priority"`
			Enabled      *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		enabled := true
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		if strings.TrimSpace(body.AccessToken) == "" && strings.TrimSpace(body.ID) == "" {
			http.Error(w, "access_token required", http.StatusBadRequest)
			return
		}
		label := strings.TrimSpace(body.Label)
		if label == "" {
			label = "SuperGrok"
		}
		if err := s.store.Upsert(store.Credential{
			ID:           strings.TrimSpace(body.ID),
			Label:        label,
			AccessToken:  strings.TrimSpace(body.AccessToken),
			RefreshToken: strings.TrimSpace(body.RefreshToken),
			Priority:     body.Priority,
			Enabled:      enabled,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		if err := s.store.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	label := "imported"
	payload := raw
	var envelope struct {
		JSON  json.RawMessage `json:"json"`
		Label string          `json:"label"`
		Raw   string          `json:"raw"`
	}
	if json.Unmarshal(raw, &envelope) == nil {
		if strings.TrimSpace(envelope.Label) != "" {
			label = envelope.Label
		}
		if len(envelope.JSON) > 0 {
			payload = envelope.JSON
		} else if strings.TrimSpace(envelope.Raw) != "" {
			payload = []byte(envelope.Raw)
		}
	}
	n, err := s.store.ImportRaw(payload, label)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "imported": n})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
