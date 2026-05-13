package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"icoo_llm_bridge/internal/service"
	"icoo_llm_bridge/internal/view"
)

type RuntimeController struct {
	runtime service.RuntimeService
}

func NewRuntimeController(runtime service.RuntimeService) *RuntimeController {
	return &RuntimeController{runtime: runtime}
}

func (c *RuntimeController) State(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, view.Response{Data: c.runtime.State(ctx.Request.Context())})
}
