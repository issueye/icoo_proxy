package repository

import (
	"context"

	"icoo_llm_bridge/internal/model/entity"

	"gorm.io/gorm"
)

type Repositories struct {
	Provider      ProviderRepository
	ProviderModel ProviderModelRepository
	Endpoint      EndpointRepository
	RoutingRule   RoutingRuleRepository
	APIKey        APIKeyRepository
	Traffic       TrafficRepository
	UIPreference  UIPreferenceRepository
}

func NewRepositories(db *gorm.DB, trafficDB *gorm.DB) Repositories {
	if trafficDB == nil {
		trafficDB = db
	}
	return Repositories{
		Provider:      &gormProviderRepository{db: db},
		ProviderModel: &gormProviderModelRepository{db: db},
		Endpoint:      &gormEndpointRepository{db: db},
		RoutingRule:   &gormRoutingRuleRepository{db: db},
		APIKey:        &gormAPIKeyRepository{db: db},
		Traffic:       &gormTrafficRepository{db: trafficDB},
		UIPreference:  &gormUIPreferenceRepository{db: db},
	}
}

// ProviderRepository 供应商仓库
type gormProviderRepository struct{ db *gorm.DB }

func (r *gormProviderRepository) List(ctx context.Context) ([]entity.Provider, error) {
	var items []entity.Provider
	return items, r.db.WithContext(ctx).Order("name asc").Find(&items).Error
}

func (r *gormProviderRepository) Find(ctx context.Context, id string) (entity.Provider, error) {
	var item entity.Provider
	return item, r.db.WithContext(ctx).First(&item, "id = ?", id).Error
}

func (r *gormProviderRepository) Save(ctx context.Context, item *entity.Provider) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *gormProviderRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Provider{}, "id = ?", id).Error
}

// ProviderModelRepository 供应商模型仓库
type gormProviderModelRepository struct{ db *gorm.DB }

func (r *gormProviderModelRepository) ListByProvider(ctx context.Context, providerID string) ([]entity.ProviderModel, error) {
	var items []entity.ProviderModel
	return items, r.db.WithContext(ctx).Where("provider_id = ?", providerID).Order("name asc").Find(&items).Error
}

func (r *gormProviderModelRepository) Save(ctx context.Context, item *entity.ProviderModel) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *gormProviderModelRepository) Delete(ctx context.Context, providerID string, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProviderModel{}, "provider_id = ? AND id = ?", providerID, id).Error
}

// EndpointRepository 入口仓库
type gormEndpointRepository struct{ db *gorm.DB }

func (r *gormEndpointRepository) List(ctx context.Context) ([]entity.IngressEndpoint, error) {
	var items []entity.IngressEndpoint
	return items, r.db.WithContext(ctx).Order("built_in desc, path asc").Find(&items).Error
}

func (r *gormEndpointRepository) Enabled(ctx context.Context) ([]entity.IngressEndpoint, error) {
	var items []entity.IngressEndpoint
	return items, r.db.WithContext(ctx).Where("enabled = ?", true).Order("built_in desc, path asc").Find(&items).Error
}

func (r *gormEndpointRepository) Find(ctx context.Context, id string) (entity.IngressEndpoint, error) {
	var item entity.IngressEndpoint
	return item, r.db.WithContext(ctx).First(&item, "id = ?", id).Error
}

func (r *gormEndpointRepository) Save(ctx context.Context, item *entity.IngressEndpoint) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *gormEndpointRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.IngressEndpoint{}, "id = ?", id).Error
}

// RoutingRuleRepository 路由规则仓库
type gormRoutingRuleRepository struct{ db *gorm.DB }

func (r *gormRoutingRuleRepository) List(ctx context.Context) ([]entity.RoutingRule, error) {
	var items []entity.RoutingRule
	return items, r.db.WithContext(ctx).Order("priority asc, name asc").Find(&items).Error
}

func (r *gormRoutingRuleRepository) ListEnabled(ctx context.Context) ([]entity.RoutingRule, error) {
	var items []entity.RoutingRule
	return items, r.db.WithContext(ctx).Where("enabled = ?", true).Order("priority asc").Find(&items).Error
}

func (r *gormRoutingRuleRepository) Find(ctx context.Context, id string) (entity.RoutingRule, error) {
	var item entity.RoutingRule
	return item, r.db.WithContext(ctx).First(&item, "id = ?", id).Error
}

func (r *gormRoutingRuleRepository) Save(ctx context.Context, item *entity.RoutingRule) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *gormRoutingRuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.RoutingRule{}, "id = ?", id).Error
}

// APIKeyRepository API密钥仓库
type gormAPIKeyRepository struct{ db *gorm.DB }

func (r *gormAPIKeyRepository) List(ctx context.Context) ([]entity.APIKey, error) {
	var items []entity.APIKey
	return items, r.db.WithContext(ctx).Order("created_at desc").Find(&items).Error
}

func (r *gormAPIKeyRepository) ListEnabled(ctx context.Context) ([]entity.APIKey, error) {
	var items []entity.APIKey
	return items, r.db.WithContext(ctx).Where("enabled = ?", true).Find(&items).Error
}

func (r *gormAPIKeyRepository) Find(ctx context.Context, id string) (entity.APIKey, error) {
	var item entity.APIKey
	return item, r.db.WithContext(ctx).First(&item, "id = ?", id).Error
}

func (r *gormAPIKeyRepository) Save(ctx context.Context, item *entity.APIKey) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *gormAPIKeyRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.APIKey{}, "id = ?", id).Error
}

// TrafficRepository 流量记录仓库
type gormTrafficRepository struct{ db *gorm.DB }

func (r *gormTrafficRepository) Record(ctx context.Context, item *entity.TrafficRecord) error {
	return r.db.Session(&gorm.Session{
		NewDB:                  true,
		SkipDefaultTransaction: true,
	}).WithContext(ctx).Create(item).Error
}

func (r *gormTrafficRepository) List(ctx context.Context, limit int) ([]entity.TrafficRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var items []entity.TrafficRecord
	return items, r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&items).Error
}

func (r *gormTrafficRepository) Clear(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("1 = 1").Delete(&entity.TrafficRecord{}).Error
}

// UIPreferenceRepository 用户偏好仓库
type gormUIPreferenceRepository struct{ db *gorm.DB }

func (r *gormUIPreferenceRepository) Find(ctx context.Context, key string) (entity.UIPreference, error) {
	var item entity.UIPreference
	return item, r.db.WithContext(ctx).First(&item, "key = ?", key).Error
}

func (r *gormUIPreferenceRepository) Save(ctx context.Context, item *entity.UIPreference) error {
	return r.db.WithContext(ctx).Save(item).Error
}
