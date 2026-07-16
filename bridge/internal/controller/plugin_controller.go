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

func (c *PluginController) Discover(ctx *gin.Context) {
	items, err := c.service.Discover(ctx.Request.Context())
	writePagedResult(ctx, items, err)
}

func (c *PluginController) Register(ctx *gin.Context) {
	var input service.PluginRegisterInput
	if !bindJSON(ctx, &input) {
		return
	}
	writeResult(ctx, gin.H{"registered": true, "id": input.ID}, c.service.Register(ctx.Request.Context(), input))
}

func (c *PluginController) Install(ctx *gin.Context) {
	var input service.PluginInstallInput
	if !bindJSON(ctx, &input) {
		return
	}
	writeResult(ctx, gin.H{"installed": true, "id": input.ID}, c.service.Install(ctx.Request.Context(), input))
}

func (c *PluginController) Unregister(ctx *gin.Context) {
	writeResult(ctx, gin.H{"unregistered": true}, c.service.Unregister(ctx.Request.Context(), pathID(ctx)))
}

func (c *PluginController) SetEnabled(ctx *gin.Context) {
	var input service.PluginEnabledInput
	if !bindJSON(ctx, &input) {
		return
	}
	writeResult(ctx, gin.H{"enabled": input.Enabled}, c.service.SetEnabled(ctx.Request.Context(), pathID(ctx), input.Enabled))
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
