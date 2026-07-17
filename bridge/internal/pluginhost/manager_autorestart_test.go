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

func buildMockPlugin(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	exeName := "mockplugin"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	exePath := filepath.Join(tmp, exeName)
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	cmd := exec.Command("go", "build", "-o", exePath, "./plugins/mock/cmd/mockplugin")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build mockplugin: %v\n%s", err, out)
	}
	return exePath
}

func waitPluginStatus(t *testing.T, m *Manager, id, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		inst, ok := m.Get(id)
		if ok {
			last = inst.Status
			if inst.Status == want {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("plugin %q status want=%s last=%s", id, want, last)
}

func TestNextRestartDelayBackoff(t *testing.T) {
	cfg := config.Config{Plugins: config.PluginsConfig{AutoRestart: true}}
	m := NewManager(cfg, nil)
	// No instance → first attempt delay 1s
	d := m.nextRestartDelay("x")
	if d != time.Second {
		t.Fatalf("first delay=%v", d)
	}

	m.mu.Lock()
	m.plugins["x"] = &Instance{ID: "x", restartAttempts: 1, lastRestart: time.Now().Add(-10 * time.Second)}
	m.mu.Unlock()
	if d := m.nextRestartDelay("x"); d != 2*time.Second {
		t.Fatalf("attempts=1 delay=%v", d)
	}

	m.mu.Lock()
	m.plugins["x"].restartAttempts = 2
	m.plugins["x"].lastRestart = time.Now().Add(-10 * time.Second)
	m.mu.Unlock()
	if d := m.nextRestartDelay("x"); d != 5*time.Second {
		t.Fatalf("attempts=2 delay=%v", d)
	}

	m.mu.Lock()
	m.plugins["x"].restartAttempts = 5
	m.plugins["x"].lastRestart = time.Now().Add(-time.Minute)
	m.mu.Unlock()
	if d := m.nextRestartDelay("x"); d != 30*time.Second {
		t.Fatalf("attempts=5 delay=%v", d)
	}

	// Cooldown remainder when lastRestart is recent.
	m.mu.Lock()
	m.plugins["x"].restartAttempts = 0
	m.plugins["x"].lastRestart = time.Now()
	m.mu.Unlock()
	if d := m.nextRestartDelay("x"); d <= 0 || d > time.Second {
		t.Fatalf("cooldown delay=%v", d)
	}

	m.Close()
	if d := m.nextRestartDelay("x"); d >= 0 {
		t.Fatalf("closed manager should suppress, delay=%v", d)
	}
}

func TestWatchProcessAutoRestartOnCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("skip process integration test in short mode")
	}
	exePath := buildMockPlugin(t)
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:             dataDir,
		MaxRequestBodyBytes: 64 << 20,
		Plugins: config.PluginsConfig{
			MaxConcurrentStreams:     8,
			HeartbeatInterval:        time.Hour,
			ShutdownPluginTimeout:    3 * time.Second,
			AutoRestart:              true,
			AutoRestartFailThreshold: 3,
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
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.Stop(ctx, "mock")
		m.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.Start(ctx, "mock"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitPluginStatus(t, m, "mock", "running", 10*time.Second)

	inst, ok := m.Get("mock")
	if !ok || inst.Cmd == nil || inst.Cmd.Process == nil {
		t.Fatal("missing process")
	}
	pid := inst.Cmd.Process.Pid
	if err := inst.Cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	// Wait until a new generation is running (auto-restart).
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := m.Get("mock")
		if ok && cur.Status == "running" && cur.Client != nil && cur.Cmd != nil && cur.Cmd.Process != nil && cur.Cmd.Process.Pid != pid {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	cur, _ := m.Get("mock")
	status := ""
	if cur != nil {
		status = cur.Status
	}
	t.Fatalf("auto-restart did not recover; status=%s", status)
}

func TestWatchProcessNoRestartWhenAutoRestartDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skip process integration test in short mode")
	}
	exePath := buildMockPlugin(t)
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:             dataDir,
		MaxRequestBodyBytes: 64 << 20,
		Plugins: config.PluginsConfig{
			MaxConcurrentStreams:  8,
			HeartbeatInterval:     time.Hour,
			ShutdownPluginTimeout: 3 * time.Second,
			AutoRestart:           false,
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
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.Stop(ctx, "mock")
		m.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.Start(ctx, "mock"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitPluginStatus(t, m, "mock", "running", 10*time.Second)

	inst, _ := m.Get("mock")
	if err := inst.Cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	waitPluginStatus(t, m, "mock", "error", 10*time.Second)

	// Ensure it stays error (no restart).
	time.Sleep(2 * time.Second)
	cur, _ := m.Get("mock")
	if cur == nil || cur.Status != "error" {
		t.Fatalf("expected error status without restart, got %#v", cur)
	}
}

func TestStopSuppressesAutoRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skip process integration test in short mode")
	}
	exePath := buildMockPlugin(t)
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:             dataDir,
		MaxRequestBodyBytes: 64 << 20,
		Plugins: config.PluginsConfig{
			MaxConcurrentStreams:  8,
			HeartbeatInterval:     time.Hour,
			ShutdownPluginTimeout: 3 * time.Second,
			AutoRestart:           true,
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
	t.Cleanup(func() { m.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.Start(ctx, "mock"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitPluginStatus(t, m, "mock", "running", 10*time.Second)
	if err := m.Stop(ctx, "mock"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitPluginStatus(t, m, "mock", "stopped", 10*time.Second)

	// Give auto-restart path time; must remain stopped.
	time.Sleep(2 * time.Second)
	cur, _ := m.Get("mock")
	if cur == nil || cur.Status != "stopped" {
		t.Fatalf("Stop should not auto-restart, status=%v", cur)
	}
}
