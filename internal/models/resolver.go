package models

import "icoo_proxy/internal/consts"

type Resolver interface {
	Resolve(id string) (Snapshot, bool)
}

type Snapshot struct {
	ID           string
	Name         string
	Protocol     consts.Protocol
	Vendor       consts.Vendor
	BaseURL      string
	APIKey       string
	OnlyStream   bool
	UserAgent    string
	IsEnabled    bool
	Models       []string
	DefaultModel string
}
