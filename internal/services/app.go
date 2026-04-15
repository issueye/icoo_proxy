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

	// Register protocol adapters
	protocol.RegisterDefaults()

	// Initialize config
	configService := GetConfigService()
	configService.Init(ctx)
	if err := audit.GetService().Init(); err != nil {
		runtime.LogWarning(a.ctx, "Failed to initialize audit store: "+err.Error())
	}

	// Initialize agent process manager
	GetAgentProcessManager().Init(ctx)

	// Set up proxy target from config
	clawCfg := configService.GetClawConnectionConfig()
	if clawCfg.APIBase != "" {
		GetAPIProxy().SetTargetBase(clawCfg.APIBase)
	}

	// Inject config provider into provider manager
	pm := provider.GetManager()
	pm.SetConfigProvider(configService)
	pm.LoadFromConfig()

	// Start gateway server
	gwCfg := configService.GetGatewayConfig()
	gw := gateway.GetServer()
	if err := gw.Start(gwCfg.ListenPort); err != nil {
		runtime.LogWarning(a.ctx, "Failed to start gateway: "+err.Error())
	}

	// Refresh models from providers
	go pm.RefreshModels(ctx)
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("你好 %s，现在是展示时间！", name)
}

func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) CloseWindow() {
	GetAgentProcessManager().Shutdown()
	gateway.GetServer().Stop()
	GetConfigService().Close()
	audit.GetService().Close()
	runtime.Quit(a.ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	GetAgentProcessManager().Shutdown()
	gateway.GetServer().Stop()
	GetConfigService().Close()
	audit.GetService().Close()
}

// --- Legacy compatibility ---

func (a *App) GetClawConnectionConfig() map[string]string {
	cfg := GetConfigService().GetClawConnectionConfig()
	agentCfg := GetConfigService().GetAgentProcessConfig()
	return map[string]string{
		"apiBase":   cfg.APIBase,
		"wsHost":    cfg.WSHost,
		"wsPort":    cfg.WSPort,
		"wsPath":    cfg.WSPath,
		"userId":    cfg.UserID,
		"agentPath": agentCfg.BinaryPath,
	}
}

func (a *App) SetClawConnectionConfig(apiBase, wsHost, wsPort, wsPath, userId, agentPath string) error {
	cfg := ClawConnectionConfig{
		APIBase: apiBase,
		WSHost:  wsHost,
		WSPort:  wsPort,
		WSPath:  wsPath,
		UserID:  userId,
	}
	GetAPIProxy().SetTargetBase(apiBase)
	if err := GetConfigService().SetClawConnectionConfig(cfg); err != nil {
		return err
	}
	return GetConfigService().SetAgentProcessConfig(AgentProcessConfig{
		BinaryPath: strings.TrimSpace(agentPath),
	})
}

// --- Agent Process ---

func (a *App) GetAgentProcessStatus() AgentProcessStatus {
	return GetAgentProcessManager().Status()
}

func (a *App) WakeAgent() (AgentProcessStatus, error) {
	return GetAgentProcessManager().Wake()
}

func (a *App) StopAgent() (AgentProcessStatus, error) {
	return GetAgentProcessManager().Stop()
}

func (a *App) RestartAgent() (AgentProcessStatus, error) {
	return GetAgentProcessManager().Restart()
}

// --- Gateway ---

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
		"port":          gw.GetPort(),
		"providerCount": len(allProviders),
		"healthyCount":  healthyCount,
	}
	data, _ := json.Marshal(status)
	return string(data)
}

func (a *App) StartGateway() error {
	gwCfg := GetConfigService().GetGatewayConfig()
	return gateway.GetServer().Start(gwCfg.ListenPort)
}

func (a *App) StopGateway() error {
	return gateway.GetServer().Stop()
}

func (a *App) GetGatewayConfig() string {
	cfg := GetConfigService().GetGatewayConfig()
	data, _ := json.Marshal(cfg)
	return string(data)
}

func (a *App) SetGatewayConfig(listenPort int, defaultProvider, logLevel string, retryCount, retryIntervalMs int, authKey string) error {
	return GetConfigService().SetGatewayConfig(config.GatewayConfig{
		ListenPort:      listenPort,
		DefaultProvider: defaultProvider,
		LogLevel:        logLevel,
		RetryCount:      retryCount,
		RetryIntervalMs: retryIntervalMs,
		AuthKey:         strings.TrimSpace(authKey),
	})
}

// --- Providers ---

func (a *App) GetProviders() string {
	pm := provider.GetManager()
	return provider.ProviderListJSON(pm.GetAll())
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

// GetProviderModels returns the model list for a specific provider.
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

// SetProviderModels updates the model list for a specific provider.
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

func (a *App) GetRouteRules() string {
	rules := GetConfigService().GetRouteRules()
	data, _ := json.Marshal(rules)
	return string(data)
}

func (a *App) SetRouteRules(rules []config.RouteRuleConfig) error {
	return GetConfigService().SetRouteRules(rules)
}
