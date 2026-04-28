package services

import (
	"icoo_proxy/internal/models"

	"gorm.io/gorm"
)

type Services struct {
	db *gorm.DB

	endpoint        *EndpointService
	authKey         *AuthKeyService
	modelAlias      *ModelAliasService
	projectSettings *ProjectSettingsService
	uiPref          *UiPrefService
	supplier        *SupplierService
	catalog         *CatalogService
	routePolicy     *RoutePolicyService
	health          *HealthService
}

func NewServices(db *gorm.DB, resolver models.Resolver) (*Services, error) {
	svc := &Services{db: db}

	supplier, err := NewSupplierService(db)
	if err != nil {
		return nil, err
	}
	if resolver == nil {
		resolver = supplier
	}

	endpoint, err := NewEndpointService(db)
	if err != nil {
		return nil, err
	}
	if err := endpoint.seedDefaults(); err != nil {
		return nil, err
	}

	authKey, err := NewAuthKeyService(db)
	if err != nil {
		return nil, err
	}

	modelAlias, err := NewModelAliasService(db, resolver)
	if err != nil {
		return nil, err
	}

	uiPref, err := NewUiPrefService(db)
	if err != nil {
		return nil, err
	}

	catalog, err := NewCatalogService()
	if err != nil {
		return nil, err
	}

	routePolicy, err := NewRoutePolicyService(db, resolver)
	if err != nil {
		return nil, err
	}
	if err := routePolicy.seedDefaults(); err != nil {
		return nil, err
	}

	health := NewHealthService(resolver)
	projectSettings := NewProjectSettingsService()

	svc.modelAlias = modelAlias
	svc.authKey = authKey
	svc.endpoint = endpoint
	svc.projectSettings = projectSettings
	svc.uiPref = uiPref
	svc.supplier = supplier
	svc.catalog = catalog
	svc.routePolicy = routePolicy
	svc.health = health
	return svc, nil
}

func (s *Services) Endpoint() *EndpointService {
	return s.endpoint
}

func (s *Services) AuthKey() *AuthKeyService {
	return s.authKey
}

func (s *Services) ModelAlias() *ModelAliasService {
	return s.modelAlias
}

func (s *Services) ProjectSettings() *ProjectSettingsService {
	return s.projectSettings
}

func (s *Services) UiPref() *UiPrefService {
	return s.uiPref
}

func (s *Services) Supplier() *SupplierService {
	return s.supplier
}

func (s *Services) Catalog() *CatalogService {
	return s.catalog
}

func (s *Services) RoutePolicy() *RoutePolicyService {
	return s.routePolicy
}

func (s *Services) Health() *HealthService {
	return s.health
}
