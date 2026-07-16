package admin

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/oauth"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/upstream"
)

// Server is a loopback-only admin UI + JSON API for credentials / OAuth / billing.
type Server struct {
	store    *store.Store
	sessions *oauth.SessionManager
	upstream *upstream.Client
	refresh  *oauth.Refresher
	ln       net.Listener
	srv      *http.Server
}

func Start(st *store.Store, sessions *oauth.SessionManager, up *upstream.Client, refresh *oauth.Refresher) (*Server, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	s := &Server{store: st, sessions: sessions, upstream: up, refresh: refresh, ln: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/credentials", s.handleCredentials)
	mux.HandleFunc("/api/credentials/import", s.handleImport)
	mux.HandleFunc("/api/oauth/device/start", s.handleDeviceStart)
	mux.HandleFunc("/api/oauth/device/status", s.handleDeviceStatus)
	mux.HandleFunc("/api/billing", s.handleBilling)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		list, _ := st.List()
		enabled := 0
		for _, c := range list {
			if c.Enabled && strings.TrimSpace(c.AccessToken) != "" {
				enabled++
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                 true,
			"plugin":             "grokbuild",
			"credentials_total":  len(list),
			"credentials_active": enabled,
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
