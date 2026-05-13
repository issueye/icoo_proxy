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
	return nil
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
