package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/issueye/icoo_proxy/common/pluginipc"
)

type listRuntime struct {
	items []PluginRuntimeInstance
}

func (r *listRuntime) List() []PluginRuntimeInstance                       { return r.items }
func (r *listRuntime) Start(ctx context.Context, id string) error          { return nil }
func (r *listRuntime) Stop(ctx context.Context, id string) error           { return nil }
func (r *listRuntime) Restart(ctx context.Context, id string) error        { return nil }
func (r *listRuntime) Health(ctx context.Context, id string) (*pluginipc.HealthResult, error) {
	return &pluginipc.HealthResult{OK: true}, nil
}
func (r *listRuntime) ListModels(ctx context.Context, id string) (*pluginipc.ModelsListResult, error) {
	return &pluginipc.ModelsListResult{}, nil
}
func (r *listRuntime) Client(id string) (*pluginipc.Client, error) { return nil, errors.New("no client") }
func (r *listRuntime) Discover() []PluginDiscoverCandidate         { return nil }
func (r *listRuntime) Register(ctx context.Context, id string, in PluginRegisterInput, autoStart bool) error {
	return nil
}
func (r *listRuntime) Unregister(ctx context.Context, id string) error              { return nil }
func (r *listRuntime) SetEnabled(ctx context.Context, id string, enabled bool) error { return nil }
func (r *listRuntime) InstallCandidate(ctx context.Context, id string, enabled bool) error {
	return nil
}

func TestPluginServiceAdminEnabledGatesUI(t *testing.T) {
	rt := &listRuntime{items: []PluginRuntimeInstance{
		{
			ID:           "ui-on",
			Enabled:      true,
			AdminEnabled: true,
			Status:       "running",
			AdminBaseURL: "http://127.0.0.1:9",
			AdminToken:   "secret",
			UIPages:      []pluginipc.UIPage{{ID: "home", Title: "Home", Path: "/"}},
		},
		{
			ID:           "ui-off",
			Enabled:      true,
			AdminEnabled: false,
			Status:       "running",
			AdminBaseURL: "http://127.0.0.1:10",
			AdminToken:   "secret2",
			UIPages:      []pluginipc.UIPage{{ID: "home", Title: "Home", Path: "/"}},
		},
	}}
	svc := NewPluginService(rt)

	pages, err := svc.ListUIPages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || pages[0].PluginID != "ui-on" {
		t.Fatalf("pages=%+v", pages)
	}

	views, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("views=%d", len(views))
	}
	for _, v := range views {
		switch v.ID {
		case "ui-on":
			if !v.AdminEnabled || v.AdminBaseURL == "" || len(v.UIPages) != 1 {
				t.Fatalf("ui-on view=%+v", v)
			}
		case "ui-off":
			if v.AdminEnabled || v.AdminBaseURL != "" || len(v.UIPages) != 0 {
				t.Fatalf("ui-off view leaked UI: %+v", v)
			}
		}
	}

	if _, _, err := svc.AdminProxyTarget("ui-off"); !errors.Is(err, ErrPluginUIDisabled) {
		t.Fatalf("want ErrPluginUIDisabled, got %v", err)
	}
	base, tok, err := svc.AdminProxyTarget("ui-on")
	if err != nil || base == "" || tok != "secret" {
		t.Fatalf("ui-on target base=%q tok=%q err=%v", base, tok, err)
	}
}

func TestCancelPluginStreamLogsWithoutPanic(t *testing.T) {
	// cancelPluginStream must tolerate nil stream.
	svc := &proxyService{}
	svc.cancelPluginStream(nil, "rid", "pid", "test")
	// Give any accidental goroutine a tick (none expected).
	time.Sleep(10 * time.Millisecond)
}
