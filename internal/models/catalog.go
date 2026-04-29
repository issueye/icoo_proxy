package models

import "icoo_proxy/internal/consts"

type Route struct {
	Name             string          `json:"name"`
	Upstream         consts.Protocol `json:"upstream"`
	Model            string          `json:"model"`
	DefaultMaxTokens int             `json:"default_max_tokens,omitempty"`
	Source           string          `json:"source,omitempty"`
	Supplier         Snapshot        `json:"-"`
}
