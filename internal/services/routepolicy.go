package services

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

type RoutePolicyService struct {
	db     *gorm.DB
	lookup models.Resolver
}

func NewRoutePolicyService(db *gorm.DB, resolver models.Resolver) (*RoutePolicyService, error) {
	svc := &RoutePolicyService{db: db, lookup: resolver}
	return svc, nil
}

func (s *RoutePolicyService) List() []models.RoutePolicyRecord {
	var rows []models.RoutePolicyModel
	if err := s.db.Order("downstream_protocol asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]models.RoutePolicyRecord, 0, len(rows))
	for _, item := range rows {
		items = append(items, s.toRecord(item))
	}
	return items
}

func (s *RoutePolicyService) Enabled() []models.RoutePolicyRecord {
	items := s.List()
	filtered := make([]models.RoutePolicyRecord, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *RoutePolicyService) FindEnabledBySupplierID(supplierID string) (models.RoutePolicyRecord, bool) {
	id := strings.TrimSpace(supplierID)
	if id == "" {
		return models.RoutePolicyRecord{}, false
	}
	for _, item := range s.Enabled() {
		if item.SupplierID == id {
			return item, true
		}
	}
	return models.RoutePolicyRecord{}, false
}

func (s *RoutePolicyService) FindEnabledByDownstream(downstream consts.Protocol) (models.RoutePolicyRecord, bool) {
	for _, item := range s.Enabled() {
		if item.DownstreamProtocol == downstream {
			return item, true
		}
	}
	return models.RoutePolicyRecord{}, false
}

func (s *RoutePolicyService) Upsert(input models.UpsertInput) (models.RoutePolicyRecord, error) {
	downstream := normalizeProtocol(input.DownstreamProtocol.ToString())
	if downstream == "" {
		return models.RoutePolicyRecord{}, fmt.Errorf("downstream protocol is required")
	}
	if strings.TrimSpace(input.SupplierID) == "" {
		return models.RoutePolicyRecord{}, fmt.Errorf("supplier id is required")
	}
	supplier, ok := s.lookup.Resolve(input.SupplierID)
	if !ok {
		return models.RoutePolicyRecord{}, fmt.Errorf("supplier not found")
	}
	if input.Enabled && strings.TrimSpace(supplier.DefaultModel) == "" {
		return models.RoutePolicyRecord{}, fmt.Errorf("supplier %q default model is required when enabling a route policy", supplier.Name)
	}

	id := strings.TrimSpace(input.ID)
	var existing models.RoutePolicyModel
	found := false
	if id != "" && s.db.Limit(1).Find(&existing, "id = ?", id).RowsAffected > 0 {
		found = true
	} else if s.db.Limit(1).Find(&existing, "downstream_protocol = ?", downstream).RowsAffected > 0 {
		found = true
	}

	now := time.Now()
	current := models.RoutePolicyModel{
		ID:                 routePolicyBuildID(downstream),
		DownstreamProtocol: downstream,
		SupplierID:         supplier.ID,
		Enabled:            input.Enabled,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if found {
		current.ID = existing.ID
		current.CreatedAt = existing.CreatedAt
		current.TargetModel = existing.TargetModel
	} else if id != "" {
		current.ID = id
	}
	if err := s.db.Save(&current).Error; err != nil {
		return models.RoutePolicyRecord{}, err
	}
	return s.toRecord(current), nil
}

func (s *RoutePolicyService) seedDefaults() error {
	for _, item := range defaultPolicies() {
		var count int64
		err := s.db.Model(&models.RoutePolicyModel{}).
			Where("downstream_protocol = ?", item.DownstreamProtocol).
			Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			if err := s.db.Create(&item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *RoutePolicyService) toRecord(item models.RoutePolicyModel) models.RoutePolicyRecord {
	record := models.RoutePolicyRecord{
		ID:                 item.ID,
		DownstreamProtocol: item.DownstreamProtocol,
		SupplierID:         item.SupplierID,
		Enabled:            item.Enabled,
		UpdatedAt:          item.UpdatedAt.Format(time.RFC3339),
		CreatedAt:          item.CreatedAt.Format(time.RFC3339),
	}
	if supplier, ok := s.lookup.Resolve(item.SupplierID); ok {
		record.SupplierName = supplier.Name
		record.UpstreamProtocol = supplier.Protocol
	}
	return record
}

func routePolicyBuildID(downstream consts.Protocol) string {
	return "policy-" + downstream.ToString()
}

func defaultPolicies() []models.RoutePolicyModel {
	now := time.Now()
	return []models.RoutePolicyModel{
		{
			ID:                 routePolicyBuildID(consts.ProtocolAnthropic),
			DownstreamProtocol: consts.ProtocolAnthropic,
			Enabled:            false,
			UpdatedAt:          now,
			CreatedAt:          now,
		},
		{
			ID:                 routePolicyBuildID(consts.ProtocolOpenAIChat),
			DownstreamProtocol: consts.ProtocolOpenAIChat,
			Enabled:            false,
			UpdatedAt:          now,
			CreatedAt:          now,
		},
		{
			ID:                 routePolicyBuildID(consts.ProtocolOpenAIResponses),
			DownstreamProtocol: consts.ProtocolOpenAIResponses,
			Enabled:            false,
			UpdatedAt:          now,
			CreatedAt:          now,
		},
	}
}
