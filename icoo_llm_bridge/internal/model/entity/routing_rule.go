package entity

import (
	"time"

	"icoo_llm_bridge/internal/constants"
)

type RoutingRule struct {
	ID                string `gorm:"primaryKey"`
	Name              string `gorm:"not null"`
	Priority          int    `gorm:"index"`
	MatchProtocol     constants.Protocol
	MatchModelPattern string
	UpstreamProtocol  constants.Protocol
	TargetProviderID  string `gorm:"index"`
	TargetModel       string
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (RoutingRule) TableName() string {
	return "routing_rules"
}
