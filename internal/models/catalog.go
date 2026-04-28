package models

import "icoo_proxy/internal/consts"

type Route struct {
	Name     string          `json:"name"`
	Upstream consts.Protocol `json:"upstream"`
	Model    string          `json:"model"`
	Source   string          `json:"source,omitempty"`
}
