package services

import (
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

// CatalogPolicyResolver 负责把下游协议解析到已启用供应商。
type CatalogPolicyResolver struct {
	routePolicy *RoutePolicyService
	supplier    *SupplierService
}

// NewCatalogPolicyResolver 创建 catalog 使用的策略解析器。
func NewCatalogPolicyResolver(routePolicy *RoutePolicyService, supplier *SupplierService) *CatalogPolicyResolver {
	return &CatalogPolicyResolver{routePolicy: routePolicy, supplier: supplier}
}

// ResolveEnabledSupplierByDownstream 根据下游协议返回已启用供应商。
func (r *CatalogPolicyResolver) ResolveEnabledSupplierByDownstream(downstream consts.Protocol) (models.SupplierRecord, bool) {
	if r == nil || r.routePolicy == nil || r.supplier == nil {
		return models.SupplierRecord{}, false
	}
	policy, ok := r.routePolicy.FindEnabledByDownstream(downstream)
	if !ok {
		return models.SupplierRecord{}, false
	}
	for _, item := range r.supplier.List() {
		if item.ID == policy.SupplierID && item.Enabled {
			return item, true
		}
	}
	return models.SupplierRecord{}, false
}
