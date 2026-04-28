package models

import (
	"icoo_proxy/internal/consts"
	"time"
)

type ModelAliasRecord struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	SupplierID       string          `json:"supplier_id"`
	SupplierName     string          `json:"supplier_name"`
	UpstreamProtocol consts.Protocol `json:"upstream_protocol"`
	Model            string          `json:"model"`
	Enabled          bool            `json:"enabled"`
	UpdatedAt        string          `json:"updated_at"`
	CreatedAt        string          `json:"created_at"`
}

type ModelAliasModel struct {
	ID         string `gorm:"primaryKey"`
	Name       string `gorm:"uniqueIndex"`
	SupplierID string
	Model      string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ModelAliasModel) TableName() string {
	return "model_aliases"
}

type ModelAliasUpsertInput struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SupplierID string `json:"supplier_id"`
	Model      string `json:"model"`
	Enabled    bool   `json:"enabled"`
}
