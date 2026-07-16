package oauth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Refresher deduplicates concurrent refresh grants per credential.
type Refresher struct {
	Client *Client
	Skew   time.Duration
	group  singleflight.Group
	mu     sync.Mutex
	cache  map[string]TokenSet
}

func NewRefresher(client *Client) *Refresher {
	if client == nil {
		client = NewClient()
	}
	return &Refresher{
		Client: client,
		Skew:   DefaultRefreshSkew,
		cache:  make(map[string]TokenSet),
	}
}

// EnsureAccess returns a usable access token, refreshing when needed.
func (r *Refresher) EnsureAccess(ctx context.Context, key string, access, refresh string, expiresAt time.Time) (TokenSet, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = strings.TrimSpace(refresh)
	}
	now := time.Now().UTC()
	skew := r.Skew
	if skew <= 0 {
		skew = DefaultRefreshSkew
	}

	if cached, ok := r.getCache(key); ok {
		if strings.TrimSpace(cached.AccessToken) != "" && !cached.Expired(now, skew) {
			return cached, nil
		}
		if strings.TrimSpace(cached.RefreshToken) != "" {
			refresh = cached.RefreshToken
			if strings.TrimSpace(cached.AccessToken) != "" {
				access = cached.AccessToken
			}
			if !cached.ExpiresAt.IsZero() {
				expiresAt = cached.ExpiresAt
			}
		}
	}

	current := TokenSet{AccessToken: access, RefreshToken: refresh, ExpiresAt: expiresAt}
	if strings.TrimSpace(current.AccessToken) != "" && !current.Expired(now, skew) {
		return current, nil
	}
	if strings.TrimSpace(current.RefreshToken) == "" {
		if strings.TrimSpace(current.AccessToken) != "" {
			// No refresh available; use existing access and hope.
			return current, nil
		}
		return TokenSet{}, fmt.Errorf("access expired and no refresh_token")
	}

	v, err, _ := r.group.Do(key, func() (any, error) {
		// Re-check cache inside flight.
		if cached, ok := r.getCache(key); ok {
			if strings.TrimSpace(cached.AccessToken) != "" && !cached.Expired(time.Now().UTC(), skew) {
				return cached, nil
			}
			if rt := strings.TrimSpace(cached.RefreshToken); rt != "" {
				current.RefreshToken = rt
			}
		}
		opCtx, cancel := context.WithTimeout(context.Background(), DefaultHTTPTimeout)
		defer cancel()
		if ctx != nil {
			// Prefer parent cancel if sooner, but keep independent deadline.
			select {
			case <-ctx.Done():
				return TokenSet{}, ctx.Err()
			default:
			}
		}
		next, err := r.Client.Refresh(opCtx, current.RefreshToken)
		if err != nil {
			return TokenSet{}, err
		}
		if strings.TrimSpace(next.RefreshToken) == "" {
			next.RefreshToken = current.RefreshToken
		}
		r.putCache(key, *next)
		return *next, nil
	})
	if err != nil {
		return TokenSet{}, err
	}
	return v.(TokenSet), nil
}

func (r *Refresher) getCache(key string) (TokenSet, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ts, ok := r.cache[key]
	return ts, ok
}

func (r *Refresher) putCache(key string, ts TokenSet) {
	r.mu.Lock()
	r.cache[key] = ts
	r.mu.Unlock()
}

// Invalidate drops cached tokens for a credential.
func (r *Refresher) Invalidate(key string) {
	r.mu.Lock()
	delete(r.cache, key)
	r.mu.Unlock()
}
