package entity

import "time"

// ModelCatalogItem is a reusable model definition that can be assigned to providers.
type ModelCatalogItem struct {
	ID          string    `json:"id" gorm:"primaryKey;column:id;comment:模型目录ID"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null;comment:模型名称"`
	Family      string    `json:"family" gorm:"column:family;comment:模型家族"`
	Icon        string    `json:"icon" gorm:"column:icon;comment:图标类型"`
	MaxTokens   int       `json:"max_tokens" gorm:"column:max_tokens;comment:默认最大令牌数"`
	Description string    `json:"description" gorm:"column:description;comment:模型说明"`
	BuiltIn     bool      `json:"built_in" gorm:"column:built_in;comment:是否内置"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;comment:创建时间"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;comment:更新时间"`
}

func (ModelCatalogItem) TableName() string {
	return "model_catalog"
}
