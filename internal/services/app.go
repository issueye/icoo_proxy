package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"icoo_proxy/internal/audit"
	"icoo_proxy/internal/config"
	"icoo_proxy/internal/gateway"
	"icoo_proxy/internal/protocol"
	"icoo_proxy/internal/provider"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	protocol.RegisterDefaults()

	configService := GetConfigService()
	configService.Init(ctx)
	if err := audit.GetService().Init(); err != nil {
		runtime.LogWarning(a.ctx, "Failed to initialize audit store: "+err.Error())
	}

	pm := provider.GetManager()
	pm.SetConfigProvider(configService)
	pm.LoadFromConfig()

	gwCfg := configService.GetGatewayConfig()
	gw := gateway.GetServer()
	if err := gw.Start(gwCfg.ListenHost, gwCfg.ListenPort); err != nil {
		runtime.LogWarning(a.ctx, "Failed to start gateway: "+err.Error())
	}

	go pm.RefreshModels(ctx)
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("你好 %s，现在是展示时间！", name)
}

func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) CloseWindow() {
	gateway.GetServer().Stop()
	GetConfigService().Close()
	audit.GetService().Close()
	runtime.Quit(a.ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	gateway.GetServer().Stop()
	GetConfigService().Close()
	audit.GetService().Close()
}

func (a *App) GetGatewayStatus() string {
	gw := gateway.GetServer()
	pm := provider.GetManager()
	allProviders := pm.GetAll()
	healthyCount := 0
	for _, p := range allProviders {
		if p.Healthy {
			healthyCount++
		}
	}
	status := map[string]interface{}{
		"running":       gw.IsRunning(),
		"host":          gw.GetHost(),
		"listenHost":    gw.GetHost(),
		"port":          gw.GetPort(),
		"providerCount": len(allProviders),
		"healthyCount":  healthyCount,
	}
	data, _ := json.Marshal(status)
	return string(data)
}

func (a *App) StartGateway() error {
	gwCfg := GetConfigService().GetGatewayConfig()
	return gateway.GetServer().Start(gwCfg.ListenHost, gwCfg.ListenPort)
}

func (a *App) StopGateway() error {
	return gateway.GetServer().Stop()
}

func (a *App) GetGatewayConfig() string {
	cfg := GetConfigService().GetGatewayConfig()
	data, _ := json.Marshal(cfg)
	return string(data)
}

func (a *App) SetGatewayConfig(listenHost string, listenPort int, defaultProvider, logLevel string, retryCount, retryIntervalMs int) error {
	return GetConfigService().SetGatewayConfig(config.GatewayConfig{
		ListenHost:      strings.TrimSpace(listenHost),
		ListenPort:      listenPort,
		DefaultProvider: defaultProvider,
		LogLevel:        logLevel,
		RetryCount:      retryCount,
		RetryIntervalMs: retryIntervalMs,
	})
}

func (a *App) GetProviders() string {
	pm := provider.GetManager()
	return provider.ProviderListJSON(pm.GetAll())
}

func (a *App) GetAPIKeys() string {
	data, _ := json.Marshal(GetConfigService().GetAPIKeys())
	return string(data)
}

func (a *App) AddAPIKey(id, name, key, description string, enabled bool, scopeMode string, providerIDs, endpointIDs []string) error {
	return GetConfigService().AddAPIKey(config.ApiKeyConfig{
		ID:          strings.TrimSpace(id),
		Name:        strings.TrimSpace(name),
		Key:         strings.TrimSpace(key),
		Description: strings.TrimSpace(description),
		Enabled:     enabled,
		ScopeMode:   strings.TrimSpace(scopeMode),
		ProviderIDs: providerIDs,
		EndpointIDs: endpointIDs,
	})
}

func (a *App) UpdateAPIKey(id, name, key, description string, enabled bool, scopeMode string, providerIDs, endpointIDs []string) error {
	return GetConfigService().UpdateAPIKey(config.ApiKeyConfig{
		ID:          strings.TrimSpace(id),
		Name:        strings.TrimSpace(name),
		Key:         strings.TrimSpace(key),
		Description: strings.TrimSpace(description),
		Enabled:     enabled,
		ScopeMode:   strings.TrimSpace(scopeMode),
		ProviderIDs: providerIDs,
		EndpointIDs: endpointIDs,
	})
}

func (a *App) DeleteAPIKey(id string) error {
	return GetConfigService().DeleteAPIKey(strings.TrimSpace(id))
}

func (a *App) GetEndpoints() string {
	data, _ := json.Marshal(GetConfigService().GetEndpoints())
	return string(data)
}

func (a *App) AddEndpoint(id, name, providerID, path, method, capability, requestProtocol, responseProtocol string, enabled bool, priority int, isDefault bool, remark string) error {
	return GetConfigService().AddEndpoint(config.EndpointConfig{
		ID:               strings.TrimSpace(id),
		Name:             strings.TrimSpace(name),
		ProviderID:       strings.TrimSpace(providerID),
		Path:             strings.TrimSpace(path),
		Method:           strings.TrimSpace(method),
		Capability:       strings.TrimSpace(capability),
		RequestProtocol:  strings.TrimSpace(requestProtocol),
		ResponseProtocol: strings.TrimSpace(responseProtocol),
		Enabled:          enabled,
		Priority:         priority,
		IsDefault:        isDefault,
		Remark:           strings.TrimSpace(remark),
	})
}

func (a *App) UpdateEndpoint(id, name, providerID, path, method, capability, requestProtocol, responseProtocol string, enabled bool, priority int, isDefault bool, remark string) error {
	return GetConfigService().UpdateEndpoint(config.EndpointConfig{
		ID:               strings.TrimSpace(id),
		Name:             strings.TrimSpace(name),
		ProviderID:       strings.TrimSpace(providerID),
		Path:             strings.TrimSpace(path),
		Method:           strings.TrimSpace(method),
		Capability:       strings.TrimSpace(capability),
		RequestProtocol:  strings.TrimSpace(requestProtocol),
		ResponseProtocol: strings.TrimSpace(responseProtocol),
		Enabled:          enabled,
		Priority:         priority,
		IsDefault:        isDefault,
		Remark:           strings.TrimSpace(remark),
	})
}

func (a *App) DeleteEndpoint(id string) error {
	return GetConfigService().DeleteEndpoint(strings.TrimSpace(id))
}

func (a *App) AddProvider(id, name, providerType, apiBase, apiKey, endpointMode string, enabled bool, priority int) error {
	return provider.GetManager().Add(config.ProviderConfig{
		ID:           id,
		Name:         name,
		Type:         providerType,
		APIBase:      apiBase,
		APIKey:       apiKey,
		EndpointMode: endpointMode,
		Enabled:      enabled,
		Priority:     priority,
	})
}

func (a *App) UpdateProvider(id, name, providerType, apiBase, apiKey, endpointMode string, enabled bool, priority int) error {
	return provider.GetManager().Update(config.ProviderConfig{
		ID:           id,
		Name:         name,
		Type:         providerType,
		APIBase:      apiBase,
		APIKey:       apiKey,
		EndpointMode: endpointMode,
		Enabled:      enabled,
		Priority:     priority,
	})
}

func (a *App) DeleteProvider(id string) error {
	return provider.GetManager().Delete(id)
}

func (a *App) TestProvider(id, name, providerType, apiBase, apiKey, endpointMode string) string {
	cfg := config.ProviderConfig{
		ID:           id,
		Name:         name,
		Type:         providerType,
		APIBase:      apiBase,
		APIKey:       apiKey,
		EndpointMode: endpointMode,
	}
	pm := provider.GetManager()
	err := pm.TestConnection(a.ctx, cfg)
	result := map[string]interface{}{
		"success": err == nil,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func (a *App) RefreshModels() string {
	pm := provider.GetManager()
	pm.RefreshModels(a.ctx)
	return provider.ProviderListJSON(pm.GetAll())
}

func (a *App) GetProviderModels(providerID string) string {
	pm := provider.GetManager()
	llms, defaultModel, err := pm.GetModels(providerID)
	result := map[string]interface{}{
		"llms":         llms,
		"defaultModel": defaultModel,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func (a *App) SetProviderModels(providerID string, llms []config.ModelEntry, defaultModel string) error {
	return provider.GetManager().SetModels(providerID, llms, defaultModel)
}

func (a *App) GetModels() string {
	models := provider.GetManager().GetAllModels()
	type modelObj struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnedBy string `json:"ownedBy"`
	}
	list := make([]modelObj, 0, len(models))
	for _, m := range models {
		list = append(list, modelObj{ID: m.ID, Name: m.Name, OwnedBy: m.OwnedBy})
	}
	data, _ := json.Marshal(list)
	return string(data)
}

func (a *App) GetGatewayRequestLogs(limit int) string {
	return audit.GetService().ListJSON(limit)
}
