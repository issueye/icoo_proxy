package service

import (
	"context"
	"sync"

	"github.com/issueye/icoo_proxy/common/domain"
	"github.com/issueye/icoo_proxy/bridge/internal/model/entity"
)

// routeSnapshot bundles the two data sets RouteResolver consults per request
// (enabled providers and enabled routing rules). They share a single entry so
// any admin mutation can invalidate both atomically, but each half is loaded and
// cached independently via the providersValid / rulesValid flags below.
type routeSnapshot struct {
	providers       []domain.ProviderSnapshot
	providersValid  bool
	rules           []entity.RoutingRule
	rulesValid      bool
}

// RouteCache caches the provider/routing snapshot used by RouteResolver so the
// hot proxy path does not re-query the database (with its N+1 model lookups) on
// every request. The cache is invalidated by the admin services immediately
// after a provider/model/rule write (write-through invalidation), so the proxy
// observes configuration changes on the next request with no stale window.
type RouteCache struct {
	mu       sync.RWMutex
	snapshot routeSnapshot
}

// CacheInvalidator is implemented by RouteCache and injected into the admin
// services so they can drop the cached snapshot after a mutation. Defining it as
// an interface keeps the admin services decoupled from the concrete cache type
// and avoids a constructor-parameter ordering problem (the cache is owned by the
// resolver, which is built after the services that invalidate it).
type CacheInvalidator interface {
	InvalidateProviders()
}

// noopInvalidator is the zero value used before wiring is complete.
type noopInvalidator struct{}

func (noopInvalidator) InvalidateProviders() {}

// providers returns a defensive copy of cached providers when present.
func (c *RouteCache) providers() ([]domain.ProviderSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.snapshot.providersValid {
		return nil, false
	}
	out := make([]domain.ProviderSnapshot, len(c.snapshot.providers))
	copy(out, c.snapshot.providers)
	return out, true
}

// rules returns a defensive copy of cached rules when present.
func (c *RouteCache) rules() ([]entity.RoutingRule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.snapshot.rulesValid {
		return nil, false
	}
	out := make([]entity.RoutingRule, len(c.snapshot.rules))
	copy(out, c.snapshot.rules)
	return out, true
}

// setProviders caches a defensive copy of providers, leaving any cached rules
// intact (rules and providers are cached independently).
func (c *RouteCache) setProviders(providers []domain.ProviderSnapshot) {
	cp := make([]domain.ProviderSnapshot, len(providers))
	copy(cp, providers)
	c.mu.Lock()
	c.snapshot.providers = cp
	c.snapshot.providersValid = true
	c.mu.Unlock()
}

// setRules caches a defensive copy of rules, leaving any cached providers intact.
func (c *RouteCache) setRules(rules []entity.RoutingRule) {
	cp := make([]entity.RoutingRule, len(rules))
	copy(cp, rules)
	c.mu.Lock()
	c.snapshot.rules = cp
	c.snapshot.rulesValid = true
	c.mu.Unlock()
}

// InvalidateProviders drops the cached provider/rule snapshot.
func (c *RouteCache) InvalidateProviders() {
	c.mu.Lock()
	c.snapshot = routeSnapshot{}
	c.mu.Unlock()
}

// loadProviders returns cached providers, or loads them on a miss. Errors are
// not cached: a failed load leaves the cache untouched so the next request
// retries.
func (c *RouteCache) loadProviders(
	ctx context.Context,
	loadProviders func(context.Context) ([]domain.ProviderSnapshot, error),
) ([]domain.ProviderSnapshot, error) {
	if c == nil {
		// Defensive: a resolver built without the cache (e.g. tests) bypasses it.
		return loadProviders(ctx)
	}
	if cached, ok := c.providers(); ok {
		return cached, nil
	}
	providers, err := loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	c.setProviders(providers)
	cached, _ := c.providers()
	return cached, nil
}

// loadRules returns cached rules, or loads them on a miss. Loaded independently
// of providers so a rule-only miss does not force a provider reload.
func (c *RouteCache) loadRules(
	ctx context.Context,
	loadRules func(context.Context) ([]entity.RoutingRule, error),
) ([]entity.RoutingRule, error) {
	if c == nil {
		// Defensive: a resolver built without the cache (e.g. tests) bypasses it.
		return loadRules(ctx)
	}
	if cached, ok := c.rules(); ok {
		return cached, nil
	}
	rules, err := loadRules(ctx)
	if err != nil {
		return nil, err
	}
	c.setRules(rules)
	cached, _ := c.rules()
	return cached, nil
}

// apiKeyCache memoizes the enabled API keys so Verify does not hit the database
// on every proxied request. It is self-invalidating: authService drops it after
// any key create/delete. The constant-time secret comparison in Verify is
// unchanged; only the (already-hashed) key list is cached.
type apiKeyCache struct {
	mu   sync.RWMutex
	keys []entity.APIKey
}

func (c *apiKeyCache) loadOrFetch(
	ctx context.Context,
	fetch func(context.Context) ([]entity.APIKey, error),
) ([]entity.APIKey, error) {
	c.mu.RLock()
	if c.keys != nil {
		keys := c.keys
		c.mu.RUnlock()
		return keys, nil
	}
	c.mu.RUnlock()

	keys, err := fetch(ctx)
	if err != nil {
		return nil, err
	}
	cp := make([]entity.APIKey, len(keys))
	copy(cp, keys)
	c.mu.Lock()
	c.keys = cp
	c.mu.Unlock()
	return cp, nil
}

func (c *apiKeyCache) invalidate() {
	c.mu.Lock()
	c.keys = nil
	c.mu.Unlock()
}
