package service

import (
	"context"
	"net/http"

	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/domain"
	"github.com/issueye/icoo_proxy/bridge/internal/model/entity"
)

type RuntimeService interface {
	State(ctx context.Context) State
}

type EndpointService interface {
	List(ctx context.Context) ([]entity.IngressEndpoint, error)
	Enabled(ctx context.Context) ([]entity.IngressEndpoint, error)
	Upsert(ctx context.Context, input EndpointUpsertInput) (entity.IngressEndpoint, error)
	Delete(ctx context.Context, id string) error
}

type ProxyService interface {
	Handle(w http.ResponseWriter, r *http.Request, downstream constants.Protocol)
}

type RouteResolver interface {
	Resolve(ctx context.Context, downstream constants.Protocol, requestedModel string) (domain.Route, error)
}

type State struct {
	Service    string   `json:"service"`
	Version    string   `json:"version"`
	Running    bool     `json:"running"`
	ListenAddr string   `json:"listen_addr"`
	Paths      []string `json:"paths"`
}
