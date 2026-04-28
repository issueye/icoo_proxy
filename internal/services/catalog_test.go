package services

import (
	"testing"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

type testPolicyResolver struct {
	suppliers map[consts.Protocol]models.SupplierRecord
}

func (r testPolicyResolver) ResolveEnabledSupplierByDownstream(downstream consts.Protocol) (models.SupplierRecord, bool) {
	item, ok := r.suppliers[downstream]
	return item, ok
}

func TestCatalogResolveQualifiedSupplierModel(t *testing.T) {
	catalog, err := NewCatalogFromEntries(map[consts.Protocol]string{
		consts.ProtocolOpenAIChat: "openai-responses:gpt-4.1",
	}, "")
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}
	cache := NewSupplierModelCache()
	if err := cache.Rebuild([]models.SupplierRecord{{
		Name:     "VendorA",
		Protocol: consts.ProtocolAnthropic,
		Enabled:  true,
		Models:   []string{"claude-3-7-sonnet"},
	}}); err != nil {
		t.Fatalf("rebuild cache: %v", err)
	}
	catalog.SetSupplierModelCache(cache)

	route, err := catalog.Resolve(consts.ProtocolOpenAIChat, "VendorA/claude-3-7-sonnet")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if route.Upstream != consts.ProtocolAnthropic {
		t.Fatalf("upstream = %s, want %s", route.Upstream, consts.ProtocolAnthropic)
	}
	if route.Model != "claude-3-7-sonnet" {
		t.Fatalf("model = %q, want %q", route.Model, "claude-3-7-sonnet")
	}
	if route.Source != "qualified-supplier-model" {
		t.Fatalf("source = %q, want %q", route.Source, "qualified-supplier-model")
	}
}

func TestCatalogResolveRoutePolicyModel(t *testing.T) {
	catalog, err := NewCatalogFromEntries(map[consts.Protocol]string{
		consts.ProtocolOpenAIChat: "openai-responses:gpt-4.1",
	}, "")
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}
	cache := NewSupplierModelCache()
	if err := cache.Rebuild([]models.SupplierRecord{{
		Name:     "VendorB",
		Protocol: consts.ProtocolOpenAIResponses,
		Enabled:  true,
		Models:   []string{"gpt-4.1-mini"},
	}}); err != nil {
		t.Fatalf("rebuild cache: %v", err)
	}
	catalog.SetSupplierModelCache(cache)
	catalog.SetPolicyResolver(testPolicyResolver{suppliers: map[consts.Protocol]models.SupplierRecord{
		consts.ProtocolOpenAIChat: {
			Name:     "VendorB",
			Protocol: consts.ProtocolOpenAIResponses,
			Enabled:  true,
		},
	}})

	route, err := catalog.Resolve(consts.ProtocolOpenAIChat, "gpt-4.1-mini")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if route.Upstream != consts.ProtocolOpenAIResponses {
		t.Fatalf("upstream = %s, want %s", route.Upstream, consts.ProtocolOpenAIResponses)
	}
	if route.Model != "gpt-4.1-mini" {
		t.Fatalf("model = %q, want %q", route.Model, "gpt-4.1-mini")
	}
	if route.Source != "route-policy-supplier-model" {
		t.Fatalf("source = %q, want %q", route.Source, "route-policy-supplier-model")
	}
}

func TestCatalogResolveFallsBackToAliasAndDefault(t *testing.T) {
	catalog, err := NewCatalogFromEntries(map[consts.Protocol]string{
		consts.ProtocolOpenAIChat: "openai-responses:gpt-4.1",
	}, "alias-a=anthropic:claude-3-7-sonnet")
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}

	aliasRoute, err := catalog.Resolve(consts.ProtocolOpenAIChat, "alias-a")
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if aliasRoute.Upstream != consts.ProtocolAnthropic {
		t.Fatalf("alias upstream = %s, want %s", aliasRoute.Upstream, consts.ProtocolAnthropic)
	}
	if aliasRoute.Source != "alias" {
		t.Fatalf("alias source = %q, want %q", aliasRoute.Source, "alias")
	}

	defaultRoute, err := catalog.Resolve(consts.ProtocolOpenAIChat, "custom-model")
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	if defaultRoute.Upstream != consts.ProtocolOpenAIResponses {
		t.Fatalf("default upstream = %s, want %s", defaultRoute.Upstream, consts.ProtocolOpenAIResponses)
	}
	if defaultRoute.Model != "custom-model" {
		t.Fatalf("default model = %q, want %q", defaultRoute.Model, "custom-model")
	}
	if defaultRoute.Source != "default-fallback" {
		t.Fatalf("default source = %q, want %q", defaultRoute.Source, "default-fallback")
	}
}
