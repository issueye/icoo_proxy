package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPluginsConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
max_request_body_bytes = 1048576
[plugins]
max_frame_bytes = 0
max_concurrent_streams = 16
heartbeat_interval_seconds = 3
[plugins.entries.grokbuild]
enabled = false
executable = "plugin-grokbuild.exe"
data_dir = ".data/plugins/grokbuild"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Plugins.MaxConcurrentStreams != 16 {
		t.Fatalf("streams=%d", cfg.Plugins.MaxConcurrentStreams)
	}
	e, ok := cfg.Plugins.Entries["grokbuild"]
	if !ok {
		t.Fatal("missing grokbuild entry")
	}
	if e.Enabled {
		t.Fatal("should be disabled by default in fixture")
	}
	if e.Executable != "plugin-grokbuild.exe" {
		t.Fatalf("exe=%s", e.Executable)
	}
	if cfg.EffectiveMaxFrameBytes() != 1048576 {
		t.Fatalf("frame bytes=%d", cfg.EffectiveMaxFrameBytes())
	}
}

func TestLoadAcceptsNestedAndDesktopFlatLogConfig(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "canonical nested log config",
			config: `[log]
chain_log_path = "nested.log"
chain_log_bodies = true
chain_log_max_body_bytes = 1234
`,
		},
		{
			name: "desktop flat log config",
			config: `chain_log_path = "nested.log"
chain_log_bodies = true
chain_log_max_body_bytes = 1234
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(tt.config), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Log.ChainLogPath != "nested.log" {
				t.Fatalf("chain log path = %q", cfg.Log.ChainLogPath)
			}
			if !cfg.Log.ChainLogBodies {
				t.Fatal("chain log bodies = false")
			}
			if cfg.Log.ChainLogMaxBodyBytes != 1234 {
				t.Fatalf("chain log max body bytes = %d", cfg.Log.ChainLogMaxBodyBytes)
			}
		})
	}
}
