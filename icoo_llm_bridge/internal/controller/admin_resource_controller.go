package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"icoo_llm_bridge/internal/service"
	"icoo_llm_bridge/internal/view"
)

type ProviderController struct {
	service service.ProviderService
}

func NewProviderController(service service.ProviderService) *ProviderController {
	return &ProviderController{service: service}
}

func (c *ProviderController) List(ctx *gin.Context) {
	items, err := c.service.List(ctx.Request.Context())
	writePagedResult(ctx, items, err)
}

func (c *ProviderController) Save(ctx *gin.Context) {
	var input service.ProviderUpsertInput
	if !bindJSON(ctx, &input) {
		return
	}
	if input.ID == "" {
		input.ID = pathID(ctx)
	}
	item, err := c.service.Upsert(ctx.Request.Context(), input)
	writeResult(ctx, item, err)
}

func (c *ProviderController) Delete(ctx *gin.Context) {
	writeResult(ctx, gin.H{"deleted": true}, c.service.Delete(ctx.Request.Context(), pathID(ctx)))
}

type ProviderModelController struct {
	service service.ProviderModelService
}

func NewProviderModelController(service service.ProviderModelService) *ProviderModelController {
	return &ProviderModelController{service: service}
}

func (c *ProviderModelController) List(ctx *gin.Context) {
	items, err := c.service.ListByProvider(ctx.Request.Context(), ctx.Param("provider_id"))
	writePagedResult(ctx, items, err)
}

func (c *ProviderModelController) Save(ctx *gin.Context) {
	var input service.ProviderModelUpsertInput
	if !bindJSON(ctx, &input) {
		return
	}
	input.ProviderID = ctx.Param("provider_id")
	if input.ID == "" {
		input.ID = ctx.Param("id")
	}
	item, err := c.service.Upsert(ctx.Request.Context(), input)
	writeResult(ctx, item, err)
}

func (c *ProviderModelController) Delete(ctx *gin.Context) {
	writeResult(ctx, gin.H{"deleted": true}, c.service.Delete(ctx.Request.Context(), ctx.Param("provider_id"), ctx.Param("id")))
}

type EndpointController struct {
	service service.EndpointService
}

func NewEndpointController(service service.EndpointService) *EndpointController {
	return &EndpointController{service: service}
}

func (c *EndpointController) List(ctx *gin.Context) {
	items, err := c.service.List(ctx.Request.Context())
	writePagedResult(ctx, items, err)
}

func (c *EndpointController) Save(ctx *gin.Context) {
	var input service.EndpointUpsertInput
	if !bindJSON(ctx, &input) {
		return
	}
	if input.ID == "" {
		input.ID = ctx.Param("id")
	}
	item, err := c.service.Upsert(ctx.Request.Context(), input)
	writeResult(ctx, item, err)
}

func (c *EndpointController) Delete(ctx *gin.Context) {
	writeResult(ctx, gin.H{"deleted": true}, c.service.Delete(ctx.Request.Context(), ctx.Param("id")))
}

type RoutingRuleController struct {
	service service.RoutingRuleService
}

func NewRoutingRuleController(service service.RoutingRuleService) *RoutingRuleController {
	return &RoutingRuleController{service: service}
}

func (c *RoutingRuleController) List(ctx *gin.Context) {
	items, err := c.service.List(ctx.Request.Context())
	writePagedResult(ctx, items, err)
}

func (c *RoutingRuleController) Save(ctx *gin.Context) {
	var input service.RoutingRuleUpsertInput
	if !bindJSON(ctx, &input) {
		return
	}
	if input.ID == "" {
		input.ID = ctx.Param("id")
	}
	item, err := c.service.Upsert(ctx.Request.Context(), input)
	writeResult(ctx, item, err)
}

func (c *RoutingRuleController) Delete(ctx *gin.Context) {
	writeResult(ctx, gin.H{"deleted": true}, c.service.Delete(ctx.Request.Context(), ctx.Param("id")))
}

type APIKeyController struct {
	service service.AuthService
}

func NewAPIKeyController(service service.AuthService) *APIKeyController {
	return &APIKeyController{service: service}
}

func (c *APIKeyController) List(ctx *gin.Context) {
	items, err := c.service.ListKeys(ctx.Request.Context())
	writePagedResult(ctx, items, err)
}

func (c *APIKeyController) Secret(ctx *gin.Context) {
	item, err := c.service.GetKeySecret(ctx.Request.Context(), ctx.Param("id"))
	writeResult(ctx, item, err)
}

func (c *APIKeyController) Create(ctx *gin.Context) {
	var input service.APIKeyCreateInput
	if !bindJSON(ctx, &input) {
		return
	}
	item, err := c.service.CreateKey(ctx.Request.Context(), input)
	writeResult(ctx, item, err)
}

func (c *APIKeyController) Delete(ctx *gin.Context) {
	writeResult(ctx, gin.H{"deleted": true}, c.service.DeleteKey(ctx.Request.Context(), ctx.Param("id")))
}

type TrafficController struct {
	service service.TrafficService
}

func NewTrafficController(service service.TrafficService) *TrafficController {
	return &TrafficController{service: service}
}

func (c *TrafficController) List(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "500"))
	items, err := c.service.List(ctx.Request.Context(), limit)
	writePagedResult(ctx, items, err)
}

func (c *TrafficController) Clear(ctx *gin.Context) {
	writeResult(ctx, gin.H{"cleared": true}, c.service.Clear(ctx.Request.Context()))
}

func bindJSON(ctx *gin.Context, target any) bool {
	if err := ctx.ShouldBindJSON(target); err != nil {
		ctx.JSON(http.StatusBadRequest, view.Response{Error: &view.Error{Code: "BAD_REQUEST", Message: err.Error()}})
		return false
	}
	return true
}

func pathID(ctx *gin.Context) string {
	if id := ctx.Param("id"); id != "" {
		return id
	}
	return ctx.Param("provider_id")
}

func writeResult(ctx *gin.Context, data any, err error) {
	if err != nil {
		ctx.JSON(http.StatusBadRequest, view.Response{Error: &view.Error{Code: "BAD_REQUEST", Message: err.Error()}})
		return
	}
	ctx.JSON(http.StatusOK, view.Response{Data: data})
}

func writePagedResult[T any](ctx *gin.Context, items []T, err error) {
	if err != nil {
		writeResult(ctx, nil, err)
		return
	}
	page, pageSize := pageParams(ctx)
	ctx.JSON(http.StatusOK, view.Response{Data: paginate(items, page, pageSize)})
}

func pageParams(ctx *gin.Context) (int, int) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", ctx.DefaultQuery("pageSize", "20")))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func paginate[T any](items []T, page int, pageSize int) view.Page[T] {
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	paged := make([]T, 0)
	if start < end {
		paged = items[start:end]
	}
	return view.Page[T]{
		Items:    paged,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}
