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

	"icoo_proxy/internal/api"
	"icoo_proxy/internal/app"
	"icoo_proxy/internal/authkey"
	"icoo_proxy/internal/bootstrap"
	"icoo_proxy/internal/catalog"
	"icoo_proxy/internal/config"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/endpoint"
	"icoo_proxy/internal/modelalias"
	"icoo_proxy/internal/projectsettings"
	"icoo_proxy/internal/proxy"
	"icoo_proxy/internal/routepolicy"
	"icoo_proxy/internal/server"
	"icoo_proxy/internal/supplier"
	"icoo_proxy/internal/uiprefs"
)

type App struct {
	ctx  context.Context // 应用上下文
	mu   sync.RWMutex    // 读写锁
	root string          // 应用根目录
	cfg  config.Config   // 应用配置
	// catalog    *catalog.Catalog        // 路由目录
	// service    *proxy.Service          // 代理服务
	// authKeys   *authkey.Service        // 认证密钥服务
	// suppliers  *supplier.Service       // 供应商服务
	// health     *supplier.HealthService // 健康检查服务
	// policies   *routepolicy.Service    // 路由策略服务
	// aliases    *modelalias.Service     // 模型别名服务
	// endpoints  *endpoint.Service       // 端点服务
	// traffic    *traffic.Service        // 流量服务
	// uiPrefs    *uiprefs.Service        // UI偏好服务
	app        *app.App     // 应用实例
	httpServer *http.Server // HTTP服务器
	chainLog   *os.File     // 请求链日志文件
	listenAddr string       // 监听地址
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
	p, err := app.NewApp(root)
	if err != nil {
		a.setLastError(err.Error())
		return
	}
	a.app = p
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.stopProxy(ctx)
}

func (a *App) GetOverview() map[string]interface{} {
	return stateToMap(a.State())
}

func (a *App) GetProjectSettings() (projectsettings.Values, error) {
	if strings.TrimSpace(a.root) == "" {
		return projectsettings.Values{}, context.Canceled
	}
	return projectsettings.Load(a.root)
}

func (a *App) SaveProjectSettings(input projectsettings.Values) (projectsettings.Values, error) {
	if strings.TrimSpace(a.root) == "" {
		return projectsettings.Values{}, context.Canceled
	}
	if err := projectsettings.Save(a.root, input); err != nil {
		return projectsettings.Values{}, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return projectsettings.Values{}, err
	}
	return projectsettings.Load(a.root)
}

func (a *App) GetUiPrefs() (uiprefs.Preferences, error) {
	if a.app == nil {
		return uiprefs.Preferences{}, context.Canceled
	}
	return a.app.Services().UiPref().Get()
}

func (a *App) SaveUiPrefs(input uiprefs.Preferences) (uiprefs.Preferences, error) {
	if a.app == nil {
		return uiprefs.Preferences{}, context.Canceled
	}
	if err := a.app.Services().UiPref().Save(input); err != nil {
		return uiprefs.Preferences{}, err
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

func (a *App) ListSuppliers() []supplier.Record {
	if a.app == nil {
		return nil
	}
	return a.app.Services().Supplier().List()
}

func (a *App) SaveSupplier(input supplier.UpsertInput) ([]supplier.Record, error) {
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

func (a *App) DeleteSupplier(id string) ([]supplier.Record, error) {
	if a.app == nil {
		return nil, context.Canceled
	}
	if a.app.Services().Policies() != nil {
		if policy, ok := a.app.Services().Policies().FindEnabledBySupplierID(id); ok {
			return nil, fmt.Errorf("supplier is used by enabled route policy %q", policy.DownstreamProtocol)
		}
	}
	if err := a.app.Services().Supplier().Delete(id); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.suppliers.List(), nil
}

func (a *App) ListSupplierHealth() []supplier.HealthRecord {
	if a.health == nil {
		return nil
	}
	return a.health.List()
}

func (a *App) CheckSupplier(id string) ([]supplier.HealthRecord, error) {
	if a.health == nil {
		return nil, context.Canceled
	}
	if _, err := a.health.Check(id); err != nil {
		return nil, err
	}
	return a.health.List(), nil
}

func (a *App) ListRoutePolicies() []routepolicy.Record {
	if a.policies == nil {
		return nil
	}
	return a.policies.List()
}

func (a *App) ListModelAliases() []modelalias.Record {
	if a.aliases == nil {
		return nil
	}
	return a.aliases.List()
}

func (a *App) SaveModelAlias(input modelalias.UpsertInput) ([]modelalias.Record, error) {
	if a.aliases == nil {
		return nil, context.Canceled
	}
	if _, err := a.aliases.Upsert(input); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.aliases.List(), nil
}

func (a *App) DeleteModelAlias(id string) ([]modelalias.Record, error) {
	if a.aliases == nil {
		return nil, context.Canceled
	}
	if err := a.aliases.Delete(id); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.aliases.List(), nil
}

func (a *App) SaveRoutePolicy(input routepolicy.UpsertInput) ([]routepolicy.Record, error) {
	if a.policies == nil {
		return nil, context.Canceled
	}
	if _, err := a.policies.Upsert(input); err != nil {
		return nil, err
	}
	if _, err := a.ReloadProxy(); err != nil {
		return nil, err
	}
	return a.policies.List(), nil
}

func (a *App) ListEndpoints() []endpoint.Record {
	if a.endpoints == nil {
		return nil
	}
	return a.endpoints.List()
}

func (a *App) SaveEndpoint(input endpoint.UpsertInput) ([]endpoint.Record, error) {
	if a.endpoints == nil {
		return nil, context.Canceled
	}
	if _, err := a.endpoints.Upsert(input); err != nil {
		return nil, err
	}
	return a.endpoints.List(), nil
}

func (a *App) DeleteEndpoint(id string) ([]endpoint.Record, error) {
	if a.endpoints == nil {
		return nil, context.Canceled
	}
	if err := a.endpoints.Delete(id); err != nil {
		return nil, err
	}
	return a.endpoints.List(), nil
}

func (a *App) ListAuthKeys() []authkey.Record {
	if a.authKeys == nil {
		return nil
	}
	return a.authKeys.List()
}

func (a *App) SaveAuthKey(input authkey.UpsertInput) ([]authkey.Record, error) {
	if a.authKeys == nil {
		return nil, context.Canceled
	}
	if _, err := a.authKeys.Upsert(input); err != nil {
		return nil, err
	}
	return a.authKeys.List(), nil
}

func (a *App) DeleteAuthKey(id string) ([]authkey.Record, error) {
	if a.authKeys == nil {
		return nil, context.Canceled
	}
	if err := a.authKeys.Delete(id); err != nil {
		return nil, err
	}
	return a.authKeys.List(), nil
}

func (a *App) GetAuthKeySecret(id string) (string, error) {
	if a.authKeys == nil {
		return "", context.Canceled
	}
	return a.authKeys.GetSecret(id)
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
				BaseURL:    a.cfg.AnthropicBaseURL,
				Configured: strings.TrimSpace(a.cfg.AnthropicAPIKey) != "",
			},
			{
				Protocol:   consts.ProtocolOpenAIChat,
				BaseURL:    a.cfg.OpenAIChatBaseURLValue(),
				Configured: strings.TrimSpace(a.cfg.OpenAIChatAPIKeyValue()) != "",
			},
			{
				Protocol:   consts.ProtocolOpenAIResponses,
				BaseURL:    a.cfg.OpenAIResponsesBaseURLValue(),
				Configured: strings.TrimSpace(a.cfg.OpenAIResponsesAPIKeyValue()) != "",
			},
		},
		Checks: map[string]interface{}{
			"proxy_running":           a.running,
			"anthropic_ready":         strings.TrimSpace(a.cfg.AnthropicAPIKey) != "",
			"openai_chat_ready":       strings.TrimSpace(a.cfg.OpenAIChatAPIKeyValue()) != "",
			"openai_responses_ready":  strings.TrimSpace(a.cfg.OpenAIResponsesAPIKeyValue()) != "",
			"route_catalog_ready":     a.app.Services().RouteCatalog() != nil,
			"supplier_store_ready":    a.suppliers != nil,
			"route_policy_ready":      a.policies != nil,
			"model_alias_store_ready": a.aliases != nil,
			"endpoint_store_ready":    a.endpoints != nil,
			"auth_key_store_ready":    a.authKeys != nil,
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
	} else if a.service != nil {
		state.RecentRequests = a.service.RecentRequests()
	}
	if a.endpoints != nil {
		for _, item := range a.endpoints.List() {
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
	}
	if a.policies != nil {
		for _, policy := range a.policies.List() {
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
	cfg, err := config.Load(a.root)
	if err != nil {
		return err
	}
	cfg, err = bootstrap.ApplyRoutePolicies(cfg, a.suppliers, a.policies)
	if err != nil {
		return err
	}
	if a.authKeys != nil {
		cfg.ProxyAPIKeys = authkey.MergeSecrets(cfg.ProxyAPIKeys, a.authKeys.EnabledSecrets())
	}
	if a.aliases != nil {
		cfg.ModelRoutes = modelalias.MergeEntries(cfg.ModelRoutes, a.aliases.EnabledEntries())
	}
	cat, err := catalog.New(cfg)
	if err != nil {
		return err
	}
	service := proxy.New(cfg, cat)
	if a.traffic != nil {
		service.SetRequestRecorder(a.traffic)
	}
	chainLogger, chainLog, err := openChainLog(cfg.ChainLogPath)
	if err != nil {
		return err
	}
	service.SetChainLogger(chainLogger)
	handler := api.NewMux(a, service, a.endpointRoutes())
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
	a.catalog = cat
	a.service = service
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
		defaults := endpoint.DefaultDefinitions()
		routes := make([]api.EndpointRoute, 0, len(defaults))
		for _, item := range defaults {
			protocol := consts.Protocol(item.Protocol)
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
	records := a.app.Services().Endpoint().Enabled()
	routes := make([]api.EndpointRoute, 0, len(records))
	for _, item := range records {
		protocol := consts.Protocol(item.Protocol)
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
		defaults := endpoint.DefaultDefinitions()
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
		"notes":                       state.Notes,
		"checks":                      state.Checks,
	}
}

type supplierResolverAdapter struct {
	svc *supplier.Service
}

func (a *supplierResolverAdapter) Resolve(id string) (modelalias.SupplierSnapshot, bool) {
	snap, ok := a.svc.Resolve(id)
	if !ok {
		return modelalias.SupplierSnapshot{}, false
	}
	return modelalias.SupplierSnapshot{
		ID:       snap.ID,
		Name:     snap.Name,
		Protocol: snap.Protocol,
	}, true
}
