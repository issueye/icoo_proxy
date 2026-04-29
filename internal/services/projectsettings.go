package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"icoo_proxy/internal/models"
)

type Values struct {
	ProxyHost                   string `json:"proxy_host"`
	ProxyPort                   int    `json:"proxy_port"`
	ProxyReadTimeoutSeconds     int    `json:"proxy_read_timeout_seconds"`
	ProxyWriteTimeoutSeconds    int    `json:"proxy_write_timeout_seconds"`
	ProxyShutdownTimeoutSeconds int    `json:"proxy_shutdown_timeout_seconds"`
	DefaultMaxTokens            int    `json:"default_max_tokens"`
	ProxyChainLogPath           string `json:"proxy_chain_log_path"`
	ProxyChainLogBodies         bool   `json:"proxy_chain_log_bodies"`
	ProxyChainLogMaxBodyBytes   int    `json:"proxy_chain_log_max_body_bytes"`
}

var managedEnvKeys = []string{
	"PROXY_HOST",
	"PROXY_PORT",
	"PROXY_READ_TIMEOUT_SECONDS",
	"PROXY_WRITE_TIMEOUT_SECONDS",
	"PROXY_SHUTDOWN_TIMEOUT_SECONDS",
	"PROXY_DEFAULT_MAX_TOKENS",
	"PROXY_CHAIN_LOG_PATH",
	"PROXY_CHAIN_LOG_BODIES",
	"PROXY_CHAIN_LOG_MAX_BODY_BYTES",
}

type ProjectSettingsService struct{}

func NewProjectSettingsService() *ProjectSettingsService {
	return &ProjectSettingsService{}
}

func (s *ProjectSettingsService) Load(root string) (Values, error) {
	env, err := s.readEnvFile(filepath.Join(root, ".env"))
	if err != nil {
		return Values{}, err
	}
	return Values{
		ProxyHost:                   stringWithDefault(env, "PROXY_HOST", "127.0.0.1"),
		ProxyPort:                   intWithDefault(env, "PROXY_PORT", 18181),
		ProxyReadTimeoutSeconds:     intWithDefault(env, "PROXY_READ_TIMEOUT_SECONDS", 15),
		ProxyWriteTimeoutSeconds:    intWithDefault(env, "PROXY_WRITE_TIMEOUT_SECONDS", 300),
		ProxyShutdownTimeoutSeconds: intWithDefault(env, "PROXY_SHUTDOWN_TIMEOUT_SECONDS", 10),
		DefaultMaxTokens:            intWithDefault(env, "PROXY_DEFAULT_MAX_TOKENS", models.DefaultSupplierModelMaxTokens),
		ProxyChainLogPath:           stringWithDefault(env, "PROXY_CHAIN_LOG_PATH", filepath.Join(root, ".data", "icoo_proxy-chain.log")),
		ProxyChainLogBodies:         boolWithDefault(env, "PROXY_CHAIN_LOG_BODIES", true),
		ProxyChainLogMaxBodyBytes:   intWithDefault(env, "PROXY_CHAIN_LOG_MAX_BODY_BYTES", 0),
	}, nil
}

func (s *ProjectSettingsService) Save(root string, values Values) error {
	if err := s.validate(values); err != nil {
		return err
	}
	envPath := filepath.Join(root, ".env")
	existing, err := os.ReadFile(envPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := s.mergeEnvContent(string(existing), values)
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		return err
	}
	s.applyProcessEnv(values)
	return nil
}

func (s *ProjectSettingsService) validate(values Values) error {
	if strings.TrimSpace(values.ProxyHost) == "" {
		return fmt.Errorf("proxy_host is required")
	}
	if values.ProxyPort <= 0 {
		return fmt.Errorf("proxy_port must be greater than 0")
	}
	if values.ProxyReadTimeoutSeconds <= 0 {
		return fmt.Errorf("proxy_read_timeout_seconds must be greater than 0")
	}
	if values.ProxyWriteTimeoutSeconds <= 0 {
		return fmt.Errorf("proxy_write_timeout_seconds must be greater than 0")
	}
	if values.ProxyShutdownTimeoutSeconds <= 0 {
		return fmt.Errorf("proxy_shutdown_timeout_seconds must be greater than 0")
	}
	if values.DefaultMaxTokens <= 0 {
		return fmt.Errorf("default_max_tokens must be greater than 0")
	}
	if values.ProxyChainLogMaxBodyBytes < 0 {
		return fmt.Errorf("proxy_chain_log_max_body_bytes must be 0 or greater")
	}
	return nil
}

func (s *ProjectSettingsService) managedEnvEntries(values Values) map[string]string {
	return map[string]string{
		"PROXY_HOST":                     strings.TrimSpace(values.ProxyHost),
		"PROXY_PORT":                     strconv.Itoa(values.ProxyPort),
		"PROXY_READ_TIMEOUT_SECONDS":     strconv.Itoa(values.ProxyReadTimeoutSeconds),
		"PROXY_WRITE_TIMEOUT_SECONDS":    strconv.Itoa(values.ProxyWriteTimeoutSeconds),
		"PROXY_SHUTDOWN_TIMEOUT_SECONDS": strconv.Itoa(values.ProxyShutdownTimeoutSeconds),
		"PROXY_DEFAULT_MAX_TOKENS":       strconv.Itoa(values.DefaultMaxTokens),
		"PROXY_CHAIN_LOG_PATH":           strings.TrimSpace(values.ProxyChainLogPath),
		"PROXY_CHAIN_LOG_BODIES":         s.formatBool(values.ProxyChainLogBodies),
		"PROXY_CHAIN_LOG_MAX_BODY_BYTES": strconv.Itoa(values.ProxyChainLogMaxBodyBytes),
	}
}

func (s *ProjectSettingsService) mergeEnvContent(existing string, values Values) string {
	entries := s.managedEnvEntries(values)
	lines := strings.Split(existing, "\n")
	output := make([]string, 0, len(lines)+len(managedEnvKeys)+1)
	written := make(map[string]bool, len(managedEnvKeys))
	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			output = append(output, rawLine)
			continue
		}
		key, _, found := strings.Cut(rawLine, "=")
		if !found {
			output = append(output, rawLine)
			continue
		}
		managedKey := strings.TrimSpace(key)
		value, ok := entries[managedKey]
		if !ok {
			output = append(output, rawLine)
			continue
		}
		if written[managedKey] {
			continue
		}
		output = append(output, managedKey+"="+value)
		written[managedKey] = true
	}
	missing := make([]string, 0, len(managedEnvKeys))
	for _, key := range managedEnvKeys {
		if !written[key] {
			missing = append(missing, key+"="+entries[key])
		}
	}
	if len(missing) > 0 && len(output) > 0 && strings.TrimSpace(output[len(output)-1]) != "" {
		output = append(output, "")
	}
	output = append(output, missing...)
	if len(output) == 0 || output[len(output)-1] != "" {
		output = append(output, "")
	}
	return strings.Join(output, "\n")
}

func (s *ProjectSettingsService) formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (s *ProjectSettingsService) applyProcessEnv(values Values) {
	set := func(key, value string) {
		_ = os.Setenv(key, value)
	}
	set("PROXY_HOST", strings.TrimSpace(values.ProxyHost))
	set("PROXY_PORT", strconv.Itoa(values.ProxyPort))
	set("PROXY_READ_TIMEOUT_SECONDS", strconv.Itoa(values.ProxyReadTimeoutSeconds))
	set("PROXY_WRITE_TIMEOUT_SECONDS", strconv.Itoa(values.ProxyWriteTimeoutSeconds))
	set("PROXY_SHUTDOWN_TIMEOUT_SECONDS", strconv.Itoa(values.ProxyShutdownTimeoutSeconds))
	set("PROXY_DEFAULT_MAX_TOKENS", strconv.Itoa(values.DefaultMaxTokens))
	set("PROXY_CHAIN_LOG_PATH", strings.TrimSpace(values.ProxyChainLogPath))
	set("PROXY_CHAIN_LOG_BODIES", s.formatBool(values.ProxyChainLogBodies))
	set("PROXY_CHAIN_LOG_MAX_BODY_BYTES", strconv.Itoa(values.ProxyChainLogMaxBodyBytes))
}

func (s *ProjectSettingsService) readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	values := make(map[string]string)
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return values, nil
}

func stringWithDefault(values map[string]string, key, fallback string) string {
	if value := strings.TrimSpace(values[key]); value != "" {
		return value
	}
	return fallback
}

func intWithDefault(values map[string]string, key string, fallback int) int {
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func boolWithDefault(values map[string]string, key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(values[key])) {
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
