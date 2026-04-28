package models

import (
	"icoo_proxy/internal/consts"
	"time"
)

type RoutePolicyRecord struct {
	ID                 string          `json:"id"`
	DownstreamProtocol consts.Protocol `json:"downstream_protocol"`
	SupplierID         string          `json:"supplier_id"`
	SupplierName       string          `json:"supplier_name"`
	UpstreamProtocol   consts.Protocol `json:"upstream_protocol"`
	Enabled            bool            `json:"enabled"`
	UpdatedAt          string          `json:"updated_at"`
	CreatedAt          string          `json:"created_at"`
}

type RoutePolicyModel struct {
	ID                 string          `gorm:"primaryKey"`
	DownstreamProtocol consts.Protocol `gorm:"uniqueIndex"`
	SupplierID         string
	TargetModel        string
	Enabled            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (RoutePolicyModel) TableName() string {
	return "route_policies"
}

type UpsertInput struct {
	ID                 string          `json:"id"`
	DownstreamProtocol consts.Protocol `json:"downstream_protocol"`
	SupplierID         string          `json:"supplier_id"`
	Enabled            bool            `json:"enabled"`
}
