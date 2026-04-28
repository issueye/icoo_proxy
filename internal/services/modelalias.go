package services

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"icoo_proxy/internal/models"
)

type ModelAliasService struct {
	db     *gorm.DB
	lookup models.Resolver
}

func NewModelAliasService(db *gorm.DB, resolver models.Resolver) (*ModelAliasService, error) {
	return &ModelAliasService{db: db, lookup: resolver}, nil
}

func (s *ModelAliasService) List() []models.ModelAliasRecord {
	var rows []models.ModelAliasModel
	if err := s.db.Order("lower(name) asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]models.ModelAliasRecord, 0, len(rows))
	for _, item := range rows {
		items = append(items, s.toRecord(item))
	}
	return items
}

func (s *ModelAliasService) EnabledEntries() []string {
	var rows []models.ModelAliasModel
	if err := s.db.Where("enabled = ?", true).Order("lower(name) asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		supplier, ok := s.lookup.Resolve(item.SupplierID)
		if !ok {
			continue
		}
		items = append(items, fmt.Sprintf("%s=%s:%s",
			item.Name,
			supplier.Protocol.ToString(),
			supplier.Name+"/"+strings.TrimSpace(item.Model),
		))
	}
	return items
}

func (s *ModelAliasService) Upsert(input models.ModelAliasUpsertInput) (models.ModelAliasRecord, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return models.ModelAliasRecord{}, fmt.Errorf("model alias name is required")
	}
	supplierID := strings.TrimSpace(input.SupplierID)
	if supplierID == "" {
		return models.ModelAliasRecord{}, fmt.Errorf("model alias supplier is required")
	}
	supplier, ok := s.lookup.Resolve(supplierID)
	if !ok {
		return models.ModelAliasRecord{}, fmt.Errorf("supplier not found")
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		return models.ModelAliasRecord{}, fmt.Errorf("model alias target model is required")
	}

	id := strings.TrimSpace(input.ID)
	var existing models.ModelAliasModel
	found := false
	if id != "" && s.db.Limit(1).Find(&existing, "id = ?", id).RowsAffected > 0 {
		found = true
	} else if s.db.Limit(1).Find(&existing, "name = ?", name).RowsAffected > 0 {
		found = true
	}

	now := time.Now()
	current := models.ModelAliasModel{
		ID:         buildID(name),
		Name:       name,
		SupplierID: supplier.ID,
		Model:      model,
		Enabled:    input.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if found {
		current.ID = existing.ID
		current.CreatedAt = existing.CreatedAt
	} else if id != "" {
		current.ID = id
	}
	if err := s.db.Save(&current).Error; err != nil {
		return models.ModelAliasRecord{}, err
	}
	return s.toRecord(current), nil
}

func (s *ModelAliasService) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("model alias id is required")
	}
	result := s.db.Delete(&models.ModelAliasModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("model alias not found")
	}
	return nil
}

func (s *ModelAliasService) toRecord(item models.ModelAliasModel) models.ModelAliasRecord {
	record := models.ModelAliasRecord{
		ID:         item.ID,
		Name:       item.Name,
		SupplierID: item.SupplierID,
		Model:      item.Model,
		Enabled:    item.Enabled,
		UpdatedAt:  item.UpdatedAt.Format(time.RFC3339),
		CreatedAt:  item.CreatedAt.Format(time.RFC3339),
	}
	if supplier, ok := s.lookup.Resolve(item.SupplierID); ok {
		record.SupplierName = supplier.Name
		record.UpstreamProtocol = supplier.Protocol
	}
	return record
}

func buildModelAliasID(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, "_", "-")
	if base == "" {
		base = "model-alias"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func MergeEntries(base string, extra []string) string {
	items := make([]string, 0)
	seen := make(map[string]struct{})
	appendEntry := func(entry string) {
		value := strings.TrimSpace(entry)
		if value == "" {
			return
		}
		alias, _, found := strings.Cut(value, "=")
		alias = strings.TrimSpace(alias)
		if !found || alias == "" {
			return
		}
		if _, ok := seen[alias]; ok {
			return
		}
		seen[alias] = struct{}{}
		items = append(items, value)
	}
	for _, entry := range modelAliasSplitEntries(base) {
		appendEntry(entry)
	}
	for _, entry := range extra {
		value := strings.TrimSpace(entry)
		alias, _, found := strings.Cut(value, "=")
		alias = strings.TrimSpace(alias)
		if !found || alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			for index, item := range items {
				currentAlias, _, _ := strings.Cut(item, "=")
				if strings.TrimSpace(currentAlias) == alias {
					items[index] = value
					break
				}
			}
			continue
		}
		seen[alias] = struct{}{}
		items = append(items, value)
	}
	return strings.Join(items, ",")
}

func modelAliasSplitEntries(raw string) []string {
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
