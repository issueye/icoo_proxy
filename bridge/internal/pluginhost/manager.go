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
	closed   bool
}

// Instance is one running (or configured) plugin process.
type Instance struct {
	ID          string
	Entry       config.PluginEntry
	Endpoint    string
	Token       string
	Client      *pluginipc.Client
	Handshake   *pluginipc.HandshakeResult
	Cmd         *exec.Cmd
	Status      string // stopped | starting | running | unhealthy | error
	LastError   string
	StartedAt   time.Time
	cancelHeart context.CancelFunc

	// Reliability bookkeeping (host-side only).
	failCount       int
	lastRestart     time.Time
	restartAttempts int
	watchGen        uint64 // bumped on each Start/Stop to ignore stale watchers
	stopping        bool   // true while Stop is tearing down (suppress auto-restart)
}

// NewManager creates a plugin host manager (does not spawn yet).
// Runtime catalog = static TOML entries, overlaid by data_dir/plugins/registry.json
// so desktop install / enable / disable survive bridge restarts without editing TOML.
func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	// Ensure package plugins/ roots exist so tools can drop one-dir-per-plugin packages.
	EnsurePluginsPackageDirs()
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
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("pluginhost: manager closed")
	}
	if inst, ok := m.plugins[id]; ok && inst.Status == "running" && inst.Client != nil {
		m.mu.Unlock()
		return nil
	}
	// Invalidate any previous process watcher for this id.
	var prevGen uint64
	if prev, ok := m.plugins[id]; ok {
		prevGen = prev.watchGen
		prev.stopping = true
		if prev.cancelHeart != nil {
			prev.cancelHeart()
			prev.cancelHeart = nil
		}
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
		watchGen:  prevGen + 1,
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

	// Dial + handshake via SDK. Plugins must Listen promptly; timeouts stay generous
	// enough for cold-start AV scans / first-run, but fail with process/log diagnostics.
	maxFrame := int(m.cfg.EffectiveMaxFrameBytes())
	maxStreams := entry.MaxConcurrentStreams
	if maxStreams <= 0 {
		maxStreams = m.cfg.Plugins.MaxConcurrentStreams
	}
	client, hs, err := pluginipc.Connect(context.WithoutCancel(ctx), pluginipc.ConnectConfig{
		Endpoint:             endpoint,
		Token:                token,
		HostVersion:          pluginipc.DefaultHostVersion,
		DialTimeout:          30 * time.Second,
		HandshakeTimeout:     15 * time.Second,
		MaxFrameBytes:        maxFrame,
		MaxConcurrentStreams: maxStreams,
	})
	if err != nil {
		detail := m.failPluginStart(inst, cmd, logPath, err)
		return fmt.Errorf("pluginhost: connect %s: %s", id, detail)
	}

	inst.Client = client
	inst.Handshake = hs
	inst.Status = "running"
	inst.LastError = ""
	inst.failCount = 0
	inst.stopping = false

	heartCtx, heartCancel := context.WithCancel(context.Background())
	inst.cancelHeart = heartCancel
	go m.heartbeatLoop(heartCtx, id)
	go m.watchProcess(id, inst.watchGen, cmd)

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

			shouldRestart := false
			m.mu.Lock()
			cur := m.plugins[id]
			if cur == nil || cur.Client == nil || cur.stopping {
				m.mu.Unlock()
				return
			}
			if err != nil {
				cur.failCount++
				cur.Status = "unhealthy"
				cur.LastError = err.Error()
				threshold := m.cfg.Plugins.AutoRestartFailThreshold
				if threshold <= 0 {
					threshold = 3
				}
				if m.cfg.Plugins.AutoRestart && cur.failCount >= threshold {
					shouldRestart = true
					// Close half-open client so hot path stops using it.
					cli := cur.Client
					cur.Client = nil
					m.mu.Unlock()
					_ = cli.Close()
				} else {
					m.mu.Unlock()
				}
			} else {
				if cur.Status == "unhealthy" {
					cur.Status = "running"
					cur.LastError = ""
				}
				cur.failCount = 0
				m.mu.Unlock()
			}

			if shouldRestart {
				m.logger.Warn("plugin unhealthy; auto-restarting",
					"plugin_id", id, "fail_count", thresholdOr(m.cfg.Plugins.AutoRestartFailThreshold, 3))
				m.scheduleAutoRestart(id, "heartbeat failures")
			}
		}
	}
}

// watchProcess waits for the child process and converges instance status.
// Auto-restarts enabled plugins when the process exits unexpectedly.
func (m *Manager) watchProcess(id string, gen uint64, cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	err := cmd.Wait()
	exitMsg := "process exited"
	if err != nil {
		exitMsg = "process exited: " + err.Error()
	} else if cmd.ProcessState != nil {
		exitMsg = "process exited: " + cmd.ProcessState.String()
	}

	m.mu.Lock()
	inst := m.plugins[id]
	if inst == nil || inst.watchGen != gen {
		// Stale watcher (Stop/Start already replaced this generation).
		m.mu.Unlock()
		return
	}
	stopping := inst.stopping
	if inst.cancelHeart != nil {
		inst.cancelHeart()
		inst.cancelHeart = nil
	}
	cli := inst.Client
	inst.Client = nil
	if stopping {
		// Expected teardown via Stop; status already handled there.
		m.mu.Unlock()
		if cli != nil {
			_ = cli.Close()
		}
		return
	}
	inst.Status = "error"
	inst.LastError = exitMsg
	entryEnabled := false
	if e, ok := m.entries[id]; ok {
		entryEnabled = e.Enabled
		inst.Entry = e
	} else {
		entryEnabled = inst.Entry.Enabled
	}
	auto := m.cfg.Plugins.AutoRestart && entryEnabled
	m.mu.Unlock()

	if cli != nil {
		_ = cli.Close()
	}
	m.logger.Warn("plugin process exited", "plugin_id", id, "error", exitMsg)
	if auto {
		m.scheduleAutoRestart(id, exitMsg)
	}
}

func (m *Manager) scheduleAutoRestart(id, reason string) {
	delay := m.nextRestartDelay(id)
	if delay < 0 {
		m.logger.Error("plugin auto-restart suppressed", "plugin_id", id, "reason", reason)
		return
	}
	m.logger.Info("plugin auto-restart scheduled", "plugin_id", id, "delay", delay.String(), "reason", reason)
	time.AfterFunc(delay, func() {
		m.mu.RLock()
		closed := m.closed
		entry, ok := m.entries[id]
		m.mu.RUnlock()
		if closed || !ok || !entry.Enabled {
			return
		}
		// Stop may be a no-op if already dead; Restart = Stop + Start.
		if err := m.Restart(context.Background(), id); err != nil {
			m.logger.Error("plugin auto-restart failed", "plugin_id", id, "error", err)
			return
		}
		m.mu.Lock()
		if inst := m.plugins[id]; inst != nil {
			inst.lastRestart = time.Now()
			inst.restartAttempts++
			inst.failCount = 0
		}
		m.mu.Unlock()
		m.logger.Info("plugin auto-restarted", "plugin_id", id)
	})
}

// nextRestartDelay returns backoff delay, or <0 when restarts are suppressed.
func (m *Manager) nextRestartDelay(id string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return -1
	}
	inst := m.plugins[id]
	attempts := 0
	var last time.Time
	if inst != nil {
		attempts = inst.restartAttempts
		last = inst.lastRestart
	}
	// Cooldown: never restart more often than min backoff after last restart.
	base := time.Second
	if attempts <= 0 {
		// first auto-restart
	} else if attempts == 1 {
		base = 2 * time.Second
	} else if attempts == 2 {
		base = 5 * time.Second
	} else {
		base = 30 * time.Second
	}
	if !last.IsZero() {
		since := time.Since(last)
		if since < base {
			return base - since
		}
	}
	// Cap runaway restarts: after many attempts keep 30s spacing (still try).
	return base
}

func thresholdOr(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// Stop gracefully shuts down a plugin.
func (m *Manager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	inst := m.plugins[id]
	if inst == nil {
		m.mu.Unlock()
		return nil
	}
	inst.stopping = true
	inst.watchGen++ // invalidate in-flight watcher auto-restart
	heartCancel := inst.cancelHeart
	inst.cancelHeart = nil
	cli := inst.Client
	inst.Client = nil
	cmd := inst.Cmd
	m.mu.Unlock()

	if heartCancel != nil {
		heartCancel()
	}
	timeout := m.cfg.Plugins.ShutdownPluginTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if cli != nil {
		sctx, cancel := context.WithTimeout(ctx, timeout)
		_ = cli.Shutdown(sctx)
		cancel()
		_ = cli.Close()
	}
	if cmd != nil && cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(timeout):
			_ = cmd.Process.Kill()
			<-done
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			<-done
		}
	}
	m.mu.Lock()
	if cur := m.plugins[id]; cur != nil {
		cur.Status = "stopped"
		cur.Client = nil
		cur.Cmd = nil
		cur.stopping = false
		cur.failCount = 0
	}
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

// Close releases host-side resources (Windows Job Object). Safe to call multiple times.
// Prefer StopAll first for graceful plugin shutdown; Close is a last-resort kill-on-job-close.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.closed = true
	job := m.job
	m.job = nil
	m.mu.Unlock()
	if job != nil {
		job.close()
	}
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
