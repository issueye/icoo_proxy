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

func (s *EndpointService) QueryPage(page int, pageSize int, keyword string, protocol string) EndpointPageResult {
	page = normalizePage(page)
	pageSize = normalizePageSize(pageSize)
	keyword = normalizeKeyword(keyword)
	protocol = normalizeEndpointFilter(protocol)

	result := EndpointPageResult{
		Items:    make([]models.EndpointRecord, 0, pageSize),
		Page:     page,
		PageSize: pageSize,
	}

	if s == nil || s.db == nil {
		return result
	}

	var totalCount int64
	_ = s.db.Model(&models.EndpointModel{}).Count(&totalCount).Error
	result.TotalCount = int(totalCount)

	var enabledCount int64
	_ = s.db.Model(&models.EndpointModel{}).Where("enabled = ?", true).Count(&enabledCount).Error
	result.EnabledCount = int(enabledCount)

	var customCount int64
	_ = s.db.Model(&models.EndpointModel{}).Where("built_in = ?", false).Count(&customCount).Error
	result.CustomCount = int(customCount)

	query := s.db.Model(&models.EndpointModel{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("lower(path) LIKE ? OR lower(description) LIKE ?", like, like)
	}
	if protocol != "all" {
		query = query.Where("protocol = ?", protocol)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return result
	}

	result.Total = int(total)
	result.Page = clampPage(page, total, pageSize)

	var rows []models.EndpointModel
	if err := query.
		Order("built_in desc, path asc").
		Offset((result.Page - 1) * result.PageSize).
		Limit(result.PageSize).
		Find(&rows).Error; err != nil {
		return result
	}

	for _, item := range rows {
		result.Items = append(result.Items, toEndpointRecord(item))
	}
	return result
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

func normalizeEndpointFilter(raw string) string {
	protocol := normalizeProtocol(raw)
	if protocol == consts.Protocol("") {
		return "all"
	}
	return protocol.ToString()
}

func buildID(path string) string {
	id := strings.Trim(strings.ReplaceAll(path, "/", "-"), "-")
	id = strings.ReplaceAll(id, "_", "-")
	if id == "" {
		id = "endpoint"
	}
	return "endpoint-" + id
}
