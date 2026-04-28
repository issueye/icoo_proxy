package services

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

type DownstreamPolicyResolver interface {
	ResolveEnabledSupplierByDownstream(downstream consts.Protocol) (models.SupplierRecord, bool)
}

type CatalogService struct {
	defaults      map[consts.Protocol]models.Route
	aliases       map[string]models.Route
	supplierCache *SupplierModelCache
	policyLookup  DownstreamPolicyResolver
}

func NewCatalogService() (*CatalogService, error) {
	defaults := make(map[consts.Protocol]models.Route)
	aliases := make(map[string]models.Route)
	return &CatalogService{
		defaults: defaults,
		aliases:  aliases,
	}, nil
}

func NewCatalogFromEntries(defaults map[consts.Protocol]string, aliasEntries string) (*CatalogService, error) {
	catalog, err := NewCatalogService()
	if err != nil {
		return nil, err
	}
	for downstream, target := range defaults {
		if strings.TrimSpace(target) == "" {
			continue
		}
		route, err := catalogParseTarget(downstream.ToString(), target)
		if err != nil {
			return nil, err
		}
		catalog.defaults[downstream] = withRouteSource(route, "default")
	}
	for _, entry := range catalogSplitEntries(aliasEntries) {
		name, target, found := strings.Cut(entry, "=")
		name = strings.TrimSpace(name)
		if !found || name == "" {
			return nil, fmt.Errorf("invalid model route %q", entry)
		}
		route, err := catalogParseTarget(name, target)
		if err != nil {
			return nil, err
		}
		catalog.aliases[name] = withRouteSource(route, "alias")
	}
	return catalog, nil
}

func (c *CatalogService) SetSupplierModelCache(cache *SupplierModelCache) {
	c.supplierCache = cache
}

func (c *CatalogService) SetPolicyResolver(resolver DownstreamPolicyResolver) {
	c.policyLookup = resolver
}

func (c *CatalogService) Resolve(downstream consts.Protocol, requestedModel string) (models.Route, error) {
	model := strings.TrimSpace(requestedModel)
	slog.Info("下游请求模型和协议", "model", model, "downstream", downstream)

	defaultRoute, hasDefault := c.defaults[downstream]
	if model == "" {
		if !hasDefault {
			return models.Route{}, fmt.Errorf("missing model and no default route for %s", downstream)
		}
		return defaultRoute, nil
	}

	if route, ok := c.resolveQualifiedSupplierModel(model); ok {
		return withRouteSource(route, "qualified-supplier-model"), nil
	}
	if route, ok := c.resolveRoutePolicyModel(downstream, model); ok {
		return withRouteSource(route, "route-policy-supplier-model"), nil
	}
	if route, ok := c.aliases[model]; ok {
		return withRouteSource(route, "alias"), nil
	}
	if !hasDefault {
		return models.Route{}, fmt.Errorf("requested model %q has no default route for %s", model, downstream)
	}

	copyRoute := defaultRoute
	copyRoute.Name = model
	copyRoute.Model = model
	copyRoute.Source = "default-fallback"

	slog.Info("最终路由", "copyRoute", copyRoute)
	return copyRoute, nil
}

func (c *CatalogService) resolveQualifiedSupplierModel(model string) (models.Route, bool) {
	if c == nil || c.supplierCache == nil {
		return models.Route{}, false
	}
	return c.supplierCache.ResolveQualified(model)
}

func (c *CatalogService) resolveRoutePolicyModel(downstream consts.Protocol, model string) (models.Route, bool) {
	if c == nil || c.supplierCache == nil || c.policyLookup == nil {
		return models.Route{}, false
	}
	supplier, ok := c.policyLookup.ResolveEnabledSupplierByDownstream(downstream)
	if !ok {
		return models.Route{}, false
	}
	return c.supplierCache.ResolveBySupplierAndModel(supplier.Name, model)
}

func (c *CatalogService) Defaults() []models.Route {
	items := make([]models.Route, 0, len(c.defaults))
	for protocol, route := range c.defaults {
		copyRoute := route
		copyRoute.Name = string(protocol)
		items = append(items, copyRoute)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func (c *CatalogService) Aliases() []models.Route {
	items := make([]models.Route, 0, len(c.aliases))
	for _, route := range c.aliases {
		items = append(items, route)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func withRouteSource(route models.Route, source string) models.Route {
	route.Source = source
	return route
}

func catalogSplitEntries(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	items := make([]string, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(field)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}

func catalogParseTarget(name, raw string) (models.Route, error) {
	value := strings.TrimSpace(raw)
	protocolRaw, model, found := strings.Cut(value, ":")
	if !found {
		return models.Route{}, fmt.Errorf("invalid route %q for %s", raw, name)
	}
	protocol := consts.Protocol(strings.TrimSpace(protocolRaw))
	switch protocol {
	case consts.ProtocolAnthropic, consts.ProtocolOpenAIChat, consts.ProtocolOpenAIResponses:
	default:
		return models.Route{}, fmt.Errorf("unsupported upstream protocol %q for %s", protocol, name)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return models.Route{}, fmt.Errorf("missing model for %s", name)
	}
	return models.Route{
		Name:     name,
		Upstream: protocol,
		Model:    model,
	}, nil
}
