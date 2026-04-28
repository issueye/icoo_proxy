package models

const DefaultKey = "main"

type Preferences struct {
	Theme      string `json:"theme"`
	ButtonSize string `json:"buttonSize"`
}

type UiPrefModel struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (UiPrefModel) TableName() string {
	return "ui_prefs"
}
