package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/issueye/icoo_proxy/bridge/internal/service"
)

// PluginController exposes process-plugin admin APIs.
type PluginController struct {
	service service.PluginService
}

func NewPluginController(service service.PluginService) *PluginController {
	return &PluginController{service: service}
}

func (c *PluginController) List(ctx *gin.Context) {
	items, err := c.service.List(ctx.Request.Context())
	writePagedResult(ctx, items, err)
}

func (c *PluginController) Start(ctx *gin.Context) {
	writeResult(ctx, gin.H{"started": true}, c.service.Start(ctx.Request.Context(), pathID(ctx)))
}

func (c *PluginController) Stop(ctx *gin.Context) {
	writeResult(ctx, gin.H{"stopped": true}, c.service.Stop(ctx.Request.Context(), pathID(ctx)))
}

func (c *PluginController) Restart(ctx *gin.Context) {
	writeResult(ctx, gin.H{"restarted": true}, c.service.Restart(ctx.Request.Context(), pathID(ctx)))
}

func (c *PluginController) Health(ctx *gin.Context) {
	item, err := c.service.Health(ctx.Request.Context(), pathID(ctx))
	writeResult(ctx, item, err)
}

func (c *PluginController) Models(ctx *gin.Context) {
	items, err := c.service.Models(ctx.Request.Context(), pathID(ctx))
	writeResult(ctx, items, err)
}
