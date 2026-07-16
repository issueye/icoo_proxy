package oauth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
)

// TestPreRefreshSkipsWhenNotNearExpiry ensures the scanner does not call
// refresh for tokens with distant expiry (no network).
func TestPreRefreshSkipsWhenNotNearExpiry(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	cred := store.Credential{
		ID:           "c1",
		Label:        "t",
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
		Enabled:      true,
	}
	if err := st.Upsert(cred); err != nil {
		t.Fatal(err)
	}
	// Refresher with nil-safe client; EnsureAccess should not be needed.
	r := NewRefresher(NewClient())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runPreRefreshOnce(ctx, st, r, 3*time.Minute)

	got, err := st.Get("c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" {
		t.Fatalf("tokens mutated unexpectedly: %+v", got)
	}
	_ = filepath.Join(dir, "credentials.json")
}
