package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

type Config struct {
	Host                      string        // 代理服务主机地址
	Port                      int           // 代理服务端口号
	ReadTimeout               time.Duration // 代理服务读取超时时间
	WriteTimeout              time.Duration // 代理服务写入超时时间
	ShutdownTimeout           time.Duration // 代理服务关闭超时时间
	ProxyAPIKeys              []string      // 代理 API 密钥列表
	AllowUnauthenticatedLocal bool          // 是否允许未认证本地请求
	ChainLogPath              string        // 链路日志路径
	ChainLogBodies            bool          // 是否记录请求和响应体
	ChainLogMaxBodyBytes      int           // 最大记录请求和响应体字节数
	DefaultMaxTokens          int           // 项目级默认 max_tokens 兜底值

	AnthropicConfig        *AnthropicConfig
	OpenAIRResponsesConfig *OpenAIRResponsesConfig
	OpenAIChatConfig       *OpenAIChatConfig
}

type AnthropicConfig struct {
	Vendor     consts.Vendor
	BaseURL    string
	APIKey     string
	OnlyStream bool
	UserAgent  string
	Version    string
}

type OpenAIRResponsesConfig struct {
	Vendor     consts.Vendor
	BaseURL    string
	APIKey     string
	OnlyStream bool
	UserAgent  string
	Version    string
}

type OpenAIChatConfig struct {
	Vendor     consts.Vendor
	BaseURL    string
	APIKey     string
	OnlyStream bool
	UserAgent  string
	Version    string
}

func Load(workdir string) (Config, error) {
	if err := loadDotEnv(filepath.Join(workdir, ".env")); err != nil {
		return Config{}, err
	}
	cfg := Config{
		Host:                      strings.TrimSpace(os.Getenv("PROXY_HOST")),
		Port:                      intFromEnv("PROXY_PORT", 18181),
		ReadTimeout:               durationFromEnv("PROXY_READ_TIMEOUT_SECONDS", 15*time.Second),
		WriteTimeout:              durationFromEnv("PROXY_WRITE_TIMEOUT_SECONDS", 0),
		ShutdownTimeout:           durationFromEnv("PROXY_SHUTDOWN_TIMEOUT_SECONDS", 10*time.Second),
		ProxyAPIKeys:              csvFromEnv("PROXY_API_KEYS"),
		AllowUnauthenticatedLocal: boolFromEnv("PROXY_ALLOW_UNAUTHENTICATED_LOCAL", true),
		ChainLogPath:              strings.TrimSpace(os.Getenv("PROXY_CHAIN_LOG_PATH")),
		ChainLogBodies:            boolFromEnv("PROXY_CHAIN_LOG_BODIES", true),
		ChainLogMaxBodyBytes:      nonNegativeIntFromEnv("PROXY_CHAIN_LOG_MAX_BODY_BYTES", 0),
		DefaultMaxTokens:          intFromEnv("PROXY_DEFAULT_MAX_TOKENS", models.DefaultSupplierModelMaxTokens),

		AnthropicConfig:        &AnthropicConfig{Vendor: consts.VendorAnthropic},
		OpenAIRResponsesConfig: &OpenAIRResponsesConfig{Vendor: consts.VendorOpenAI},
		OpenAIChatConfig:       &OpenAIChatConfig{Vendor: consts.VendorOpenAI},
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.ChainLogPath == "" {
		cfg.ChainLogPath = filepath.Join(workdir, ".data", "icoo_proxy-chain.log")
	}
	return cfg, nil
}

func (c Config) AuthKeys() []string {
	return slices.Clone(c.ProxyAPIKeys)
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		value = strings.Trim(value, "'")
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func intFromEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func nonNegativeIntFromEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func boolFromEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}

func csvFromEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	return mergeUniqueValues([]string{raw})
}

func mergeUniqueValues(groups ...[]string) []string {
	values := make([]string, 0)
	for _, group := range groups {
		for _, item := range group {
			for _, part := range strings.Split(item, ",") {
				value := strings.TrimSpace(part)
				if value != "" && !slices.Contains(values, value) {
					values = append(values, value)
				}
			}
		}
	}
	return values
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	seconds := intFromEnv(key, int(fallback/time.Second))
	return time.Duration(seconds) * time.Second
}
