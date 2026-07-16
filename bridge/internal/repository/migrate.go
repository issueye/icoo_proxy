package repository

import (
	"github.com/issueye/icoo_proxy/bridge/internal/model/entity"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.Provider{},
		&entity.ProviderModel{},
		&entity.ModelCatalogItem{},
		&entity.IngressEndpoint{},
		&entity.RoutingRule{},
		&entity.APIKey{},
		&entity.UIPreference{},
	)
}

func AutoMigrateTraffic(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.TrafficRecord{},
	)
}
