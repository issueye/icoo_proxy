package controller

import "github.com/issueye/icoo_proxy/bridge/internal/service"

type Controllers struct {
	Health        *HealthController
	Runtime       *RuntimeController
	Proxy         *ProxyController
	Provider      *ProviderController
	ProviderModel *ProviderModelController
	ModelCatalog  *ModelCatalogController
	Endpoint      *EndpointController
	RoutingRule   *RoutingRuleController
	APIKey        *APIKeyController
	Traffic       *TrafficController
	UIPreference  *UIPreferenceController
	Plugin        *PluginController
	ModelList     *ModelListController
}

func NewControllers(services service.Services) Controllers {
	return Controllers{
		Health:        NewHealthController(services.Runtime),
		Runtime:       NewRuntimeController(services.Runtime),
		Proxy:         NewProxyController(services.Proxy, services.Endpoint),
		Provider:      NewProviderController(services.Provider, services.ProviderChat),
		ProviderModel: NewProviderModelController(services.ProviderModel),
		ModelCatalog:  NewModelCatalogController(services.ModelCatalog),
		Endpoint:      NewEndpointController(services.Endpoint),
		RoutingRule:   NewRoutingRuleController(services.RoutingRule),
		APIKey:        NewAPIKeyController(services.Auth),
		Traffic:       NewTrafficController(services.Traffic),
		UIPreference:  NewUIPreferenceController(services.UIPreference),
		Plugin:        NewPluginController(services.Plugins),
		ModelList:     NewModelListController(services.ModelList),
	}
}
