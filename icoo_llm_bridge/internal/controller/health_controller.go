package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"icoo_llm_bridge/internal/service"
)

type HealthController struct {
	runtime service.RuntimeService
}

func NewHealthController(runtime service.RuntimeService) *HealthController {
	return &HealthController{runtime: runtime}
}

func (c *HealthController) Index(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.runtime.State(ctx.Request.Context()))
}

func (c *HealthController) Healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"service": "icoo_llm_bridge", "status": "ok"})
}

func (c *HealthController) Readyz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"service": "icoo_llm_bridge", "ready": true})
}
