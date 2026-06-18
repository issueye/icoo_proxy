package router

import (
	"github.com/gin-gonic/gin"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/controller"
	"icoo_llm_bridge/internal/middleware"
)

func New(controllers controller.Controllers, middlewares middleware.Middlewares) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middlewares.Recovery, middlewares.RequestID, middlewares.CORS)

	engine.GET("/", controllers.Health.Index)
	engine.GET("/healthz", controllers.Health.Healthz)
	engine.GET("/readyz", controllers.Health.Readyz)

	engine.POST("/v1/messages", controllers.Proxy.Handle(constants.ProtocolAnthropic))
	engine.POST("/v1/chat/completions", controllers.Proxy.Handle(constants.ProtocolOpenAIChat))
	engine.POST("/v1/responses", controllers.Proxy.Handle(constants.ProtocolOpenAIResponses))

	api := engine.Group("/api/v1", middlewares.AdminAuth)
	api.GET("/runtime/state", controllers.Runtime.State)
	api.GET("/providers", controllers.Provider.List)
	api.POST("/providers", controllers.Provider.Save)
	api.PUT("/providers/:provider_id", controllers.Provider.Save)
	api.DELETE("/providers/:provider_id", controllers.Provider.Delete)
	api.GET("/providers/:provider_id/models", controllers.ProviderModel.List)
	api.POST("/providers/:provider_id/models", controllers.ProviderModel.Save)
	api.POST("/providers/:provider_id/fetch-models", controllers.ProviderModel.Fetch)
	api.PUT("/providers/:provider_id/models/:id", controllers.ProviderModel.Save)
	api.DELETE("/providers/:provider_id/models/:id", controllers.ProviderModel.Delete)
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

	engine.NoRoute(controllers.Proxy.HandleDynamic())

	return engine
}
