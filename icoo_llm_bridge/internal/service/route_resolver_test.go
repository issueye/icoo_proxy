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
			model("p-openai", "gpt-4o", 128000, true, true),
			model("p-openai", "gpt-4o-mini", 64000, false, true),
			model("p-openai", "disabled-model", 1000, false, false),
		},
		"p-anthropic": {
			model("p-anthropic", "claude-3-5-sonnet", 200000, true, true),
		},
		"p-disabled": {
			model("p-disabled", "ghost", 1, true, true),
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
				rule("r-catch", "catch all", 1, constants.ProtocolOpenAIChat, "*", "p-anthropic", "claude-3-5-sonnet", true),
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
				rule("r-disabled", "disabled", 0, constants.ProtocolOpenAIChat, "claude*", "p-openai", "gpt-4o-mini", false),
				rule("r-low", "low priority", 20, constants.ProtocolOpenAIChat, "claude*", "p-openai", "gpt-4o-mini", true),
				rule("r-high", "high priority", 10, constants.ProtocolOpenAIChat, "claude*", "p-anthropic", "claude-3-5-sonnet", true),
				rule("r-other-protocol", "other protocol", 1, constants.ProtocolAnthropic, "claude*", "p-openai", "gpt-4o", true),
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
			name:      "empty requested model uses default rule and provider default model",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-default", "default", 1, constants.ProtocolOpenAIChat, "", "p-openai", "", true),
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
			name:      "star rule can act as default rule for empty requested model",
			providers: baseProviders,
			models:    baseModels,
			rules: []entity.RoutingRule{
				rule("r-star", "star default", 1, constants.ProtocolOpenAIChat, "*", "p-openai", "gpt-4o-mini", true),
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
			rules:          []entity.RoutingRule{rule("r-catch", "catch all", 1, constants.ProtocolOpenAIChat, "*", "p-openai", "gpt-4o", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "missing/gpt-4o",
			wantErr:        `direct route provider "missing" was not found or is disabled`,
		},
		{
			name:           "missing rule match returns clear error",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-anthropic", "anthropic only", 1, constants.ProtocolAnthropic, "gpt-*", "p-openai", "gpt-4o", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "gpt-4o",
			wantErr:        `no route matched downstream protocol "openai-chat" and model "gpt-4o"`,
		},
		{
			name:           "disabled target model is rejected",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-disabled-model", "disabled model", 1, constants.ProtocolOpenAIChat, "*", "p-openai", "disabled-model", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "anything",
			wantErr:        `routing rule "disabled model" targets missing or disabled model "disabled-model" for provider "openai"`,
		},
		{
			name:           "disabled target provider is rejected",
			providers:      baseProviders,
			models:         baseModels,
			rules:          []entity.RoutingRule{rule("r-disabled-provider", "disabled provider", 1, constants.ProtocolOpenAIChat, "*", "p-disabled", "ghost", true)},
			downstream:     constants.ProtocolOpenAIChat,
			requestedModel: "anything",
			wantErr:        `routing rule "disabled provider" targets missing or disabled provider "p-disabled"`,
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
	return entity.Provider{
		ID:       id,
		Name:     name,
		Protocol: protocol,
		Vendor:   constants.VendorCustom,
		Enabled:  enabled,
	}
}

func model(providerID string, name string, maxTokens int, isDefault bool, enabled bool) entity.ProviderModel {
	return entity.ProviderModel{
		ID:         providerID + "-" + name,
		ProviderID: providerID,
		Name:       name,
		MaxTokens:  maxTokens,
		IsDefault:  isDefault,
		Enabled:    enabled,
	}
}

func rule(
	id string,
	name string,
	priority int,
	protocol constants.Protocol,
	pattern string,
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
