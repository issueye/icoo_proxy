package entity

import (
	"time"

	"icoo_llm_bridge/internal/constants"
)

// Provider 供应商
type Provider struct {
	ID           string             `gorm:"primaryKey;column:id;comment:供应商ID"`
	Name         string             `gorm:"uniqueIndex;not null;comment:供应商名称"`
	Protocol     constants.Protocol `gorm:"column:protocol;comment:协议"`
	Vendor       constants.Vendor   `gorm:"column:vendor;comment:供应商厂商类型"`
	BaseURL      string             `gorm:"column:base_url;comment:基础URL"`
	APIKeyCipher string             `gorm:"column:api_key_cipher;comment:API密钥加密"`
	OnlyStream   bool               `gorm:"column:only_stream;comment:是否仅支持流式输出"`
	UserAgent    string             `gorm:"column:user_agent;comment:用户代理"`
	Enabled      bool               `gorm:"column:enabled;comment:是否启用供应商"`
	Description  string             `gorm:"column:description;comment:供应商描述"`
	CreatedAt    time.Time          `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt    time.Time          `gorm:"column:updated_at;comment:更新时间"`
}

func (Provider) TableName() string {
	return "providers"
}

type ProviderModel struct {
	ID         string    `json:"id" gorm:"primaryKey;column:id;comment:供应商模型ID"`
	ProviderID string    `json:"provider_id" gorm:"index:idx_provider_model,unique;not null;comment:供应商ID"`
	Name       string    `json:"name" gorm:"index:idx_provider_model,unique;not null;comment:模型名称"`
	MaxTokens  int       `json:"max_tokens" gorm:"column:max_tokens;comment:最大令牌数"`
	Enabled    bool      `json:"enabled" gorm:"column:enabled;comment:是否启用模型"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;comment:创建时间"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at;comment:更新时间"`
}

func (ProviderModel) TableName() string {
	return "provider_models"
}
