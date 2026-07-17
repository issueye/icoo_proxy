package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/bridge/internal/model/entity"
	"github.com/issueye/icoo_proxy/common/constants"
)

var errModelListNotFound = errors.New("not found")

type fakeProvidersForModels struct {
	items []entity.Provider
}

func (f *fakeProvidersForModels) List(ctx context.Context) ([]entity.Provider, error) {
	return append([]entity.Provider(nil), f.items...), nil
}
func (f *fakeProvidersForModels) Find(ctx context.Context, id string) (entity.Provider, error) {
	for _, p := range f.items {
		if p.ID == id {
			return p, nil
		}
	}
	return entity.Provider{}, errModelListNotFound
}
func (f *fakeProvidersForModels) Save(ctx context.Context, item *entity.Provider) error {
	return nil
}
func (f *fakeProvidersForModels) Delete(ctx context.Context, id string) error { return nil }

type fakeModelsForList struct {
	byProvider map[string][]entity.ProviderModel
}

func (f *fakeModelsForList) ListByProvider(ctx context.Context, providerID string) ([]entity.ProviderModel, error) {
	return append([]entity.ProviderModel(nil), f.byProvider[providerID]...), nil
}
func (f *fakeModelsForList) Save(ctx context.Context, item *entity.ProviderModel) error { return nil }
func (f *fakeModelsForList) Delete(ctx context.Context, providerID string, id string) error {
	return nil
}

func TestModelListService_ListModels(t *testing.T) {
	now := time.Now()
	svc := NewModelListService(
		config.Config{AllowLocalWithoutAuth: true},
		allowAuth{},
		&fakeProvidersForModels{items: []entity.Provider{
			{ID: "p1", Name: "测试", Protocol: constants.ProtocolOpenAIChat, Vendor: constants.VendorPlugin, Enabled: true},
			{ID: "p2", Name: "OpenAI", Protocol: constants.ProtocolOpenAIResponses, Vendor: constants.VendorOpenAI, Enabled: true},
			{ID: "p3", Name: "disabled", Enabled: false},
		}},
		&fakeModelsForList{byProvider: map[string][]entity.ProviderModel{
			"p1": {
				{ID: "m1", ProviderID: "p1", Name: "grok-4.5", MaxTokens: 131072, Enabled: true, CreatedAt: now},
				{ID: "m2", ProviderID: "p1", Name: "grok-4", MaxTokens: 131072, Enabled: false, CreatedAt: now},
			},
			"p2": {
				{ID: "m3", ProviderID: "p2", Name: "gpt-4o", MaxTokens: 128000, Enabled: true, CreatedAt: now},
				{ID: "m4", ProviderID: "p2", Name: "grok-4.5", MaxTokens: 1, Enabled: true, CreatedAt: now}, // name collision
			},
		}},
	)

	res, err := svc.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Object != "list" {
		t.Fatalf("object = %q", res.Object)
	}

	ids := map[string]OpenAIModelRef{}
	for _, m := range res.Data {
		ids[m.ID] = m
	}

	// Short ids: first provider wins for grok-4.5 (测试 sorts after OpenAI? "OpenAI" < "测试" by lower)
	// sort is by ToLower name: "OpenAI" then "disabled" then "测试" — OpenAI first.
	// So short grok-4.5 is owned by OpenAI.
	if m, ok := ids["grok-4.5"]; !ok {
		t.Fatal("missing short id grok-4.5")
	} else if m.OwnedBy != "OpenAI" {
		t.Fatalf("grok-4.5 owned_by = %q, want OpenAI (first sorted provider)", m.OwnedBy)
	}
	if _, ok := ids["gpt-4o"]; !ok {
		t.Fatal("missing gpt-4o")
	}
	if _, ok := ids["测试/grok-4.5"]; !ok {
		t.Fatal("missing direct route 测试/grok-4.5")
	}
	if _, ok := ids["OpenAI/gpt-4o"]; !ok {
		t.Fatal("missing direct route OpenAI/gpt-4o")
	}
	// disabled model omitted
	if _, ok := ids["grok-4"]; ok {
		t.Fatal("disabled model should be omitted")
	}

	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:18181/v1/models", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	if !svc.Authorize(req) {
		t.Fatal("local without auth should allow")
	}
}
