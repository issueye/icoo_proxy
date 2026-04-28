package services

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

type DefaultDefinition struct {
	Path        string
	Protocol    consts.Protocol
	Description string
}

var defaultDefinitions = []DefaultDefinition{
	{Path: "/v1/messages", Protocol: consts.ProtocolAnthropic, Description: "Anthropic Messages official-compatible endpoint."},
	{Path: "/anthropic/v1/messages", Protocol: consts.ProtocolAnthropic, Description: "Anthropic namespaced Messages endpoint."},
	{Path: "/v1/chat/completions", Protocol: consts.ProtocolOpenAIChat, Description: "OpenAI Chat Completions official-compatible endpoint."},
	{Path: "/openai/v1/chat/completions", Protocol: consts.ProtocolOpenAIChat, Description: "OpenAI namespaced Chat Completions endpoint."},
	{Path: "/v1/responses", Protocol: consts.ProtocolOpenAIResponses, Description: "OpenAI Responses official-compatible endpoint."},
	{Path: "/openai/v1/responses", Protocol: consts.ProtocolOpenAIResponses, Description: "OpenAI namespaced Responses endpoint."},
}

func DefaultDefinitions() []DefaultDefinition {
	return append([]DefaultDefinition(nil), defaultDefinitions...)
}

type EndpointService struct {
	db *gorm.DB
}

func NewEndpointService(db *gorm.DB) (*EndpointService, error) {
	svc := &EndpointService{db: db}
	return svc, nil
}

func (s *EndpointService) List() []models.EndpointRecord {
	var rows []models.EndpointModel
	if err := s.db.Order("built_in desc, path asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]models.EndpointRecord, 0, len(rows))
	for _, item := range rows {
		items = append(items, toEndpointRecord(item))
	}
	return items
}

func (s *EndpointService) Enabled() []models.EndpointRecord {
	var rows []models.EndpointModel
	if err := s.db.Where("enabled = ?", true).Order("built_in desc, path asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]models.EndpointRecord, 0, len(rows))
	for _, item := range rows {
		items = append(items, toEndpointRecord(item))
	}
	return items
}

func (s *EndpointService) Upsert(input models.EndpointUpsertInput) (models.EndpointRecord, error) {
	path := normalizePath(input.Path)
	if path == "" {
		return models.EndpointRecord{}, fmt.Errorf("endpoint path is required")
	}
	protocol := normalizeProtocol(input.Protocol)
	if protocol == consts.Protocol("") {
		return models.EndpointRecord{}, fmt.Errorf("endpoint protocol is required")
	}

	id := strings.TrimSpace(input.ID)
	var existing models.EndpointModel
	found := false
	if id != "" && s.db.Limit(1).Find(&existing, "id = ?", id).RowsAffected > 0 {
		found = true
	} else if s.db.Limit(1).Find(&existing, "path = ?", path).RowsAffected > 0 {
		found = true
	}

	now := time.Now()
	current := models.EndpointModel{
		ID:          buildID(path),
		Path:        path,
		Protocol:    protocol,
		Description: strings.TrimSpace(input.Description),
		Enabled:     input.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if found {
		current.ID = existing.ID
		current.BuiltIn = existing.BuiltIn
		current.CreatedAt = existing.CreatedAt
	} else if id != "" {
		current.ID = id
	}
	if err := s.db.Save(&current).Error; err != nil {
		return models.EndpointRecord{}, err
	}
	return toEndpointRecord(current), nil
}

func (s *EndpointService) seedDefaults() error {
	for _, item := range DefaultDefinitions() {
		var count int64
		if err := s.db.Model(&models.EndpointModel{}).Where("path = ?", item.Path).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		now := time.Now()
		current := models.EndpointModel{
			ID:          buildID(item.Path),
			Path:        item.Path,
			Protocol:    item.Protocol,
			Description: item.Description,
			Enabled:     true,
			BuiltIn:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.db.Create(&current).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *EndpointService) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("endpoint id is required")
	}
	var item models.EndpointModel
	if err := s.db.First(&item, "id = ?", id).Error; err != nil {
		return fmt.Errorf("endpoint not found")
	}
	if item.BuiltIn {
		return fmt.Errorf("built-in endpoint cannot be deleted")
	}
	return s.db.Delete(&models.EndpointModel{}, "id = ?", id).Error
}

func toEndpointRecord(item models.EndpointModel) models.EndpointRecord {
	return models.EndpointRecord{
		ID:          item.ID,
		Path:        item.Path,
		Protocol:    item.Protocol,
		Description: item.Description,
		Enabled:     item.Enabled,
		BuiltIn:     item.BuiltIn,
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
	}
}

func normalizePath(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func normalizeProtocol(raw string) consts.Protocol {
	value := consts.Protocol(strings.TrimSpace(raw))
	switch value {
	case consts.ProtocolAnthropic, consts.ProtocolOpenAIChat, consts.ProtocolOpenAIResponses:
		return value
	default:
		return ""
	}
}

func buildID(path string) string {
	id := strings.Trim(strings.ReplaceAll(path, "/", "-"), "-")
	id = strings.ReplaceAll(id, "_", "-")
	if id == "" {
		id = "endpoint"
	}
	return "endpoint-" + id
}
