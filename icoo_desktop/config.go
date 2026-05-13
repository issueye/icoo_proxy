package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ServerConfig struct {
	Host                      string   `toml:"host"`
	Port                      int      `toml:"port"`
	ReadTimeoutSeconds        int      `toml:"read_timeout_seconds"`
	WriteTimeoutSeconds       int      `toml:"write_timeout_seconds"`
	ShutdownTimeoutSeconds    int      `toml:"shutdown_timeout_seconds"`
	APIKeys                   []string `toml:"api_keys"`
	AllowUnauthenticatedLocal bool     `toml:"allow_local_without_auth"`
	ChainLogPath              string   `toml:"chain_log_path"`
	ChainLogBodies            bool     `toml:"chain_log_bodies"`
	ChainLogMaxBodyBytes      int      `toml:"chain_log_max_body_bytes"`
	DefaultMaxTokens          int      `toml:"default_max_tokens"`
}

type rawServerConfig struct {
	ServerConfig
	LegacyAllowUnauthenticatedLocal *bool `toml:"allow_unauthenticated_local"`
}

func defaultConfig() ServerConfig {
	return ServerConfig{
		Host:                      "127.0.0.1",
		Port:                      18181,
		ReadTimeoutSeconds:        15,
		WriteTimeoutSeconds:       300,
		ShutdownTimeoutSeconds:    10,
		APIKeys:                   nil,
		AllowUnauthenticatedLocal: true,
		ChainLogPath:              ".data/bridge-chain.log",
		ChainLogBodies:            false,
		ChainLogMaxBodyBytes:      8192,
		DefaultMaxTokens:          32768,
	}
}

func (c ServerConfig) URL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
}

func configPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "config.toml"), nil
}

func loadConfig() ServerConfig {
	cfg := defaultConfig()
	path, err := configPath()
	if err != nil {
		_ = saveConfig(cfg)
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		_ = saveConfig(cfg)
		return cfg
	}
	var loaded rawServerConfig
	if err := toml.Unmarshal(data, &loaded); err != nil {
		_ = saveConfig(cfg)
		return cfg
	}
	if loaded.Host != "" {
		cfg.Host = loaded.Host
	}
	if loaded.Port > 0 {
		cfg.Port = loaded.Port
	}
	if loaded.ReadTimeoutSeconds > 0 {
		cfg.ReadTimeoutSeconds = loaded.ReadTimeoutSeconds
	}
	if loaded.WriteTimeoutSeconds > 0 {
		cfg.WriteTimeoutSeconds = loaded.WriteTimeoutSeconds
	}
	if loaded.ShutdownTimeoutSeconds > 0 {
		cfg.ShutdownTimeoutSeconds = loaded.ShutdownTimeoutSeconds
	}
	if loaded.ServerConfig.APIKeys != nil {
		cfg.APIKeys = loaded.ServerConfig.APIKeys
	}
	cfg.AllowUnauthenticatedLocal = loaded.ServerConfig.AllowUnauthenticatedLocal
	if loaded.LegacyAllowUnauthenticatedLocal != nil {
		cfg.AllowUnauthenticatedLocal = *loaded.LegacyAllowUnauthenticatedLocal
	}
	if loaded.ChainLogPath != "" {
		cfg.ChainLogPath = loaded.ChainLogPath
	}
	cfg.ChainLogBodies = loaded.ChainLogBodies
	if loaded.ChainLogMaxBodyBytes >= 0 {
		cfg.ChainLogMaxBodyBytes = loaded.ChainLogMaxBodyBytes
	}
	if loaded.DefaultMaxTokens > 0 {
		cfg.DefaultMaxTokens = loaded.DefaultMaxTokens
	}
	return normalizeConfig(cfg)
}

func saveConfig(cfg ServerConfig) error {
	cfg = normalizeConfig(cfg)
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func normalizeConfig(cfg ServerConfig) ServerConfig {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port <= 0 {
		cfg.Port = 18181
	}
	if cfg.ReadTimeoutSeconds <= 0 {
		cfg.ReadTimeoutSeconds = 15
	}
	if cfg.WriteTimeoutSeconds <= 0 {
		cfg.WriteTimeoutSeconds = 300
	}
	if cfg.ShutdownTimeoutSeconds <= 0 {
		cfg.ShutdownTimeoutSeconds = 10
	}
	if cfg.DefaultMaxTokens <= 0 {
		cfg.DefaultMaxTokens = 32768
	}
	if len(cfg.APIKeys) == 0 && isLocalHost(cfg.Host) {
		cfg.AllowUnauthenticatedLocal = true
	}
	return cfg
}

func isLocalHost(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}
