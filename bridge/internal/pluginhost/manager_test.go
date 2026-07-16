package pluginhost

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
)

func TestManagerStartStopMockPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skip process integration test in short mode")
	}

	// Build mock plugin next to test binary.
	tmp := t.TempDir()
	exeName := "mockplugin"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	exePath := filepath.Join(tmp, exeName)
	// Build from workspace root so go.work resolves icoo/common.
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	cmd := exec.Command("go", "build", "-o", exePath, "./plugins/mock/cmd/mockplugin")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build mockplugin: %v\n%s", err, out)
	}

	dataDir := filepath.Join(tmp, "data")
	cfg := config.Config{
		DataDir:             dataDir,
		MaxRequestBodyBytes: 64 << 20,
		Plugins: config.PluginsConfig{
			MaxConcurrentStreams:  8,
			HeartbeatInterval:     time.Hour, // disable noisy heartbeat during test
			ShutdownPluginTimeout: 3 * time.Second,
			Entries: map[string]config.PluginEntry{
				"mock": {
					Enabled:    true,
					Executable: exePath,
					DataDir:    filepath.Join(dataDir, "plugins", "mock"),
				},
			},
		},
	}
	m := NewManager(cfg, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := m.Start(ctx, "mock"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	cli, err := m.Client("mock")
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	models, err := cli.ListModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(models.Models) == 0 {
		t.Fatal("expected models")
	}
	if err := m.Stop(ctx, "mock"); err != nil {
		t.Fatal(err)
	}
}
