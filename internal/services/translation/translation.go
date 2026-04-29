package translation

import (
	"encoding/json"
	"fmt"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
	anthropicmodel "icoo_proxy/internal/models/anthropic"
	openaichatmodel "icoo_proxy/internal/models/openai_chat"
	openairesponsesmodel "icoo_proxy/internal/models/openai_responses"
	"strings"
	"time"
)

// ConvertRequest 根据下游协议和路由目标协议转换请求体。
// 同协议请求会保留原协议格式，仅将 model 改写为路由解析后的目标模型。
func ConvertRequest(downstream consts.Protocol, route models.Route, body []byte, globalDefaultMaxTokens int) ([]byte, error) {
	upstream := route.Upstream
	model := route.Model
	defaultMaxTokens := route.DefaultMaxTokens
	if defaultMaxTokens <= 0 {
		defaultMaxTokens = globalDefaultMaxTokens
	}
	if defaultMaxTokens <= 0 {
		defaultMaxTokens = models.DefaultSupplierModelMaxTokens
	}

	switch {
	// anthropic -> anthropic
	case downstream == consts.ProtocolAnthropic && upstream == consts.ProtocolAnthropic:
		return RewriteModel(body, model)
	// anthropic -> openai chat
	case downstream == consts.ProtocolAnthropic && upstream == consts.ProtocolOpenAIChat:
		return translateAnthropicToChatRequest(body, model)
	// anthropic -> openai responses
	case downstream == consts.ProtocolAnthropic && upstream == consts.ProtocolOpenAIResponses:
		return translateAnthropicToResponsesRequest(body, model)
	// openai chat -> anthropic
	case downstream == consts.ProtocolOpenAIChat && upstream == consts.ProtocolAnthropic:
		return translateChatToAnthropicRequest(body, model, defaultMaxTokens)
	// openai chat -> openai chat
	case downstream == consts.ProtocolOpenAIChat && upstream == consts.ProtocolOpenAIChat:
		return RewriteModel(body, model)
	// openai chat -> openai responses
	case downstream == consts.ProtocolOpenAIChat && upstream == consts.ProtocolOpenAIResponses:
		return translateChatToResponsesRequest(body, model)
	// openai responses -> anthropic
	case downstream == consts.ProtocolOpenAIResponses && upstream == consts.ProtocolAnthropic:
		return translateResponsesToAnthropicRequest(body, model, defaultMaxTokens)
	// openai responses -> openai chat
	case downstream == consts.ProtocolOpenAIResponses && upstream == consts.ProtocolOpenAIChat:
		return translateResponsesToChatRequest(body, model)
	// openai responses -> openai responses
	case downstream == consts.ProtocolOpenAIResponses && upstream == consts.ProtocolOpenAIResponses:
		return RewriteResponsesRequest(body, model)
	default:
		return nil, fmt.Errorf("request protocol conversion from %s to %s is not implemented", downstream, upstream)
	}
}

// ConvertResponse 根据上游协议和下游协议转换响应体。
// 同协议响应不改写协议格式，直接原样返回响应体。
func ConvertResponse(downstream, upstream consts.Protocol, model string, body []byte) ([]byte, error) {
	switch {
	// anthropic -> anthropic
	case upstream == consts.ProtocolAnthropic && downstream == consts.ProtocolAnthropic:
		return body, nil
	// anthropic -> openai chat
	case upstream == consts.ProtocolAnthropic && downstream == consts.ProtocolOpenAIChat:
		return translateAnthropicToChatResponse(body, model)
	// anthropic -> openai responses
	case upstream == consts.ProtocolAnthropic && downstream == consts.ProtocolOpenAIResponses:
		return translateAnthropicToResponsesResponse(body, model)
	// openai chat -> anthropic
	case upstream == consts.ProtocolOpenAIChat && downstream == consts.ProtocolAnthropic:
		return translateChatToAnthropicResponse(body, model)
	// openai chat -> openai chat
	case upstream == consts.ProtocolOpenAIChat && downstream == consts.ProtocolOpenAIChat:
		return body, nil
	// openai chat -> openai responses
	case upstream == consts.ProtocolOpenAIChat && downstream == consts.ProtocolOpenAIResponses:
		return translateChatToResponsesResponse(body, model)
	// openai responses -> anthropic
	case upstream == consts.ProtocolOpenAIResponses && downstream == consts.ProtocolAnthropic:
		return translateResponsesToAnthropicResponse(body, model)
	// openai responses -> openai chat
	case upstream == consts.ProtocolOpenAIResponses && downstream == consts.ProtocolOpenAIChat:
		return translateResponsesToChatResponse(body, model)
	// openai responses -> openai responses
	case upstream == consts.ProtocolOpenAIResponses && downstream == consts.ProtocolOpenAIResponses:
		return body, nil
	default:
		return nil, fmt.Errorf("response protocol conversion from %s to %s is not implemented", upstream, downstream)
	}
}

type chatReasoningPart struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      string `json:"data,omitempty"`
}

const defaultResponsesReasoningEffort = "medium"

type anthropicRequestEnvelope struct {
	Typed anthropicmodel.RequestMessagesRequest
}

type anthropicResponseEnvelope struct {
	Typed    anthropicmodel.ResponseBody
	HasUsage bool
}

type openAIChatRequestEnvelope struct {
	Typed               openaichatmodel.ReqeustBody
	MaxTokens           *int
	MaxCompletionTokens *int
}

type openAIChatResponseEnvelope struct {
	Typed    openaichatmodel.Response
	HasUsage bool
}

type openAIResponsesRequestEnvelope struct {
	Typed           openairesponsesmodel.RequestBody
	MaxOutputTokens *int
}

type openAIResponsesResponseEnvelope struct {
	Typed    openairesponsesmodel.ResponseBody
	HasUsage bool
}

func decodeAnthropicRequest(body []byte) (*anthropicRequestEnvelope, error) {
	var typed anthropicmodel.RequestMessagesRequest
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, fmt.Errorf("invalid json body")
	}
	return &anthropicRequestEnvelope{Typed: typed}, nil
}

func decodeAnthropicResponse(body []byte) (*anthropicResponseEnvelope, error) {
	var typed anthropicmodel.ResponseBody
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, fmt.Errorf("invalid upstream json body")
	}
	var presence struct {
		Usage *json.RawMessage `json:"usage,omitempty"`
	}
	_ = json.Unmarshal(body, &presence)
	return &anthropicResponseEnvelope{
		Typed:    typed,
		HasUsage: presence.Usage != nil,
	}, nil
}

func decodeOpenAIChatRequest(body []byte) (*openAIChatRequestEnvelope, error) {
	var typed openaichatmodel.ReqeustBody
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, fmt.Errorf("invalid json body")
	}
	var raw struct {
		MaxTokens           *int `json:"max_tokens,omitempty"`
		MaxCompletionTokens *int `json:"max_completion_tokens,omitempty"`
	}
	_ = json.Unmarshal(body, &raw)
	return &openAIChatRequestEnvelope{
		Typed:               typed,
		MaxTokens:           raw.MaxTokens,
		MaxCompletionTokens: raw.MaxCompletionTokens,
	}, nil
}

func decodeOpenAIChatResponse(body []byte) (*openAIChatResponseEnvelope, error) {
	var typed openaichatmodel.Response
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, fmt.Errorf("invalid upstream json body")
	}
	var presence struct {
		Usage *json.RawMessage `json:"usage,omitempty"`
	}
	_ = json.Unmarshal(body, &presence)
	return &openAIChatResponseEnvelope{
		Typed:    typed,
		HasUsage: presence.Usage != nil,
	}, nil
}

func decodeOpenAIResponsesRequest(body []byte) (*openAIResponsesRequestEnvelope, error) {
	var typed openairesponsesmodel.RequestBody
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, fmt.Errorf("invalid json body")
	}
	var raw struct {
		MaxOutputTokens *int `json:"max_output_tokens,omitempty"`
	}
	_ = json.Unmarshal(body, &raw)
	return &openAIResponsesRequestEnvelope{
		Typed:           typed,
		MaxOutputTokens: raw.MaxOutputTokens,
	}, nil
}

func decodeOpenAIResponsesResponse(body []byte) (*openAIResponsesResponseEnvelope, error) {
	var typed openairesponsesmodel.ResponseBody
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, fmt.Errorf("invalid upstream json body")
	}
	var presence struct {
		Usage *json.RawMessage `json:"usage,omitempty"`
	}
	_ = json.Unmarshal(body, &presence)
	return &openAIResponsesResponseEnvelope{
		Typed:    typed,
		HasUsage: presence.Usage != nil,
	}, nil
}

func responseRequestToMap(payload *openAIResponsesRequestEnvelope) map[string]interface{} {
	result := map[string]interface{}{}
	if payload.MaxOutputTokens != nil {
		result["max_output_tokens"] = *payload.MaxOutputTokens
	} else {
		appendIfNotEmpty(result, "max_output_tokens", payload.Typed.MaxOutputTokens)
	}
	return result
}

func responsesResponseToMap(payload *openAIResponsesResponseEnvelope) map[string]interface{} {
	result := map[string]interface{}{}
	appendIfNotEmpty(result, "id", payload.Typed.ID)
	appendIfNotEmpty(result, "status", payload.Typed.Status.ToString())
	if output := toJSONArray(payload.Typed.Output); len(output) > 0 {
		result["output"] = output
	}
	if payload.HasUsage {
		result["usage"] = responsesUsageToMap(payload.Typed.Usage)
	}
	return result
}

func toInterfacesFromObjectMaps(items []map[string]interface{}) []interface{} {
	if len(items) == 0 {
		return nil
	}
	result := make([]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func toJSONValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil
	}
	return decoded
}

func toJSONArray(value interface{}) []interface{} {
	items, _ := toJSONValue(value).([]interface{})
	return items
}

func toJSONObject(value interface{}) map[string]interface{} {
	object, _ := toJSONValue(value).(map[string]interface{})
	return object
}

func chatUsageToMap(usage openaichatmodel.ResponseUsage) map[string]interface{} {
	return toJSONObject(usage)
}

func responsesUsageToMap(usage openairesponsesmodel.ResponseUsage) map[string]interface{} {
	return toJSONObject(usage)
}

func anthropicUsageToMap(usage anthropicmodel.ResponseUsage) map[string]interface{} {
	return toJSONObject(usage)
}

func appendIfNotEmpty(target map[string]interface{}, key string, value interface{}) {
	switch v := value.(type) {
	case nil:
		return
	case string:
		if strings.TrimSpace(v) == "" {
			return
		}
	case int:
		if v == 0 {
			return
		}
	case int64:
		if v == 0 {
			return
		}
	case float64:
		if v == 0 {
			return
		}
	case bool:
		if !v {
			return
		}
	case []map[string]interface{}:
		if len(v) == 0 {
			return
		}
	case []interface{}:
		if len(v) == 0 {
			return
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return
		}
	}
	target[key] = value
}

func translateAnthropicToResponsesRequest(body []byte, model string) ([]byte, error) {
	payload, err := decodeAnthropicRequest(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	messages := toJSONArray(typed.Messages)
	request := map[string]interface{}{
		"model": model,
		"input": normalizeAnthropicMessages(messages),
	}
	if typed.Stream {
		request["stream"] = true
	}
	if system := anthropicSystemToInstructions(toJSONValue(typed.System)); system != "" {
		request["instructions"] = system
	}
	if typed.Temperature != nil {
		request["temperature"] = *typed.Temperature
	}
	if typed.TopP != nil {
		request["top_p"] = *typed.TopP
	}
	if typed.MaxTokens > 0 {
		request["max_output_tokens"] = typed.MaxTokens
	}
	if len(typed.Tools) > 0 {
		request["tools"] = anthropicToolsToResponsesTools(toJSONArray(typed.Tools))
	}
	ApplyDefaultResponsesReasoning(defaultResponsesReasoningEffort, request)
	return json.Marshal(request)
}

func translateResponsesToAnthropicRequest(body []byte, model string, defaultMaxTokens int) ([]byte, error) {
	payload, err := decodeOpenAIResponsesRequest(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	maxTokens, err := resolveAnthropicMaxTokens(responseRequestToMap(payload), defaultMaxTokens, "max_output_tokens")
	if err != nil {
		return nil, err
	}

	request := map[string]interface{}{
		"model":      model,
		"messages":   normalizeResponsesInput(toJSONValue(typed.Input)),
		"max_tokens": maxTokens,
	}
	if typed.Stream {
		request["stream"] = true
	}
	instructions := typed.Instructions
	if strings.TrimSpace(instructions) != "" {
		request["system"] = instructions
	}
	appendIfNotEmpty(request, "temperature", typed.Temperature)
	appendIfNotEmpty(request, "top_p", typed.TopP)
	if len(typed.Tools) > 0 {
		request["tools"] = responsesToolsToAnthropicTools(toJSONArray(typed.Tools))
	}
	return json.Marshal(request)
}

func translateAnthropicToChatRequest(body []byte, model string) ([]byte, error) {
	responsesBody, err := translateAnthropicToResponsesRequest(body, model)
	if err != nil {
		return nil, err
	}
	return translateResponsesToChatRequest(responsesBody, model)
}

func translateChatToAnthropicRequest(body []byte, model string, defaultMaxTokens int) ([]byte, error) {
	responsesBody, err := translateChatToResponsesRequest(body, model)
	if err != nil {
		return nil, err
	}
	return translateResponsesToAnthropicRequest(responsesBody, model, defaultMaxTokens)
}

func translateChatToResponsesRequest(body []byte, model string) ([]byte, error) {
	payload, err := decodeOpenAIChatRequest(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	messages := toJSONArray(typed.Messages)
	instructions := collectSystemMessages(messages)
	input := make([]map[string]interface{}, 0, len(messages))
	for _, raw := range messages {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if strings.EqualFold(role, "system") {
			continue
		}
		input = append(input, chatMessageToResponsesInput(msg)...)
	}

	request := map[string]interface{}{
		"model": model,
		"input": input,
	}
	if typed.Stream {
		request["stream"] = true
	}
	if instructions != "" {
		request["instructions"] = instructions
	}
	appendIfNotEmpty(request, "temperature", typed.Temperature)
	appendIfNotEmpty(request, "top_p", typed.TopP)
	if typed.ToolChoice != nil {
		request["tool_choice"] = toJSONValue(typed.ToolChoice)
	}
	if payload.MaxCompletionTokens != nil {
		request["max_output_tokens"] = *payload.MaxCompletionTokens
	} else if payload.MaxTokens != nil {
		request["max_output_tokens"] = *payload.MaxTokens
	} else if typed.MaxCompletionTokens > 0 {
		request["max_output_tokens"] = typed.MaxCompletionTokens
	} else if typed.MaxTokens > 0 {
		request["max_output_tokens"] = typed.MaxTokens
	}
	if len(typed.Tools) > 0 {
		request["tools"] = chatToolsToResponsesTools(toJSONArray(typed.Tools))
	}
	ApplyDefaultResponsesReasoning(defaultResponsesReasoningEffort, request)
	return json.Marshal(request)
}

func translateResponsesToChatRequest(body []byte, model string) ([]byte, error) {
	payload, err := decodeOpenAIResponsesRequest(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed
	if typed.Stream {
		return nil, fmt.Errorf("streaming cross protocol translation is not implemented yet")
	}

	messages := make([]map[string]interface{}, 0)
	instructions := typed.Instructions
	if strings.TrimSpace(instructions) != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": instructions,
		})
	}
	for _, item := range normalizeResponsesInputToChatMessages(toJSONValue(typed.Input)) {
		messages = append(messages, item)
	}

	request := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}
	appendIfNotEmpty(request, "temperature", typed.Temperature)
	appendIfNotEmpty(request, "top_p", typed.TopP)
	if typed.ToolChoice != nil {
		request["tool_choice"] = toJSONValue(typed.ToolChoice)
	}
	if typed.MaxOutputTokens > 0 {
		request["max_tokens"] = typed.MaxOutputTokens
	}
	if len(typed.Tools) > 0 {
		request["tools"] = responsesToolsToChatTools(toJSONArray(typed.Tools))
	}
	return json.Marshal(request)
}

func translateResponsesToChatResponse(body []byte, model string) ([]byte, error) {
	payload, err := decodeOpenAIResponsesResponse(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	payloadMap := responsesResponseToMap(payload)
	output := toJSONArray(typed.Output)
	content := extractResponsesText(payloadMap)
	toolCalls := extractResponsesFunctionCalls(output)
	reasoning := extractResponsesReasoningContent(output)
	message := map[string]interface{}{
		"role":    "assistant",
		"content": content,
	}
	if len(reasoning) > 0 {
		message["reasoning"] = reasoning
	}
	finishReason := mapResponsesStatusToFinishReason(payloadMap)
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
		finishReason = "tool_calls"
		if content == "" && len(reasoning) == 0 {
			message["content"] = nil
		}
	}
	response := map[string]interface{}{
		"id":      stringValue(typed.ID, "chatcmpl-proxy"),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
	}
	if payload.HasUsage {
		response["usage"] = mapUsageToChat(responsesUsageToMap(typed.Usage))
	}
	return json.Marshal(response)
}

func translateChatToResponsesResponse(body []byte, model string) ([]byte, error) {
	payload, err := decodeOpenAIChatResponse(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	choices := toJSONArray(typed.Choices)
	text := extractChatAssistantText(choices)
	output := make([]map[string]interface{}, 0)
	if text != "" {
		output = append(output, map[string]interface{}{
			"id":   "msg_proxy_1",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{"type": "output_text", "text": text},
			},
		})
	}
	output = append(output, chatChoicesToResponsesFunctionCalls(choices)...)
	status := "completed"
	if len(output) == 0 {
		output = append(output, map[string]interface{}{
			"id":      "msg_proxy_1",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]interface{}{{"type": "output_text", "text": ""}},
		})
	}
	response := map[string]interface{}{
		"id":     stringValue(typed.ID, "resp-proxy"),
		"object": "response",
		"model":  model,
		"status": status,
		"output": output,
	}
	if payload.HasUsage {
		response["usage"] = mapUsageToResponses(chatUsageToMap(typed.Usage))
	}
	return json.Marshal(response)
}

func translateAnthropicToChatResponse(body []byte, model string) ([]byte, error) {
	responsesBody, err := translateAnthropicToResponsesResponse(body, model)
	if err != nil {
		return nil, err
	}
	return translateResponsesToChatResponse(responsesBody, model)
}

func translateChatToAnthropicResponse(body []byte, model string) ([]byte, error) {
	responsesBody, err := translateChatToResponsesResponse(body, model)
	if err != nil {
		return nil, err
	}
	return translateResponsesToAnthropicResponse(responsesBody, model)
}

func translateResponsesToAnthropicResponse(body []byte, model string) ([]byte, error) {
	payload, err := decodeOpenAIResponsesResponse(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	payloadMap := responsesResponseToMap(payload)
	output := toJSONArray(typed.Output)
	text := extractResponsesText(payloadMap)
	content := make([]map[string]interface{}, 0)
	if text != "" {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": text,
		})
	}
	content = append(content, responsesOutputToAnthropicToolUse(output)...)
	stopReason := mapResponsesStatusToAnthropicStopReason(payloadMap)
	if len(content) == 0 {
		content = append(content, map[string]interface{}{"type": "text", "text": ""})
	}
	if hasAnthropicToolUse(content) {
		stopReason = "tool_use"
	}
	response := map[string]interface{}{
		"id":            stringValue(typed.ID, "msg_proxy"),
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"content":       content,
	}
	if payload.HasUsage {
		response["usage"] = mapUsageToAnthropic(responsesUsageToMap(typed.Usage))
	}
	return json.Marshal(response)
}

func translateAnthropicToResponsesResponse(body []byte, model string) ([]byte, error) {
	payload, err := decodeAnthropicResponse(body)
	if err != nil {
		return nil, err
	}
	typed := payload.Typed

	contentItems := toJSONArray(typed.Content)
	text := extractAnthropicText(contentItems)
	reasoning := extractAnthropicReasoningContent(contentItems)
	output := make([]map[string]interface{}, 0)
	if text != "" || len(reasoning) > 0 {
		message := map[string]interface{}{
			"id":   "msg_proxy_1",
			"type": "message",
			"role": "assistant",
		}
		if text != "" {
			message["content"] = []map[string]interface{}{{"type": "output_text", "text": text}}
		}
		if len(reasoning) > 0 {
			message["reasoning"] = reasoning
		}
		if _, ok := message["content"]; !ok {
			message["content"] = []map[string]interface{}{}
		}
		output = append(output, message)
	}
	output = append(output, anthropicContentToResponsesFunctionCalls(contentItems)...)
	if len(output) == 0 {
		output = append(output, map[string]interface{}{
			"id":      "msg_proxy_1",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]interface{}{{"type": "output_text", "text": ""}},
		})
	}
	response := map[string]interface{}{
		"id":     stringValue(typed.ID, "resp-proxy"),
		"object": "response",
		"model":  model,
		"status": "completed",
		"output": output,
	}
	if payload.HasUsage {
		response["usage"] = mapAnthropicUsageToResponses(anthropicUsageToMap(typed.Usage))
	}
	return json.Marshal(response)
}

func collectSystemMessages(messages []interface{}) string {
	parts := make([]string, 0)
	for _, raw := range messages {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if !strings.EqualFold(role, "system") {
			continue
		}
		parts = append(parts, contentToText(msg["content"]))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func normalizeResponsesInput(raw interface{}) []map[string]interface{} {
	switch value := raw.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return nil
		}
		return []map[string]interface{}{{
			"role":    "user",
			"content": text,
		}}
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(value))
		for _, rawItem := range value {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			itemType, _ := item["type"].(string)
			switch itemType {
			case "function_call_output":
				items = append(items, map[string]interface{}{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"type":        "tool_result",
							"tool_use_id": stringValue(item["call_id"], ""),
							"content":     responseToolOutputToText(item["output"]),
						},
					},
				})
				continue
			case "function_call":
				items = append(items, map[string]interface{}{
					"role": "assistant",
					"content": []map[string]interface{}{
						{
							"type":  "tool_use",
							"id":    stringValue(item["call_id"], ""),
							"name":  stringValue(item["name"], ""),
							"input": parseJSONStringObject(stringValue(item["arguments"], "{}")),
						},
					},
				})
				continue
			}
			role, _ := item["role"].(string)
			if role == "" {
				role = "user"
			}
			if role == "assistant" {
				reasoning := extractAssistantReasoningFromMessage(item)
				if len(reasoning) > 0 {
					items = append(items, map[string]interface{}{
						"role":    role,
						"content": buildAnthropicAssistantContent(item["content"], reasoning),
					})
					continue
				}
			}
			items = append(items, map[string]interface{}{
				"role":    role,
				"content": normalizeMessageContent(item["content"]),
			})
		}
		return items
	default:
		return nil
	}
}

func normalizeResponsesInputToChatMessages(raw interface{}) []map[string]interface{} {
	switch value := raw.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return nil
		}
		return []map[string]interface{}{{"role": "user", "content": text}}
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(value))
		for _, rawItem := range value {
			item, ok := rawItem.(map[string]interface{})
			if !ok {
				continue
			}
			itemType, _ := item["type"].(string)
			switch itemType {
			case "function_call_output":
				items = append(items, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": stringValue(item["call_id"], ""),
					"content":      responseToolOutputToText(item["output"]),
				})
			case "function_call":
				items = append(items, map[string]interface{}{
					"role":    "assistant",
					"content": nil,
					"tool_calls": []map[string]interface{}{
						{
							"id":   stringValue(item["call_id"], ""),
							"type": "function",
							"function": map[string]interface{}{
								"name":      stringValue(item["name"], ""),
								"arguments": stringValue(item["arguments"], "{}"),
							},
						},
					},
				})
			default:
				role, _ := item["role"].(string)
				if role == "" {
					role = "user"
				}
				message := map[string]interface{}{
					"role":    role,
					"content": normalizeMessageContent(item["content"]),
				}
				if role == "assistant" {
					if reasoning := extractResponsesReasoningContent([]interface{}{item}); len(reasoning) > 0 {
						message["reasoning"] = reasoning
					}
				}
				items = append(items, message)
			}
		}
		return items
	default:
		return nil
	}
}

func chatMessageToResponsesInput(msg map[string]interface{}) []map[string]interface{} {
	role, _ := msg["role"].(string)
	if role == "" {
		role = "user"
	}
	if role == "tool" {
		return []map[string]interface{}{{
			"type":    "function_call_output",
			"call_id": stringValue(msg["tool_call_id"], ""),
			"output":  contentToText(msg["content"]),
		}}
	}

	items := make([]map[string]interface{}, 0)
	content := normalizeMessageContent(msg["content"])
	assistantContentHandled := false
	if role == "assistant" {
		reasoning := normalizeChatReasoning(msg["reasoning"])
		if len(reasoning) > 0 {
			assistantMessage := map[string]interface{}{
				"role":      role,
				"content":   content,
				"reasoning": reasoning,
			}
			if content == nil {
				assistantMessage["content"] = ""
			}
			items = append(items, assistantMessage)
			assistantContentHandled = true
		}
	}
	if !assistantContentHandled {
		if content := contentToText(content); content != "" {
			items = append(items, map[string]interface{}{
				"role":    role,
				"content": content,
			})
		}
	}
	if role == "assistant" {
		toolCalls, _ := msg["tool_calls"].([]interface{})
		for _, rawToolCall := range toolCalls {
			toolCall, ok := rawToolCall.(map[string]interface{})
			if !ok {
				continue
			}
			function, _ := toolCall["function"].(map[string]interface{})
			items = append(items, map[string]interface{}{
				"type":      "function_call",
				"call_id":   stringValue(toolCall["id"], ""),
				"name":      stringValue(function["name"], ""),
				"arguments": stringValue(function["arguments"], "{}"),
			})
		}
	}
	if len(items) == 0 {
		items = append(items, map[string]interface{}{
			"role":    role,
			"content": content,
		})
	}
	return items
}

func normalizeAnthropicMessages(raw []interface{}) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(raw))
	for _, rawItem := range raw {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := item["role"].(string)
		if role == "" {
			role = "user"
		}
		content, _ := item["content"].([]interface{})
		if role == "assistant" {
			items = append(items, anthropicAssistantContentToResponsesInput(content)...)
			continue
		}
		items = append(items, anthropicUserContentToResponsesInput(content, role)...)
	}
	return items
}

func anthropicSystemToInstructions(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case []interface{}:
		parts := make([]string, 0, len(value))
		for _, rawPart := range value {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func normalizeMessageContent(raw interface{}) interface{} {
	switch value := raw.(type) {
	case string:
		return value
	case []interface{}:
		textParts := make([]string, 0, len(value))
		for _, part := range value {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := partMap["type"].(string)
			switch partType {
			case "text", "input_text", "output_text":
				if text, ok := partMap["text"].(string); ok && text != "" {
					textParts = append(textParts, text)
				}
			}
		}
		if len(textParts) == 1 {
			return textParts[0]
		}
		if len(textParts) > 1 {
			return strings.Join(textParts, "\n")
		}
	}
	return raw
}

func anthropicAssistantContentToResponsesInput(content []interface{}) []map[string]interface{} {
	items := make([]map[string]interface{}, 0)
	textParts := make([]string, 0)
	reasoning := make([]map[string]interface{}, 0)
	for _, rawPart := range content {
		part, ok := rawPart.(map[string]interface{})
		if !ok {
			continue
		}
		partType, _ := part["type"].(string)
		switch partType {
		case "text":
			if text, ok := part["text"].(string); ok && text != "" {
				textParts = append(textParts, text)
			}
		case "thinking":
			reasoning = append(reasoning, map[string]interface{}{
				"type":      "thinking",
				"thinking":  stringValue(part["thinking"], ""),
				"signature": stringValue(part["signature"], ""),
			})
		case "redacted_thinking":
			reasoning = append(reasoning, map[string]interface{}{
				"type": "redacted_thinking",
				"data": stringValue(part["data"], ""),
			})
		case "tool_use":
			items = append(items, map[string]interface{}{
				"type":      "function_call",
				"call_id":   stringValue(part["id"], ""),
				"name":      stringValue(part["name"], ""),
				"arguments": marshalToJSONString(part["input"]),
			})
		}
	}
	if len(textParts) > 0 || len(reasoning) > 0 {
		message := map[string]interface{}{
			"role":    "assistant",
			"content": strings.Join(textParts, "\n"),
		}
		if len(reasoning) > 0 {
			message["reasoning"] = reasoning
		}
		items = append([]map[string]interface{}{message}, items...)
	}
	return items
}

func anthropicUserContentToResponsesInput(content []interface{}, role string) []map[string]interface{} {
	items := make([]map[string]interface{}, 0)
	textParts := make([]string, 0)
	for _, rawPart := range content {
		part, ok := rawPart.(map[string]interface{})
		if !ok {
			continue
		}
		partType, _ := part["type"].(string)
		switch partType {
		case "text":
			if text, ok := part["text"].(string); ok && text != "" {
				textParts = append(textParts, text)
			}
		case "tool_result":
			items = append(items, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": stringValue(part["tool_use_id"], ""),
				"output":  anthropicToolResultContentToOutput(part["content"]),
			})
		}
	}
	if len(textParts) > 0 {
		items = append([]map[string]interface{}{{
			"role":    role,
			"content": strings.Join(textParts, "\n"),
		}}, items...)
	}
	return items
}

func contentToText(raw interface{}) string {
	switch value := normalizeMessageContent(raw).(type) {
	case string:
		return value
	default:
		return ""
	}
}

func extractResponsesOutputText(raw interface{}) string {
	items, _ := raw.([]interface{})
	parts := make([]string, 0)
	for _, item := range items {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		content, _ := msg["content"].([]interface{})
		for _, rawPart := range content {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := part["type"].(string)
			if partType == "output_text" || partType == "text" {
				if text, ok := part["text"].(string); ok && text != "" {
					parts = append(parts, text)
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

func extractResponsesText(payload map[string]interface{}) string {
	if text := strings.TrimSpace(extractResponsesOutputText(payload["output"])); text != "" {
		return text
	}
	return strings.TrimSpace(stringValue(payload["output_text"], ""))
}

func extractResponsesFunctionCalls(raw interface{}) []map[string]interface{} {
	items, _ := raw.([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		if stringValue(item["type"], "") != "function_call" {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":   stringValue(item["call_id"], ""),
			"type": "function",
			"function": map[string]interface{}{
				"name":      stringValue(item["name"], ""),
				"arguments": stringValue(item["arguments"], "{}"),
			},
		})
	}
	return result
}

func responsesOutputToAnthropicToolUse(raw interface{}) []map[string]interface{} {
	items, _ := raw.([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		if stringValue(item["type"], "") != "function_call" {
			continue
		}
		result = append(result, map[string]interface{}{
			"type":  "tool_use",
			"id":    stringValue(item["call_id"], ""),
			"name":  stringValue(item["name"], ""),
			"input": parseJSONStringObject(stringValue(item["arguments"], "{}")),
		})
	}
	return result
}

func anthropicContentToResponsesFunctionCalls(raw interface{}) []map[string]interface{} {
	parts, _ := raw.([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]interface{})
		if !ok {
			continue
		}
		if stringValue(part["type"], "") != "tool_use" {
			continue
		}
		result = append(result, map[string]interface{}{
			"type":      "function_call",
			"call_id":   stringValue(part["id"], ""),
			"name":      stringValue(part["name"], ""),
			"arguments": marshalToJSONString(part["input"]),
		})
	}
	return result
}

func extractResponsesReasoningContent(raw interface{}) []map[string]interface{} {
	items, _ := raw.([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		reasoning, _ := item["reasoning"].([]interface{})
		for _, rawPart := range reasoning {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			partType := stringValue(part["type"], "")
			switch partType {
			case "thinking":
				result = append(result, map[string]interface{}{
					"type":      "thinking",
					"thinking":  stringValue(part["thinking"], ""),
					"signature": stringValue(part["signature"], ""),
				})
			case "redacted_thinking":
				result = append(result, map[string]interface{}{
					"type": "redacted_thinking",
					"data": stringValue(part["data"], ""),
				})
			}
		}
	}
	return result
}

func extractAnthropicReasoningContent(raw interface{}) []map[string]interface{} {
	parts, _ := raw.([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]interface{})
		if !ok {
			continue
		}
		switch stringValue(part["type"], "") {
		case "thinking":
			result = append(result, map[string]interface{}{
				"type":      "thinking",
				"thinking":  stringValue(part["thinking"], ""),
				"signature": stringValue(part["signature"], ""),
			})
		case "redacted_thinking":
			result = append(result, map[string]interface{}{
				"type": "redacted_thinking",
				"data": stringValue(part["data"], ""),
			})
		}
	}
	return result
}

func extractAssistantReasoningFromMessage(message map[string]interface{}) []chatReasoningPart {
	rawParts, _ := message["reasoning"].([]interface{})
	parts := make([]chatReasoningPart, 0, len(rawParts))
	for _, rawPart := range rawParts {
		part, ok := rawPart.(map[string]interface{})
		if !ok {
			continue
		}
		typeName := stringValue(part["type"], "")
		switch typeName {
		case "thinking":
			parts = append(parts, chatReasoningPart{
				Type:      "thinking",
				Thinking:  stringValue(part["thinking"], ""),
				Signature: stringValue(part["signature"], ""),
			})
		case "redacted_thinking":
			parts = append(parts, chatReasoningPart{
				Type: "redacted_thinking",
				Data: stringValue(part["data"], ""),
			})
		}
	}
	return parts
}

func normalizeChatReasoning(raw interface{}) []chatReasoningPart {
	switch value := raw.(type) {
	case []chatReasoningPart:
		return value
	case []interface{}:
		parts := make([]chatReasoningPart, 0, len(value))
		for _, rawPart := range value {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			typeName := stringValue(part["type"], "")
			switch typeName {
			case "thinking":
				parts = append(parts, chatReasoningPart{
					Type:      "thinking",
					Thinking:  stringValue(part["thinking"], ""),
					Signature: stringValue(part["signature"], ""),
				})
			case "redacted_thinking":
				parts = append(parts, chatReasoningPart{
					Type: "redacted_thinking",
					Data: stringValue(part["data"], ""),
				})
			}
		}
		return parts
	default:
		return nil
	}
}

func buildAnthropicAssistantContent(rawContent interface{}, reasoning []chatReasoningPart) []map[string]interface{} {
	content := make([]map[string]interface{}, 0, len(reasoning)+1)
	for _, part := range reasoning {
		switch part.Type {
		case "thinking":
			content = append(content, map[string]interface{}{
				"type":      "thinking",
				"thinking":  part.Thinking,
				"signature": part.Signature,
			})
		case "redacted_thinking":
			content = append(content, map[string]interface{}{
				"type": "redacted_thinking",
				"data": part.Data,
			})
		}
	}
	if text := contentToText(rawContent); text != "" {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": text,
		})
		return content
	}
	if normalized, ok := normalizeMessageContent(rawContent).([]interface{}); ok {
		for _, rawPart := range normalized {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			if stringValue(part["type"], "") == "text" || stringValue(part["type"], "") == "input_text" || stringValue(part["type"], "") == "output_text" {
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": stringValue(part["text"], ""),
				})
			}
		}
	}
	return content
}

func appendAnthropicContentPart(message map[string]interface{}, part map[string]interface{}) {
	if message == nil || part == nil {
		return
	}
	existing, _ := message["content"].([]map[string]interface{})
	existing = append(existing, part)
	message["content"] = existing
}

func extractAnthropicText(raw interface{}) string {
	parts, _ := raw.([]interface{})
	texts := make([]string, 0, len(parts))
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]interface{})
		if !ok {
			continue
		}
		partType, _ := part["type"].(string)
		if partType != "text" {
			continue
		}
		if text, ok := part["text"].(string); ok && text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}

func extractChatAssistantText(raw interface{}) string {
	choices, _ := raw.([]interface{})
	for _, choice := range choices {
		item, ok := choice.(map[string]interface{})
		if !ok {
			continue
		}
		message, _ := item["message"].(map[string]interface{})
		if message == nil {
			continue
		}
		return contentToText(message["content"])
	}
	return ""
}

func chatChoicesToResponsesFunctionCalls(raw interface{}) []map[string]interface{} {
	choices, _ := raw.([]interface{})
	result := make([]map[string]interface{}, 0)
	for _, choice := range choices {
		item, ok := choice.(map[string]interface{})
		if !ok {
			continue
		}
		message, _ := item["message"].(map[string]interface{})
		if message == nil {
			continue
		}
		toolCalls, _ := message["tool_calls"].([]interface{})
		for _, rawToolCall := range toolCalls {
			toolCall, ok := rawToolCall.(map[string]interface{})
			if !ok {
				continue
			}
			function, _ := toolCall["function"].(map[string]interface{})
			result = append(result, map[string]interface{}{
				"type":      "function_call",
				"call_id":   stringValue(toolCall["id"], ""),
				"name":      stringValue(function["name"], ""),
				"arguments": stringValue(function["arguments"], "{}"),
			})
		}
	}
	return result
}

func mapResponsesStatusToFinishReason(payload map[string]interface{}) string {
	if status, _ := payload["status"].(string); status == "completed" {
		return "stop"
	}
	return "length"
}

func mapResponsesStatusToAnthropicStopReason(payload map[string]interface{}) string {
	if status, _ := payload["status"].(string); status == "completed" {
		return "end_turn"
	}
	return "max_tokens"
}

func mapUsageToChat(raw interface{}) map[string]interface{} {
	usageMap, _ := raw.(map[string]interface{})
	return map[string]interface{}{
		"prompt_tokens":     intValue(usageMap["input_tokens"]),
		"completion_tokens": intValue(usageMap["output_tokens"]),
		"total_tokens":      intValue(usageMap["total_tokens"]),
	}
}

func mapUsageToResponses(raw interface{}) map[string]interface{} {
	usageMap, _ := raw.(map[string]interface{})
	prompt := intValue(usageMap["prompt_tokens"])
	completion := intValue(usageMap["completion_tokens"])
	total := intValue(usageMap["total_tokens"])
	if total == 0 {
		total = prompt + completion
	}
	return map[string]interface{}{
		"input_tokens":  prompt,
		"output_tokens": completion,
		"total_tokens":  total,
	}
}

func mapUsageToAnthropic(raw interface{}) map[string]interface{} {
	usageMap, _ := raw.(map[string]interface{})
	return map[string]interface{}{
		"input_tokens":  intValue(usageMap["input_tokens"]),
		"output_tokens": intValue(usageMap["output_tokens"]),
	}
}

func mapAnthropicUsageToResponses(raw interface{}) map[string]interface{} {
	usageMap, _ := raw.(map[string]interface{})
	input := intValue(usageMap["input_tokens"])
	output := intValue(usageMap["output_tokens"])
	return map[string]interface{}{
		"input_tokens":  input,
		"output_tokens": output,
		"total_tokens":  input + output,
	}
}

func intValue(raw interface{}) int {
	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func positiveIntField(payload map[string]interface{}, key string) (int, bool) {
	value, ok := payload[key]
	if !ok {
		return 0, false
	}
	parsed := intValue(value)
	if parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func resolveAnthropicMaxTokens(payload map[string]interface{}, defaultMaxTokens int, keys ...string) (int, error) {
	for _, key := range keys {
		if _, exists := payload[key]; !exists {
			continue
		}
		maxTokens, ok := positiveIntField(payload, key)
		if !ok {
			return 0, fmt.Errorf("anthropic request requires max_tokens (or max_completion_tokens) to be a positive integer")
		}
		return maxTokens, nil
	}
	if defaultMaxTokens <= 0 {
		defaultMaxTokens = models.DefaultSupplierModelMaxTokens
	}
	return defaultMaxTokens, nil
}

func stringValue(raw interface{}, fallback string) string {
	if value, ok := raw.(string); ok && value != "" {
		return value
	}
	return fallback
}

func chatToolsToResponsesTools(raw interface{}) []map[string]interface{} {
	tools, _ := raw.([]interface{})
	items := make([]map[string]interface{}, 0, len(tools))
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]interface{})
		if !ok {
			continue
		}
		if fn, ok := tool["function"].(map[string]interface{}); ok {
			items = append(items, map[string]interface{}{
				"type":        "function",
				"name":        stringValue(fn["name"], ""),
				"description": stringValue(fn["description"], ""),
				"parameters":  objectValue(fn["parameters"]),
			})
			continue
		}
		if stringValue(tool["type"], "") == "function" {
			items = append(items, map[string]interface{}{
				"type":        "function",
				"name":        stringValue(tool["name"], ""),
				"description": stringValue(tool["description"], ""),
				"parameters":  objectValue(tool["parameters"]),
			})
		}
	}
	return items
}

func responsesToolsToChatTools(raw interface{}) []map[string]interface{} {
	tools, _ := raw.([]interface{})
	items := make([]map[string]interface{}, 0, len(tools))
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]interface{})
		if !ok {
			continue
		}
		if stringValue(tool["type"], "") != "function" {
			continue
		}
		items = append(items, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        stringValue(tool["name"], ""),
				"description": stringValue(tool["description"], ""),
				"parameters":  objectValue(firstNonNil(tool["parameters"], tool["input_schema"])),
			},
		})
	}
	return items
}

func anthropicToolsToResponsesTools(raw interface{}) []map[string]interface{} {
	tools, _ := raw.([]interface{})
	items := make([]map[string]interface{}, 0, len(tools))
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, map[string]interface{}{
			"type":        "function",
			"name":        stringValue(tool["name"], ""),
			"description": stringValue(tool["description"], ""),
			"parameters":  objectValue(tool["input_schema"]),
		})
	}
	return items
}

func responsesToolsToAnthropicTools(raw interface{}) []map[string]interface{} {
	tools, _ := raw.([]interface{})
	items := make([]map[string]interface{}, 0, len(tools))
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]interface{})
		if !ok {
			continue
		}
		if stringValue(tool["type"], "") != "function" {
			continue
		}
		items = append(items, map[string]interface{}{
			"name":         stringValue(tool["name"], ""),
			"description":  stringValue(tool["description"], ""),
			"input_schema": objectValue(firstNonNil(tool["parameters"], tool["input_schema"])),
		})
	}
	return items
}

func responseToolOutputToText(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return value
	case []interface{}:
		parts := make([]string, 0, len(value))
		for _, rawPart := range value {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func anthropicToolResultContentToOutput(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return value
	case []interface{}:
		parts := make([]string, 0, len(value))
		for _, rawPart := range value {
			part, ok := rawPart.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func marshalToJSONString(raw interface{}) string {
	if raw == nil {
		return "{}"
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func parseJSONStringObject(raw string) map[string]interface{} {
	if strings.TrimSpace(raw) == "" {
		return map[string]interface{}{}
	}
	var value map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]interface{}{}
	}
	return value
}

func objectValue(raw interface{}) map[string]interface{} {
	if value, ok := raw.(map[string]interface{}); ok {
		return value
	}
	return map[string]interface{}{}
}

func firstNonNil(values ...interface{}) interface{} {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func hasAnthropicToolUse(content []map[string]interface{}) bool {
	for _, part := range content {
		if stringValue(part["type"], "") == "tool_use" {
			return true
		}
	}
	return false
}
