package services

import (
	"encoding/json"
	"fmt"
	"icoo_proxy/internal/models"
	"strings"

	"gorm.io/gorm"
)

type UiPrefService struct {
	db *gorm.DB
}

func NewUiPrefService(db *gorm.DB) (*UiPrefService, error) {
	s := &UiPrefService{db: db}
	return s, nil
}

func (s *UiPrefService) Get() (models.Preferences, error) {
	var row models.UiPrefModel
	err := s.db.Limit(1).Find(&row, "key = ?", models.DefaultKey).Error
	if err != nil {
		return models.Preferences{}, fmt.Errorf("load ui preferences: %w", err)
	}
	if row.Key == "" {
		return models.Preferences{}, fmt.Errorf("no ui preferences found")
	}
	var prefs models.Preferences
	if err := json.Unmarshal([]byte(row.Value), &prefs); err != nil {
		return models.Preferences{}, fmt.Errorf("parse ui preferences: %w", err)
	}
	return prefs, nil
}

func (s *UiPrefService) Save(input models.Preferences) error {
	theme := strings.TrimSpace(input.Theme)
	if theme == "" {
		theme = "blue"
	}
	buttonSize := strings.TrimSpace(input.ButtonSize)
	if buttonSize == "" {
		buttonSize = "md"
	}
	normalized := models.Preferences{Theme: theme, ButtonSize: buttonSize}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("serialize ui preferences: %w", err)
	}
	row := models.UiPrefModel{
		Key:   models.DefaultKey,
		Value: string(raw),
	}
	return s.db.Save(&row).Error
}
