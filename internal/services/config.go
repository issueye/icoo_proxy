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
	Providers []config.ProviderConfig `json:"providers" toml:"providers"`
}

// Re-export config types for convenience
type GatewayConfig = config.GatewayConfig
type ProviderConfig = config.ProviderConfig

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
		Providers: []config.ProviderConfig{},
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

	if empty, err := s.isDatabaseEmptyLocked(); err != nil {
		return err
	} else if empty {
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
	s.applyDefaultsLocked(cfg)
	return nil
}

func (s *ConfigService) Save() error {
	if err := s.ensureDatabase(); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.applyDefaultsLocked(&current)
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
	if cfg.Providers == nil {
		cfg.Providers = []config.ProviderConfig{}
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
	if err := s.db.AutoMigrate(&settingRecord{}, &providerRecord{}); err != nil {
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
	s.applyDefaultsLocked(cfg)
	return true, s.saveLocked()
}

func (s *ConfigService) loadFromDBLocked() (*FullConfig, error) {
	cfg := defaultConfig()
	if err := s.loadGatewayConfigLocked(&cfg.Gateway); err != nil {
		return nil, err
	}
	providers, err := s.loadProvidersLocked()
	if err != nil {
		return nil, err
	}
	cfg.Providers = providers
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

func (s *ConfigService) loadProvidersLocked() ([]config.ProviderConfig, error) {
	var records []providerRecord
	if err := s.db.Order("priority DESC").Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	var providers []config.ProviderConfig
	for _, record := range records {
		var p config.ProviderConfig
		p.ID = record.ID
		p.Name = record.Name
		p.Type = record.Type
		p.APIBase = record.APIBase
		p.EndpointMode = config.NormalizeProviderEndpointMode(record.Type, record.EndpointMode)
		apiKey, err := s.decryptSecret(record.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKey = apiKey
		p.Enabled = record.Enabled
		p.Priority = record.Priority
		p.DefaultModel = record.DefaultModel
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

func (s *ConfigService) saveLocked() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.saveSettingsLocked(tx); err != nil {
			return err
		}
		if err := s.saveProvidersLocked(tx); err != nil {
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
	encryptedGatewayAuthKey, err := s.encryptSecret(gatewayCfg.AuthKey)
	if err != nil {
		return err
	}
	gatewayCfg.AuthKey = encryptedGatewayAuthKey

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
	if err := s.loadSettingJSONLocked("gateway", target); err != nil {
		return err
	}
	authKey, err := s.decryptSecret(target.AuthKey)
	if err != nil {
		return err
	}
	target.AuthKey = authKey
	return nil
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