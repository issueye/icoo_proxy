package repository

import (
	"context"

	"icoo_llm_bridge/internal/model/entity"
)

type ProviderRepository interface {
	List(ctx context.Context) ([]entity.Provider, error)
	Find(ctx context.Context, id string) (entity.Provider, error)
	Save(ctx context.Context, item *entity.Provider) error
	Delete(ctx context.Context, id string) error
}

type ProviderModelRepository interface {
	ListByProvider(ctx context.Context, providerID string) ([]entity.ProviderModel, error)
	Save(ctx context.Context, item *entity.ProviderModel) error
	Delete(ctx context.Context, providerID string, id string) error
}

type ModelCatalogRepository interface {
	List(ctx context.Context) ([]entity.ModelCatalogItem, error)
	Find(ctx context.Context, id string) (entity.ModelCatalogItem, error)
	Save(ctx context.Context, item *entity.ModelCatalogItem) error
	Delete(ctx context.Context, id string) error
}

type EndpointRepository interface {
	List(ctx context.Context) ([]entity.IngressEndpoint, error)
	Enabled(ctx context.Context) ([]entity.IngressEndpoint, error)
	Find(ctx context.Context, id string) (entity.IngressEndpoint, error)
	Save(ctx context.Context, item *entity.IngressEndpoint) error
	Delete(ctx context.Context, id string) error
}

type RoutingRuleRepository interface {
	List(ctx context.Context) ([]entity.RoutingRule, error)
	ListEnabled(ctx context.Context) ([]entity.RoutingRule, error)
	Find(ctx context.Context, id string) (entity.RoutingRule, error)
	Save(ctx context.Context, item *entity.RoutingRule) error
	Delete(ctx context.Context, id string) error
}

type APIKeyRepository interface {
	List(ctx context.Context) ([]entity.APIKey, error)
	ListEnabled(ctx context.Context) ([]entity.APIKey, error)
	Find(ctx context.Context, id string) (entity.APIKey, error)
	Save(ctx context.Context, item *entity.APIKey) error
	Delete(ctx context.Context, id string) error
}

type TrafficRepository interface {
	Record(ctx context.Context, item *entity.TrafficRecord) error
	List(ctx context.Context, limit int) ([]entity.TrafficRecord, error)
	Clear(ctx context.Context) error
}

type UIPreferenceRepository interface {
	Find(ctx context.Context, key string) (entity.UIPreference, error)
	Save(ctx context.Context, item *entity.UIPreference) error
}
