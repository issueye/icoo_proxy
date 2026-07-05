package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/entity"
	"icoo_llm_bridge/internal/repository"
)

var providerChatClient = &http.Client{Timeout: 120 * time.Second}

type providerChatService struct {
	providerRepo repository.ProviderRepository
}

func NewProviderChatService(providerRepo repository.ProviderRepository) ProviderChatService {
	return &providerChatService{providerRepo: providerRepo}
}

func (s *providerChatService) Chat(ctx context.Context, providerID string, input ProviderChatInput) (ProviderChatResult, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ProviderChatResult{}, fmt.Errorf("provider_id is required")
	}
	provider, err := s.providerRepo.Find(ctx, providerID)
	if err != nil {
		return ProviderChatResult{}, fmt.Errorf("provider not found: %w", err)
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		return ProviderChatResult{}, fmt.Errorf("model is required")
	}
	messages := normalizeChatMessages(input.Messages)
	if len(messages) == 0 {
		return ProviderChatResult{}, fmt.Errorf("message is required")
	}

	body, err := buildProviderChatRequest(provider.Protocol, model, messages, input)
	if err != nil {
		return ProviderChatResult{}, err
	}
	url := joinUpstreamURL(provider.BaseURL, provider.Protocol)
	if url == "" {
		return ProviderChatResult{}, fmt.Errorf("upstream base_url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ProviderChatResult{}, fmt.Errorf("build upstream request: %w", err)
	}
	applyProviderChatHeaders(req, provider)

	client := providerChatClient
	if strings.TrimSpace(provider.ProxyURL) != "" {
		client, err = newProxiedHTTPClient(providerChatClient.Timeout, provider.ProxyURL)
		if err != nil {
			return ProviderChatResult{}, err
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return ProviderChatResult{}, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := readLimitedBody(resp.Body, maxUpstreamErrorBodyBytes)
	if err != nil {
		return ProviderChatResult{}, fmt.Errorf("read upstream response failed: %w", err)
	}
	if !isHTTPSuccess(resp.StatusCode) {
		return ProviderChatResult{}, errors.New(upstreamErrorMessage(resp.StatusCode, respBody))
	}

	content := extractProviderChatText(provider.Protocol, respBody)
	if strings.TrimSpace(content) == "" {
		return ProviderChatResult{}, fmt.Errorf("upstream returned no text content")
	}

	return ProviderChatResult{
		SupplierID: provider.ID,
		Model:      model,
		Message: ProviderChatMessage{
			Role:    "assistant",
			Content: content,
		},
		StatusCode: resp.StatusCode,
		DurationMS: duration,
	}, nil
}

func normalizeChatMessages(messages []ProviderChatMessage) []ProviderChatMessage {
	result := make([]ProviderChatMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		switch role {
		case "system", "user", "assistant":
			result = append(result, ProviderChatMessage{Role: role, Content: content})
		default:
			result = append(result, ProviderChatMessage{Role: "user", Content: content})
		}
	}
	return result
}

func buildProviderChatRequest(protocol constants.Protocol, model string, messages []ProviderChatMessage, input ProviderChatInput) ([]byte, error) {
	maxTokens := input.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	switch protocol {
	case constants.ProtocolAnthropic:
		payload := map[string]any{
			"model":      model,
			"max_tokens": maxTokens,
			"messages":   anthropicMessages(messages),
			"stream":     false,
		}
		if system := systemPrompt(messages); system != "" {
			payload["system"] = system
		}
		if input.Temperature != nil {
			payload["temperature"] = *input.Temperature
		}
		return json.Marshal(payload)
	case constants.ProtocolOpenAIChat:
		payload := map[string]any{
			"model":      model,
			"messages":   openAIChatMessages(messages),
			"max_tokens": maxTokens,
			"stream":     false,
		}
		if input.Temperature != nil {
			payload["temperature"] = *input.Temperature
		}
		return json.Marshal(payload)
	case constants.ProtocolOpenAIResponses:
		payload := map[string]any{
			"model":             model,
			"input":             openAIResponsesInput(messages),
			"max_output_tokens": maxTokens,
			"stream":            false,
		}
		if input.Temperature != nil {
			payload["temperature"] = *input.Temperature
		}
		return json.Marshal(payload)
	default:
		return nil, fmt.Errorf("protocol is invalid")
	}
}

func systemPrompt(messages []ProviderChatMessage) string {
	var parts []string
	for _, message := range messages {
		if message.Role == "system" {
			parts = append(parts, message.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func anthropicMessages(messages []ProviderChatMessage) []map[string]string {
	result := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" {
			continue
		}
		result = append(result, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}
	return result
}

func openAIChatMessages(messages []ProviderChatMessage) []map[string]string {
	result := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		result = append(result, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}
	return result
}

func openAIResponsesInput(messages []ProviderChatMessage) []map[string]string {
	result := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		result = append(result, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}
	return result
}

func applyProviderChatHeaders(req *http.Request, provider entity.Provider) {
	req.Header.Set("Content-Type", "application/json")
	switch provider.Protocol {
	case constants.ProtocolAnthropic:
		req.Header.Set("x-api-key", provider.APIKeyCipher)
		req.Header.Set("anthropic-version", "2023-06-01")
	case constants.ProtocolOpenAIChat, constants.ProtocolOpenAIResponses:
		req.Header.Set("Authorization", "Bearer "+provider.APIKeyCipher)
	}
	if provider.UserAgent != "" {
		req.Header.Set("User-Agent", provider.UserAgent)
	}
}

func extractProviderChatText(protocol constants.Protocol, body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if text, _ := payload["output_text"].(string); strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	switch protocol {
	case constants.ProtocolAnthropic:
		return extractChatTextParts(payload["content"])
	case constants.ProtocolOpenAIChat:
		return extractChatCompletionText(payload)
	case constants.ProtocolOpenAIResponses:
		return extractResponsesText(payload)
	default:
		return ""
	}
}

func extractChatCompletionText(payload map[string]any) string {
	choices, _ := payload["choices"].([]any)
	if len(choices) == 0 {
		return ""
	}
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)
	return textContent(message["content"])
}

func extractResponsesText(payload map[string]any) string {
	output, _ := payload["output"].([]any)
	var parts []string
	for _, item := range output {
		outputItem, _ := item.(map[string]any)
		if text := extractChatTextParts(outputItem["content"]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func extractChatTextParts(value any) string {
	items, _ := value.([]any)
	var parts []string
	for _, item := range items {
		block, _ := item.(map[string]any)
		if text := textContent(block["text"]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func textContent(value any) string {
	switch item := value.(type) {
	case string:
		return strings.TrimSpace(item)
	case []any:
		return extractChatTextParts(item)
	default:
		return ""
	}
}
