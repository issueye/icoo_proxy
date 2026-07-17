package pluginhost

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/bridge/internal/service"
)

// entryFor returns the current catalog entry for id (registry-aware).
func (m *Manager) entryFor(id string) (config.PluginEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entries[id]
	return e, ok
}

// setEntry updates the in-memory catalog and persists dynamic registry.
func (m *Manager) setEntry(id string, entry config.PluginEntry, persist bool) error {
	m.mu.Lock()
	if m.entries == nil {
		m.entries = map[string]config.PluginEntry{}
	}
	if m.removed == nil {
		m.removed = map[string]struct{}{}
	}
	// Re-register clears uninstall tombstone.
	delete(m.removed, id)
	m.entries[id] = entry
	// Snapshot for persist without holding lock during IO.
	snap := cloneEntries(m.entries)
	tomb := cloneRemoved(m.removed)
	// Persist all runtime entries so desktop installs and enable toggles survive restart.
	m.mu.Unlock()
	if !persist || m.registry == nil {
		return nil
	}
	return m.registry.Save(snap, tomb)
}

// Register implements service.PluginRuntime — adds/updates catalog and optionally starts.
func (m *Manager) Register(ctx context.Context, id string, in service.PluginRegisterInput, autoStart bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("pluginhost: plugin id is required")
	}
	if strings.TrimSpace(in.Executable) == "" {
		return fmt.Errorf("pluginhost: executable is required")
	}
	entry := config.PluginEntry{
		Enabled:      in.Enabled || autoStart,
		Executable:   strings.TrimSpace(in.Executable),
		Args:         append([]string(nil), in.Args...),
		DataDir:      strings.TrimSpace(in.DataDir),
		AdminEnabled: in.AdminEnabled,
	}
	// Normalize absolute path when possible.
	if !filepath.IsAbs(entry.Executable) {
		if abs, err := filepath.Abs(entry.Executable); err == nil {
			entry.Executable = abs
		}
	}
	if entry.DataDir == "" {
		entry.DataDir = filepath.Join(m.cfg.DataDir, "plugins", id)
	}
	if err := m.setEntry(id, entry, true); err != nil {
		return err
	}
	if autoStart || entry.Enabled {
		entry.Enabled = true
		_ = m.setEntry(id, entry, true)
		return m.Start(ctx, id)
	}
	return nil
}

// Unregister stops a plugin (if running) and removes it from the dynamic catalog.
func (m *Manager) Unregister(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	_ = m.Stop(ctx, id)
	m.mu.Lock()
	delete(m.entries, id)
	delete(m.plugins, id)
	if m.removed == nil {
		m.removed = map[string]struct{}{}
	}
	m.removed[id] = struct{}{}
	snap := cloneEntries(m.entries)
	tomb := cloneRemoved(m.removed)
	m.mu.Unlock()
	if m.registry != nil {
		return m.registry.Save(snap, tomb)
	}
	return nil
}

func cloneRemoved(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for k := range in {
		out[k] = struct{}{}
	}
	return out
}

// SetEnabled toggles enabled flag; when enabling, starts the process; when
// disabling, stops it. Persists to registry.
func (m *Manager) SetEnabled(ctx context.Context, id string, enabled bool) error {
	entry, ok := m.entryFor(id)
	if !ok {
		return fmt.Errorf("pluginhost: unknown plugin %q", id)
	}
	entry.Enabled = enabled
	if err := m.setEntry(id, entry, true); err != nil {
		return err
	}
	if enabled {
		return m.Start(ctx, id)
	}
	return m.Stop(ctx, id)
}

// Discover implements service.PluginRuntime — returns candidates found on disk.
func (m *Manager) Discover() []service.PluginDiscoverCandidate {
	m.mu.RLock()
	reg := cloneEntries(m.entries)
	m.mu.RUnlock()
	raw := DiscoverPlugins(m.cfg.DataDir, reg)
	out := make([]service.PluginDiscoverCandidate, 0, len(raw))
	for _, c := range raw {
		out = append(out, service.PluginDiscoverCandidate{
			ID:               c.ID,
			Name:             c.Name,
			Version:          c.Version,
			Description:      c.Description,
			Executable:       c.Executable,
			ManifestPath:     c.ManifestPath,
			Capabilities:     append([]string(nil), c.Capabilities...),
			SupportedIngress: append([]string(nil), c.SupportedIngress...),
			Registered:       c.Registered,
			Source:           c.Source,
		})
	}
	return out
}

// InstallCandidate registers a discovered plugin by id (from Discover results).
func (m *Manager) InstallCandidate(ctx context.Context, id string, enabled bool) error {
	cands := m.Discover()
	var found *service.PluginDiscoverCandidate
	for i := range cands {
		if cands[i].ID == id {
			found = &cands[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("pluginhost: candidate %q not found on disk", id)
	}
	return m.Register(ctx, id, service.PluginRegisterInput{
		ID:         id,
		Executable: found.Executable,
		Enabled:    enabled,
		DataDir:    filepath.Join(m.cfg.DataDir, "plugins", id),
		AutoStart:  enabled,
	}, enabled)
}
