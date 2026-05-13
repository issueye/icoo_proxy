package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/entity"
	"icoo_llm_bridge/internal/repository"
	"icoo_llm_bridge/internal/utils/idgen"
)

type endpointService struct {
	repo repository.EndpointRepository
}

func NewEndpointService(repo repository.EndpointRepository) EndpointService {
	return &endpointService{repo: repo}
}

func (s *endpointService) List(ctx context.Context) ([]entity.IngressEndpoint, error) {
	return s.repo.List(ctx)
}

func (s *endpointService) Enabled(ctx context.Context) ([]entity.IngressEndpoint, error) {
	return s.repo.Enabled(ctx)
}

func (s *endpointService) Upsert(ctx context.Context, input EndpointUpsertInput) (entity.IngressEndpoint, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return entity.IngressEndpoint{}, fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if _, ok := constants.ParseProtocol(input.DownstreamProtocol.String()); !ok {
		return entity.IngressEndpoint{}, fmt.Errorf("downstream_protocol is invalid")
	}
	now := time.Now()
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = idgen.New("endpoint")
	}
	item := entity.IngressEndpoint{
		ID:                 id,
		Path:               path,
		DownstreamProtocol: input.DownstreamProtocol,
		Enabled:            input.Enabled,
		Protected:          input.Protected,
		BuiltIn:            false,
		Description:        strings.TrimSpace(input.Description),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.Save(ctx, &item); err != nil {
		return entity.IngressEndpoint{}, err
	}
	return item, nil
}

func (s *endpointService) Delete(ctx context.Context, id string) error {
	item, err := s.repo.Find(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if item.BuiltIn {
		return fmt.Errorf("built-in endpoint cannot be deleted")
	}
	return s.repo.Delete(ctx, item.ID)
}
