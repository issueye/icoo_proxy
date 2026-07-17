package router

import (
	"github.com/gin-gonic/gin"

	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/bridge/internal/controller"
	"github.com/issueye/icoo_proxy/bridge/internal/middleware"
)

func New(controllers controller.Controllers, middlewares middleware.Middlewares) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	engine.Use(middlewares.Recovery, middlewares.RequestID, middlewares.CORS)

	engine.GET("/", controllers.Health.Index)
	engine.GET("/healthz", controllers.Health.Healthz)
	engine.GET("/readyz", controllers.Health.Readyz)

	engine.POST("/v1/messages", controllers.Proxy.Handle(constants.ProtocolAnthropic))
	engine.POST("/v1/chat/completions", controllers.Proxy.Handle(constants.ProtocolOpenAIChat))
	engine.POST("/v1/responses", controllers.Proxy.Handle(constants.ProtocolOpenAIResponses))
	// OpenAI-compatible model list for clients / tools (Cursor, OpenAI SDK, etc.).
	engine.GET("/v1/models", controllers.ModelList.List)
	engine.GET("/models", controllers.ModelList.List)

	api := engine.Group("/api/v1", middlewares.AdminAuth)
	api.GET("/runtime/state", controllers.Runtime.State)
	api.GET("/providers", controllers.Provider.List)
	api.POST("/providers", controllers.Provider.Save)
	api.PUT("/providers/:provider_id", controllers.Provider.Save)
	api.DELETE("/providers/:provider_id", controllers.Provider.Delete)
	api.POST("/providers/:provider_id/check", controllers.Provider.Check)
	api.POST("/providers/:provider_id/chat", controllers.Provider.Chat)
	api.GET("/providers/:provider_id/models", controllers.ProviderModel.List)
	api.POST("/providers/:provider_id/models", controllers.ProviderModel.Save)
	api.POST("/providers/:provider_id/fetch-models", controllers.ProviderModel.Fetch)
	api.PUT("/providers/:provider_id/models/:id", controllers.ProviderModel.Save)
	api.DELETE("/providers/:provider_id/models/:id", controllers.ProviderModel.Delete)
	api.GET("/model-catalog", controllers.ModelCatalog.List)
	api.POST("/model-catalog", controllers.ModelCatalog.Save)
	api.PUT("/model-catalog/:id", controllers.ModelCatalog.Save)
	api.DELETE("/model-catalog/:id", controllers.ModelCatalog.Delete)
	api.GET("/ingress-endpoints", controllers.Endpoint.List)
	api.POST("/ingress-endpoints", controllers.Endpoint.Save)
	api.PUT("/ingress-endpoints/:id", controllers.Endpoint.Save)
	api.DELETE("/ingress-endpoints/:id", controllers.Endpoint.Delete)
	api.GET("/routing-rules", controllers.RoutingRule.List)
	api.POST("/routing-rules", controllers.RoutingRule.Save)
	api.PUT("/routing-rules/:id", controllers.RoutingRule.Save)
	api.DELETE("/routing-rules/:id", controllers.RoutingRule.Delete)
	api.GET("/api-keys", controllers.APIKey.List)
	api.GET("/api-keys/:id/secret", controllers.APIKey.Secret)
	api.POST("/api-keys", controllers.APIKey.Create)
	api.DELETE("/api-keys/:id", controllers.APIKey.Delete)
	api.GET("/traffic", controllers.Traffic.List)
	api.DELETE("/traffic", controllers.Traffic.Clear)
	api.GET("/ui-prefs", controllers.UIPreference.Get)
	api.PUT("/ui-prefs", controllers.UIPreference.Save)

	// Static plugin collection routes MUST be registered before "/plugins/:id/..."
	// so Gin does not treat "ui-pages" / "discover" / "install" as :id values.
	api.GET("/plugins", controllers.Plugin.List)
	api.POST("/plugins", controllers.Plugin.Register)
	api.GET("/plugins/ui-pages", controllers.Plugin.ListUIPages)
	api.GET("/plugins/discover", controllers.Plugin.Discover)
	api.POST("/plugins/install", controllers.Plugin.Install)
	api.POST("/plugins/:id/start", controllers.Plugin.Start)
	api.POST("/plugins/:id/stop", controllers.Plugin.Stop)
	api.POST("/plugins/:id/restart", controllers.Plugin.Restart)
	api.PUT("/plugins/:id/enabled", controllers.Plugin.SetEnabled)
	api.DELETE("/plugins/:id", controllers.Plugin.Unregister)
	api.GET("/plugins/:id/health", controllers.Plugin.Health)
	api.GET("/plugins/:id/models", controllers.Plugin.Models)
	// Plugin-provided extension UI (iframe target).
	// Gin forbids registering both "/ui/" and "/ui/*filepath" (conflict on '').
	// Cover exact "/ui" plus catch-all nested paths under "/ui/*filepath".
	api.Any("/plugins/:id/ui", controllers.Plugin.ProxyUI)
	api.Any("/plugins/:id/ui/*filepath", controllers.Plugin.ProxyUI)

	engine.NoRoute(controllers.Proxy.HandleDynamic())

	return engine
}
