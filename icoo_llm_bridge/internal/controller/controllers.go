package controller

import "icoo_llm_bridge/internal/service"

type Controllers struct {
	Health        *HealthController
	Runtime       *RuntimeController
	Proxy         *ProxyController
	Provider      *ProviderController
	ProviderModel *ProviderModelController
	Endpoint      *EndpointController
	RoutingRule   *RoutingRuleController
	APIKey        *APIKeyController
	Traffic       *TrafficController
	UIPreference  *UIPreferenceController
}

func NewControllers(services service.Services) Controllers {
	return Controllers{
		Health:        NewHealthController(services.Runtime),
		Runtime:       NewRuntimeController(services.Runtime),
		Proxy:         NewProxyController(services.Proxy, services.Endpoint),
		Provider:      NewProviderController(services.Provider, services.ProviderChat),
		ProviderModel: NewProviderModelController(services.ProviderModel),
		Endpoint:      NewEndpointController(services.Endpoint),
		RoutingRule:   NewRoutingRuleController(services.RoutingRule),
		APIKey:        NewAPIKeyController(services.Auth),
		Traffic:       NewTrafficController(services.Traffic),
		UIPreference:  NewUIPreferenceController(services.UIPreference),
	}
}
