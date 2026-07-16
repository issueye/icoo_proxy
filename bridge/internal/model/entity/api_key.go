package entity

import "time"

// APIKey API密钥
type APIKey struct {
	ID            string     `gorm:"primaryKey;column:id;comment:API密钥ID"`
	Name          string     `gorm:"index;not null;comment:API密钥名称"`
	SecretHash    string     `gorm:"uniqueIndex;not null;comment:API密钥哈希"`
	SecretPreview string     `gorm:"column:secret_preview;comment:API密钥预览"`
	SecretCipher  string     `gorm:"column:secret_cipher;comment:API密钥加密"`
	Scopes        string     `gorm:"column:scopes;comment:API密钥作用"`
	Enabled       bool       `gorm:"column:enabled;comment:是否启用API密钥"`
	ExpiresAt     *time.Time `gorm:"column:expires_at;comment:过期时间"`
	CreatedAt     time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;comment:更新时间"`
}

func (APIKey) TableName() string {
	return "api_keys"
}
