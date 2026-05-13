package service

import (
	"log/slog"

	"icoo_llm_bridge/internal/config"
	"icoo_llm_bridge/internal/repository"
	"icoo_llm_bridge/internal/utils/ai_llm_proxy"
)

type Services struct {
	Auth          AuthService
	Runtime       RuntimeService
	Endpoint      EndpointService
	Provider      ProviderService
	ProviderModel ProviderModelService
	RoutingRule   RoutingRuleService
	Routing       RouteResolver
	Traffic       TrafficService
	Proxy         ProxyService
}

type Deps struct {
	Config    config.Config
	Logger    *slog.Logger
	Repos     repository.Repositories
	Converter ai_llm_proxy.Converter
}

func NewServices(deps Deps) Services {
	auth := NewAuthService(deps.Repos.APIKey)
	endpoints := NewEndpointService(deps.Repos.Endpoint)
	providers := NewProviderService(deps.Repos.Provider)
	providerModels := NewProviderModelService(deps.Repos.ProviderModel)
	rules := NewRoutingRuleService(deps.Repos.RoutingRule)
	routing := NewRouteResolver(deps.Repos.Provider, deps.Repos.ProviderModel, deps.Repos.RoutingRule)
	traffic := NewTrafficService(deps.Repos.Traffic)
	runtime := NewRuntimeService(deps.Config, endpoints)
	proxy := NewProxyService(deps.Config, deps.Logger, deps.Converter, auth, routing, traffic)
	return Services{
		Auth:          auth,
		Runtime:       runtime,
		Endpoint:      endpoints,
		Provider:      providers,
		ProviderModel: providerModels,
		RoutingRule:   rules,
		Routing:       routing,
		Traffic:       traffic,
		Proxy:         proxy,
	}
}
