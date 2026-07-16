package pluginhost

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

// Manager owns process plugin lifecycle for the bridge host.
type Manager struct {
	cfg    config.Config
	logger *slog.Logger
	job    *jobHolder // Windows: KILL_ON_JOB_CLOSE; Unix: no-op

	mu       sync.RWMutex
	plugins  map[string]*Instance
	entries  map[string]config.PluginEntry // runtime catalog (TOML + registry overlay)
	removed  map[string]struct{}           // uninstall tombstones (do not re-seed from TOML)
	registry *Registry
}

// Instance is one running (or configured) plugin process.
type Instance struct {
	ID           string
	Entry        config.PluginEntry
	Endpoint     string
	Token        string
	Client       *pluginipc.Client
	Handshake    *pluginipc.HandshakeResult
	Cmd          *exec.Cmd
	Status       string // stopped | starting | running | unhealthy | error
	LastError    string
	StartedAt    time.Time
	cancelHeart  context.CancelFunc
}

// NewManager creates a plugin host manager (does not spawn yet).
// Runtime catalog = static TOML entries, overlaid by data_dir/plugins/registry.json
// so desktop install / enable / disable survive bridge restarts without editing TOML.
func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	// Catalog: TOML seed → overlay registry entries → drop uninstall tombstones.
	// Desktop Register/Unregister/SetEnabled rewrite registry.json so choices stick.
	entries := make(map[string]config.PluginEntry)
	for id, e := range cfg.Plugins.Entries {
		entries[id] = e
	}
	reg := newRegistry(cfg.DataDir)
	removed := map[string]struct{}{}
	if loaded, tomb, err := reg.LoadSnapshot(); err != nil {
		logger.Warn("pluginhost: load registry failed", "error", err)
	} else {
		for id, e := range loaded {
			entries[id] = e
		}
		removed = tomb
		for id := range removed {
			delete(entries, id)
		}
	}
	m := &Manager{
		cfg:      cfg,
		logger:   logger,
		plugins:  make(map[string]*Instance),
		entries:  entries,
		registry: reg,
		removed:  removed,
	}
	if job, err := newKillOnCloseJob(); err == nil {
		m.job = job
	} else if logger != nil {
		logger.Warn("pluginhost: job object unavailable", "error", err)
	}
	return m
}

// StartEnabled spawns all catalog entries with enabled=true.
// Failures are logged; partial start is allowed (other plugins continue).
func (m *Manager) StartEnabled(ctx context.Context) error {
	m.mu.RLock()
	ids := make([]string, 0, len(m.entries))
	for id, entry := range m.entries {
		if entry.Enabled {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	for _, id := range ids {
		if err := m.Start(ctx, id); err != nil {
			m.logger.Error("plugin start failed", "plugin_id", id, "error", err)
			// continue others
		}
	}
	return nil
}

// Start spawns and handshakes a single plugin by id.
// Enabled=true is only required for StartEnabled (boot); admin may Start any catalog entry.
func (m *Manager) Start(ctx context.Context, id string) error {
	entry, ok := m.entryFor(id)
	if !ok {
		return fmt.Errorf("pluginhost: unknown plugin %q", id)
	}

	m.mu.Lock()
	if inst, ok := m.plugins[id]; ok && inst.Status == "running" && inst.Client != nil {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	dataDir := entry.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(m.cfg.DataDir, "plugins", id)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}

	endpoint, err := pluginipc.NewEndpoint(id, m.cfg.DataDir)
	if err != nil {
		return err
	}
	token, err := pluginipc.NewHostToken()
	if err != nil {
		return err
	}

	exe := entry.Executable
	if exe == "" {
		return fmt.Errorf("pluginhost: plugin %q missing executable", id)
	}
	if !filepath.IsAbs(exe) {
		// Prefer next to bridge binary, then cwd.
		if self, err := os.Executable(); err == nil {
			cand := filepath.Join(filepath.Dir(self), exe)
			if _, err := os.Stat(cand); err == nil {
				exe = cand
			}
		}
	}

	args := []string{
		"--endpoint", endpoint,
		"--data-dir", dataDir,
		"--plugin-id", id,
	}
	args = append(args, entry.Args...)

	// Do not bind process lifetime to the caller's ctx (often a short dial
	// timeout). Plugins outlive Start(); Stop/Job Object handle teardown.
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(),
		"ICOO_PLUGIN_TOKEN="+token,
		"ICOO_PLUGIN_ENDPOINT="+endpoint,
	)
	cmd.Dir = dataDir
	configurePluginCommand(cmd) // Job Object / PGID

	logPath := filepath.Join(dataDir, "plugin.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	inst := &Instance{
		ID:        id,
		Entry:     entry,
		Endpoint:  endpoint,
		Token:     token,
		Cmd:       cmd,
		Status:    "starting",
		StartedAt: time.Now(),
	}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		inst.Status = "error"
		inst.LastError = err.Error()
		m.mu.Lock()
		m.plugins[id] = inst
		m.mu.Unlock()
		return fmt.Errorf("pluginhost: start %s: %w", id, err)
	}
	// Detach log file lifetime from parent after start; process keeps FD.
	_ = logFile.Close()
	if cmd.Process != nil {
		if err := attachProcessToJob(m.job, cmd.Process.Pid); err != nil {
			m.logger.Warn("pluginhost: assign job object failed", "plugin_id", id, "error", err)
		}
	}

	// Dial with timeout. Plugins must Listen promptly; keep this generous enough
	// for cold-start AV scans / first-run, but fail with process/log diagnostics.
	dialCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	conn, err := pluginipc.Dial(dialCtx, pluginipc.DialConfig{Endpoint: endpoint})
	if err != nil {
		detail := m.failPluginStart(inst, cmd, logPath, err)
		return fmt.Errorf("pluginhost: dial %s: %s", id, detail)
	}

	maxFrame := int(m.cfg.EffectiveMaxFrameBytes())
	maxStreams := entry.MaxConcurrentStreams
	if maxStreams <= 0 {
		maxStreams = m.cfg.Plugins.MaxConcurrentStreams
	}
	client := pluginipc.NewClient(conn, pluginipc.ClientOptions{
		MaxFrameBytes:        maxFrame,
		MaxConcurrentStreams: maxStreams,
	})

	hsCtx, hsCancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer hsCancel()
	hs, err := client.Handshake(hsCtx, token, "icoo_llm_bridge")
	if err != nil {
		_ = client.Close()
		detail := m.failPluginStart(inst, cmd, logPath, err)
		return fmt.Errorf("pluginhost: handshake %s: %s", id, detail)
	}

	inst.Client = client
	inst.Handshake = hs
	inst.Status = "running"
	inst.LastError = ""

	heartCtx, heartCancel := context.WithCancel(context.Background())
	inst.cancelHeart = heartCancel
	go m.heartbeatLoop(heartCtx, id)

	m.mu.Lock()
	m.plugins[id] = inst
	m.mu.Unlock()
	m.logger.Info("plugin started", "plugin_id", id, "endpoint", endpoint, "version", hs.PluginVersion)
	return nil
}

// failPluginStart kills the child, records diagnostics on inst, and returns a detail string.
func (m *Manager) failPluginStart(inst *Instance, cmd *exec.Cmd, logPath string, cause error) string {
	// Read log before kill so short-lived messages are preserved.
	tail := tailFile(logPath, 2048)
	msg := cause.Error()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		if err := cmd.Wait(); err != nil && cmd.ProcessState == nil {
			msg += "; wait_error=" + err.Error()
		}
	}
	if cmd != nil && cmd.ProcessState != nil {
		msg += "; process_exited=" + cmd.ProcessState.String()
	}
	if tail != "" {
		msg += "; plugin.log: " + tail
	} else {
		msg += "; plugin.log empty (crashed before Listen, or wrong executable path)"
	}
	inst.Status = "error"
	inst.LastError = msg
	inst.Client = nil
	m.mu.Lock()
	m.plugins[inst.ID] = inst
	m.mu.Unlock()
	return msg
}

func tailFile(path string, max int) string {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		return ""
	}
	if len(raw) > max {
		raw = raw[len(raw)-max:]
	}
	// Collapse whitespace for single-line API errors.
	s := strings.TrimSpace(string(raw))
	s = strings.ReplaceAll(s, "\r\n", " | ")
	s = strings.ReplaceAll(s, "\n", " | ")
	return s
}

func (m *Manager) heartbeatLoop(ctx context.Context, id string) {
	interval := m.cfg.Plugins.HeartbeatInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.mu.RLock()
			inst := m.plugins[id]
			m.mu.RUnlock()
			if inst == nil || inst.Client == nil {
				return
			}
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := inst.Client.Ping(pingCtx)
			cancel()
			m.mu.Lock()
			if cur := m.plugins[id]; cur != nil {
				if err != nil {
					cur.Status = "unhealthy"
					cur.LastError = err.Error()
				} else if cur.Status == "unhealthy" {
					cur.Status = "running"
					cur.LastError = ""
				}
			}
			m.mu.Unlock()
		}
	}
}

// Stop gracefully shuts down a plugin.
func (m *Manager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	inst := m.plugins[id]
	m.mu.Unlock()
	if inst == nil {
		return nil
	}
	if inst.cancelHeart != nil {
		inst.cancelHeart()
	}
	timeout := m.cfg.Plugins.ShutdownPluginTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if inst.Client != nil {
		sctx, cancel := context.WithTimeout(ctx, timeout)
		_ = inst.Client.Shutdown(sctx)
		cancel()
		_ = inst.Client.Close()
	}
	if inst.Cmd != nil && inst.Cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- inst.Cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(timeout):
			_ = inst.Cmd.Process.Kill()
			<-done
		case <-ctx.Done():
			_ = inst.Cmd.Process.Kill()
			<-done
		}
	}
	m.mu.Lock()
	inst.Status = "stopped"
	inst.Client = nil
	m.mu.Unlock()
	return nil
}

// StopAll shuts down every running plugin (call after HTTP drain).
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	ids := make([]string, 0, len(m.plugins))
	for id := range m.plugins {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	var first error
	for _, id := range ids {
		if err := m.Stop(ctx, id); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Get returns a snapshot of a plugin instance.
func (m *Manager) Get(id string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.plugins[id]
	return inst, ok
}

// Client returns the IPC client for a running plugin.
func (m *Manager) Client(id string) (*pluginipc.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.plugins[id]
	if !ok || inst.Client == nil || (inst.Status != "running" && inst.Status != "unhealthy") {
		return nil, fmt.Errorf("pluginhost: plugin %q not available", id)
	}
	return inst.Client, nil
}

// Restart stops then starts a plugin.
func (m *Manager) Restart(ctx context.Context, id string) error {
	_ = m.Stop(ctx, id)
	return m.Start(ctx, id)
}

// Health probes plugin.health on a running instance.
func (m *Manager) Health(ctx context.Context, id string) (*pluginipc.HealthResult, error) {
	cli, err := m.Client(id)
	if err != nil {
		return nil, err
	}
	return cli.Health(ctx)
}

// ListModels calls models.list on a running plugin.
func (m *Manager) ListModels(ctx context.Context, id string) (*pluginipc.ModelsListResult, error) {
	cli, err := m.Client(id)
	if err != nil {
		return nil, err
	}
	return cli.ListModels(ctx)
}
