package config

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DefaultMaxTokens = 32768

type Config struct {
	Host                   string
	Port                   int
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	StreamPreflightTimeout time.Duration
	ShutdownTimeout        time.Duration
	AllowLocalWithoutAuth  bool
	DefaultMaxTokens       int
	DataDir                string
	DBPath                 string
	TrafficDBPath          string
	Log                    LogConfig
	Archive                ArchiveConfig
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
