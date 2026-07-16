package pluginhost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/issueye/icoo_proxy/bridge/internal/config"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

// registryFile is the durable dynamic plugin catalog under data_dir.
// Static TOML entries still win on id collision for boot defaults, then
// runtime registry overlays (desktop installs) take precedence after load.
const registryFileName = "registry.json"

type registryDoc struct {
	// Entries is the full runtime catalog snapshot (TOML seed + desktop installs).
	Entries map[string]config.PluginEntry `json:"entries"`
	// Removed lists plugin ids the user uninstalled via desktop. On boot, TOML
	// entries matching these ids are not re-seeded.
	Removed []string `json:"removed,omitempty"`
}

// Registry persists dynamically registered plugins (install / enable / disable).
type Registry struct {
	path string
	mu   sync.Mutex
}

func newRegistry(dataDir string) *Registry {
	dir := filepath.Join(dataDir, "plugins")
	return &Registry{path: filepath.Join(dir, registryFileName)}
}

// LoadSnapshot returns catalog entries and the uninstall tombstone set.
func (r *Registry) LoadSnapshot() (entries map[string]config.PluginEntry, removed map[string]struct{}, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	removed = map[string]struct{}{}
	raw, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]config.PluginEntry{}, removed, nil
		}
		return nil, nil, err
	}
	var doc registryDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, err
	}
	if doc.Entries == nil {
		doc.Entries = map[string]config.PluginEntry{}
	}
	for _, id := range doc.Removed {
		id = strings.TrimSpace(id)
		if id != "" {
			removed[id] = struct{}{}
		}
	}
	return cloneEntries(doc.Entries), removed, nil
}

// Load returns entries only (tombstones ignored). Prefer LoadSnapshot for boot.
func (r *Registry) Load() (map[string]config.PluginEntry, error) {
	entries, _, err := r.LoadSnapshot()
	return entries, err
}

func (r *Registry) Save(entries map[string]config.PluginEntry, removed map[string]struct{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	doc := registryDoc{Entries: cloneEntries(entries)}
	if len(removed) > 0 {
		doc.Removed = make([]string, 0, len(removed))
		for id := range removed {
			doc.Removed = append(doc.Removed, id)
		}
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func cloneEntries(in map[string]config.PluginEntry) map[string]config.PluginEntry {
	out := make(map[string]config.PluginEntry, len(in))
	for k, v := range in {
		e := v
		if len(v.Args) > 0 {
			e.Args = append([]string(nil), v.Args...)
		}
		out[k] = e
	}
	return out
}

// DiscoverCandidate is a plugin found on disk but not necessarily registered.
type DiscoverCandidate struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	Version          string   `json:"version,omitempty"`
	Executable       string   `json:"executable"`
	ManifestPath     string   `json:"manifest_path,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	SupportedIngress []string `json:"supported_ingress,omitempty"`
	Registered       bool     `json:"registered"`
	Source           string   `json:"source"` // bridge_dir | data_dir | path
}

// DiscoverPlugins scans known locations for plugin binaries / manifests.
func DiscoverPlugins(dataDir string, registered map[string]config.PluginEntry) []DiscoverCandidate {
	seen := map[string]DiscoverCandidate{}

	// 1) Next to bridge executable: plugin-*.exe and *.manifest.json
	if self, err := os.Executable(); err == nil {
		dir := filepath.Dir(self)
		scanPluginDir(dir, "bridge_dir", seen)
	}
	// 2) CWD
	if cwd, err := os.Getwd(); err == nil {
		scanPluginDir(cwd, "cwd", seen)
	}
	// 3) data_dir/plugins/<id>/
	pluginsRoot := filepath.Join(dataDir, "plugins")
	if entries, err := os.ReadDir(pluginsRoot); err == nil {
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			sub := filepath.Join(pluginsRoot, ent.Name())
			scanPluginDir(sub, "data_dir", seen)
			// Also try plugin.manifest.json in subdir with exe nearby.
			if man, err := pluginipc.LoadManifest(filepath.Join(sub, "plugin.manifest.json")); err == nil {
				exe := man.Executable
				if exe == "" {
					exe = "plugin-" + man.PluginID
					if isWindows() {
						exe += ".exe"
					}
				}
				if !filepath.IsAbs(exe) {
					cand := filepath.Join(sub, exe)
					if st, err := os.Stat(cand); err == nil && !st.IsDir() {
						exe = cand
					} else if self, err := os.Executable(); err == nil {
						// fall back to bridge dir
						alt := filepath.Join(filepath.Dir(self), filepath.Base(exe))
						if st, err := os.Stat(alt); err == nil && !st.IsDir() {
							exe = alt
						}
					}
				}
				c := DiscoverCandidate{
					ID:               man.PluginID,
					Name:             man.Name,
					Version:          man.Version,
					Executable:       exe,
					ManifestPath:     filepath.Join(sub, "plugin.manifest.json"),
					Capabilities:     append([]string(nil), man.Capabilities...),
					SupportedIngress: append([]string(nil), man.SupportedIngress...),
					Source:           "data_dir",
				}
				if prev, ok := seen[c.ID]; !ok || prev.ManifestPath == "" {
					seen[c.ID] = c
				}
			}
		}
	}

	out := make([]DiscoverCandidate, 0, len(seen))
	for id, c := range seen {
		_, c.Registered = registered[id]
		// Prefer absolute executable path when resolvable.
		if !filepath.IsAbs(c.Executable) {
			if abs, err := filepath.Abs(c.Executable); err == nil {
				if st, err := os.Stat(abs); err == nil && !st.IsDir() {
					c.Executable = abs
				}
			}
		}
		out = append(out, c)
	}
	return out
}

func scanPluginDir(dir, source string, seen map[string]DiscoverCandidate) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	// manifests first
	for _, ent := range entries {
		name := ent.Name()
		if ent.IsDir() {
			continue
		}
		if strings.EqualFold(name, "plugin.manifest.json") || strings.HasSuffix(strings.ToLower(name), ".manifest.json") {
			path := filepath.Join(dir, name)
			man, err := pluginipc.LoadManifest(path)
			if err != nil {
				continue
			}
			exe := man.Executable
			if exe == "" {
				exe = "plugin-" + man.PluginID
				if isWindows() {
					exe += ".exe"
				}
			}
			if !filepath.IsAbs(exe) {
				cand := filepath.Join(dir, filepath.Base(exe))
				if st, err := os.Stat(cand); err == nil && !st.IsDir() {
					exe = cand
				}
			}
			seen[man.PluginID] = DiscoverCandidate{
				ID:               man.PluginID,
				Name:             man.Name,
				Version:          man.Version,
				Executable:       exe,
				ManifestPath:     path,
				Capabilities:     append([]string(nil), man.Capabilities...),
				SupportedIngress: append([]string(nil), man.SupportedIngress...),
				Source:           source,
			}
		}
	}
	// bare binaries plugin-*.exe
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "plugin-") {
			continue
		}
		if isWindows() && !strings.HasSuffix(lower, ".exe") {
			continue
		}
		// strip plugin- prefix and .exe
		id := strings.TrimSuffix(strings.TrimPrefix(name, "plugin-"), ".exe")
		id = strings.TrimSuffix(id, ".EXE")
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			// keep manifest-backed entry
			continue
		}
		full := filepath.Join(dir, name)
		seen[id] = DiscoverCandidate{
			ID:         id,
			Name:       id,
			Executable: full,
			Source:     source,
		}
	}
}

func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}
