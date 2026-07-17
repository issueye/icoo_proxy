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
	Description      string   `json:"description,omitempty"`
	Executable       string   `json:"executable"`
	ManifestPath     string   `json:"manifest_path,omitempty"` // info.toml or legacy manifest path
	Capabilities     []string `json:"capabilities,omitempty"`
	SupportedIngress []string `json:"supported_ingress,omitempty"`
	Registered       bool     `json:"registered"`
	Source           string   `json:"source"` // plugins_dir | bridge_dir | data_dir | cwd
}

// DiscoverPlugins scans package plugins/ directories (preferred) then legacy locations.
// Preferred layout:
//
//	plugins/<plugin_id>/info.toml
//	plugins/<plugin_id>/<executable>
func DiscoverPlugins(dataDir string, registered map[string]config.PluginEntry) []DiscoverCandidate {
	EnsurePluginsPackageDirs()
	seen := map[string]DiscoverCandidate{}

	// 1) Primary: package plugins/<id>/info.toml next to bridge and under cwd.
	for _, root := range PluginsPackageRoots() {
		scanPluginsTree(root, "plugins_dir", seen)
	}

	// 2) Legacy: flat binaries / plugin.manifest.json next to bridge and in cwd.
	if self, err := os.Executable(); err == nil {
		scanPluginDir(filepath.Dir(self), "bridge_dir", seen)
	}
	if cwd, err := os.Getwd(); err == nil {
		scanPluginDir(cwd, "cwd", seen)
	}

	// 3) Runtime data_dir/plugins/<id>/ — may hold copies or legacy installs.
	// Skip registry.json and non-plugin dirs without info/manifest.
	pluginsRoot := filepath.Join(dataDir, "plugins")
	if entries, err := os.ReadDir(pluginsRoot); err == nil {
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			// registry / credential dirs without info.toml are not candidates.
			sub := filepath.Join(pluginsRoot, ent.Name())
			if infoPath := filepath.Join(sub, InfoFileName); fileExists(infoPath) {
				if c, ok := candidateFromInfoDir(sub, "data_dir"); ok {
					mergeCandidate(seen, c)
				}
				continue
			}
			scanPluginDir(sub, "data_dir", seen)
			if man, err := pluginipc.LoadManifest(filepath.Join(sub, "plugin.manifest.json")); err == nil {
				exe := ResolvePluginExecutable(sub, man.Executable, man.PluginID)
				c := DiscoverCandidate{
					ID:               man.PluginID,
					Name:             man.Name,
					Version:          man.Version,
					Description:      man.Disclaimer,
					Executable:       exe,
					ManifestPath:     filepath.Join(sub, "plugin.manifest.json"),
					Capabilities:     append([]string(nil), man.Capabilities...),
					SupportedIngress: append([]string(nil), man.SupportedIngress...),
					Source:           "data_dir",
				}
				mergeCandidate(seen, c)
			}
		}
	}

	out := make([]DiscoverCandidate, 0, len(seen))
	for id, c := range seen {
		_, c.Registered = registered[id]
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

// scanPluginsTree walks plugins/<id>/ directories for info.toml.
func scanPluginsTree(root, source string, seen map[string]DiscoverCandidate) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		// Skip hidden / system-ish names.
		name := ent.Name()
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		sub := filepath.Join(root, name)
		if c, ok := candidateFromInfoDir(sub, source); ok {
			mergeCandidate(seen, c)
		}
	}
}

func candidateFromInfoDir(dir, source string) (DiscoverCandidate, bool) {
	infoPath := filepath.Join(dir, InfoFileName)
	info, err := LoadPluginInfo(infoPath)
	if err != nil {
		return DiscoverCandidate{}, false
	}
	// Prefer directory name as id when info id missing was already filled;
	// if info.id disagrees with folder name, info.toml wins.
	exe := ResolvePluginExecutable(dir, info.Executable, info.ID)
	return DiscoverCandidate{
		ID:               info.ID,
		Name:             info.Name,
		Version:          info.Version,
		Description:      info.Description,
		Executable:       exe,
		ManifestPath:     infoPath,
		Capabilities:     append([]string(nil), info.Capabilities...),
		SupportedIngress: append([]string(nil), info.SupportedIngress...),
		Source:           source,
	}, true
}

func mergeCandidate(seen map[string]DiscoverCandidate, c DiscoverCandidate) {
	if c.ID == "" {
		return
	}
	prev, ok := seen[c.ID]
	if !ok {
		seen[c.ID] = c
		return
	}
	// Prefer info.toml / plugins_dir over bare binary or legacy sources.
	preferNew := false
	if strings.HasSuffix(strings.ToLower(c.ManifestPath), InfoFileName) &&
		!strings.HasSuffix(strings.ToLower(prev.ManifestPath), InfoFileName) {
		preferNew = true
	}
	if c.Source == "plugins_dir" && prev.Source != "plugins_dir" {
		preferNew = true
	}
	if c.Description != "" && prev.Description == "" {
		preferNew = true
	}
	if preferNew {
		seen[c.ID] = c
	}
}

func scanPluginDir(dir, source string, seen map[string]DiscoverCandidate) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	// Prefer info.toml in this dir (single-plugin layout without nesting).
	if c, ok := candidateFromInfoDir(dir, source); ok {
		mergeCandidate(seen, c)
	}
	// Legacy manifests.
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
			exe := ResolvePluginExecutable(dir, man.Executable, man.PluginID)
			mergeCandidate(seen, DiscoverCandidate{
				ID:               man.PluginID,
				Name:             man.Name,
				Version:          man.Version,
				Description:      man.Disclaimer,
				Executable:       exe,
				ManifestPath:     path,
				Capabilities:     append([]string(nil), man.Capabilities...),
				SupportedIngress: append([]string(nil), man.SupportedIngress...),
				Source:           source,
			})
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
		id := strings.TrimSuffix(strings.TrimPrefix(name, "plugin-"), ".exe")
		id = strings.TrimSuffix(id, ".EXE")
		if id == "" {
			continue
		}
		if prev, ok := seen[id]; ok && prev.ManifestPath != "" {
			continue
		}
		full := filepath.Join(dir, name)
		if abs, err := filepath.Abs(full); err == nil {
			full = abs
		}
		mergeCandidate(seen, DiscoverCandidate{
			ID:         id,
			Name:       id,
			Executable: full,
			Source:     source,
		})
	}
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}
