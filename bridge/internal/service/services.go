package service

import (
	"log/slog"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/bridge/internal/repository"
	"github.com/issueye/icoo_proxy/common/ai_llm_proxy"
)

type Services struct {
	Auth          AuthService
	Runtime       RuntimeService
	Endpoint      EndpointService
	Provider      ProviderService
	ProviderModel ProviderModelService
	ModelCatalog  ModelCatalogService
	ProviderChat  ProviderChatService
	RoutingRule   RoutingRuleService
	Routing       RouteResolver
	Traffic       TrafficService
	UIPreference  UIPreferenceService
	Proxy         ProxyService
	Plugins       PluginService
	ModelList     ModelListService
}

type Deps struct {
	Config    config.Config
	Logger    *slog.Logger
	Repos     repository.Repositories
	Converter ai_llm_proxy.Converter
	Plugins   PluginRuntime // optional process-plugin host
}

func NewServices(deps Deps) Services {
	tracker := NewRequestTracker()
	auth := NewAuthService(deps.Repos.APIKey)
	endpoints := NewEndpointService(deps.Repos.Endpoint)
	providers := NewProviderService(deps.Repos.Provider)
	providerModels := NewProviderModelService(deps.Repos.ProviderModel, deps.Repos.Provider, deps.Plugins)
	modelCatalog := NewModelCatalogService(deps.Repos.ModelCatalog)
	providerChat := NewProviderChatService(deps.Repos.Provider, deps.Repos.ProviderModel, deps.Plugins)
	rules := NewRoutingRuleService(deps.Repos.RoutingRule, tracker)
	// Hold the concrete resolver so its route cache can be wired into the admin
	// services below; it still satisfies the RouteResolver interface.
	resolver := newRouteResolver(deps.Repos.Provider, deps.Repos.ProviderModel, deps.Repos.RoutingRule)

	// Wire the resolver's route cache into the mutating admin services so any
	// provider/model/rule write immediately drops the cached snapshot and the
	// next proxied request re-reads fresh data (write-through invalidation).
	invalidator := resolver.cache
	providers.SetCacheInvalidator(invalidator)
	providerModels.SetCacheInvalidator(invalidator)
	rules.SetCacheInvalidator(invalidator)

	traffic := NewTrafficService(deps.Repos.Traffic)
	uiPreference := NewUIPreferenceService(deps.Repos.UIPreference)
	runtime := NewRuntimeService(deps.Config, endpoints)
	proxy := NewProxyService(deps.Config, deps.Logger, deps.Converter, auth, resolver, traffic, tracker, deps.Plugins)
	plugins := NewPluginService(deps.Plugins)
	modelList := NewModelListService(deps.Config, auth, deps.Repos.Provider, deps.Repos.ProviderModel)
	return Services{
		Auth:          auth,
		Runtime:       runtime,
		Endpoint:      endpoints,
		Provider:      providers,
		ProviderModel: providerModels,
		ModelCatalog:  modelCatalog,
		ProviderChat:  providerChat,
		RoutingRule:   rules,
		Routing:       resolver,
		Traffic:       traffic,
		UIPreference:  uiPreference,
		Proxy:         proxy,
		Plugins:       plugins,
		ModelList:     modelList,
	}
}
