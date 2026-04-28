package services

import (
	"sync"
	"testing"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

func TestSupplierModelCacheResolveQualified(t *testing.T) {
	cache := NewSupplierModelCache()
	if err := cache.Rebuild([]models.SupplierRecord{{
		ID:       "s1",
		Name:     "OpenAI",
		Protocol: consts.ProtocolOpenAIResponses,
		Enabled:  true,
		Models:   []string{"gpt-4.1", "gpt-4o-mini"},
	}}); err != nil {
		t.Fatalf("rebuild cache: %v", err)
	}

	route, ok := cache.ResolveQualified("  openai / gpt-4.1 ")
	if !ok {
		t.Fatalf("expected qualified model to resolve")
	}
	if route.Upstream != consts.ProtocolOpenAIResponses {
		t.Fatalf("upstream = %s, want %s", route.Upstream, consts.ProtocolOpenAIResponses)
	}
	if route.Model != "gpt-4.1" {
		t.Fatalf("model = %q, want %q", route.Model, "gpt-4.1")
	}
	if route.Source != "qualified-supplier-model" {
		t.Fatalf("source = %q, want %q", route.Source, "qualified-supplier-model")
	}
}

func TestSupplierModelCacheRejectsInvalidQualifiedModel(t *testing.T) {
	cache := NewSupplierModelCache()
	if _, ok := cache.ResolveQualified("/gpt-4.1"); ok {
		t.Fatal("expected invalid qualified model to fail")
	}
	if _, ok := cache.ResolveQualified("openai/"); ok {
		t.Fatal("expected invalid qualified model to fail")
	}
	if _, ok := cache.ResolveQualified("a/b/c"); ok {
		t.Fatal("expected invalid qualified model to fail")
	}
}

func TestSupplierModelCacheResolveBySupplierAndModel(t *testing.T) {
	cache := NewSupplierModelCache()
	if err := cache.Rebuild([]models.SupplierRecord{{
		ID:       "s1",
		Name:     "Claude",
		Protocol: consts.ProtocolAnthropic,
		Enabled:  true,
		Models:   []string{"claude-sonnet-4-5"},
	}}); err != nil {
		t.Fatalf("rebuild cache: %v", err)
	}

	route, ok := cache.ResolveBySupplierAndModel(" claude ", " claude-sonnet-4-5 ")
	if !ok {
		t.Fatalf("expected supplier/model lookup to resolve")
	}
	if route.Upstream != consts.ProtocolAnthropic {
		t.Fatalf("upstream = %s, want %s", route.Upstream, consts.ProtocolAnthropic)
	}
	if route.Source != "route-policy-supplier-model" {
		t.Fatalf("source = %q, want %q", route.Source, "route-policy-supplier-model")
	}
}

func TestSupplierModelCacheConcurrentRead(t *testing.T) {
	cache := NewSupplierModelCache()
	if err := cache.Rebuild([]models.SupplierRecord{{
		ID:       "s1",
		Name:     "OpenAI",
		Protocol: consts.ProtocolOpenAIChat,
		Enabled:  true,
		Models:   []string{"gpt-4.1"},
	}}); err != nil {
		t.Fatalf("rebuild cache: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := cache.ResolveQualified("openai/gpt-4.1"); !ok {
				t.Error("expected concurrent qualified lookup to resolve")
			}
		}()
	}
	wg.Wait()
}
