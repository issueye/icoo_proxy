package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/issueye/icoo_proxy/bridge/internal/service"
)

// ModelListController exposes OpenAI-compatible GET /v1/models for clients/tools.
type ModelListController struct {
	service service.ModelListService
}

func NewModelListController(service service.ModelListService) *ModelListController {
	return &ModelListController{service: service}
}

// List handles GET /v1/models (and optional aliases).
func (c *ModelListController) List(ctx *gin.Context) {
	if c.service == nil {
		ctx.JSON(http.StatusOK, service.OpenAIModelsListResponse{Object: "list", Data: []service.OpenAIModelRef{}})
		return
	}
	if !c.service.Authorize(ctx.Request) {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "invalid proxy api key",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}
	result, err := c.service.ListModels(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "server_error",
				"code":    "models_list_failed",
			},
		})
		return
	}
	if result.Data == nil {
		result.Data = []service.OpenAIModelRef{}
	}
	ctx.JSON(http.StatusOK, result)
}
