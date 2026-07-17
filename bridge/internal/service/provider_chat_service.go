package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/issueye/icoo_proxy/bridge/internal/model/entity"
	"github.com/issueye/icoo_proxy/bridge/internal/repository"
	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

var providerChatClient = &http.Client{Timeout: 120 * time.Second}

type providerChatService struct {
	providerRepo repository.ProviderRepository
	modelRepo    repository.ProviderModelRepository
	plugins      PluginRuntime
}

func NewProviderChatService(providerRepo repository.ProviderRepository, modelRepo repository.ProviderModelRepository, plugins ...PluginRuntime) ProviderChatService {
	var runtime PluginRuntime
	if len(plugins) > 0 {
		runtime = plugins[0]
	}
	return &providerChatService{providerRepo: providerRepo, modelRepo: modelRepo, plugins: runtime}
}

func (s *providerChatService) Check(ctx context.Context, providerID string) (ProviderHealthResult, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ProviderHealthResult{}, fmt.Errorf("provider_id is required")
	}
	provider, err := s.providerRepo.Find(ctx, providerID)
	if err != nil {
		return ProviderHealthResult{}, fmt.Errorf("provider not found: %w", err)
	}
	checkedAt := time.Now()
	result := ProviderHealthResult{
		SupplierID: provider.ID,
		Status:     "warning",
		Message:    "provider has no enabled model",
		CheckedAt:  checkedAt.Format(time.RFC3339),
	}
	if !provider.Enabled {
		result.Message = "provider is disabled"
		return result, nil
	}

	// Process plugin providers: probe plugin.health instead of HTTP upstream.
	if provider.Vendor == constants.VendorPlugin {
		return s.checkPluginProvider(ctx, provider, checkedAt)
	}
	models, err := s.modelRepo.ListByProvider(ctx, provider.ID)
	if err != nil {
		return ProviderHealthResult{}, fmt.Errorf("list provider models: %w", err)
	}
	model := ""
	for _, item := range models {
		if item.Enabled {
			model = strings.TrimSpace(item.Name)
			if model != "" {
				break
			}
		}
	}
	if model == "" {
		return result, nil
	}

	body, err := buildProviderChatRequest(provider.Protocol, model, []ProviderChatMessage{{
		Role:    "user",
		Content: "Reply with OK.",
	}}, ProviderChatInput{MaxTokens: 1})
	if err != nil {
		return ProviderHealthResult{}, err
	}
	url := joinUpstreamURL(provider.BaseURL, provider.Protocol)
	if url == "" {
		return ProviderHealthResult{}, fmt.Errorf("upstream base_url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ProviderHealthResult{}, fmt.Errorf("build upstream request: %w", err)
	}
	applyProviderChatHeaders(req, provider)
	client := providerChatClient
	if strings.TrimSpace(provider.ProxyURL) != "" {
		client, err = newProxiedHTTPClient(providerChatClient.Timeout, provider.ProxyURL)
		if err != nil {
			return ProviderHealthResult{}, err
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	result.DurationMS = time.Since(start).Milliseconds()
	result.CheckedAt = time.Now().Format(time.RFC3339)
	if err != nil {
		result.Status = "unreachable"
		result.Message = err.Error()
		return result, nil
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	if !isHTTPSuccess(resp.StatusCode) {
		respBody, readErr := readLimitedBody(resp.Body, maxUpstreamErrorBodyBytes)
		result.Status = "unreachable"
		if readErr != nil {
			result.Message = "read upstream error response failed"
		} else {
			result.Message = upstreamErrorMessage(resp.StatusCode, respBody)
		}
		return result, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxUpstreamErrorBodyBytes))
	result.Status = "reachable"
	result.Message = "upstream request succeeded"
	return result, nil
}

func (s *providerChatService) checkPluginProvider(ctx context.Context, provider entity.Provider, checkedAt time.Time) (ProviderHealthResult, error) {
	pluginID := ResolveProviderPluginID(provider.Vendor, provider.PluginID, provider.BaseURL)
	result := ProviderHealthResult{
		SupplierID: provider.ID,
		CheckedAt:  checkedAt.Format(time.RFC3339),
	}
	if pluginID == "" {
		result.Status = "warning"
		result.Message = "plugin_id is not configured"
		return result, nil
	}
	if s.plugins == nil {
		result.Status = "unreachable"
		result.Message = "plugin runtime is not configured"
		return result, nil
	}
	start := time.Now()
	health, err := s.plugins.Health(ctx, pluginID)
	result.DurationMS = time.Since(start).Milliseconds()
	result.CheckedAt = time.Now().Format(time.RFC3339)
	if err != nil {
		result.Status = "unreachable"
		result.Message = err.Error()
		return result, nil
	}
	if health != nil && health.OK {
		result.Status = "reachable"
		result.Message = health.Status
		if result.Message == "" {
			result.Message = "plugin healthy"
		}
		return result, nil
	}
	result.Status = "warning"
	result.Message = "plugin reported unhealthy"
	if health != nil && health.Status != "" {
		result.Message = health.Status
	}
	return result, nil
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

	// Process plugin providers: build the preferred-ingress body and call
	// proxy.complete over IPC (same path as the gateway hot path).
	if provider.Vendor == constants.VendorPlugin {
		return s.chatPluginProvider(ctx, provider, model, messages, input)
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

func (s *providerChatService) chatPluginProvider(
	ctx context.Context,
	provider entity.Provider,
	model string,
	messages []ProviderChatMessage,
	input ProviderChatInput,
) (ProviderChatResult, error) {
	pluginID := ResolveProviderPluginID(provider.Vendor, provider.PluginID, provider.BaseURL)
	if pluginID == "" {
		return ProviderChatResult{}, fmt.Errorf("plugin_id is not configured")
	}
	if s.plugins == nil {
		return ProviderChatResult{}, fmt.Errorf("plugin runtime is not configured")
	}
	if provider.OnlyStream {
		return ProviderChatResult{}, fmt.Errorf("provider only_stream is enabled; non-stream admin chat is not allowed")
	}

	protocol := provider.Protocol
	if protocol == "" {
		protocol = constants.ProtocolOpenAIResponses
	}
	body, err := buildProviderChatRequest(protocol, model, messages, input)
	if err != nil {
		return ProviderChatResult{}, err
	}

	cli, err := s.plugins.Client(pluginID)
	if err != nil {
		return ProviderChatResult{}, fmt.Errorf("plugin %q unavailable: %w", pluginID, err)
	}

	req := pluginipc.NewProxyRequest(pluginipc.ProxyRequestInput{
		Ingress: protocol.String(),
		Path:    providerChatIngressPath(protocol),
		Method:  http.MethodPost,
		Headers: map[string]string{"content-type": "application/json"},
		Body:    body,
		Model:   model,
		Stream:  false,
	})

	start := time.Now()
	resp, err := cli.Complete(ctx, req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		_, msg := pluginipc.MapCallError(err)
		return ProviderChatResult{}, fmt.Errorf("plugin chat failed: %s", msg)
	}
	if resp == nil {
		return ProviderChatResult{}, fmt.Errorf("empty plugin response")
	}
	status := resp.StatusOrOK()
	if !resp.Success() {
		return ProviderChatResult{}, errors.New(upstreamErrorMessage(status, resp.Body))
	}

	content := extractProviderChatText(protocol, resp.Body)
	if strings.TrimSpace(content) == "" {
		// Plugin may respond in Responses shape even when ingress was Chat/Anthropic
		// conversion failed partial; try sibling extractors before giving up.
		for _, p := range []constants.Protocol{
			constants.ProtocolOpenAIResponses,
			constants.ProtocolOpenAIChat,
			constants.ProtocolAnthropic,
		} {
			if p == protocol {
				continue
			}
			if text := extractProviderChatText(p, resp.Body); strings.TrimSpace(text) != "" {
				content = text
				break
			}
		}
	}
	if strings.TrimSpace(content) == "" {
		return ProviderChatResult{}, fmt.Errorf("plugin returned no text content")
	}

	return ProviderChatResult{
		SupplierID: provider.ID,
		Model:      model,
		Message: ProviderChatMessage{
			Role:    "assistant",
			Content: content,
		},
		StatusCode: status,
		DurationMS: duration,
	}, nil
}

func providerChatIngressPath(protocol constants.Protocol) string {
	switch protocol {
	case constants.ProtocolAnthropic:
		return "/v1/messages"
	case constants.ProtocolOpenAIChat:
		return "/v1/chat/completions"
	case constants.ProtocolOpenAIResponses:
		return "/v1/responses"
	default:
		return "/v1/responses"
	}
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
