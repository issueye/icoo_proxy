package translation

import (
	"encoding/json"
	"strings"
	"testing"

	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

func TestConvertRequestChatToAnthropicPreservesStream(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 2048,
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"stream":true,
		"messages":[
			{"role":"system","content":"You are helpful."},
			{"role":"user","content":"Hello"}
		],
		"max_tokens":128,
		"temperature":0.7
	}`)

	converted, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if got := payload["model"]; got != "claude-sonnet-4-20250514" {
		t.Fatalf("expected rewritten model, got %#v", got)
	}
	if got, ok := payload["stream"].(bool); !ok || !got {
		t.Fatalf("expected stream=true in anthropic payload, got %#v", payload["stream"])
	}
	if got := payload["system"]; got != "You are helpful." {
		t.Fatalf("expected system instructions to be preserved, got %#v", got)
	}
	if got := intValue(payload["max_tokens"]); got != 128 {
		t.Fatalf("expected max_tokens=128, got %d", got)
	}
	if got := payload["temperature"]; got != 0.7 {
		t.Fatalf("expected temperature=0.7, got %#v", got)
	}

	messages, ok := payload["messages"].([]interface{})
	if !ok || len(messages) != 1 {
		t.Fatalf("expected one non-system message, got %#v", payload["messages"])
	}
	message, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected message object, got %#v", messages[0])
	}
	if got := message["role"]; got != "user" {
		t.Fatalf("expected user role, got %#v", got)
	}
}

func TestConvertRequestChatToAnthropicUsesMaxCompletionTokens(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 2048,
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"stream":true,
		"messages":[
			{"role":"user","content":"Hello"}
		],
		"max_completion_tokens":256
	}`)

	converted, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if got := intValue(payload["max_tokens"]); got != 256 {
		t.Fatalf("expected max_tokens=256 from max_completion_tokens, got %d", got)
	}
}

func TestConvertRequestChatToAnthropicUsesRouteDefaultMaxTokens(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 8192,
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"stream":true,
		"messages":[
			{"role":"user","content":"Hello"}
		]
	}`)

	converted, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if got := intValue(payload["max_tokens"]); got != 8192 {
		t.Fatalf("expected default max_tokens=8192, got %d", got)
	}
}

func TestConvertRequestChatToAnthropicUsesGlobalDefaultMaxTokensWhenRouteMissing(t *testing.T) {
	route := models.Route{
		Upstream: consts.ProtocolAnthropic,
		Model:    "claude-sonnet-4-20250514",
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"messages":[
			{"role":"user","content":"Hello"}
		]
	}`)

	converted, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if got := intValue(payload["max_tokens"]); got != models.DefaultSupplierModelMaxTokens {
		t.Fatalf("expected global default max_tokens=%d, got %d", models.DefaultSupplierModelMaxTokens, got)
	}
}

func TestConvertRequestChatToAnthropicUsesConfiguredGlobalDefaultMaxTokensWhenRouteMissing(t *testing.T) {
	route := models.Route{
		Upstream: consts.ProtocolAnthropic,
		Model:    "claude-sonnet-4-20250514",
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"messages":[
			{"role":"user","content":"Hello"}
		]
	}`)

	const configuredGlobalDefault = 65536
	converted, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, configuredGlobalDefault)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if got := intValue(payload["max_tokens"]); got != configuredGlobalDefault {
		t.Fatalf("expected configured global default max_tokens=%d, got %d", configuredGlobalDefault, got)
	}
}

func TestConvertRequestChatToAnthropicRejectsZeroMaxTokens(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 8192,
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"stream":true,
		"messages":[
			{"role":"user","content":"Hello"}
		],
		"max_tokens":0
	}`)

	_, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, models.DefaultSupplierModelMaxTokens)
	if err == nil {
		t.Fatal("expected zero max_tokens error, got nil")
	}
	if !strings.Contains(err.Error(), "anthropic request requires max_tokens") {
		t.Fatalf("expected anthropic max_tokens error, got %v", err)
	}
}

func TestConvertRequestResponsesToAnthropicUsesRouteDefaultMaxTokens(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 4096,
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"input":[{"role":"user","content":"Hello"}]
	}`)

	converted, err := ConvertRequest(consts.ProtocolOpenAIResponses, route, body, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if got := intValue(payload["max_tokens"]); got != 4096 {
		t.Fatalf("expected route default max_tokens=4096, got %d", got)
	}
}

func TestConvertRequestChatToAnthropicDoesNotInjectStreamWhenDisabled(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 2048,
	}
	body := []byte(`{
		"model":"gpt-4.1",
		"messages":[
			{"role":"user","content":"Hello"}
		],
		"max_tokens":64
	}`)

	converted, err := ConvertRequest(consts.ProtocolOpenAIChat, route, body, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(converted, &payload); err != nil {
		t.Fatalf("failed to decode converted payload: %v", err)
	}

	if _, exists := payload["stream"]; exists {
		t.Fatalf("expected stream to be omitted for non-streaming request, got %#v", payload["stream"])
	}
	if got := payload["model"]; got != "claude-sonnet-4-20250514" {
		t.Fatalf("expected rewritten model, got %#v", got)
	}
	if got := intValue(payload["max_tokens"]); got != 64 {
		t.Fatalf("expected max_tokens=64, got %d", got)
	}
}

func TestConvertRequestChatToAnthropicRoundTripsAnthropicThinking(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 2048,
	}
	anthropicResponse := []byte(`{
		"id":"msg_thinking_1",
		"type":"message",
		"role":"assistant",
		"content":[
			{"type":"thinking","thinking":"step 1","signature":"sig_1"},
			{"type":"redacted_thinking","data":"blob_1"},
			{"type":"text","text":"Need tool"},
			{"type":"tool_use","id":"toolu_1","name":"search","input":{"q":"weather"}}
		],
		"usage":{"input_tokens":12,"output_tokens":8}
	}`)

	chatResponse, err := ConvertResponse(consts.ProtocolOpenAIChat, consts.ProtocolAnthropic, "deepseek-v4-pro", anthropicResponse)
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}

	var chatPayload map[string]interface{}
	if err := json.Unmarshal(chatResponse, &chatPayload); err != nil {
		t.Fatalf("failed to decode chat response: %v", err)
	}

	choices, ok := chatPayload["choices"].([]interface{})
	if !ok || len(choices) != 1 {
		t.Fatalf("expected one chat choice, got %#v", chatPayload["choices"])
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected choice object, got %#v", choices[0])
	}
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected assistant message object, got %#v", choice["message"])
	}

	chatRequestBody, err := json.Marshal(map[string]interface{}{
		"model": "deepseek-v4-pro",
		"messages": []interface{}{
			message,
			map[string]interface{}{
				"role":         "tool",
				"tool_call_id": "toolu_1",
				"content":      "{\"result\":\"sunny\"}",
			},
		},
		"max_tokens": 256,
	})
	if err != nil {
		t.Fatalf("failed to encode chat request: %v", err)
	}

	anthropicRequest, err := ConvertRequest(consts.ProtocolOpenAIChat, route, chatRequestBody, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var anthropicPayload map[string]interface{}
	if err := json.Unmarshal(anthropicRequest, &anthropicPayload); err != nil {
		t.Fatalf("failed to decode anthropic request: %v", err)
	}

	messages, ok := anthropicPayload["messages"].([]interface{})
	if !ok || len(messages) != 3 {
		t.Fatalf("expected three anthropic messages after round trip, got %#v", anthropicPayload["messages"])
	}

	assistantHistory, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first message object, got %#v", messages[0])
	}
	if got := assistantHistory["role"]; got != "assistant" {
		t.Fatalf("expected first message role assistant, got %#v", got)
	}
	assistantContent, ok := assistantHistory["content"].([]interface{})
	if !ok || len(assistantContent) != 3 {
		t.Fatalf("expected assistant thinking/text content blocks, got %#v", assistantHistory["content"])
	}
	firstPart, _ := assistantContent[0].(map[string]interface{})
	if got := firstPart["type"]; got != "thinking" {
		t.Fatalf("expected first assistant block thinking, got %#v", got)
	}
	if got := firstPart["thinking"]; got != "step 1" {
		t.Fatalf("expected thinking text to round trip, got %#v", got)
	}
	if got := firstPart["signature"]; got != "sig_1" {
		t.Fatalf("expected thinking signature to round trip, got %#v", got)
	}
	secondPart, _ := assistantContent[1].(map[string]interface{})
	if got := secondPart["type"]; got != "redacted_thinking" {
		t.Fatalf("expected second assistant block redacted_thinking, got %#v", got)
	}
	if got := secondPart["data"]; got != "blob_1" {
		t.Fatalf("expected redacted thinking data to round trip, got %#v", got)
	}
	thirdPart, _ := assistantContent[2].(map[string]interface{})
	if got := thirdPart["type"]; got != "text" {
		t.Fatalf("expected third assistant block text, got %#v", got)
	}
	if got := thirdPart["text"]; got != "Need tool" {
		t.Fatalf("expected assistant text to round trip, got %#v", got)
	}

	assistantToolUse, ok := messages[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected second message object, got %#v", messages[1])
	}
	toolContent, ok := assistantToolUse["content"].([]interface{})
	if !ok || len(toolContent) != 1 {
		t.Fatalf("expected tool_use content block, got %#v", assistantToolUse["content"])
	}
	toolUse, _ := toolContent[0].(map[string]interface{})
	if got := toolUse["type"]; got != "tool_use" {
		t.Fatalf("expected tool_use block, got %#v", got)
	}
	if got := toolUse["id"]; got != "toolu_1" {
		t.Fatalf("expected tool_use id toolu_1, got %#v", got)
	}
	if got := toolUse["name"]; got != "search" {
		t.Fatalf("expected tool_use name search, got %#v", got)
	}
	toolInput, ok := toolUse["input"].(map[string]interface{})
	if !ok || toolInput["q"] != "weather" {
		t.Fatalf("expected tool_use input to round trip, got %#v", toolUse["input"])
	}

	toolResultMessage, ok := messages[2].(map[string]interface{})
	if !ok {
		t.Fatalf("expected third message object, got %#v", messages[2])
	}
	if got := toolResultMessage["role"]; got != "user" {
		t.Fatalf("expected tool result message role user, got %#v", got)
	}
	toolResultContent, ok := toolResultMessage["content"].([]interface{})
	if !ok || len(toolResultContent) != 1 {
		t.Fatalf("expected one tool_result block, got %#v", toolResultMessage["content"])
	}
	toolResult, _ := toolResultContent[0].(map[string]interface{})
	if got := toolResult["type"]; got != "tool_result" {
		t.Fatalf("expected tool_result block, got %#v", got)
	}
	if got := toolResult["tool_use_id"]; got != "toolu_1" {
		t.Fatalf("expected tool_result tool_use_id toolu_1, got %#v", got)
	}
	if got := toolResult["content"]; got != "{\"result\":\"sunny\"}" {
		t.Fatalf("expected tool_result content to round trip, got %#v", got)
	}
}

func TestConvertRequestResponsesToAnthropicRoundTripsAnthropicThinking(t *testing.T) {
	route := models.Route{
		Upstream:         consts.ProtocolAnthropic,
		Model:            "claude-sonnet-4-20250514",
		DefaultMaxTokens: 2048,
	}
	anthropicResponse := []byte(`{
		"id":"msg_thinking_2",
		"type":"message",
		"role":"assistant",
		"content":[
			{"type":"thinking","thinking":"step 2","signature":"sig_2"},
			{"type":"redacted_thinking","data":"blob_2"},
			{"type":"text","text":"Use search"},
			{"type":"tool_use","id":"toolu_2","name":"search","input":{"q":"forecast"}}
		],
		"usage":{"input_tokens":10,"output_tokens":6}
	}`)

	responsesResponse, err := ConvertResponse(consts.ProtocolOpenAIResponses, consts.ProtocolAnthropic, "deepseek-v4-pro", anthropicResponse)
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}

	var responsesPayload map[string]interface{}
	if err := json.Unmarshal(responsesResponse, &responsesPayload); err != nil {
		t.Fatalf("failed to decode responses response: %v", err)
	}

	responsesRequestBody, err := json.Marshal(map[string]interface{}{
		"model":             "deepseek-v4-pro",
		"input":             responsesPayload["output"],
		"max_output_tokens": 256,
	})
	if err != nil {
		t.Fatalf("failed to encode responses request: %v", err)
	}

	anthropicRequest, err := ConvertRequest(consts.ProtocolOpenAIResponses, route, responsesRequestBody, models.DefaultSupplierModelMaxTokens)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	var anthropicPayload map[string]interface{}
	if err := json.Unmarshal(anthropicRequest, &anthropicPayload); err != nil {
		t.Fatalf("failed to decode anthropic request: %v", err)
	}

	messages, ok := anthropicPayload["messages"].([]interface{})
	if !ok || len(messages) != 2 {
		t.Fatalf("expected two anthropic messages after round trip, got %#v", anthropicPayload["messages"])
	}

	assistantHistory, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first message object, got %#v", messages[0])
	}
	assistantContent, ok := assistantHistory["content"].([]interface{})
	if !ok || len(assistantContent) != 3 {
		t.Fatalf("expected assistant thinking/text content blocks, got %#v", assistantHistory["content"])
	}
	firstPart, _ := assistantContent[0].(map[string]interface{})
	if got := firstPart["type"]; got != "thinking" || firstPart["thinking"] != "step 2" || firstPart["signature"] != "sig_2" {
		t.Fatalf("expected thinking block to round trip, got %#v", firstPart)
	}
	secondPart, _ := assistantContent[1].(map[string]interface{})
	if got := secondPart["type"]; got != "redacted_thinking" || secondPart["data"] != "blob_2" {
		t.Fatalf("expected redacted thinking block to round trip, got %#v", secondPart)
	}
	thirdPart, _ := assistantContent[2].(map[string]interface{})
	if got := thirdPart["type"]; got != "text" || thirdPart["text"] != "Use search" {
		t.Fatalf("expected text block to round trip, got %#v", thirdPart)
	}

	assistantToolUse, ok := messages[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected second message object, got %#v", messages[1])
	}
	toolContent, ok := assistantToolUse["content"].([]interface{})
	if !ok || len(toolContent) != 1 {
		t.Fatalf("expected one tool_use block, got %#v", assistantToolUse["content"])
	}
	toolUse, _ := toolContent[0].(map[string]interface{})
	if got := toolUse["type"]; got != "tool_use" || toolUse["id"] != "toolu_2" || toolUse["name"] != "search" {
		t.Fatalf("expected tool_use block to round trip, got %#v", toolUse)
	}
	toolInput, ok := toolUse["input"].(map[string]interface{})
	if !ok || toolInput["q"] != "forecast" {
		t.Fatalf("expected tool_use input to round trip, got %#v", toolUse["input"])
	}
}
