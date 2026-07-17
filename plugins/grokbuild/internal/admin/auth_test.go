package admin

import (
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/issueye/icoo_proxy/common/pluginipc"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
)

func TestAdminTokenRequired(t *testing.T) {
	dir := t.TempDir()
	st := store.New(filepath.Join(dir, "creds"))
	settings := store.NewSettingsStore(dir)
	token := "test-admin-token-0123456789abcdef"

	srv, base, err := Start(StartOpts{
		Store:      st,
		Settings:   settings,
		AdminToken: token,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	// Missing token → 401
	resp, err := http.Get(base + "/api/health")
	if err != nil {
		t.Fatalf("GET without token: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d body=%s", resp.StatusCode, body)
	}

	// Wrong token → 401
	req, _ := http.NewRequest(http.MethodGet, base+"/api/health", nil)
	req.Header.Set(pluginipc.HeaderPluginAdminToken, "wrong")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET wrong token: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token status=%d body=%s", resp.StatusCode, body)
	}

	// Correct token → 200
	req, _ = http.NewRequest(http.MethodGet, base+"/api/health", nil)
	req.Header.Set(pluginipc.HeaderPluginAdminToken, token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET with token: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid token status=%d body=%s", resp.StatusCode, body)
	}
}

func TestAdminTokenEmptyAllowsLocalDev(t *testing.T) {
	dir := t.TempDir()
	st := store.New(filepath.Join(dir, "creds"))
	srv, base, err := Start(StartOpts{Store: st})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	resp, err := http.Get(base + "/api/health")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("empty token should allow, status=%d", resp.StatusCode)
	}
}
