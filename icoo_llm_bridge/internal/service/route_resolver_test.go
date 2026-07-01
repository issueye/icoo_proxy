package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/domain"
	"icoo_llm_bridge/internal/model/entity"
)

func TestRouteResolverResolve(t *testing.T) {
	ctx := context.Background()
	baseProviders := []entity.Provider{
		provider("p-openai", "openai", constants.ProtocolOpenAIChat, true),
		provider("p-anthropic", "anthropic", constants.ProtocolAnthropic, true),
		provider("p-disabled", "disabled", constants.ProtocolOpenAIChat, false),
	}
	baseModels := map[string][]entity.ProviderModel{
		"p-openai": {
			model("p-openai", "gpt-4o", 128000, true),
			model("p-openai", "gpt-4o-mini", 64000, true),
			model("p-openai", "disabled-model", 1000, false),
		},
		"p-anthropic": {
			model("p-anthropic", "claude-3-5-sonnet", 200000, true),
		},
		"p-disabled": {
			model("p-disabled", "ghost", 1, true),
		},
	}

	tests := []struct {
		name           string
		providers      []entity.Provider
		models         map[string][]entity.ProviderModel
		rules          []entity.RoutingRule
		downstream     constants.Protocol
		requestedModel string
		want           domain.Route
		wantErr        string
	}{
		{
			name:      "provider model direct route wins before rules",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-catch", "catch all", 1, constants.ProtocolOpenAIChat, "*", "", "p-anthropic", "claude-3-5-sonnet", true),
			},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "openai/gpt-4o-mini",
			want: domain.Route{
				Name:             "openai/gpt-4o-mini",
				UpstreamProtocol: constants.ProtocolOpenAIChat,
				Model:            "gpt-4o-mini",
				DefaultMaxTokens: 64000,
				Source:           "direct",
			},
		},
		{
			name:      "enabled rules match by protocol pattern and priority",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-disabled", "disabled", 0, constants.ProtocolOpenAIChat, "claude*", "", "p-openai", "gpt-4o-mini", false),
				rule("r-low", "low priority", 20, constants.ProtocolOpenAIChat, "claude*", "", "p-openai", "gpt-4o-mini", true),
				rule("r-high", "high priority", 10, constants.ProtocolOpenAIChat, "claude*", "", "p-anthropic", "claude-3-5-sonnet", true),
				rule("r-other-protocol", "other protocol", 1, constants.ProtocolAnthropic, "claude*", "", "p-openai", "gpt-4o", true),
			},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "claude-3-opus",
			want: domain.Route{
				Name:             "high priority",
				UpstreamProtocol: constants.ProtocolAnthropic,
				Model:            "claude-3-5-sonnet",
				DefaultMaxTokens: 200000,
				Source:           "routing_rule:r-high",
			},
		},
		{
			name:      "empty requested model uses explicit target model",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-default", "default", 1, constants.ProtocolOpenAIChat, "", "", "p-openai", "gpt-4o", true),
			},
			downstream: constants.ProtocolOpenAIChat,
			want: domain.Route{
				Name:             "default",
				UpstreamProtocol: constants.ProtocolOpenAIChat,
				Model:            "gpt-4o",
				DefaultMaxTokens: 128000,
				Source:           "routing_rule:r-default",
			},
		},
		{
			name:      "star rule can act as default rule with explicit model",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-star", "star default", 1, constants.ProtocolOpenAIChat, "*", "", "p-openai", "gpt-4o-mini", true),
			},
			downstream: constants.ProtocolOpenAIChat,
			want: domain.Route{
				Name:             "star default",
				UpstreamProtocol: constants.ProtocolOpenAIChat,
				Model:            "gpt-4o-mini",
				DefaultMaxTokens: 64000,
				Source:           "routing_rule:r-star",
			},
		},
		{
			name:           "missing direct provider returns clear error before rules",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-catch", "catch all", 1, constants.ProtocolOpenAIChat, "*", "", "p-openai", "gpt-4o", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "missing/gpt-4o",
			wantErr:        `direct route provider "missing" was not found or is disabled`,
		},
		{
			name:           "missing rule match returns clear error",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-anthropic", "anthropic only", 1, constants.ProtocolAnthropic, "gpt-*", "", "p-openai", "gpt-4o", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "gpt-4o",
			wantErr:        `no route matched downstream protocol "openai-chat" and model "gpt-4o"`,
		},
		{
			name:           "disabled target model is rejected",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-disabled-model", "disabled model", 1, constants.ProtocolOpenAIChat, "*", "", "p-openai", "disabled-model", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "anything",
			wantErr:        `routing rule "disabled model" targets missing or disabled model "disabled-model" for provider "openai"`,
		},
		{
			name:           "empty requested model without target model is rejected",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-default", "default", 1, constants.ProtocolOpenAIChat, "*", "", "p-openai", "", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "",
			wantErr:        `routing rule "default" did not specify a target model`,
		},
		{
			name:           "disabled target provider is rejected",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-disabled-provider", "disabled provider", 1, constants.ProtocolOpenAIChat, "*", "", "p-disabled", "ghost", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "anything",
			wantErr:        `routing rule "disabled provider" targets missing or disabled provider "p-disabled"`,
		},
		{
			name:      "rule upstream protocol overrides provider protocol",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-cross-protocol", "cross protocol", 1, constants.ProtocolOpenAIResponses, "*", constants.ProtocolAnthropic, "p-openai", "gpt-4o", true),
			},
			downstream:     constants.ProtocolOpenAIResponses,
			requestedModel: "any-model",
			want: domain.Route{
				Name:             "cross protocol",
				UpstreamProtocol: constants.ProtocolAnthropic,
				Model:            "gpt-4o",
				DefaultMaxTokens: 128000,
				Source:           "routing_rule:r-cross-protocol",
			},
		},
		{
			name:      "rule upstream protocol falls back to provider protocol when empty",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-same-protocol", "same protocol", 1, constants.ProtocolOpenAIChat, "gpt-*", "", "p-openai", "gpt-4o", true),
			},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "gpt-4o-mini",
			want: domain.Route{
				Name:             "same protocol",
				UpstreamProtocol: constants.ProtocolOpenAIChat,
				Model:            "gpt-4o",
				DefaultMaxTokens: 128000,
				Source:           "routing_rule:r-same-protocol",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewRouteResolver(
				&fakeProviderRepository{items: tt.providers},
				&fakeProviderModelRepository{items: tt.models},
				&fakeRoutingRuleRepository{items: tt.rules},
			)

			got, err := resolver.Resolve(ctx, tt.downstream, tt.requestedModel)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() unexpected error: %v", err)
			}
			assertRoute(t, got, tt.want)
		})
	}
}

func TestRouteResolverResolvePlan(t *testing.T) {
	ctx := context.Background()
	providers := []entity.Provider{
		providerWithRuntime("p-openai", "openai", constants.ProtocolOpenAIChat, "https://openai.example", "openai-key", true),
		providerWithRuntime("p-anthropic", "anthropic", constants.ProtocolAnthropic, "https://anthropic.example", "anthropic-key", true),
		providerWithRuntime("p-responses", "responses", constants.ProtocolOpenAIResponses, "https://responses.example", "responses-key", true),
	}
	models := map[string][]entity.ProviderModel{
		"p-openai": {
			model("p-openai", "gpt-4o", 128000, true),
			model("p-openai", "gpt-4o-mini", 64000, true),
		},
		"p-anthropic": {
			model("p-anthropic", "claude-3-5-sonnet", 200000, true),
		},
		"p-responses": {
			model("p-responses", "gpt-5", 200000, true),
		},
	}

	t.Run("direct route produces a single direct candidate", func(t *testing.T) {
		resolver := newConcreteRouteResolver(providers, models, []entity.RoutingRule{
			rule("r-catch", "catch all", 1, constants.ProtocolOpenAIChat, "*", "", "p-anthropic", "claude-3-5-sonnet", true),
		})

		plan, err := resolver.ResolvePlan(ctx, constants.ProtocolOpenAIChat, "openai/gpt-4o")
		if err != nil {
			t.Fatalf("ResolvePlan() unexpected error: %v", err)
		}
		if plan.DownstreamProtocol != constants.ProtocolOpenAIChat || plan.RequestedModel != "openai/gpt-4o" {
			t.Fatalf("plan request metadata = %#v", plan)
		}
		if len(plan.Candidates) != 1 {
			t.Fatalf("candidates = %d, want 1", len(plan.Candidates))
		}

		candidate := plan.Candidates[0]
		if candidate.Source != "direct" || candidate.Priority != 0 || candidate.Model != "gpt-4o" {
			t.Fatalf("candidate = %#v", candidate)
		}
		if candidate.Endpoint.BaseURL != "https://openai.example" || !candidate.Endpoint.Enabled {
			t.Fatalf("endpoint snapshot = %#v", candidate.Endpoint)
		}
		if candidate.Credential.APIKey != "openai-key" || !candidate.Credential.Enabled {
			t.Fatalf("credential snapshot = %#v", candidate.Credential)
		}
		if got := candidate.Route(); got.Provider.BaseURL != "https://openai.example" || got.Provider.APIKey != "openai-key" {
			t.Fatalf("candidate Route() provider = %#v", got.Provider)
		}
	})

	t.Run("routing rule candidates are ordered by priority", func(t *testing.T) {
		resolver := newConcreteRouteResolver(providers, models, []entity.RoutingRule{
			rule("r-low", "low priority", 20, constants.ProtocolOpenAIChat, "claude*", "", "p-openai", "gpt-4o-mini", true),
			rule("r-high", "high priority", 10, constants.ProtocolOpenAIChat, "claude*", "", "p-anthropic", "claude-3-5-sonnet", true),
			rule("r-same-a", "same priority a", 15, constants.ProtocolOpenAIChat, "claude*", constants.ProtocolAnthropic, "p-responses", "gpt-5", true),
			rule("r-same-b", "same priority b", 15, constants.ProtocolOpenAIChat, "claude*", "", "p-openai", "gpt-4o", true),
			rule("r-other-protocol", "other protocol", 1, constants.ProtocolAnthropic, "claude*", "", "p-openai", "gpt-4o", true),
		})

		plan, err := resolver.ResolvePlan(ctx, constants.ProtocolOpenAIChat, "claude-3-opus")
		if err != nil {
			t.Fatalf("ResolvePlan() unexpected error: %v", err)
		}
		if len(plan.Candidates) != 4 {
			t.Fatalf("candidates = %d, want 4", len(plan.Candidates))
		}

		wantNames := []string{"high priority", "same priority a", "same priority b", "low priority"}
		wantPriorities := []int{10, 15, 15, 20}
		for i, candidate := range plan.Candidates {
			if candidate.Name != wantNames[i] || candidate.Priority != wantPriorities[i] {
				t.Fatalf("candidate[%d] = %q/%d, want %q/%d", i, candidate.Name, candidate.Priority, wantNames[i], wantPriorities[i])
			}
		}
		if plan.Candidates[1].UpstreamProtocol != constants.ProtocolAnthropic {
			t.Fatalf("upstream protocol override = %q", plan.Candidates[1].UpstreamProtocol)
		}
	})
}

func assertRoute(t *testing.T, got domain.Route, want domain.Route) {
	t.Helper()
	if got.Name != want.Name ||
		got.UpstreamProtocol != want.UpstreamProtocol ||
		got.Model != want.Model ||
		got.DefaultMaxTokens != want.DefaultMaxTokens ||
		got.Source != want.Source {
		t.Fatalf("route = %#v, want fields %#v", got, want)
	}
}

func provider(id string, name string, protocol constants.Protocol, enabled bool) entity.Provider {
	return providerWithRuntime(id, name, protocol, "", "", enabled)
}

func providerWithRuntime(id string, name string, protocol constants.Protocol, baseURL string, apiKey string, enabled bool) entity.Provider {
	return entity.Provider{
		ID:           id,
		Name:         name,
		Protocol:     protocol,
		Vendor:       constants.VendorCustom,
		BaseURL:      baseURL,
		APIKeyCipher: apiKey,
		Enabled:      enabled,
	}
}

func newConcreteRouteResolver(
	providers []entity.Provider,
	models map[string][]entity.ProviderModel,
	rules []entity.RoutingRule,
) *routeResolver {
	return &routeResolver{
		providers: &fakeProviderRepository{items: providers},
		models:    &fakeProviderModelRepository{items: models},
		rules:     &fakeRoutingRuleRepository{items: rules},
		cache:     &RouteCache{},
	}
}

func model(providerID string, name string, maxTokens int, enabled bool) entity.ProviderModel {
	return entity.ProviderModel{
		ID:         providerID + "-" + name,
		ProviderID: providerID,
		Name:       name,
		MaxTokens:  maxTokens,
		Enabled:    enabled,
	}
}

func rule(
	id string,
	name string,
	priority int,
	protocol constants.Protocol,
	pattern string,
	upstreamProtocol constants.Protocol,
	targetProviderID string,
	targetModel string,
	enabled bool,
) entity.RoutingRule {
	return entity.RoutingRule{
		ID:                id,
		Name:              name,
		Priority:          priority,
		MatchProtocol:     protocol,
		MatchModelPattern: pattern,
		UpstreamProtocol:  upstreamProtocol,
		TargetProviderID:  targetProviderID,
		TargetModel:       targetModel,
		Enabled:           enabled,
	}
}

type fakeProviderRepository struct {
	items []entity.Provider
	err   error
}

func (r *fakeProviderRepository) List(context.Context) ([]entity.Provider, error) {
	return append([]entity.Provider(nil), r.items...), r.err
}

func (r *fakeProviderRepository) Find(context.Context, string) (entity.Provider, error) {
	return entity.Provider{}, errors.New("not implemented")
}

func (r *fakeProviderRepository) Save(context.Context, *entity.Provider) error {
	return errors.New("not implemented")
}

func (r *fakeProviderRepository) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

type fakeProviderModelRepository struct {
	items map[string][]entity.ProviderModel
	err   error
}

func (r *fakeProviderModelRepository) ListByProvider(_ context.Context, providerID string) ([]entity.ProviderModel, error) {
	return append([]entity.ProviderModel(nil), r.items[providerID]...), r.err
}

func (r *fakeProviderModelRepository) Save(context.Context, *entity.ProviderModel) error {
	return errors.New("not implemented")
}

func (r *fakeProviderModelRepository) Delete(context.Context, string, string) error {
	return errors.New("not implemented")
}

type fakeRoutingRuleRepository struct {
	items []entity.RoutingRule
	err   error
}

func (r *fakeRoutingRuleRepository) List(context.Context) ([]entity.RoutingRule, error) {
	return append([]entity.RoutingRule(nil), r.items...), r.err
}

func (r *fakeRoutingRuleRepository) ListEnabled(context.Context) ([]entity.RoutingRule, error) {
	if r.err != nil {
		return nil, r.err
	}
	var enabled []entity.RoutingRule
	for _, item := range r.items {
		if item.Enabled {
			enabled = append(enabled, item)
		}
	}
	return enabled, nil
}

func (r *fakeRoutingRuleRepository) Find(context.Context, string) (entity.RoutingRule, error) {
	return entity.RoutingRule{}, errors.New("not implemented")
}

func (r *fakeRoutingRuleRepository) Save(context.Context, *entity.RoutingRule) error {
	return errors.New("not implemented")
}

func (r *fakeRoutingRuleRepository) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

// countingProviderRepository wraps the fake and records how many times List is
// called, so cache hit/miss behavior can be asserted.
type countingProviderRepository struct {
	inner *fakeProviderRepository
	calls int
}

func (r *countingProviderRepository) List(ctx context.Context) ([]entity.Provider, error) {
	r.calls++
	return r.inner.List(ctx)
}

func (r *countingProviderRepository) Find(ctx context.Context, id string) (entity.Provider, error) {
	return r.inner.Find(ctx, id)
}

func (r *countingProviderRepository) Save(ctx context.Context, item *entity.Provider) error {
	return r.inner.Save(ctx, item)
}

func (r *countingProviderRepository) Delete(ctx context.Context, id string) error {
	return r.inner.Delete(ctx, id)
}

// TestRouteResolverCachesProvidersAndInvalidates verifies the route cache serves
// repeated resolves from memory and drops the snapshot on invalidation so a
// fresh read happens after a config mutation.
func TestRouteResolverCachesProvidersAndInvalidates(t *testing.T) {
	ctx := context.Background()
	providers := &countingProviderRepository{
		inner: &fakeProviderRepository{
			items: []entity.Provider{
				provider("p-openai", "openai", constants.ProtocolOpenAIChat, true),
			},
		},
	}
	models := &fakeProviderModelRepository{
		items: map[string][]entity.ProviderModel{
			"p-openai": {model("p-openai", "gpt-4o", 128000, true)},
		},
	}
	resolver := newConcreteRouteResolver(nil, nil, nil)
	resolver.providers = providers
	resolver.models = models

	// First resolve: one DB hit to load providers.
	if _, err := resolver.Resolve(ctx, constants.ProtocolOpenAIChat, "openai/gpt-4o"); err != nil {
		t.Fatalf("first Resolve error: %v", err)
	}
	if providers.calls != 1 {
		t.Fatalf("provider List calls after first resolve = %d, want 1", providers.calls)
	}

	// Second resolve: served from cache, no extra DB hit.
	if _, err := resolver.Resolve(ctx, constants.ProtocolOpenAIChat, "openai/gpt-4o"); err != nil {
		t.Fatalf("second Resolve error: %v", err)
	}
	if providers.calls != 1 {
		t.Fatalf("provider List calls after cached resolve = %d, want 1", providers.calls)
	}

	// Invalidate: the next resolve re-reads from the DB.
	resolver.cache.InvalidateProviders()
	if _, err := resolver.Resolve(ctx, constants.ProtocolOpenAIChat, "openai/gpt-4o"); err != nil {
		t.Fatalf("post-invalidate Resolve error: %v", err)
	}
	if providers.calls != 2 {
		t.Fatalf("provider List calls after invalidate = %d, want 2", providers.calls)
	}
}

