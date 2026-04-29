package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"icoo_proxy/internal/api"
	appcore "icoo_proxy/internal/app"
	"icoo_proxy/internal/config"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
	"icoo_proxy/internal/server"
	"icoo_proxy/internal/services"
	"icoo_proxy/internal/traffic"
)

type App struct {
	ctx  context.Context
	mu   sync.RWMutex
	root string
	cfg  config.Config

	catalog *services.CatalogService
	service *services.ProxyService
	traffic *traffic.Service
	app     *appcore.App

	httpServer *http.Server
	chainLog   *os.File
	listenAddr string
	running    bool
	lastError  string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	root, err := os.Getwd()
	if err != nil {
		a.setLastError(err.Error())
		return
	}
	a.root = root
	p, err := appcore.NewApp(root)
	if err != nil {
		a.setLastError(err.Error())
		return
	}
	a.app = p
	trafficService, err := traffic.NewService(root)
	if err != nil {
		a.setLastError(err.Error())
		return
	}
	a.traffic = trafficService
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.stopProxy(ctx)
	if a.traffic != nil {
		_ = a.traffic.Close()
	}
}

func (a *App) GetOverview() map[string]interface{} {
	return stateToMap(a.State())
}

func (a *App) GetTrafficPage(page int, pageSize int, filter string) map[string]interface{} {
	lastUpdatedAt := time.Now().Format(time.RFC3339)
	filter = normalizeTrafficFilter(filter)

	if a.traffic != nil {
		result := a.traffic.QueryPage(filter, page, pageSize)
		return map[string]interface{}{
			"items":            result.Items,
			"total":            result.Total,
			"page":             result.Page,
			"page_size":        result.PageSize,
			"filter":           filter,
			"protocol_options": result.ProtocolOptions,
			"token_stats":      result.TokenStats,
			"total_requests":   result.TotalRequests,
			"success_count":    result.SuccessCount,
			"error_count":      result.ErrorCount,
			"average_latency":  result.AverageLatency,
			"last_updated_at":  lastUpdatedAt,
		}
	}

	items := []api.RequestView{}
	if a.service != nil {
		items = a.service.RecentRequests()
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	protocolOptions := []string{"all"}
	seenProtocols := map[string]struct{}{"all": {}}
	filtered := make([]api.RequestView, 0, len(items))
	tokenStats := api.TokenStatsView{}
	successCount := 0
	errorCount := 0
	totalDuration := int64(0)

	for _, item := range items {
		tokenStats.InputTokens += item.InputTokens
		tokenStats.OutputTokens += item.OutputTokens
		tokenStats.TotalTokens += item.TotalTokens
		totalDuration += item.DurationMS

		if item.StatusCode > 0 && item.StatusCode < 400 {
			successCount++
		}
		if item.StatusCode >= 400 {
			errorCount++
		}

		appendProtocolOption(&protocolOptions, seenProtocols, item.Downstream)
		appendProtocolOption(&protocolOptions, seenProtocols, item.Upstream)

		if filter != "all" && item.Downstream != filter && item.Upstream != filter {
			continue
		}
		filtered = append(filtered, item)
	}

	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	averageLatency := 0
	if len(items) > 0 {
		averageLatency = int(totalDuration / int64(len(items)))
	}

	return map[string]interface{}{
		"items":            filtered[start:end],
		"total":            len(filtered),
		"page":             page,
		"page_size":        pageSize,
		"filter":           filter,
		"protocol_options": protocolOptions,
		"token_stats":      tokenStats,
		"total_requests":   len(items),
		"success_count":    successCount,
		"error_count":      errorCount,
		"average_latency":  averageLatency,
		"last_updated_at":  lastUpdatedAt,
	}
}

func (a *App) GetSuppliersPage(page int, pageSize int, keyword string, protocol string) services.SupplierPageResult {
	if a.app == nil {
		return services.SupplierPageResult{
			Items:    []models.SupplierRecord{},
			Page:     1,
			PageSize: 10,
		}
	}
	return a.app.Services().Supplier().QueryPage(page, pageSize, keyword, protocol)
}

func (a *App) GetEndpointsPage(page int, pageSize int, keyword string, protocol string) services.EndpointPageResult {
	if a.app == nil {
		return services.EndpointPageResult{
			Items:    []models.EndpointRecord{},
			Page:     1,
			PageSize: 10,
		}
	}
	return a.app.Services().Endpoint().QueryPage(page, pageSize, keyword, protocol)
}

func (a *App) GetAuthKeysPage(page int, pageSize int, keyword string, status string) services.AuthKeyPageResult {
	if a.app == nil {
		return services.AuthKeyPageResult{
			Items:    []models.AuthKeyRecord{},
			Page:     1,
			PageSize: 10,
		}
	}
	return a.app.Services().AuthKey().QueryPage(page, pageSize, keyword, status)
}

func (a *App) GetProjectSettings() (services.Values, error) {
	if a.app == nil || strings.TrimSpace(a.root) == "" {
		return services.Values{}, context.Canceled
	}
	return a.app.Services().ProjectSettings().Load(a.root)
}

func (a *App) SaveProjectSettings(input services.Values) (services.Values, error) {
	if a.app == nil || strings.TrimSpace(a.root) == "" {
		return services.Values{}, context.Canceled
	}
	if err := a.app.Services().ProjectSettings().Save(a.root, input); err != nil {
		return services.Values{}, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return services.Values{}, err
	}
	return a.app.Services().ProjectSettings().Load(a.root)
}

func (a *App) GetUiPrefs() (models.Preferences, error) {
	if a.app == nil {
		return models.Preferences{}, context.Canceled
	}
	return a.app.Services().UiPref().Get()
}

func (a *App) SaveUiPrefs(input models.Preferences) (models.Preferences, error) {
	if a.app == nil {
		return models.Preferences{}, context.Canceled
	}
	if err := a.app.Services().UiPref().Save(input); err != nil {
		return models.Preferences{}, err
	}
	return a.GetUiPrefs()
}

func (a *App) ReloadProxy() (map[string]interface{}, error) {
	if err := a.stopProxy(context.Background()); err != nil {
		a.setLastError(err.Error())
		return stateToMap(a.State()), err
	}
	if err := a.startProxy(); err != nil {
		a.setLastError(err.Error())
		return stateToMap(a.State()), err
	}
	return stateToMap(a.State()), nil
}

func (a *App) ListSuppliers() []models.SupplierRecord {
	if a.app == nil {
		return nil
	}
	return a.app.Services().Supplier().List()
}

func (a *App) SaveSupplier(input models.SupplierUpsertInput) ([]models.SupplierRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if _, err := a.app.Services().Supplier().Upsert(input); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.ListSuppliers(), nil
}

func (a *App) DeleteSupplier(id string) ([]models.SupplierRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if policy, ok := a.app.Services().RoutePolicy().FindEnabledBySupplierID(id); ok {
		return nil, fmt.Errorf("supplier is used by enabled route policy %q", policy.DownstreamProtocol)
	}
	if err := a.app.Services().Supplier().Delete(id); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.ListSuppliers(), nil
}

func (a *App) ListSupplierHealth() []services.HealthRecord {
	if a.app == nil {
		return nil
	}
	return a.app.Services().Health().List()
}

func (a *App) CheckSupplier(id string) ([]services.HealthRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if _, err := a.app.Services().Health().Check(id); err != nil {
		return nil, err
	}
	return a.app.Services().Health().List(), nil
}

func (a *App) ListRoutePolicies() []models.RoutePolicyRecord {
	if a.app == nil {
		return nil
	}
	return a.app.Services().RoutePolicy().List()
}

func (a *App) ListModelAliases() []models.ModelAliasRecord {
	if a.app == nil {
		return nil
	}
	return a.app.Services().ModelAlias().List()
}

func (a *App) SaveModelAlias(input models.ModelAliasUpsertInput) ([]models.ModelAliasRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if _, err := a.app.Services().ModelAlias().Upsert(input); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.app.Services().ModelAlias().List(), nil
}

func (a *App) DeleteModelAlias(id string) ([]models.ModelAliasRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if err := a.app.Services().ModelAlias().Delete(id); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.app.Services().ModelAlias().List(), nil
}

func (a *App) SaveRoutePolicy(input models.UpsertInput) ([]models.RoutePolicyRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if _, err := a.app.Services().RoutePolicy().Upsert(input); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.app.Services().RoutePolicy().List(), nil
}

func (a *App) ListEndpoints() []models.EndpointRecord {
	if a.app == nil {
		return nil
	}
	return a.app.Services().Endpoint().List()
}

func (a *App) SaveEndpoint(input models.EndpointUpsertInput) ([]models.EndpointRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if _, err := a.app.Services().Endpoint().Upsert(input); err != nil {
		return nil, err
	}
	return a.app.Services().Endpoint().List(), nil
}

func (a *App) DeleteEndpoint(id string) ([]models.EndpointRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if err := a.app.Services().Endpoint().Delete(id); err != nil {
		return nil, err
	}
	return a.app.Services().Endpoint().List(), nil
}

func (a *App) ListAuthKeys() []models.AuthKeyRecord {
	if a.app == nil {
		return nil
	}
	return a.app.Services().AuthKey().List()
}

func (a *App) SaveAuthKey(input models.AuthKeyUpsertInput) ([]models.AuthKeyRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if _, err := a.app.Services().AuthKey().Upsert(input); err != nil {
		return nil, err
	}
	return a.app.Services().AuthKey().List(), nil
}

func (a *App) DeleteAuthKey(id string) ([]models.AuthKeyRecord, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if err := a.app.Services().AuthKey().Delete(id); err != nil {
		return nil, err
	}
	return a.app.Services().AuthKey().List(), nil
}

func (a *App) GetAuthKeySecret(id string) (string, error) {
	if a.app == nil {
		return "", context.Canceled
	}
	return a.app.Services().AuthKey().GetSecret(id)
}

func (a *App) State() api.State {
	a.mu.RLock()
	defer a.mu.RUnlock()

	state := api.State{
		Service:                   "icoo_proxy",
		Version:                   Version,
		Running:                   a.running,
		ListenAddr:                a.listenAddr,
		ProxyURL:                  proxyURL(a.listenAddr),
		LastError:                 a.lastError,
		AuthRequired:              len(a.cfg.AuthKeys()) > 0,
		AuthKeyCount:              len(a.cfg.AuthKeys()),
		AllowUnauthenticatedLocal: a.cfg.AllowUnauthenticatedLocal,
		SupportedPaths: append([]string{
			"/healthz",
			"/readyz",
			"/admin/models",
			"/admin/routes",
			"/admin/requests",
		}, a.enabledEndpointPathsLocked()...),
		Upstreams: []api.UpstreamView{
			{
				Protocol:   consts.ProtocolAnthropic,
				BaseURL:    anthropicBaseURL(a.cfg),
				Configured: strings.TrimSpace(anthropicAPIKey(a.cfg)) != "",
			},
			{
				Protocol:   consts.ProtocolOpenAIChat,
				BaseURL:    openAIChatBaseURL(a.cfg),
				Configured: strings.TrimSpace(openAIChatAPIKey(a.cfg)) != "",
			},
			{
				Protocol:   consts.ProtocolOpenAIResponses,
				BaseURL:    openAIResponsesBaseURL(a.cfg),
				Configured: strings.TrimSpace(openAIResponsesAPIKey(a.cfg)) != "",
			},
		},
		Checks: map[string]interface{}{
			"proxy_running":           a.running,
			"anthropic_ready":         strings.TrimSpace(anthropicAPIKey(a.cfg)) != "",
			"openai_chat_ready":       strings.TrimSpace(openAIChatAPIKey(a.cfg)) != "",
			"openai_responses_ready":  strings.TrimSpace(openAIResponsesAPIKey(a.cfg)) != "",
			"route_catalog_ready":     a.catalog != nil,
			"supplier_store_ready":    a.app != nil,
			"route_policy_ready":      a.app != nil,
			"model_alias_store_ready": a.app != nil,
			"endpoint_store_ready":    a.app != nil,
			"auth_key_store_ready":    a.app != nil,
		},
	}
	if a.catalog != nil {
		for _, route := range a.catalog.Defaults() {
			state.Defaults = append(state.Defaults, api.RouteView{
				Name:     route.Name,
				Upstream: string(route.Upstream),
				Model:    route.Model,
			})
		}
		for _, route := range a.catalog.Aliases() {
			state.Aliases = append(state.Aliases, api.RouteView{
				Name:     route.Name,
				Upstream: string(route.Upstream),
				Model:    route.Model,
			})
		}
	}
	if a.traffic != nil {
		state.RecentRequests = a.traffic.ListRecent(100)
		state.TokenStats = a.traffic.TokenStats()
	} else if a.service != nil {
		state.RecentRequests = a.service.RecentRequests()
	}
	if a.app != nil {
		for _, item := range a.app.Services().Endpoint().List() {
			state.Endpoints = append(state.Endpoints, api.EndpointView{
				ID:          item.ID,
				Path:        item.Path,
				Protocol:    item.Protocol,
				Description: item.Description,
				Enabled:     item.Enabled,
				BuiltIn:     item.BuiltIn,
				UpdatedAt:   item.UpdatedAt,
				CreatedAt:   item.CreatedAt,
			})
		}
		for _, policy := range a.app.Services().RoutePolicy().List() {
			state.RoutePolicies = append(state.RoutePolicies, api.RoutePolicyView{
				ID:                 policy.ID,
				DownstreamProtocol: policy.DownstreamProtocol,
				SupplierID:         policy.SupplierID,
				SupplierName:       policy.SupplierName,
				UpstreamProtocol:   policy.UpstreamProtocol,
				Enabled:            policy.Enabled,
				UpdatedAt:          policy.UpdatedAt,
				CreatedAt:          policy.CreatedAt,
			})
		}
	}
	return state
}

func (a *App) startProxy() error {
	if a.app == nil {
		return context.Canceled
	}
	cfg, err := config.Load(a.root)
	if err != nil {
		return err
	}
	svc := a.app.Services()
	cfg.ProxyAPIKeys = services.MergeSecrets(cfg.ProxyAPIKeys, svc.AuthKey().EnabledSecrets())
	defaults, err := applyRoutePolicies(&cfg, svc)
	if err != nil {
		return err
	}
	aliasEntries := services.MergeEntries("", svc.ModelAlias().EnabledEntries())
	catalog, err := services.NewCatalogFromRoutes(defaults, aliasEntries)
	if err != nil {
		return err
	}
	supplierCache := services.NewSupplierModelCache()
	if err := supplierCache.Rebuild(svc.Supplier().ListSnapshots()); err != nil {
		return err
	}
	catalog.SetSupplierModelCache(supplierCache)
	catalog.SetPolicyResolver(services.NewCatalogPolicyResolver(svc.RoutePolicy(), svc.Supplier()))
	proxyService := services.New(cfg, catalog)
	if a.traffic != nil {
		proxyService.SetRequestRecorder(a.traffic)
	}
	chainLogger, chainLog, err := openChainLog(cfg.ChainLogPath)
	if err != nil {
		return err
	}
	proxyService.SetChainLogger(chainLogger)
	handler := api.NewMux(a, proxyService, a.endpointRoutes())
	srv := server.New(cfg, handler)
	listener, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		if chainLog != nil {
			_ = chainLog.Close()
		}
		return err
	}
	listenAddr := listener.Addr().String()

	a.mu.Lock()
	a.cfg = cfg
	a.catalog = catalog
	a.service = proxyService
	a.httpServer = srv
	a.chainLog = chainLog
	a.listenAddr = listenAddr
	a.running = true
	a.lastError = ""
	a.mu.Unlock()

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			a.setLastError(err.Error())
		}
	}()
	return nil
}

func applyRoutePolicies(cfg *config.Config, svc *services.Services) (map[consts.Protocol]models.Route, error) {
	defaults := make(map[consts.Protocol]models.Route)
	for _, policy := range svc.RoutePolicy().Enabled() {
		supplier, ok := svc.Supplier().Resolve(policy.SupplierID)
		if !ok {
			return nil, fmt.Errorf("supplier %q not found for route policy %q", policy.SupplierID, policy.DownstreamProtocol)
		}
		if !supplier.IsEnabled {
			return nil, fmt.Errorf("supplier %q is disabled for route policy %q", supplier.Name, policy.DownstreamProtocol)
		}
		if strings.TrimSpace(supplier.DefaultModel) == "" {
			return nil, fmt.Errorf("supplier %q default model is required for route policy %q", supplier.Name, policy.DownstreamProtocol)
		}
		configureUpstream(cfg, supplier)
		defaults[policy.DownstreamProtocol] = models.Route{
			Name:             policy.DownstreamProtocol.ToString(),
			Upstream:         supplier.Protocol,
			Model:            supplier.DefaultModel,
			DefaultMaxTokens: defaultSupplierModelMaxTokens(supplier),
			Source:           "default",
			Supplier:         supplier,
		}
	}
	return defaults, nil
}

func defaultSupplierModelMaxTokens(supplier models.Snapshot) int {
	item, ok := models.FindSupplierModel(supplier.Models, supplier.DefaultModel)
	if !ok || item.MaxTokens <= 0 {
		return models.DefaultSupplierModelMaxTokens
	}
	return item.MaxTokens
}

func configureUpstream(cfg *config.Config, supplier models.Snapshot) {
	switch supplier.Protocol {
	case consts.ProtocolAnthropic:
		version := "2023-06-01"
		if cfg.AnthropicConfig != nil && strings.TrimSpace(cfg.AnthropicConfig.Version) != "" {
			version = cfg.AnthropicConfig.Version
		}
		cfg.AnthropicConfig = &config.AnthropicConfig{
			Vendor:     supplier.Vendor,
			BaseURL:    strings.TrimSpace(supplier.BaseURL),
			APIKey:     strings.TrimSpace(supplier.APIKey),
			OnlyStream: supplier.OnlyStream,
			UserAgent:  strings.TrimSpace(supplier.UserAgent),
			Version:    version,
		}
	case consts.ProtocolOpenAIChat:
		cfg.OpenAIChatConfig = &config.OpenAIChatConfig{
			Vendor:     supplier.Vendor,
			BaseURL:    strings.TrimSpace(supplier.BaseURL),
			APIKey:     strings.TrimSpace(supplier.APIKey),
			OnlyStream: supplier.OnlyStream,
			UserAgent:  strings.TrimSpace(supplier.UserAgent),
		}
	case consts.ProtocolOpenAIResponses:
		cfg.OpenAIRResponsesConfig = &config.OpenAIRResponsesConfig{
			Vendor:     supplier.Vendor,
			BaseURL:    strings.TrimSpace(supplier.BaseURL),
			APIKey:     strings.TrimSpace(supplier.APIKey),
			OnlyStream: supplier.OnlyStream,
			UserAgent:  strings.TrimSpace(supplier.UserAgent),
		}
	}
}

func openChainLog(path string) (*slog.Logger, *os.File, error) {
	if strings.TrimSpace(path) == "" {
		return slog.Default(), nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug})), file, nil
}

func (a *App) endpointRoutes() []api.EndpointRoute {
	if a.app == nil {
		defaults := services.DefaultDefinitions()
		routes := make([]api.EndpointRoute, 0, len(defaults))
		for _, item := range defaults {
			routes = append(routes, api.EndpointRoute{Path: item.Path, Protocol: item.Protocol})
		}
		return routes
	}
	records := a.app.Services().Endpoint().Enabled()
	routes := make([]api.EndpointRoute, 0, len(records))
	for _, item := range records {
		protocol := item.Protocol
		switch protocol {
		case consts.ProtocolAnthropic, consts.ProtocolOpenAIChat, consts.ProtocolOpenAIResponses:
			routes = append(routes, api.EndpointRoute{
				Path:     item.Path,
				Protocol: protocol,
			})
		}
	}
	return routes
}

func (a *App) enabledEndpointPathsLocked() []string {
	if a.app == nil {
		defaults := services.DefaultDefinitions()
		paths := make([]string, 0, len(defaults))
		for _, item := range defaults {
			paths = append(paths, item.Path)
		}
		return paths
	}
	items := a.app.Services().Endpoint().Enabled()
	paths := make([]string, 0, len(items))
	for _, item := range items {
		paths = append(paths, item.Path)
	}
	return paths
}

func (a *App) stopProxy(ctx context.Context) error {
	a.mu.Lock()
	srv := a.httpServer
	chainLog := a.chainLog
	a.httpServer = nil
	a.chainLog = nil
	a.catalog = nil
	a.service = nil
	a.running = false
	a.listenAddr = ""
	a.mu.Unlock()

	if srv == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	err := srv.Shutdown(ctx)
	if chainLog != nil {
		if closeErr := chainLog.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (a *App) setLastError(message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastError = message
}

func proxyURL(addr string) string {
	if strings.TrimSpace(addr) == "" {
		return ""
	}
	return "http://" + addr
}

func stateToMap(state api.State) map[string]interface{} {
	return map[string]interface{}{
		"service":                     state.Service,
		"version":                     state.Version,
		"running":                     state.Running,
		"listen_addr":                 state.ListenAddr,
		"proxy_url":                   state.ProxyURL,
		"last_error":                  state.LastError,
		"auth_required":               state.AuthRequired,
		"auth_key_count":              state.AuthKeyCount,
		"allow_unauthenticated_local": state.AllowUnauthenticatedLocal,
		"supported_paths":             state.SupportedPaths,
		"defaults":                    state.Defaults,
		"aliases":                     state.Aliases,
		"upstreams":                   state.Upstreams,
		"endpoints":                   state.Endpoints,
		"route_policies":              state.RoutePolicies,
		"recent_requests":             state.RecentRequests,
		"token_stats":                 state.TokenStats,
		"notes":                       state.Notes,
		"checks":                      state.Checks,
	}
}

func anthropicBaseURL(cfg config.Config) string {
	if cfg.AnthropicConfig == nil {
		return ""
	}
	return cfg.AnthropicConfig.BaseURL
}

func anthropicAPIKey(cfg config.Config) string {
	if cfg.AnthropicConfig == nil {
		return ""
	}
	return cfg.AnthropicConfig.APIKey
}

func openAIChatBaseURL(cfg config.Config) string {
	if cfg.OpenAIChatConfig == nil {
		return ""
	}
	return cfg.OpenAIChatConfig.BaseURL
}

func openAIChatAPIKey(cfg config.Config) string {
	if cfg.OpenAIChatConfig == nil {
		return ""
	}
	return cfg.OpenAIChatConfig.APIKey
}

func openAIResponsesBaseURL(cfg config.Config) string {
	if cfg.OpenAIRResponsesConfig == nil {
		return ""
	}
	return cfg.OpenAIRResponsesConfig.BaseURL
}

func openAIResponsesAPIKey(cfg config.Config) string {
	if cfg.OpenAIRResponsesConfig == nil {
		return ""
	}
	return cfg.OpenAIRResponsesConfig.APIKey
}

func normalizeTrafficFilter(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "all"
	}
	return filter
}

func appendProtocolOption(options *[]string, seen map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := seen[value]; ok {
		return
	}
	seen[value] = struct{}{}
	*options = append(*options, value)
}
