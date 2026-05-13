package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/service"
)

type ProxyController struct {
	proxy     service.ProxyService
	endpoints service.EndpointService
}

func NewProxyController(proxy service.ProxyService, endpoints service.EndpointService) *ProxyController {
	return &ProxyController{proxy: proxy, endpoints: endpoints}
}

func (c *ProxyController) Handle(protocol constants.Protocol) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Method != http.MethodPost {
			ctx.Status(http.StatusMethodNotAllowed)
			return
		}
		c.proxy.Handle(ctx.Writer, ctx.Request, protocol)
	}
}

func (c *ProxyController) HandleDynamic() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Method != http.MethodPost {
			ctx.Status(http.StatusNotFound)
			return
		}
		protocol, ok := c.matchEndpoint(ctx)
		if !ok {
			ctx.Status(http.StatusNotFound)
			return
		}
		c.proxy.Handle(ctx.Writer, ctx.Request, protocol)
	}
}

func (c *ProxyController) matchEndpoint(ctx *gin.Context) (constants.Protocol, bool) {
	if c.endpoints == nil {
		return "", false
	}
	items, err := c.endpoints.Enabled(ctx.Request.Context())
	if err != nil {
		return "", false
	}
	requestPath := normalizeEndpointPath(ctx.Request.URL.Path)
	for _, item := range items {
		if normalizeEndpointPath(item.Path) == requestPath {
			return item.DownstreamProtocol, true
		}
	}
	return "", false
}

func normalizeEndpointPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}
