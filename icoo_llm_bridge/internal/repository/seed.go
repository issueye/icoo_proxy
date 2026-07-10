package repository

import (
	"context"
	"strings"
	"time"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/entity"
)

func SeedDefaults(ctx context.Context, repos Repositories) error {
	for _, item := range defaultEndpoints() {
		if err := repos.Endpoint.Save(ctx, &item); err != nil {
			return err
		}
	}
	for _, item := range defaultModelCatalog() {
		if err := repos.ModelCatalog.Save(ctx, &item); err != nil {
			return err
		}
	}
	return nil
}

func defaultModelCatalog() []entity.ModelCatalogItem {
	now := time.Now()
	return []entity.ModelCatalogItem{
		newCatalogModel("model-gpt", "GPT", "OpenAI", "gpt", 128000, "OpenAI GPT 系列模型", now),
		newCatalogModel("model-deepseek", "DeepSeek", "DeepSeek", "deepseek", 65536, "DeepSeek 通用模型系列", now),
		newCatalogModel("model-glm", "GLM", "智谱 AI", "glm", 128000, "智谱 GLM 模型系列", now),
		newCatalogModel("model-claude", "Claude", "Anthropic", "claude", 200000, "Anthropic Claude 模型系列", now),
	}
}

func newCatalogModel(id string, name string, family string, icon string, maxTokens int, description string, now time.Time) entity.ModelCatalogItem {
	return entity.ModelCatalogItem{
		ID: id, Name: name, Family: family, Icon: icon, MaxTokens: maxTokens,
		Description: description, BuiltIn: true, CreatedAt: now, UpdatedAt: now,
	}
}

func defaultEndpoints() []entity.IngressEndpoint {
	now := time.Now()
	return []entity.IngressEndpoint{
		newEndpoint("/v1/messages", constants.ProtocolAnthropic, "Anthropic Messages compatible endpoint.", now),
		newEndpoint("/v1/chat/completions", constants.ProtocolOpenAIChat, "OpenAI Chat Completions compatible endpoint.", now),
		newEndpoint("/v1/responses", constants.ProtocolOpenAIResponses, "OpenAI Responses compatible endpoint.", now),
	}
}

func newEndpoint(path string, protocol constants.Protocol, description string, now time.Time) entity.IngressEndpoint {
	return entity.IngressEndpoint{
		ID:                 "endpoint-" + strings.Trim(strings.ReplaceAll(path, "/", "-"), "-"),
		Path:               path,
		DownstreamProtocol: protocol,
		Enabled:            true,
		Protected:          true,
		BuiltIn:            true,
		Description:        description,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
