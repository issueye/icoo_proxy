package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"icoo_proxy/internal/appdb"
	"icoo_proxy/internal/config"

	"github.com/glebarez/sqlite"
	"github.com/pelletier/go-toml/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FullConfig struct {
	Gateway   config.GatewayConfig    `json:"gateway" toml:"gateway"`
	APIKeys   []config.ApiKeyConfig   `json:"apiKeys" toml:"api_keys"`
	Providers []config.ProviderConfig `json:"providers" toml:"providers"`
	Endpoints []config.EndpointConfig `json:"endpoints" toml:"endpoints"`
}

// Re-export config types for convenience.
type GatewayConfig = config.GatewayConfig
type APIKeyConfig = config.ApiKeyConfig
type ProviderConfig = config.ProviderConfig
type EndpointConfig = config.EndpointConfig

type ConfigService struct {
	config           *FullConfig
	configPath       string
	legacyConfigPath string
	keyPath          string
	db               *gorm.DB
	mu               sync.RWMutex
	ctx              context.Context
}

type settingRecord struct {
	Key   string `gorm:"primaryKey;column:key"`
	Value string `gorm:"not null;column:value"`
}

func (settingRecord) TableName() string { return "settings" }

type providerRecord struct {
	ID           string `gorm:"primaryKey;column:id"`
	Name         string `gorm:"not null;column:name"`
	Type         string `gorm:"not null;column:type"`
	APIBase      string `gorm:"not null;column:api_base"`
	APIKey       string `gorm:"not null;column:api_key"`
	EndpointMode string `gorm:"not null;column:endpoint_mode"`
	Enabled      bool   `gorm:"not null;column:enabled"`
	Priority     int    `gorm:"not null;column:priority"`
	ExtraConfig  string `gorm:"not null;column:extra_config"`
	LLMs         string `gorm:"not null;column:llms"`
	DefaultModel string `gorm:"not null;column:default_model"`
}

func (providerRecord) TableName() string { return "providers" }

type apiKeyRecord struct {
	ID          string    `gorm:"primaryKey;column:id"`
	Name        string    `gorm:"not null;column:name"`
	Key         string    `gorm:"not null;column:key_value"`
	Description string    `gorm:"not null;column:description"`
	Enabled     bool      `gorm:"not null;column:enabled"`
	ScopeMode   string    `gorm:"not null;column:scope_mode"`
	ProviderIDs string    `gorm:"not null;column:provider_ids"`
	EndpointIDs string    `gorm:"not null;column:endpoint_ids"`
	LastUsedAt  time.Time `gorm:"column:last_used_at"`
	CreatedAt   time.Time `gorm:"not null;column:created_at"`
	UpdatedAt   time.Time `gorm:"not null;column:updated_at"`
}

func (apiKeyRecord) TableName() string { return "api_keys" }

type endpointRecord struct {
	ID               string `gorm:"primaryKey;column:id"`
	Name             string `gorm:"not null;column:name"`
	ProviderID       string `gorm:"not null;column:provider_id"`
	Path             string `gorm:"not null;column:path"`
	Method           string `gorm:"not null;column:method"`
	Capability       string `gorm:"not null;column:capability"`
	RequestProtocol  string `gorm:"not null;column:request_protocol"`
	ResponseProtocol string `gorm:"not null;column:response_protocol"`
	Enabled          bool   `gorm:"not null;column:enabled"`
	Priority         int    `gorm:"not null;column:priority"`
	IsDefault        bool   `gorm:"not null;column:is_default"`
	Remark           string `gorm:"not null;column:remark"`
}

func (endpointRecord) TableName() string { return "endpoints" }

var configService *ConfigService
var configOnce sync.Once

func defaultConfig() *FullConfig {
	return &FullConfig{
		Gateway: config.GatewayConfig{
			ListenHost:      "127.0.0.1",
			ListenPort:      16790,
			LogLevel:        "info",
			RetryCount:      2,
			RetryIntervalMs: 500,
		},
		APIKeys:   []config.ApiKeyConfig{},
		Providers: []config.ProviderConfig{},
		Endpoints: []config.EndpointConfig{},
	}
}

func GetConfigService() *ConfigService {
	configOnce.Do(func() {
		configService = &ConfigService{config: defaultConfig()}
	})
	return configService
}

var _ config.ConfigProvider = (*ConfigService)(nil)

func (s *ConfigService) Init(ctx context.Context) {
	s.ctx = ctx
	workingDir := appdb.WorkingDir()
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		runtime.LogWarning(s.ctx, "Failed to create config directory: "+err.Error())
	}
	s.configPath = appdb.DBPath()
	s.legacyConfigPath = appdb.LegacyConfigPath()
	s.keyPath = appdb.KeyPath()
	if err := s.ensureDatabase(); err != nil {
		runtime.LogWarning(s.ctx, "Failed to initialize config database: "+err.Error())
		return
	}
	if err := s.Load(); err != nil {
		runtime.LogWarning(s.ctx, "Failed to load config database: "+err.Error())
	}
}

func (s *ConfigService) ensureDatabase() error {
	if s.configPath == "" {
		return fmt.Errorf("config path is empty")
	}
	if s.db != nil {
		return nil
	}

	db, err := gorm.Open(sqlite.Open(s.configPath), &gorm.Config{})
	if err != nil {
		return err
	}

	s.db = db
	return s.initSchema()
}

func (s *ConfigService) Load() error {
	if err := s.ensureDatabase(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	empty, err := s.isDatabaseEmptyLocked()
	if err != nil {
		return err
	}
	if empty {
		migrated, err := s.migrateLegacyConfigLocked()
		if err != nil {
			return err
		}
		if migrated {
			return nil
		}
		s.applyDefaultsLocked(defaultConfig())
		return s.saveLocked()
	}

	cfg, err := s.loadFromDBLocked()
	if err != nil {
		return err
	}
	changed := s.upgradeLoadedConfigLocked(cfg)
	s.applyDefaultsLocked(cfg)
	if changed {
		return s.saveLocked()
	}
	return nil
}

func (s *ConfigService) Save() error {
	if err := s.ensureDatabase(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *ConfigService) GetConfig() *FullConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := *s.config
	return &cfg
}

func (s *ConfigService) GetGatewayConfig() config.GatewayConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Gateway
}

func (s *ConfigService) SetGatewayConfig(cfg config.GatewayConfig) error {
	s.mu.Lock()
	current := *s.config
	current.Gateway = cfg
	migrateGatewayAuthKeyToAPIKey(&current, time.Now())
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) GetProviders() []config.ProviderConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]config.ProviderConfig, len(s.config.Providers))
	copy(result, s.config.Providers)
	return result
}

func (s *ConfigService) AddProvider(p config.ProviderConfig) error {
	s.mu.Lock()
	current := *s.config
	if p.ID == "" {
		p.ID = fmt.Sprintf("provider-%d", time.Now().UnixMilli())
	}
	current.Providers = append(current.Providers, p)
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) UpdateProvider(p config.ProviderConfig) error {
	s.mu.Lock()
	current := *s.config
	found := false
	for i, existing := range current.Providers {
		if existing.ID == p.ID {
			current.Providers[i] = p
			found = true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return fmt.Errorf("provider %s not found", p.ID)
	}
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) DeleteProvider(id string) error {
	s.mu.Lock()
	current := *s.config
	filtered := make([]config.ProviderConfig, 0, len(current.Providers))
	for _, p := range current.Providers {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	current.Providers = filtered
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) GetAPIKeys() []config.ApiKeyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]config.ApiKeyConfig, len(s.config.APIKeys))
	copy(result, s.config.APIKeys)
	return result
}

func (s *ConfigService) AddAPIKey(k config.ApiKeyConfig) error {
	s.mu.Lock()
	current := *s.config
	if strings.TrimSpace(k.ID) == "" {
		k.ID = fmt.Sprintf("apikey-%d", time.Now().UnixMilli())
	}
	now := time.Now()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = now
	}
	k.UpdatedAt = now
	k.ScopeMode = config.NormalizeAPIKeyScopeMode(k.ScopeMode)
	current.APIKeys = append(current.APIKeys, k)
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) UpdateAPIKey(k config.ApiKeyConfig) error {
	s.mu.Lock()
	current := *s.config
	found := false
	for i, existing := range current.APIKeys {
		if existing.ID == k.ID {
			if strings.TrimSpace(k.Key) == "" {
				k.Key = existing.Key
			}
			if k.CreatedAt.IsZero() {
				k.CreatedAt = existing.CreatedAt
			}
			k.UpdatedAt = time.Now()
			k.ScopeMode = config.NormalizeAPIKeyScopeMode(k.ScopeMode)
			current.APIKeys[i] = k
			found = true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return fmt.Errorf("api key %s not found", k.ID)
	}
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) DeleteAPIKey(id string) error {
	s.mu.Lock()
	current := *s.config
	filtered := make([]config.ApiKeyConfig, 0, len(current.APIKeys))
	for _, k := range current.APIKeys {
		if k.ID != id {
			filtered = append(filtered, k)
		}
	}
	current.APIKeys = filtered
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) GetEndpoints() []config.EndpointConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]config.EndpointConfig, len(s.config.Endpoints))
	copy(result, s.config.Endpoints)
	return result
}

func (s *ConfigService) AddEndpoint(e config.EndpointConfig) error {
	s.mu.Lock()
	current := *s.config
	if strings.TrimSpace(e.ID) == "" {
		e.ID = fmt.Sprintf("endpoint-%d", time.Now().UnixMilli())
	}
	current.Endpoints = append(current.Endpoints, e)
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) UpdateEndpoint(e config.EndpointConfig) error {
	s.mu.Lock()
	current := *s.config
	found := false
	for i, existing := range current.Endpoints {
		if existing.ID == e.ID {
			current.Endpoints[i] = e
			found = true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return fmt.Errorf("endpoint %s not found", e.ID)
	}
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) DeleteEndpoint(id string) error {
	s.mu.Lock()
	current := *s.config
	filtered := make([]config.EndpointConfig, 0, len(current.Endpoints))
	for _, e := range current.Endpoints {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	current.Endpoints = filtered
	s.applyDefaultsLocked(&current)
	s.config = &current
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) GetConfigJSON() string {
	s.mu.RLock()
	cfg := *s.config
	s.mu.RUnlock()
	data, _ := json.Marshal(cfg)
	return string(data)
}

func (s *ConfigService) applyDefaultsLocked(cfg *FullConfig) {
	defaults := defaultConfig()
	if strings.TrimSpace(cfg.Gateway.ListenHost) == "" {
		cfg.Gateway.ListenHost = defaults.Gateway.ListenHost
	}
	if cfg.Gateway.ListenPort == 0 {
		cfg.Gateway.ListenPort = defaults.Gateway.ListenPort
	}
	if cfg.Gateway.LogLevel == "" {
		cfg.Gateway.LogLevel = defaults.Gateway.LogLevel
	}
	if cfg.Gateway.RetryCount == 0 {
		cfg.Gateway.RetryCount = defaults.Gateway.RetryCount
	}
	if cfg.Gateway.RetryIntervalMs == 0 {
		cfg.Gateway.RetryIntervalMs = defaults.Gateway.RetryIntervalMs
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = []config.ApiKeyConfig{}
	}
	if cfg.Providers == nil {
		cfg.Providers = []config.ProviderConfig{}
	}
	if cfg.Endpoints == nil {
		cfg.Endpoints = []config.EndpointConfig{}
	}
	for i := range cfg.APIKeys {
		cfg.APIKeys[i].ScopeMode = config.NormalizeAPIKeyScopeMode(cfg.APIKeys[i].ScopeMode)
	}
	for i := range cfg.Providers {
		cfg.Providers[i].EndpointMode = config.NormalizeProviderEndpointMode(cfg.Providers[i].Type, cfg.Providers[i].EndpointMode)
	}
	s.config = cfg
}

func (s *ConfigService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		s.db = nil
		return err
	}
	err = sqlDB.Close()
	s.db = nil
	return err
}

func (s *ConfigService) initSchema() error {
	if err := s.db.AutoMigrate(&settingRecord{}, &providerRecord{}, &apiKeyRecord{}, &endpointRecord{}); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}

func (s *ConfigService) isDatabaseEmptyLocked() (bool, error) {
	var count int64
	if err := s.db.Model(&settingRecord{}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	if err := s.db.Model(&providerRecord{}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	if err := s.db.Model(&apiKeyRecord{}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	if err := s.db.Model(&endpointRecord{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *ConfigService) migrateLegacyConfigLocked() (bool, error) {
	if s.legacyConfigPath == "" {
		return false, nil
	}

	data, err := os.ReadFile(s.legacyConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	cfg := defaultConfig()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return false, fmt.Errorf("failed to decode legacy config: %w", err)
	}

	now := time.Now()
	if strings.TrimSpace(cfg.Gateway.AuthKey) != "" {
		cfg.APIKeys = append(cfg.APIKeys, config.ApiKeyConfig{
			ID:        fmt.Sprintf("migrated-apikey-%d", now.UnixMilli()),
			Name:      "Default Imported Key",
			Key:       strings.TrimSpace(cfg.Gateway.AuthKey),
			Enabled:   true,
			ScopeMode: config.ApiKeyScopeAll,
			CreatedAt: now,
			UpdatedAt: now,
		})
		cfg.Gateway.AuthKey = ""
	}

	for _, p := range cfg.Providers {
		mode := config.NormalizeProviderEndpointMode(p.Type, p.EndpointMode)
		cfg.Endpoints = append(cfg.Endpoints, config.EndpointConfig{
			ID:               fmt.Sprintf("migrated-endpoint-%s", p.ID),
			Name:             p.Name + " default",
			ProviderID:       p.ID,
			Path:             defaultPathForEndpointMode(mode),
			Method:           httpMethodForEndpointMode(mode),
			Capability:       capabilityForEndpointMode(mode),
			RequestProtocol:  requestProtocolForEndpointMode(mode),
			ResponseProtocol: responseProtocolForEndpointMode(mode),
			Enabled:          p.Enabled,
			Priority:         p.Priority,
			IsDefault:        true,
		})
	}

	s.applyDefaultsLocked(cfg)
	return true, s.saveLocked()
}

func (s *ConfigService) upgradeLoadedConfigLocked(cfg *FullConfig) bool {
	changed := false
	now := time.Now()
	if migrated := migrateGatewayAuthKeyToAPIKey(cfg, now); migrated {
		changed = true
	}
	if len(cfg.Endpoints) == 0 && len(cfg.Providers) > 0 {
		for _, p := range cfg.Providers {
			cfg.Endpoints = append(cfg.Endpoints, defaultEndpointForProvider(p))
		}
		changed = true
	}
	return changed
}

func migrateGatewayAuthKeyToAPIKey(cfg *FullConfig, now time.Time) bool {
	legacyKey := strings.TrimSpace(cfg.Gateway.AuthKey)
	if legacyKey == "" {
		return false
	}
	if !apiKeyValueExists(cfg.APIKeys, legacyKey) {
		cfg.APIKeys = append(cfg.APIKeys, config.ApiKeyConfig{
			ID:        fmt.Sprintf("migrated-apikey-%d", now.UnixMilli()),
			Name:      "Default Imported Key",
			Key:       legacyKey,
			Enabled:   true,
			ScopeMode: config.ApiKeyScopeAll,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	cfg.Gateway.AuthKey = ""
	return true
}

func apiKeyValueExists(apiKeys []config.ApiKeyConfig, key string) bool {
	for _, item := range apiKeys {
		if strings.TrimSpace(item.Key) == key {
			return true
		}
	}
	return false
}

func defaultEndpointForProvider(p config.ProviderConfig) config.EndpointConfig {
	mode := config.NormalizeProviderEndpointMode(p.Type, p.EndpointMode)
	return config.EndpointConfig{
		ID:               fmt.Sprintf("migrated-endpoint-%s", p.ID),
		Name:             p.Name + " default",
		ProviderID:       p.ID,
		Path:             defaultPathForEndpointMode(mode),
		Method:           httpMethodForEndpointMode(mode),
		Capability:       capabilityForEndpointMode(mode),
		RequestProtocol:  requestProtocolForEndpointMode(mode),
		ResponseProtocol: responseProtocolForEndpointMode(mode),
		Enabled:          p.Enabled,
		Priority:         p.Priority,
		IsDefault:        true,
	}
}

func (s *ConfigService) loadFromDBLocked() (*FullConfig, error) {
	cfg := defaultConfig()
	if err := s.loadGatewayConfigLocked(&cfg.Gateway); err != nil {
		return nil, err
	}
	apiKeys, err := s.loadAPIKeysLocked()
	if err != nil {
		return nil, err
	}
	providers, err := s.loadProvidersLocked()
	if err != nil {
		return nil, err
	}
	endpoints, err := s.loadEndpointsLocked()
	if err != nil {
		return nil, err
	}
	cfg.APIKeys = apiKeys
	cfg.Providers = providers
	cfg.Endpoints = endpoints
	return cfg, nil
}

func (s *ConfigService) loadSettingJSONLocked(key string, target any) error {
	var record settingRecord
	err := s.db.Where("key = ?", key).Take(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	return json.Unmarshal([]byte(record.Value), target)
}

func (s *ConfigService) loadAPIKeysLocked() ([]config.ApiKeyConfig, error) {
	var records []apiKeyRecord
	if err := s.db.Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]config.ApiKeyConfig, 0, len(records))
	for _, record := range records {
		keyValue, err := s.decryptSecret(record.Key)
		if err != nil {
			return nil, err
		}
		item := config.ApiKeyConfig{
			ID:          record.ID,
			Name:        record.Name,
			Key:         keyValue,
			Description: record.Description,
			Enabled:     record.Enabled,
			ScopeMode:   config.NormalizeAPIKeyScopeMode(record.ScopeMode),
			LastUsedAt:  record.LastUsedAt,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
		}
		if record.ProviderIDs != "" {
			if err := json.Unmarshal([]byte(record.ProviderIDs), &item.ProviderIDs); err != nil {
				return nil, err
			}
		}
		if record.EndpointIDs != "" {
			if err := json.Unmarshal([]byte(record.EndpointIDs), &item.EndpointIDs); err != nil {
				return nil, err
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *ConfigService) loadProvidersLocked() ([]config.ProviderConfig, error) {
	var records []providerRecord
	if err := s.db.Order("priority DESC").Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	providers := make([]config.ProviderConfig, 0, len(records))
	for _, record := range records {
		p := config.ProviderConfig{
			ID:           record.ID,
			Name:         record.Name,
			Type:         record.Type,
			APIBase:      record.APIBase,
			EndpointMode: config.NormalizeProviderEndpointMode(record.Type, record.EndpointMode),
			Enabled:      record.Enabled,
			Priority:     record.Priority,
			DefaultModel: record.DefaultModel,
		}
		apiKey, err := s.decryptSecret(record.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKey = apiKey
		if record.ExtraConfig != "" {
			if err := json.Unmarshal([]byte(record.ExtraConfig), &p.ExtraConfig); err != nil {
				return nil, err
			}
		}
		if record.LLMs != "" {
			if err := json.Unmarshal([]byte(record.LLMs), &p.LLMs); err != nil {
				return nil, err
			}
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (s *ConfigService) loadEndpointsLocked() ([]config.EndpointConfig, error) {
	var records []endpointRecord
	if err := s.db.Order("priority DESC").Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]config.EndpointConfig, 0, len(records))
	for _, record := range records {
		result = append(result, config.EndpointConfig{
			ID:               record.ID,
			Name:             record.Name,
			ProviderID:       record.ProviderID,
			Path:             record.Path,
			Method:           record.Method,
			Capability:       record.Capability,
			RequestProtocol:  record.RequestProtocol,
			ResponseProtocol: record.ResponseProtocol,
			Enabled:          record.Enabled,
			Priority:         record.Priority,
			IsDefault:        record.IsDefault,
			Remark:           record.Remark,
		})
	}
	return result, nil
}

func (s *ConfigService) saveLocked() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.saveSettingsLocked(tx); err != nil {
			return err
		}
		if err := s.saveAPIKeysLocked(tx); err != nil {
			return err
		}
		if err := s.saveProvidersLocked(tx); err != nil {
			return err
		}
		if err := s.saveEndpointsLocked(tx); err != nil {
			return err
		}
		return nil
	})
}

func (s *ConfigService) saveSettingsLocked(tx *gorm.DB) error {
	type kv struct {
		key   string
		value any
	}

	gatewayCfg := s.config.Gateway
	settings := []kv{{key: "gateway", value: gatewayCfg}}
	for _, item := range settings {
		payload, err := json.Marshal(item.value)
		if err != nil {
			return err
		}
		record := settingRecord{Key: item.key, Value: string(payload)}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ConfigService) loadGatewayConfigLocked(target *config.GatewayConfig) error {
	return s.loadSettingJSONLocked("gateway", target)
}

func (s *ConfigService) saveAPIKeysLocked(tx *gorm.DB) error {
	if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&apiKeyRecord{}).Error; err != nil {
		return err
	}

	records := make([]apiKeyRecord, 0, len(s.config.APIKeys))
	for _, k := range s.config.APIKeys {
		providerIDsJSON, err := json.Marshal(k.ProviderIDs)
		if err != nil {
			return err
		}
		endpointIDsJSON, err := json.Marshal(k.EndpointIDs)
		if err != nil {
			return err
		}
		encryptedKey, err := s.encryptSecret(k.Key)
		if err != nil {
			return err
		}
		records = append(records, apiKeyRecord{
			ID:          k.ID,
			Name:        k.Name,
			Key:         encryptedKey,
			Description: k.Description,
			Enabled:     k.Enabled,
			ScopeMode:   config.NormalizeAPIKeyScopeMode(k.ScopeMode),
			ProviderIDs: string(providerIDsJSON),
			EndpointIDs: string(endpointIDsJSON),
			LastUsedAt:  k.LastUsedAt,
			CreatedAt:   k.CreatedAt,
			UpdatedAt:   k.UpdatedAt,
		})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Create(&records).Error
}

func (s *ConfigService) saveProvidersLocked(tx *gorm.DB) error {
	if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&providerRecord{}).Error; err != nil {
		return err
	}

	records := make([]providerRecord, 0, len(s.config.Providers))
	for _, p := range s.config.Providers {
		extraJSON, err := json.Marshal(p.ExtraConfig)
		if err != nil {
			return err
		}
		llmsJSON, err := json.Marshal(p.LLMs)
		if err != nil {
			return err
		}
		encryptedAPIKey, err := s.encryptSecret(p.APIKey)
		if err != nil {
			return err
		}
		records = append(records, providerRecord{
			ID:           p.ID,
			Name:         p.Name,
			Type:         p.Type,
			APIBase:      p.APIBase,
			APIKey:       encryptedAPIKey,
			EndpointMode: config.NormalizeProviderEndpointMode(p.Type, p.EndpointMode),
			Enabled:      p.Enabled,
			Priority:     p.Priority,
			ExtraConfig:  string(extraJSON),
			LLMs:         string(llmsJSON),
			DefaultModel: p.DefaultModel,
		})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Create(&records).Error
}

func (s *ConfigService) saveEndpointsLocked(tx *gorm.DB) error {
	if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&endpointRecord{}).Error; err != nil {
		return err
	}

	records := make([]endpointRecord, 0, len(s.config.Endpoints))
	for _, e := range s.config.Endpoints {
		records = append(records, endpointRecord{
			ID:               e.ID,
			Name:             e.Name,
			ProviderID:       e.ProviderID,
			Path:             e.Path,
			Method:           e.Method,
			Capability:       e.Capability,
			RequestProtocol:  e.RequestProtocol,
			ResponseProtocol: e.ResponseProtocol,
			Enabled:          e.Enabled,
			Priority:         e.Priority,
			IsDefault:        e.IsDefault,
			Remark:           e.Remark,
		})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Create(&records).Error
}

func (s *ConfigService) encryptionKey() ([]byte, error) {
	if s.keyPath == "" {
		return nil, fmt.Errorf("key path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.keyPath), 0755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.keyPath)
	if err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid encryption key length: %d", len(data))
		}
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.keyPath, key, 0600); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *ConfigService) encryptSecret(plaintext string) (string, error) {
	if plaintext == "" || strings.HasPrefix(plaintext, "enc:v1:") {
		return plaintext, nil
	}

	key, err := s.encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return "enc:v1:" + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (s *ConfigService) decryptSecret(ciphertext string) (string, error) {
	if ciphertext == "" || !strings.HasPrefix(ciphertext, "enc:v1:") {
		return ciphertext, nil
	}

	key, err := s.encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:v1:"))
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("encrypted payload is too short")
	}
	nonce := payload[:gcm.NonceSize()]
	data := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func defaultPathForEndpointMode(mode string) string {
	switch mode {
	case config.ProviderEndpointModeResponses:
		return "/v1/responses"
	case config.ProviderEndpointModeAnthropicMessages:
		return "/v1/messages"
	case config.ProviderEndpointModeGeminiGenerate:
		return "/v1beta/models"
	default:
		return "/v1/chat/completions"
	}
}

func httpMethodForEndpointMode(string) string {
	return "POST"
}

func capabilityForEndpointMode(mode string) string {
	switch mode {
	case config.ProviderEndpointModeResponses:
		return "responses"
	default:
		return "chat"
	}
}

func requestProtocolForEndpointMode(mode string) string {
	switch mode {
	case config.ProviderEndpointModeResponses:
		return "openai_responses"
	case config.ProviderEndpointModeAnthropicMessages:
		return "anthropic_messages"
	case config.ProviderEndpointModeGeminiGenerate:
		return "gemini_generate_content"
	default:
		return "openai_chat"
	}
}

func responseProtocolForEndpointMode(mode string) string {
	return requestProtocolForEndpointMode(mode)
}