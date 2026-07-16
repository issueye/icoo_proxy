package service

import (
	"context"

	"github.com/issueye/icoo_proxy/common/pluginipc"
)

// PluginRuntime is the host-facing process plugin control plane used by admin
// APIs and the proxy hot path. Implemented by pluginhost.Manager.
type PluginRuntime interface {
	List() []PluginRuntimeInstance
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Health(ctx context.Context, id string) (*pluginipc.HealthResult, error)
	ListModels(ctx context.Context, id string) (*pluginipc.ModelsListResult, error)
	Client(id string) (*pluginipc.Client, error)
}

// PluginRuntimeInstance is a stable DTO so service does not import pluginhost types.
type PluginRuntimeInstance struct {
	ID            string
	Enabled       bool
	Executable    string
	Status        string
	LastError     string
	Endpoint      string
	PluginVersion string
	Capabilities  []string
	SupportedIngress []string
	StartedAt     string
}

// PluginView is the admin API representation of a process plugin.
type PluginView struct {
	ID               string   `json:"id"`
	Enabled          bool     `json:"enabled"`
	Executable       string   `json:"executable,omitempty"`
	Status           string   `json:"status"`
	LastError        string   `json:"last_error,omitempty"`
	Endpoint         string   `json:"endpoint,omitempty"`
	PluginVersion    string   `json:"plugin_version,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	SupportedIngress []string `json:"supported_ingress,omitempty"`
	StartedAt        string   `json:"started_at,omitempty"`
}

// PluginService manages process plugins via PluginRuntime.
type PluginService interface {
	List(ctx context.Context) ([]PluginView, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Health(ctx context.Context, id string) (ProviderHealthResult, error)
	Models(ctx context.Context, id string) ([]FetchedModel, error)
}

type pluginService struct {
	runtime PluginRuntime
}

// NewPluginService wraps a PluginRuntime. runtime may be nil (no plugins).
func NewPluginService(runtime PluginRuntime) PluginService {
	return &pluginService{runtime: runtime}
}

func (s *pluginService) List(ctx context.Context) ([]PluginView, error) {
	_ = ctx
	if s.runtime == nil {
		return []PluginView{}, nil
	}
	items := s.runtime.List()
	out := make([]PluginView, 0, len(items))
	for _, item := range items {
		out = append(out, PluginView{
			ID:               item.ID,
			Enabled:          item.Enabled,
			Executable:       item.Executable,
			Status:           item.Status,
			LastError:        item.LastError,
			Endpoint:         item.Endpoint,
			PluginVersion:    item.PluginVersion,
			Capabilities:     item.Capabilities,
			SupportedIngress: item.SupportedIngress,
			StartedAt:        item.StartedAt,
		})
	}
	return out, nil
}

func (s *pluginService) Start(ctx context.Context, id string) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	return s.runtime.Start(ctx, id)
}

func (s *pluginService) Stop(ctx context.Context, id string) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	return s.runtime.Stop(ctx, id)
}

func (s *pluginService) Restart(ctx context.Context, id string) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	return s.runtime.Restart(ctx, id)
}

func (s *pluginService) Health(ctx context.Context, id string) (ProviderHealthResult, error) {
	if s.runtime == nil {
		return ProviderHealthResult{}, errPluginRuntimeUnavailable
	}
	res, err := s.runtime.Health(ctx, id)
	if err != nil {
		return ProviderHealthResult{
			SupplierID: id,
			Status:     "unreachable",
			Message:    err.Error(),
		}, nil
	}
	status := "reachable"
	if res == nil || !res.OK {
		status = "warning"
	}
	msg := "ok"
	if res != nil && res.Status != "" {
		msg = res.Status
	}
	return ProviderHealthResult{
		SupplierID: id,
		Status:     status,
		Message:    msg,
	}, nil
}

func (s *pluginService) Models(ctx context.Context, id string) ([]FetchedModel, error) {
	if s.runtime == nil {
		return nil, errPluginRuntimeUnavailable
	}
	res, err := s.runtime.ListModels(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]FetchedModel, 0)
	if res == nil {
		return out, nil
	}
	for _, m := range res.Models {
		name := m.ID
		if name == "" {
			name = m.DisplayName
		}
		if name == "" {
			continue
		}
		out = append(out, FetchedModel{ID: name, Name: name})
	}
	return out, nil
}
