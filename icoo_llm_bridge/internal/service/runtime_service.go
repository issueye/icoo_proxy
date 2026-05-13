package service

import (
	"context"

	"icoo_llm_bridge/internal/config"
)

const Version = "0.1.0"

type runtimeService struct {
	cfg       config.Config
	endpoints EndpointService
}

func NewRuntimeService(cfg config.Config, endpoints EndpointService) RuntimeService {
	return &runtimeService{cfg: cfg, endpoints: endpoints}
}

func (s *runtimeService) State(ctx context.Context) State {
	paths := []string{"/healthz", "/readyz", "/api/v1/runtime/state"}
	if s.endpoints != nil {
		if items, err := s.endpoints.Enabled(ctx); err == nil {
			for _, item := range items {
				paths = append(paths, item.Path)
			}
		}
	}
	return State{
		Service:    "icoo_llm_bridge",
		Version:    Version,
		Running:    true,
		ListenAddr: s.cfg.Addr(),
		Paths:      paths,
	}
}
