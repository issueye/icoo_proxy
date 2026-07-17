package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/bridge/internal/repository"
)

// OpenAIModelsListResponse is the OpenAI-compatible GET /v1/models body.
// https://platform.openai.com/docs/api-reference/models/list
type OpenAIModelsListResponse struct {
	Object string           `json:"object"`
	Data   []OpenAIModelRef `json:"data"`
}

// OpenAIModelRef is one model entry in the list.
type OpenAIModelRef struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
	// Extra non-breaking metadata for desktop/tools (ignored by strict clients).
	ProviderID   string `json:"provider_id,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
	Vendor       string `json:"vendor,omitempty"`
}

// ModelListService serves the proxy-compatible model catalog.
type ModelListService interface {
	// ListModels returns OpenAI-style models aggregated from enabled providers.
	ListModels(ctx context.Context) (OpenAIModelsListResponse, error)
	// Authorize reports whether the request may call the public models API.
	Authorize(r *http.Request) bool
}

type modelListService struct {
	cfg          config.Config
	auth         proxyAuth
	providers    repository.ProviderRepository
	providerMods repository.ProviderModelRepository
}

func NewModelListService(
	cfg config.Config,
	auth proxyAuth,
	providers repository.ProviderRepository,
	providerMods repository.ProviderModelRepository,
) ModelListService {
	return &modelListService{
		cfg:          cfg,
		auth:         auth,
		providers:    providers,
		providerMods: providerMods,
	}
}

func (s *modelListService) Authorize(r *http.Request) bool {
	if s.cfg.AllowLocalWithoutAuth && isLoopbackRemote(r.RemoteAddr) {
		return true
	}
	key := extractRequestAPIKey(r)
	return key != "" && s.auth != nil && s.auth.Verify(r.Context(), key, "proxy")
}

func (s *modelListService) ListModels(ctx context.Context) (OpenAIModelsListResponse, error) {
	out := OpenAIModelsListResponse{
		Object: "list",
		Data:   make([]OpenAIModelRef, 0),
	}
	if s.providers == nil || s.providerMods == nil {
		return out, nil
	}

	providers, err := s.providers.List(ctx)
	if err != nil {
		return out, fmt.Errorf("list providers: %w", err)
	}

	// Preserve first-seen short model name; always emit provider/model for direct routes.
	seenShort := map[string]struct{}{}
	seenID := map[string]struct{}{}
	now := time.Now().Unix()

	// Stable order: provider name, then model name.
	sort.SliceStable(providers, func(i, j int) bool {
		return strings.ToLower(providers[i].Name) < strings.ToLower(providers[j].Name)
	})

	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		models, err := s.providerMods.ListByProvider(ctx, provider.ID)
		if err != nil {
			return out, fmt.Errorf("list models for provider %q: %w", provider.ID, err)
		}
		sort.SliceStable(models, func(i, j int) bool {
			return strings.ToLower(models[i].Name) < strings.ToLower(models[j].Name)
		})
		ownedBy := strings.TrimSpace(provider.Name)
		if ownedBy == "" {
			ownedBy = provider.ID
		}
		for _, model := range models {
			if !model.Enabled {
				continue
			}
			name := strings.TrimSpace(model.Name)
			if name == "" {
				continue
			}
			created := now
			if !model.CreatedAt.IsZero() {
				created = model.CreatedAt.Unix()
			}

			// Short id (what most clients put in `model`) — first provider wins on collision.
			if _, ok := seenShort[name]; !ok {
				seenShort[name] = struct{}{}
				if _, exists := seenID[name]; !exists {
					seenID[name] = struct{}{}
					out.Data = append(out.Data, OpenAIModelRef{
						ID:           name,
						Object:       "model",
						Created:      created,
						OwnedBy:      ownedBy,
						ProviderID:   provider.ID,
						ProviderName: ownedBy,
						MaxTokens:    model.MaxTokens,
						Protocol:     provider.Protocol.String(),
						Vendor:       provider.Vendor.String(),
					})
				}
			}

			// Direct-route id: provider-name/model-name (route_resolver resolveDirect).
			directID := ownedBy + "/" + name
			if _, exists := seenID[directID]; exists {
				continue
			}
			// Skip if identical to short id (provider name empty edge case).
			if directID == name {
				continue
			}
			seenID[directID] = struct{}{}
			out.Data = append(out.Data, OpenAIModelRef{
				ID:           directID,
				Object:       "model",
				Created:      created,
				OwnedBy:      ownedBy,
				ProviderID:   provider.ID,
				ProviderName: ownedBy,
				MaxTokens:    model.MaxTokens,
				Protocol:     provider.Protocol.String(),
				Vendor:       provider.Vendor.String(),
			})
		}
	}

	return out, nil
}
