package pluginhost

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/bridge/internal/service"
)

func TestDynamicRegisterPersistAndList(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	// Fake plugin binary under data_dir/plugins/<id>/ so Discover can find it.
	pluginDir := filepath.Join(dataDir, "plugins", "demo")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatal(err)
	}
	exeName := "plugin-demo"
	if isWindows() {
		exeName += ".exe"
	}
	exe := filepath.Join(pluginDir, exeName)
	if err := os.WriteFile(exe, []byte("not-a-real-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DataDir: dataDir,
		Plugins: config.PluginsConfig{
			Entries: map[string]config.PluginEntry{
				"static": {Enabled: false, Executable: "static.exe"},
			},
		},
	}
	m := NewManager(cfg, nil)

	// Catalog has static entry.
	list := m.List()
	if len(list) != 1 || list[0].ID != "static" {
		t.Fatalf("expected static only, got %+v", list)
	}

	ctx := context.Background()
	// Register without auto-start (would fail on fake binary).
	if err := m.Register(ctx, "demo", service.PluginRegisterInput{
		ID:         "demo",
		Executable: exe,
		Enabled:    false,
	}, false); err != nil {
		t.Fatal(err)
	}

	list = m.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}

	// Registry file exists.
	regPath := filepath.Join(cfg.DataDir, "plugins", "registry.json")
	if _, err := os.Stat(regPath); err != nil {
		t.Fatalf("registry not written: %v", err)
	}

	// Reload manager — registry overlay must restore demo.
	m2 := NewManager(cfg, nil)
	if _, ok := m2.entryFor("demo"); !ok {
		t.Fatal("demo missing after reload")
	}
	if _, ok := m2.entryFor("static"); !ok {
		t.Fatal("static missing after reload")
	}

	// Discover sees the fake binary.
	cands := m2.Discover()
	found := false
	for _, c := range cands {
		if c.ID == "demo" {
			found = true
			if !c.Registered {
				t.Fatal("demo should be registered")
			}
		}
	}
	if !found {
		t.Fatalf("discover missed demo: %+v", cands)
	}

	// Unregister removes from catalog.
	if err := m2.Unregister(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
	if _, ok := m2.entryFor("demo"); ok {
		t.Fatal("demo still registered")
	}

	// After registry is non-empty, unregister of a TOML-seeded id must stick
	// across restart (registry is authoritative; TOML must not resurrect it).
	if err := m2.Unregister(ctx, "static"); err != nil {
		t.Fatal(err)
	}
	m3 := NewManager(cfg, nil)
	if _, ok := m3.entryFor("static"); ok {
		t.Fatal("static should stay unregistered after reload")
	}
}

func TestSetEnabledUnknown(t *testing.T) {
	m := NewManager(config.Config{DataDir: t.TempDir()}, nil)
	err := m.SetEnabled(context.Background(), "nope", true)
	if err == nil {
		t.Fatal("expected error")
	}
}
