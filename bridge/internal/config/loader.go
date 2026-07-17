package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type fileConfig struct {
	Host                          string            `toml:"host"`
	Port                          int               `toml:"port"`
	ReadTimeoutSeconds            int               `toml:"read_timeout_seconds"`
	WriteTimeoutSeconds           int               `toml:"write_timeout_seconds"`
	StreamPreflightTimeoutSeconds int               `toml:"stream_preflight_timeout_seconds"`
	ShutdownTimeoutSeconds        int               `toml:"shutdown_timeout_seconds"`
	AllowLocalWithoutAuth         *bool             `toml:"allow_local_without_auth"`
	AllowUnauthLocal              *bool             `toml:"allow_unauthenticated_local"`
	DefaultMaxTokens              int               `toml:"default_max_tokens"`
	MaxRequestBodyBytes           int64             `toml:"max_request_body_bytes"`
	DataDir                       string            `toml:"data_dir"`
	DBPath                        string            `toml:"db_path"`
	TrafficDBPath                 string            `toml:"traffic_db_path"`
	ChainLogPath                  string            `toml:"chain_log_path"`
	ChainLogBodies                *bool             `toml:"chain_log_bodies"`
	ChainLogMaxBodyBytes          *int              `toml:"chain_log_max_body_bytes"`
	Log                           fileLogConfig     `toml:"log"`
	Archive                       fileArchiveConfig `toml:"archive"`
	Plugins                       filePluginsConfig `toml:"plugins"`
}

type filePluginsConfig struct {
	MaxFrameBytes            int64                      `toml:"max_frame_bytes"`
	MaxConcurrentStreams     int                        `toml:"max_concurrent_streams"`
	HeartbeatIntervalSeconds int                        `toml:"heartbeat_interval_seconds"`
	ShutdownPluginTimeoutSec int                        `toml:"shutdown_plugin_timeout_seconds"`
	AutoRestart              *bool                      `toml:"auto_restart"`
	AutoRestartFailThreshold int                        `toml:"auto_restart_fail_threshold"`
	Entries                  map[string]filePluginEntry `toml:"entries"`
}

type filePluginEntry struct {
	Enabled              bool     `toml:"enabled"`
	Executable           string   `toml:"executable"`
	Args                 []string `toml:"args"`
	DataDir              string   `toml:"data_dir"`
	AdminEnabled         bool     `toml:"admin_enabled"`
	MaxConcurrentStreams int      `toml:"max_concurrent_streams"`
}

type fileLogConfig struct {
	ChainLogPath         string `toml:"chain_log_path"`
	ChainLogBodies       *bool  `toml:"chain_log_bodies"`
	ChainLogMaxBodyBytes *int   `toml:"chain_log_max_body_bytes"`
}

type fileArchiveConfig struct {
	Enabled        bool   `toml:"enabled"`
	DownRequestDir string `toml:"down_request_dir"`
	UpRequestDir   string `toml:"up_request_dir"`
}

func Load(path string) (Config, error) {
	cfg := defaults()
	if path == "" {
		path = "config.toml"
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, err
	}
	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	applyFileConfig(&cfg, fc)
	return cfg, nil
}

func defaults() Config {
	dataDir := ".data"
	return Config{
		Host:                   "127.0.0.1",
		Port:                   18181,
		ReadTimeout:            15 * time.Second,
		WriteTimeout:           300 * time.Second,
		StreamPreflightTimeout: 30 * time.Second,
		ShutdownTimeout:        10 * time.Second,
		AllowLocalWithoutAuth:  true,
		DefaultMaxTokens:       DefaultMaxTokens,
		MaxRequestBodyBytes:    DefaultMaxRequestBytes,
		DataDir:                dataDir,
		DBPath:                 filepath.Join(dataDir, "icoo_llm_bridge.db"),
		TrafficDBPath:          filepath.Join(dataDir, "icoo_llm_bridge_traffic.db"),
		Log: LogConfig{
			ChainLogPath:         filepath.Join(dataDir, "bridge-chain.log"),
			ChainLogBodies:       false,
			ChainLogMaxBodyBytes: 8192,
		},
		Archive: ArchiveConfig{
			Enabled:        false,
			DownRequestDir: filepath.Join(dataDir, "down_request"),
			UpRequestDir:   filepath.Join(dataDir, "up_request"),
		},
		Plugins: PluginsConfig{
			MaxFrameBytes:            0, // follow MaxRequestBodyBytes
			MaxConcurrentStreams:     32,
			HeartbeatInterval:        5 * time.Second,
			ShutdownPluginTimeout:    5 * time.Second,
			AutoRestart:              true,
			AutoRestartFailThreshold: 3,
			Entries:                  map[string]PluginEntry{},
		},
	}
}

func applyFileConfig(cfg *Config, fc fileConfig) {
	if fc.Host != "" {
		cfg.Host = fc.Host
	}
	if fc.Port > 0 {
		cfg.Port = fc.Port
	}
	if fc.ReadTimeoutSeconds > 0 {
		cfg.ReadTimeout = time.Duration(fc.ReadTimeoutSeconds) * time.Second
	}
	if fc.WriteTimeoutSeconds > 0 {
		cfg.WriteTimeout = time.Duration(fc.WriteTimeoutSeconds) * time.Second
	}
	if fc.StreamPreflightTimeoutSeconds > 0 {
		cfg.StreamPreflightTimeout = time.Duration(fc.StreamPreflightTimeoutSeconds) * time.Second
	}
	if fc.ShutdownTimeoutSeconds > 0 {
		cfg.ShutdownTimeout = time.Duration(fc.ShutdownTimeoutSeconds) * time.Second
	}
	if fc.AllowLocalWithoutAuth != nil {
		cfg.AllowLocalWithoutAuth = *fc.AllowLocalWithoutAuth
	} else if fc.AllowUnauthLocal != nil {
		cfg.AllowLocalWithoutAuth = *fc.AllowUnauthLocal
	}
	if fc.DefaultMaxTokens > 0 {
		cfg.DefaultMaxTokens = fc.DefaultMaxTokens
	}
	if fc.MaxRequestBodyBytes > 0 {
		cfg.MaxRequestBodyBytes = fc.MaxRequestBodyBytes
	}
	if fc.DataDir != "" {
		cfg.ApplyDataDir(fc.DataDir)
	}
	if fc.DBPath != "" {
		cfg.DBPath = fc.DBPath
	}
	if fc.TrafficDBPath != "" {
		cfg.TrafficDBPath = fc.TrafficDBPath
	}
	hasNestedLog := fc.Log.ChainLogPath != "" || fc.Log.ChainLogBodies != nil || fc.Log.ChainLogMaxBodyBytes != nil
	if hasNestedLog {
		if fc.Log.ChainLogPath != "" {
			cfg.Log.ChainLogPath = fc.Log.ChainLogPath
		}
		if fc.Log.ChainLogBodies != nil {
			cfg.Log.ChainLogBodies = *fc.Log.ChainLogBodies
		}
		if fc.Log.ChainLogMaxBodyBytes != nil && *fc.Log.ChainLogMaxBodyBytes >= 0 {
			cfg.Log.ChainLogMaxBodyBytes = *fc.Log.ChainLogMaxBodyBytes
		}
	} else {
		if fc.ChainLogPath != "" {
			cfg.Log.ChainLogPath = fc.ChainLogPath
		}
		if fc.ChainLogBodies != nil {
			cfg.Log.ChainLogBodies = *fc.ChainLogBodies
		}
		if fc.ChainLogMaxBodyBytes != nil && *fc.ChainLogMaxBodyBytes >= 0 {
			cfg.Log.ChainLogMaxBodyBytes = *fc.ChainLogMaxBodyBytes
		}
	}
	cfg.Archive.Enabled = fc.Archive.Enabled
	if fc.Archive.DownRequestDir != "" {
		cfg.Archive.DownRequestDir = fc.Archive.DownRequestDir
	}
	if fc.Archive.UpRequestDir != "" {
		cfg.Archive.UpRequestDir = fc.Archive.UpRequestDir
	}

	// Plugins (default off; only entries with enabled=true are spawned).
	if fc.Plugins.MaxFrameBytes > 0 {
		cfg.Plugins.MaxFrameBytes = fc.Plugins.MaxFrameBytes
	}
	if fc.Plugins.MaxConcurrentStreams > 0 {
		cfg.Plugins.MaxConcurrentStreams = fc.Plugins.MaxConcurrentStreams
	}
	if fc.Plugins.HeartbeatIntervalSeconds > 0 {
		cfg.Plugins.HeartbeatInterval = time.Duration(fc.Plugins.HeartbeatIntervalSeconds) * time.Second
	}
	if fc.Plugins.ShutdownPluginTimeoutSec > 0 {
		cfg.Plugins.ShutdownPluginTimeout = time.Duration(fc.Plugins.ShutdownPluginTimeoutSec) * time.Second
	}
	if fc.Plugins.AutoRestart != nil {
		cfg.Plugins.AutoRestart = *fc.Plugins.AutoRestart
	}
	if fc.Plugins.AutoRestartFailThreshold > 0 {
		cfg.Plugins.AutoRestartFailThreshold = fc.Plugins.AutoRestartFailThreshold
	}
	if len(fc.Plugins.Entries) > 0 {
		cfg.Plugins.Entries = make(map[string]PluginEntry, len(fc.Plugins.Entries))
		for id, e := range fc.Plugins.Entries {
			cfg.Plugins.Entries[id] = PluginEntry{
				Enabled:              e.Enabled,
				Executable:           e.Executable,
				Args:                 append([]string(nil), e.Args...),
				DataDir:              e.DataDir,
				AdminEnabled:         e.AdminEnabled,
				MaxConcurrentStreams: e.MaxConcurrentStreams,
			}
		}
	}
}

// EffectiveMaxFrameBytes returns the IPC frame limit (bytes).
func (c Config) EffectiveMaxFrameBytes() int64 {
	if c.Plugins.MaxFrameBytes > 0 {
		return c.Plugins.MaxFrameBytes
	}
	if c.MaxRequestBodyBytes > 0 {
		return c.MaxRequestBodyBytes
	}
	return DefaultMaxRequestBytes
}
