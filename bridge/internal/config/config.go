package config

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultMaxTokens       = 32768
	DefaultMaxRequestBytes = 64 << 20
)

type Config struct {
	Host                   string
	Port                   int
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	StreamPreflightTimeout time.Duration
	ShutdownTimeout        time.Duration
	AllowLocalWithoutAuth  bool
	DefaultMaxTokens       int
	MaxRequestBodyBytes    int64
	DataDir                string
	DBPath                 string
	TrafficDBPath          string
	Log                    LogConfig
	Archive                ArchiveConfig
	Plugins                PluginsConfig
}

// PluginsConfig is the host-side process plugin configuration.
type PluginsConfig struct {
	// MaxFrameBytes is 0 to follow MaxRequestBodyBytes.
	MaxFrameBytes int64
	// MaxConcurrentStreams is 0 for default 32.
	MaxConcurrentStreams int
	// HeartbeatInterval is 0 for default 5s.
	HeartbeatInterval time.Duration
	// ShutdownPluginTimeout is 0 for default 5s.
	ShutdownPluginTimeout time.Duration
	// Entries maps plugin_id -> config. Empty / disabled by default.
	Entries map[string]PluginEntry
}

// PluginEntry configures one process plugin.
type PluginEntry struct {
	Enabled              bool     `json:"enabled"`
	Executable           string   `json:"executable"` // absolute or relative to data dir / binary dir
	Args                 []string `json:"args,omitempty"`
	DataDir              string   `json:"data_dir,omitempty"` // empty => <data_dir>/plugins/<id>
	AdminEnabled         bool     `json:"admin_enabled,omitempty"`
	MaxConcurrentStreams int      `json:"max_concurrent_streams,omitempty"`
}

type LogConfig struct {
	ChainLogPath         string
	ChainLogBodies       bool
	ChainLogMaxBodyBytes int
}

type ArchiveConfig struct {
	Enabled        bool
	DownRequestDir string
	UpRequestDir   string
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c *Config) ApplyDataDir(dataDir string) {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		return
	}
	c.DataDir = dataDir
	c.DBPath = filepath.Join(dataDir, "icoo_llm_bridge.db")
	c.TrafficDBPath = filepath.Join(dataDir, "icoo_llm_bridge_traffic.db")
	if c.Log.ChainLogPath == "" {
		c.Log.ChainLogPath = filepath.Join(dataDir, "bridge-chain.log")
	}
	if c.Archive.DownRequestDir == "" {
		c.Archive.DownRequestDir = filepath.Join(dataDir, "down_request")
	}
	if c.Archive.UpRequestDir == "" {
		c.Archive.UpRequestDir = filepath.Join(dataDir, "up_request")
	}
}

func (c *Config) ApplyAddrOverride(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse listen address: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		return fmt.Errorf("parse listen address: invalid port")
	}
	c.Host = host
	c.Port = port
	return nil
}
