package oauth

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
)

// PreRefreshConfig controls the background token pre-refresh loop.
type PreRefreshConfig struct {
	// Interval between scans. Default 2m.
	Interval time.Duration
	// Lead time before ExpiresAt to trigger refresh. Default DefaultRefreshSkew.
	Lead time.Duration
}

// StartPreRefresh scans enabled credentials with refresh tokens and renews
// those nearing expiry. Stops when ctx is canceled.
func StartPreRefresh(ctx context.Context, st *store.Store, r *Refresher, cfg PreRefreshConfig) {
	if st == nil || r == nil {
		return
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	lead := cfg.Lead
	if lead <= 0 {
		lead = DefaultRefreshSkew
		if r.Skew > 0 {
			lead = r.Skew
		}
	}

	// Initial delay so startup handshake is not contended.
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runPreRefreshOnce(ctx, st, r, lead)
			timer.Reset(interval)
		}
	}
}

func runPreRefreshOnce(ctx context.Context, st *store.Store, r *Refresher, lead time.Duration) {
	list, err := st.List()
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, c := range list {
		if ctx.Err() != nil {
			return
		}
		if !c.Enabled || strings.TrimSpace(c.RefreshToken) == "" {
			continue
		}
		// Refresh when no expiry (unknown) is skipped to avoid hammering; only
		// when ExpiresAt is known and within lead window.
		if c.ExpiresAt.IsZero() {
			continue
		}
		if now.Before(c.ExpiresAt.Add(-lead)) {
			continue
		}
		ts, err := r.EnsureAccess(ctx, c.ID, c.AccessToken, c.RefreshToken, c.ExpiresAt)
		if err != nil {
			log.Printf("prerefresh: credential %s: %v", c.ID, err)
			continue
		}
		if ts.AccessToken != c.AccessToken || ts.RefreshToken != c.RefreshToken || !ts.ExpiresAt.Equal(c.ExpiresAt) {
			if err := st.ApplyTokens(c.ID, ts.AccessToken, ts.RefreshToken, ts.ExpiresAt); err != nil {
				log.Printf("prerefresh: persist %s: %v", c.ID, err)
				continue
			}
			log.Printf("prerefresh: renewed credential %s (expires %s)", c.ID, ts.ExpiresAt.Format(time.RFC3339))
		}
	}
}
