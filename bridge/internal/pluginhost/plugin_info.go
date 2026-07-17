package pluginhost

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// InfoFileName is the standard plugin metadata file inside each plugin directory.
const InfoFileName = "info.toml"

// PluginInfo is the on-disk metadata for a packaged process plugin.
// Layout (one plugin per directory):
//
//	plugins/<id>/
//	  info.toml
//	  <executable>
type PluginInfo struct {
	ID               string   `toml:"id"`
	Name             string   `toml:"name"`
	Version          string   `toml:"version"`
	Description      string   `toml:"description"`
	Executable       string   `toml:"executable"`
	Author           string   `toml:"author,omitempty"`
	Homepage         string   `toml:"homepage,omitempty"`
	Disclaimer       string   `toml:"disclaimer,omitempty"`
	Capabilities     []string `toml:"capabilities,omitempty"`
	SupportedIngress []string `toml:"supported_ingress,omitempty"`
}

// LoadPluginInfo reads info.toml from path (file path, not directory).
func LoadPluginInfo(path string) (*PluginInfo, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info PluginInfo
	if _, err := toml.Decode(string(raw), &info); err != nil {
		return nil, fmt.Errorf("pluginhost: decode %s: %w", path, err)
	}
	info.ID = strings.TrimSpace(info.ID)
	info.Name = strings.TrimSpace(info.Name)
	info.Version = strings.TrimSpace(info.Version)
	info.Description = strings.TrimSpace(info.Description)
	info.Executable = strings.TrimSpace(info.Executable)
	if info.ID == "" {
		// Default id from parent directory name.
		info.ID = filepath.Base(filepath.Dir(path))
	}
	if info.ID == "" || info.ID == "." || info.ID == string(filepath.Separator) {
		return nil, fmt.Errorf("pluginhost: info.toml missing id")
	}
	if info.Name == "" {
		info.Name = info.ID
	}
	return &info, nil
}

// ResolvePluginExecutable returns an absolute path to the plugin binary under dir.
func ResolvePluginExecutable(dir string, exeHint string, pluginID string) string {
	dir = filepath.Clean(dir)
	candidates := make([]string, 0, 6)
	hint := strings.TrimSpace(exeHint)
	if hint != "" {
		if filepath.IsAbs(hint) {
			candidates = append(candidates, hint)
		} else {
			candidates = append(candidates, filepath.Join(dir, hint))
			// Allow info.toml without .exe on Windows.
			if isWindows() && !strings.HasSuffix(strings.ToLower(hint), ".exe") {
				candidates = append(candidates, filepath.Join(dir, hint+".exe"))
			}
		}
	}
	// Conventional names.
	base := "plugin-" + pluginID
	candidates = append(candidates, filepath.Join(dir, base))
	if isWindows() {
		candidates = append(candidates, filepath.Join(dir, base+".exe"))
	}
	// Also accept bare id.exe (e.g. mockplugin.exe).
	if isWindows() {
		candidates = append(candidates, filepath.Join(dir, pluginID+".exe"))
	} else {
		candidates = append(candidates, filepath.Join(dir, pluginID))
	}

	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			if abs, err := filepath.Abs(c); err == nil {
				return abs
			}
			return c
		}
	}
	// Return best-effort path even if missing (install UI can show it).
	if hint != "" {
		if filepath.IsAbs(hint) {
			return hint
		}
		p := filepath.Join(dir, hint)
		if isWindows() && !strings.HasSuffix(strings.ToLower(hint), ".exe") {
			p += ".exe"
		}
		return p
	}
	if isWindows() {
		return filepath.Join(dir, base+".exe")
	}
	return filepath.Join(dir, base)
}

// PluginsPackageRoots returns directories that may contain packaged plugins/
// (one subdirectory per plugin with info.toml). Does not include data_dir
// runtime storage (registry + credentials live under data_dir/plugins).
func PluginsPackageRoots() []string {
	roots := make([]string, 0, 4)
	seen := map[string]struct{}{}
	add := func(dir string) {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "" || dir == "." {
			return
		}
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		roots = append(roots, dir)
	}
	if self, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(self), "plugins"))
	}
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "plugins"))
	}
	return roots
}

// EnsurePluginsPackageDirs creates the default package plugins/ directories if missing.
func EnsurePluginsPackageDirs() {
	for _, root := range PluginsPackageRoots() {
		_ = os.MkdirAll(root, 0o755)
	}
}
