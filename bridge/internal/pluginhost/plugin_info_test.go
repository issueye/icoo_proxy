package pluginhost

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPluginInfoAndDiscoverTree(t *testing.T) {
	root := t.TempDir()
	pluginsRoot := filepath.Join(root, "plugins")
	dir := filepath.Join(pluginsRoot, "grokbuild")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	info := `
id = "grokbuild"
name = "GrokBuild"
version = "0.3.2"
description = "test plugin"
executable = "plugin-grokbuild.exe"
capabilities = ["proxy.complete", "health"]
supported_ingress = ["openai-chat"]
`
	if err := os.WriteFile(filepath.Join(dir, InfoFileName), []byte(info), 0o644); err != nil {
		t.Fatal(err)
	}
	exeName := "plugin-grokbuild.exe"
	if !isWindows() {
		// ResolvePluginExecutable still looks for .exe hint path first on non-windows
		// via exact name from info.toml; create that file name for the test.
	}
	if err := os.WriteFile(filepath.Join(dir, exeName), []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPluginInfo(filepath.Join(dir, InfoFileName))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "grokbuild" || loaded.Version != "0.3.2" {
		t.Fatalf("unexpected info: %+v", loaded)
	}

	// Point package roots only via cwd.
	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	cands := DiscoverPlugins(filepath.Join(root, "data"), nil)
	found := false
	for _, c := range cands {
		if c.ID == "grokbuild" {
			found = true
			if c.Name != "GrokBuild" {
				t.Fatalf("name = %q", c.Name)
			}
			if c.Description != "test plugin" {
				t.Fatalf("description = %q", c.Description)
			}
			if c.Source != "plugins_dir" {
				t.Fatalf("source = %q", c.Source)
			}
			if !fileExists(c.Executable) {
				t.Fatalf("executable missing: %s", c.Executable)
			}
		}
	}
	if !found {
		t.Fatalf("discover missed grokbuild: %+v", cands)
	}
}
