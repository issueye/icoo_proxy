package services

import (
	"strings"
	"sync"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

// SupplierModelCache 按“供应商/模型”缓存可用路由，并提供线程安全查询。
type SupplierModelCache struct {
	mu              sync.RWMutex
	suppliersByName map[string]cachedSupplier
	qualifiedRoutes map[string]models.Route
}

type cachedSupplier struct {
	name     string
	protocol consts.Protocol
	enabled  bool
	models   map[string]models.SupplierModelItem
	snapshot models.Snapshot
}

// NewSupplierModelCache 创建供应商模型缓存。
func NewSupplierModelCache() *SupplierModelCache {
	return &SupplierModelCache{
		suppliersByName: make(map[string]cachedSupplier),
		qualifiedRoutes: make(map[string]models.Route),
	}
}

// Rebuild 根据供应商列表重建缓存快照。
func (c *SupplierModelCache) Rebuild(suppliers []models.Snapshot) error {
	if c == nil {
		return nil
	}

	byName := make(map[string]cachedSupplier, len(suppliers))
	qualified := make(map[string]models.Route)
	for _, supplier := range suppliers {
		supplierName := normalizeCacheSegment(supplier.Name)
		if supplierName == "" {
			continue
		}
		modelItems := supplierModelItems(supplier)
		entry := cachedSupplier{
			name:     strings.TrimSpace(supplier.Name),
			protocol: supplier.Protocol,
			enabled:  supplier.IsEnabled,
			models:   make(map[string]models.SupplierModelItem, len(modelItems)),
			snapshot: supplier,
		}
		for _, item := range modelItems {
			modelKey := normalizeCacheSegment(item.Name)
			modelName := strings.TrimSpace(item.Name)
			if modelKey == "" || modelName == "" {
				continue
			}
			entry.models[modelKey] = item
			if !supplier.IsEnabled {
				continue
			}
			qualifiedKey := buildQualifiedCacheKey(supplier.Name, modelName)
			if qualifiedKey == "" {
				continue
			}
			qualified[qualifiedKey] = models.Route{
				Name:             strings.TrimSpace(supplier.Name) + "/" + modelName,
				Upstream:         supplier.Protocol,
				Model:            modelName,
				DefaultMaxTokens: item.MaxTokens,
				Source:           "qualified-supplier-model",
				Supplier:         supplier,
			}
		}
		byName[supplierName] = entry
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.suppliersByName = byName
	c.qualifiedRoutes = qualified
	return nil
}

// ResolveQualified 解析并命中“供应商/模型”格式的路由。
func (c *SupplierModelCache) ResolveQualified(model string) (models.Route, bool) {
	if c == nil {
		return models.Route{}, false
	}
	key := normalizeQualifiedModel(model)
	if key == "" {
		return models.Route{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	route, ok := c.qualifiedRoutes[key]
	return route, ok
}

// ResolveBySupplierAndModel 根据供应商名和模型名构造路由。
func (c *SupplierModelCache) ResolveBySupplierAndModel(supplierName, model string) (models.Route, bool) {
	if c == nil {
		return models.Route{}, false
	}
	supplierKey := normalizeCacheSegment(supplierName)
	modelKey := normalizeCacheSegment(model)
	if supplierKey == "" || modelKey == "" {
		return models.Route{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.suppliersByName[supplierKey]
	if !ok || !entry.enabled {
		return models.Route{}, false
	}
	item, ok := entry.models[modelKey]
	if !ok {
		return models.Route{}, false
	}
	return models.Route{
		Name:             entry.name + "/" + item.Name,
		Upstream:         entry.protocol,
		Model:            item.Name,
		DefaultMaxTokens: item.MaxTokens,
		Source:           "route-policy-supplier-model",
		Supplier:         entry.snapshot,
	}, true
}

func supplierModelItems(supplier models.Snapshot) []models.SupplierModelItem {
	items := make([]models.SupplierModelItem, 0, len(supplier.Models)+1)
	items = append(items, supplier.Models...)
	if strings.TrimSpace(supplier.DefaultModel) != "" {
		if _, ok := models.FindSupplierModel(items, supplier.DefaultModel); !ok {
			items = append(items, models.SupplierModelItem{
				Name:      supplier.DefaultModel,
				MaxTokens: models.DefaultSupplierModelMaxTokens,
			})
		}
	}
	return models.NormalizeSupplierModelItems(items)
}

func normalizeQualifiedModel(raw string) string {
	supplier, model, ok := splitQualifiedModel(raw)
	if !ok {
		return ""
	}
	return buildQualifiedCacheKey(supplier, model)
}

func buildQualifiedCacheKey(supplierName, model string) string {
	supplierKey := normalizeCacheSegment(supplierName)
	modelKey := normalizeCacheSegment(model)
	if supplierKey == "" || modelKey == "" {
		return ""
	}
	return supplierKey + "/" + modelKey
}

func splitQualifiedModel(raw string) (string, string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", false
	}
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	supplier := strings.TrimSpace(parts[0])
	model := strings.TrimSpace(parts[1])
	if supplier == "" || model == "" {
		return "", "", false
	}
	return supplier, model, true
}

func normalizeCacheSegment(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
