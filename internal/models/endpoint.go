package models

import (
	"icoo_proxy/internal/consts"
	"time"
)

type EndpointModel struct {
	ID          string `gorm:"primaryKey"`
	Path        string `gorm:"uniqueIndex"`
	Protocol    consts.Protocol
	Description string
	Enabled     bool
	BuiltIn     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (EndpointModel) TableName() string {
	return "endpoints"
}

type EndpointRecord struct {
	ID          string          `json:"id"`
	Path        string          `json:"path"`
	Protocol    consts.Protocol `json:"protocol"`
	Description string          `json:"description"`
	Enabled     bool            `json:"enabled"`
	BuiltIn     bool            `json:"built_in"`
	UpdatedAt   string          `json:"updated_at"`
	CreatedAt   string          `json:"created_at"`
}

type EndpointUpsertInput struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Protocol    string `json:"protocol"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}
