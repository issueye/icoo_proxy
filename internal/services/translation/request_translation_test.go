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
