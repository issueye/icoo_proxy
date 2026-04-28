package services

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

type SupplierService struct {
	db *gorm.DB
}

func NewSupplierService(db *gorm.DB) (*SupplierService, error) {
	svc := &SupplierService{db: db}
	return svc, nil
}

func (s *SupplierService) List() []models.SupplierRecord {
	var rows []models.SupplierModel
	if err := s.db.Order("lower(name) asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]models.SupplierRecord, 0, len(rows))
	for _, item := range rows {
		items = append(items, toSupplierRecord(item))
	}
	return items
}

func (s *SupplierService) Resolve(id string) (models.Snapshot, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return models.Snapshot{}, false
	}
	var item models.SupplierModel
	if err := s.db.First(&item, "id = ?", id).Error; err != nil {
		return models.Snapshot{}, false
	}
	return models.Snapshot{
		ID:           item.ID,
		Name:         item.Name,
		Protocol:     item.Protocol,
		BaseURL:      item.BaseURL,
		APIKey:       item.APIKey,
		OnlyStream:   item.OnlyStream,
		UserAgent:    item.UserAgent,
		IsEnabled:    item.Enabled,
		DefaultModel: strings.TrimSpace(item.DefaultModel),
	}, true
}

func (s *SupplierService) Upsert(input models.SupplierUpsertInput) (models.SupplierRecord, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return models.SupplierRecord{}, fmt.Errorf("supplier name is required")
	}
	protocol := normalizeProtocol(input.Protocol)
	if protocol == consts.Protocol("") {
		return models.SupplierRecord{}, fmt.Errorf("supplier protocol is required")
	}
	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" {
		return models.SupplierRecord{}, fmt.Errorf("supplier base_url is required")
	}
	modelArr := splitSupplierCSVLike(input.Models)
	defaultModel := strings.TrimSpace(input.DefaultModel)
	if defaultModel != "" && !slices.Contains(modelArr, defaultModel) {
		return models.SupplierRecord{}, fmt.Errorf("supplier default_model must exist in models list")
	}

	now := time.Now()
	id := strings.TrimSpace(input.ID)
	var existing models.SupplierModel
	found := false
	if id != "" {
		found = s.db.Limit(1).Find(&existing, "id = ?", id).RowsAffected > 0
	}

	current := models.SupplierModel{
		ID:           generateID(name),
		Name:         name,
		Protocol:     protocol,
		BaseURL:      baseURL,
		APIKey:       strings.TrimSpace(input.APIKey),
		OnlyStream:   input.OnlyStream,
		UserAgent:    strings.TrimSpace(input.UserAgent),
		Enabled:      input.Enabled,
		Description:  strings.TrimSpace(input.Description),
		Models:       strings.Join(modelArr, ","),
		DefaultModel: defaultModel,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if found {
		current.ID = existing.ID
		current.CreatedAt = existing.CreatedAt
		if strings.TrimSpace(input.APIKey) == "" {
			current.APIKey = existing.APIKey
		}
	} else if id != "" {
		current.ID = id
	}
	if err := s.db.Save(&current).Error; err != nil {
		return models.SupplierRecord{}, err
	}
	return toSupplierRecord(current), nil
}

func (s *SupplierService) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("supplier id is required")
	}
	result := s.db.Delete(&models.SupplierModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

func toSupplierRecord(item models.SupplierModel) models.SupplierRecord {
	return models.SupplierRecord{
		ID:           item.ID,
		Name:         item.Name,
		Protocol:     item.Protocol,
		BaseURL:      item.BaseURL,
		APIKeyMasked: maskSecret(item.APIKey),
		OnlyStream:   item.OnlyStream,
		UserAgent:    item.UserAgent,
		Enabled:      item.Enabled,
		Description:  item.Description,
		Models:       slices.Clone(splitSupplierCSVLike(item.Models)),
		DefaultModel: strings.TrimSpace(item.DefaultModel),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
	}
}

func normalizeSupplierProtocol(raw string) consts.Protocol {
	value := consts.Protocol(raw)
	switch value {
	case consts.ProtocolAnthropic, consts.ProtocolOpenAIChat, consts.ProtocolOpenAIResponses:
		return value
	default:
		return consts.Protocol("")
	}
}

func splitSupplierCSVLike(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	items := make([]string, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(field)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}

func maskSupplierSecret(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return strings.Repeat("*", len(value))
	}
	return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
}

func generateSupplierID(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, "_", "-")
	if base == "" {
		base = "supplier"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}
