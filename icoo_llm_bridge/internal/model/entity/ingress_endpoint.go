package entity

import (
	"time"

	"icoo_llm_bridge/internal/constants"
)

type IngressEndpoint struct {
	ID                 string `gorm:"primaryKey"`
	Path               string `gorm:"uniqueIndex;not null"`
	DownstreamProtocol constants.Protocol
	Enabled            bool
	Protected          bool
	BuiltIn            bool
	Description        string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (IngressEndpoint) TableName() string {
	return "ingress_endpoints"
}
