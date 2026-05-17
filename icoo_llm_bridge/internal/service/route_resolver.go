package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/domain"
	"icoo_llm_bridge/internal/model/entity"
	"icoo_llm_bridge/internal/repository"
)

type routeResolver struct {
	providers repository.ProviderRepository
	models    repository.ProviderModelRepository
	rules     repository.RoutingRuleRepository
}

func NewRouteResolver(
	providers repository.ProviderRepository,
	models repository.ProviderModelRepository,
	rules repository.RoutingRuleRepository,
) RouteResolver {
	return &routeResolver{
		providers: providers,
		models:    models,
		rules:     rules,
	}
}

func (r *routeResolver) Resolve(ctx context.Context, downstream constants.Protocol, requestedModel string) (domain.Route, error) {
	requestedModel = strings.TrimSpace(requestedModel)
	providers, err := r.loadProviders(ctx)
	if err != nil {
		return domain.Route{}, err
	}

	if route, ok, err := r.resolveDirect(providers, requestedModel); ok || err != nil {
		return route, err
	}

	rules, err := r.rules.ListEnabled(ctx)
	if err != nil {
		return domain.Route{}, fmt.Errorf("list enabled routing rules: %w", err)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	for _, rule := range rules {
		if !ruleMatches(rule, downstream, requestedModel) {
			continue
		}
		return r.routeFromRule(providers, rule, requestedModel)
	}

	if requestedModel == "" {
		return domain.Route{}, fmt.Errorf("no route matched downstream protocol %q", downstream)
	}
	return domain.Route{}, fmt.Errorf("no route matched downstream protocol %q and model %q", downstream, requestedModel)
}

func (r *routeResolver) ResolvePlan(ctx context.Context, downstream constants.Protocol, requestedModel string) (domain.RoutePlan, error) {
	requestedModel = strings.TrimSpace(requestedModel)
	plan := domain.RoutePlan{
		DownstreamProtocol: downstream,
		RequestedModel:     requestedModel,
	}

	providers, err := r.loadProviders(ctx)
	if err != nil {
		return plan, err
	}

	if candidate, ok, err := r.resolveDirectCandidate(providers, requestedModel); ok || err != nil {
		if err != nil {
			return plan, err
		}
		plan.Candidates = append(plan.Candidates, candidate)
		return plan, nil
	}

	rules, err := r.rules.ListEnabled(ctx)
	if err != nil {
		return plan, fmt.Errorf("list enabled routing rules: %w", err)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	for _, rule := range rules {
		if !ruleMatches(rule, downstream, requestedModel) {
			continue
		}
		candidate, err := r.candidateFromRule(providers, rule, requestedModel)
		if err != nil {
			return plan, err
		}
		plan.Candidates = append(plan.Candidates, candidate)
	}

	if len(plan.Candidates) > 0 {
		return plan, nil
	}
	if requestedModel == "" {
		return plan, fmt.Errorf("no route matched downstream protocol %q", downstream)
	}
	return plan, fmt.Errorf("no route matched downstream protocol %q and model %q", downstream, requestedModel)
}

func (r *routeResolver) loadProviders(ctx context.Context) ([]domain.ProviderSnapshot, error) {
	items, err := r.providers.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	providers := make([]domain.ProviderSnapshot, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		models, err := r.models.ListByProvider(ctx, item.ID)
		if err != nil {
			return nil, fmt.Errorf("list models for provider %q: %w", item.ID, err)
		}
		providers = append(providers, providerSnapshot(item, models))
	}
	return providers, nil
}

func (r *routeResolver) resolveDirect(providers []domain.ProviderSnapshot, requestedModel string) (domain.Route, bool, error) {
	candidate, ok, err := r.resolveDirectCandidate(providers, requestedModel)
	if err != nil || !ok {
		return domain.Route{}, ok, err
	}
	return candidate.Route(), true, nil
}

func (r *routeResolver) resolveDirectCandidate(providers []domain.ProviderSnapshot, requestedModel string) (domain.RouteCandidate, bool, error) {
	providerName, modelName, ok := strings.Cut(requestedModel, "/")
	if !ok || strings.TrimSpace(providerName) == "" || strings.TrimSpace(modelName) == "" {
		return domain.RouteCandidate{}, false, nil
	}

	provider, ok := findProvider(providers, providerName)
	if !ok {
		return domain.RouteCandidate{}, true, fmt.Errorf("direct route provider %q was not found or is disabled", providerName)
	}
	model, ok := findModel(provider.Models, modelName)
	if !ok {
		return domain.RouteCandidate{}, true, fmt.Errorf("direct route model %q was not found or is disabled for provider %q", modelName, providerName)
	}
	return buildRouteCandidate(provider.Name+"/"+model.Name, provider, provider.Protocol, model.Name, model.MaxTokens, "direct", 0), true, nil
}

func (r *routeResolver) routeFromRule(providers []domain.ProviderSnapshot, rule entity.RoutingRule, requestedModel string) (domain.Route, error) {
	candidate, err := r.candidateFromRule(providers, rule, requestedModel)
	if err != nil {
		return domain.Route{}, err
	}
	return candidate.Route(), nil
}

func (r *routeResolver) candidateFromRule(providers []domain.ProviderSnapshot, rule entity.RoutingRule, requestedModel string) (domain.RouteCandidate, error) {
	provider, ok := findProvider(providers, rule.TargetProviderID)
	if !ok {
		return domain.RouteCandidate{}, fmt.Errorf("routing rule %q targets missing or disabled provider %q", rule.Name, rule.TargetProviderID)
	}

	targetModel := strings.TrimSpace(rule.TargetModel)
	if targetModel == "" {
		targetModel = requestedModel
	}
	if targetModel == "" {
		return domain.RouteCandidate{}, fmt.Errorf("routing rule %q did not specify a target model", rule.Name)
	}

	model, ok := findModel(provider.Models, targetModel)
	if !ok {
		return domain.RouteCandidate{}, fmt.Errorf("routing rule %q targets missing or disabled model %q for provider %q", rule.Name, targetModel, provider.Name)
	}

	upstreamProtocol := rule.UpstreamProtocol
	if upstreamProtocol == "" {
		upstreamProtocol = provider.Protocol
	}

	return buildRouteCandidate(rule.Name, provider, upstreamProtocol, model.Name, model.MaxTokens, "routing_rule:"+rule.ID, rule.Priority), nil
}

func providerSnapshot(item entity.Provider, models []entity.ProviderModel) domain.ProviderSnapshot {
	snapshot := domain.ProviderSnapshot{
		ID:          item.ID,
		Name:        item.Name,
		Protocol:    item.Protocol,
		Vendor:      item.Vendor,
		BaseURL:     item.BaseURL,
		APIKey:      item.APIKeyCipher,
		OnlyStream:  item.OnlyStream,
		UserAgent:   item.UserAgent,
		Enabled:     item.Enabled,
		Description: item.Description,
		Models:      make([]domain.ProviderModelSnapshot, 0, len(models)),
	}
	for _, model := range models {
		if !model.Enabled {
			continue
		}
		snapshot.Models = append(snapshot.Models, domain.ProviderModelSnapshot{
			Name:      model.Name,
			MaxTokens: model.MaxTokens,
			Enabled:   model.Enabled,
		})
	}
	return snapshot
}

func ruleMatches(rule entity.RoutingRule, downstream constants.Protocol, requestedModel string) bool {
	if rule.MatchProtocol != "" && rule.MatchProtocol != downstream {
		return false
	}
	return modelPatternMatches(rule.MatchModelPattern, requestedModel)
}

func modelPatternMatches(pattern string, model string) bool {
	pattern = strings.TrimSpace(pattern)
	model = strings.TrimSpace(model)
	if pattern == "" {
		return model == ""
	}
	if pattern == "*" {
		return true
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return pattern == model
	}
	matched, err := filepath.Match(pattern, model)
	return err == nil && matched
}

func findProvider(providers []domain.ProviderSnapshot, key string) (domain.ProviderSnapshot, bool) {
	key = strings.TrimSpace(key)
	for _, provider := range providers {
		if provider.ID == key || provider.Name == key {
			return provider, true
		}
	}
	return domain.ProviderSnapshot{}, false
}

func findModel(models []domain.ProviderModelSnapshot, name string) (domain.ProviderModelSnapshot, bool) {
	name = strings.TrimSpace(name)
	for _, model := range models {
		if model.Name == name {
			return model, true
		}
	}
	return domain.ProviderModelSnapshot{}, false
}

func buildRoute(name string, provider domain.ProviderSnapshot, upstreamProtocol constants.Protocol, model string, maxTokens int, source string) domain.Route {
	return buildRouteCandidate(name, provider, upstreamProtocol, model, maxTokens, source, 0).Route()
}

func buildRouteCandidate(name string, provider domain.ProviderSnapshot, upstreamProtocol constants.Protocol, model string, maxTokens int, source string, priority int) domain.RouteCandidate {
	return domain.RouteCandidate{
		Name:             name,
		UpstreamProtocol: upstreamProtocol,
		Model:            model,
		DefaultMaxTokens: maxTokens,
		Source:           source,
		Priority:         priority,
		Provider:         provider,
		Endpoint: domain.ProviderEndpointSnapshot{
			ID:         defaultResourceID(provider, "endpoint"),
			ProviderID: provider.ID,
			BaseURL:    provider.BaseURL,
			Priority:   0,
			Weight:     1,
			Enabled:    true,
		},
		Credential: domain.ProviderCredentialSnapshot{
			ID:         defaultResourceID(provider, "credential"),
			ProviderID: provider.ID,
			APIKey:     provider.APIKey,
			Enabled:    true,
		},
	}
}

func defaultResourceID(provider domain.ProviderSnapshot, suffix string) string {
	key := provider.ID
	if key == "" {
		key = provider.Name
	}
	if key == "" {
		return "default-" + suffix
	}
	return key + ":default-" + suffix
}
