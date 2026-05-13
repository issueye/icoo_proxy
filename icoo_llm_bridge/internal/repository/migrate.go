package repository

import (
	"icoo_llm_bridge/internal/model/entity"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.Provider{},
		&entity.ProviderModel{},
		&entity.IngressEndpoint{},
		&entity.RoutingRule{},
		&entity.APIKey{},
		&entity.TrafficRecord{},
		&entity.UIPreference{},
	)
}
