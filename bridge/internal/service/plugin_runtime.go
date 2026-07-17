package service

import (
	"context"
	"fmt"
	"strings"

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
	// Dynamic catalog (desktop install / enable / discover).
	Discover() []PluginDiscoverCandidate
	Register(ctx context.Context, id string, in PluginRegisterInput, autoStart bool) error
	Unregister(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	InstallCandidate(ctx context.Context, id string, enabled bool) error
}

// PluginDiscoverCandidate is a plugin package found on disk (plugins/<id>/info.toml).
type PluginDiscoverCandidate struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	Version          string   `json:"version,omitempty"`
	Description      string   `json:"description,omitempty"`
	Executable       string   `json:"executable"`
	ManifestPath     string   `json:"manifest_path,omitempty"` // info.toml or legacy manifest
	Capabilities     []string `json:"capabilities,omitempty"`
	SupportedIngress []string `json:"supported_ingress,omitempty"`
	Registered       bool     `json:"registered"`
	Source           string   `json:"source"`
}

// PluginRegisterInput is the body for POST /plugins (manual install).
type PluginRegisterInput struct {
	ID           string   `json:"id"`
	Executable   string   `json:"executable"`
	Args         []string `json:"args,omitempty"`
	DataDir      string   `json:"data_dir,omitempty"`
	Enabled      bool     `json:"enabled"`
	AdminEnabled bool     `json:"admin_enabled,omitempty"`
	AutoStart    bool     `json:"auto_start,omitempty"`
}

// PluginInstallInput installs a discovered candidate by id.
type PluginInstallInput struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// PluginEnabledInput toggles enabled flag.
type PluginEnabledInput struct {
	Enabled bool `json:"enabled"`
}

// PluginRuntimeInstance is a stable DTO so service does not import pluginhost types.
type PluginRuntimeInstance struct {
	ID               string
	Enabled          bool
	Executable       string
	Status           string
	LastError        string
	Endpoint         string
	PluginVersion    string
	Capabilities     []string
	SupportedIngress []string
	StartedAt        string
	AdminBaseURL     string
	// AdminToken is host-internal only; never serialize into PluginView JSON.
	AdminToken string
	UIPages    []pluginipc.UIPage
}

// PluginUIPageView is a desktop-facing extension page descriptor.
type PluginUIPageView struct {
	PluginID    string `json:"plugin_id"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	Icon        string `json:"icon,omitempty"`
	Group       string `json:"group,omitempty"`
	Description string `json:"description,omitempty"`
	// EmbedURL is the bridge-relative path for iframe embedding.
	EmbedURL string `json:"embed_url"`
}

// PluginView is the admin API representation of a process plugin.
type PluginView struct {
	ID               string              `json:"id"`
	Enabled          bool                `json:"enabled"`
	Executable       string              `json:"executable,omitempty"`
	Status           string              `json:"status"`
	LastError        string              `json:"last_error,omitempty"`
	Endpoint         string              `json:"endpoint,omitempty"`
	PluginVersion    string              `json:"plugin_version,omitempty"`
	Capabilities     []string            `json:"capabilities,omitempty"`
	SupportedIngress []string            `json:"supported_ingress,omitempty"`
	StartedAt        string              `json:"started_at,omitempty"`
	AdminBaseURL     string              `json:"admin_base_url,omitempty"`
	UIPages          []PluginUIPageView  `json:"ui_pages,omitempty"`
}

// PluginService manages process plugins via PluginRuntime.
type PluginService interface {
	List(ctx context.Context) ([]PluginView, error)
	ListUIPages(ctx context.Context) ([]PluginUIPageView, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Health(ctx context.Context, id string) (ProviderHealthResult, error)
	Models(ctx context.Context, id string) ([]FetchedModel, error)
	// AdminBaseURL returns the plugin loopback UI base when running.
	AdminBaseURL(id string) (string, error)
	// AdminProxyTarget returns loopback base + admin token for UI reverse-proxy inject.
	// Token must never be returned from public list/ui-pages endpoints.
	AdminProxyTarget(id string) (baseURL, adminToken string, err error)
	// Dynamic plug/unplug (desktop).
	Discover(ctx context.Context) ([]PluginDiscoverCandidate, error)
	Register(ctx context.Context, in PluginRegisterInput) error
	Unregister(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	Install(ctx context.Context, in PluginInstallInput) error
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
		out = append(out, toPluginView(item))
	}
	return out, nil
}

func (s *pluginService) ListUIPages(ctx context.Context) ([]PluginUIPageView, error) {
	_ = ctx
	if s.runtime == nil {
		return []PluginUIPageView{}, nil
	}
	var pages []PluginUIPageView
	for _, item := range s.runtime.List() {
		if item.Status != "running" && item.Status != "unhealthy" {
			continue
		}
		pages = append(pages, toPluginUIPages(item)...)
	}
	if pages == nil {
		pages = []PluginUIPageView{}
	}
	return pages, nil
}

func (s *pluginService) AdminBaseURL(id string) (string, error) {
	base, _, err := s.AdminProxyTarget(id)
	return base, err
}

func (s *pluginService) AdminProxyTarget(id string) (string, string, error) {
	if s.runtime == nil {
		return "", "", errPluginRuntimeUnavailable
	}
	for _, item := range s.runtime.List() {
		if item.ID == id {
			if item.AdminBaseURL == "" {
				return "", "", fmt.Errorf("plugin %q has no admin UI", id)
			}
			return item.AdminBaseURL, item.AdminToken, nil
		}
	}
	return "", "", fmt.Errorf("plugin %q not found", id)
}

func toPluginView(item PluginRuntimeInstance) PluginView {
	return PluginView{
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
		AdminBaseURL:     item.AdminBaseURL,
		UIPages:          toPluginUIPages(item),
	}
}

func toPluginUIPages(item PluginRuntimeInstance) []PluginUIPageView {
	if len(item.UIPages) == 0 {
		return nil
	}
	out := make([]PluginUIPageView, 0, len(item.UIPages))
	for _, p := range item.UIPages {
		path := strings.TrimSpace(p.Path)
		if path == "" {
			path = "/"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		pageID := p.ID
		if pageID == "" {
			pageID = strings.Trim(path, "/")
			if pageID == "" {
				pageID = "home"
			}
		}
		embed := "/api/v1/plugins/" + item.ID + "/ui" + path
		out = append(out, PluginUIPageView{
			PluginID:    item.ID,
			ID:          pageID,
			Title:       firstNonEmpty(p.Title, item.ID),
			Path:        path,
			Icon:        p.Icon,
			Group:       firstNonEmpty(p.Group, "插件"),
			Description: p.Description,
			EmbedURL:    embed,
		})
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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

func (s *pluginService) Discover(ctx context.Context) ([]PluginDiscoverCandidate, error) {
	_ = ctx
	if s.runtime == nil {
		return []PluginDiscoverCandidate{}, nil
	}
	items := s.runtime.Discover()
	if items == nil {
		return []PluginDiscoverCandidate{}, nil
	}
	return items, nil
}

func (s *pluginService) Register(ctx context.Context, in PluginRegisterInput) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	if strings.TrimSpace(in.Executable) == "" {
		return fmt.Errorf("executable is required")
	}
	return s.runtime.Register(ctx, id, in, in.AutoStart || in.Enabled)
}

func (s *pluginService) Unregister(ctx context.Context, id string) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	return s.runtime.Unregister(ctx, id)
}

func (s *pluginService) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	return s.runtime.SetEnabled(ctx, id, enabled)
}

func (s *pluginService) Install(ctx context.Context, in PluginInstallInput) error {
	if s.runtime == nil {
		return errPluginRuntimeUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	return s.runtime.InstallCandidate(ctx, id, in.Enabled)
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
