package ai_llm_proxy

import (
	"encoding/json"
	"strings"
	"testing"

	"icoo_llm_bridge/internal/constants"
)

func TestProtocolConverterChatRequestToResponses(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"model":"chat-model","messages":[{"role":"user","content":"hello"}]}`)

	out, err := converter.ConvertRequest(RequestInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      "target-model",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload["model"] != "target-model" {
		t.Fatalf("model = %v", payload["model"])
	}
	if _, ok := payload["input"]; !ok {
		t.Fatalf("expected responses input in payload: %s", string(out))
	}
}

func TestProtocolConverterAnthropicRequestToResponses(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"model":"claude","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`)

	out, err := converter.ConvertRequest(RequestInput{
		Downstream: constants.ProtocolAnthropic,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      "target-model",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload["model"] != "target-model" {
		t.Fatalf("model = %v", payload["model"])
	}
	if _, ok := payload["input"]; !ok {
		t.Fatalf("expected responses input in payload: %s", string(out))
	}
}

func TestProtocolConverterChatRequestToAnthropic(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"model":"chat-model","max_tokens":64,"messages":[{"role":"user","content":"hello"}]}`)

	out, err := converter.ConvertRequest(RequestInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolAnthropic,
		Model:      "target-claude",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}
	var payload AnthropicRequest
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload.Model != "target-claude" {
		t.Fatalf("model = %v", payload.Model)
	}
	if payload.MaxTokens != minMaxOutputTokens {
		t.Fatalf("max_tokens = %d", payload.MaxTokens)
	}
	if len(payload.Messages) != 1 || payload.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", payload.Messages)
	}
}

func TestProtocolConverterResponsesResponseToChat(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"id":"resp_1","object":"response","model":"gpt","status":"completed","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}]}`)

	out, err := converter.ConvertResponse(ResponseInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      "target-model",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload["object"] != "chat.completion" {
		t.Fatalf("object = %v, body = %s", payload["object"], string(out))
	}
}

func TestProtocolConverterAnthropicResponseToChat(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[{"type":"text","text":"hello"}],"stop_reason":"end_turn","usage":{"input_tokens":2,"output_tokens":3}}`)

	out, err := converter.ConvertResponse(ResponseInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolAnthropic,
		Model:      "target-claude",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}
	var payload ChatCompletionsResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload.Object != "chat.completion" {
		t.Fatalf("object = %v, body = %s", payload.Object, string(out))
	}
	if len(payload.Choices) != 1 {
		t.Fatalf("choices = %+v", payload.Choices)
	}
	var content string
	if err := json.Unmarshal(payload.Choices[0].Message.Content, &content); err != nil {
		t.Fatalf("unmarshal message content: %v", err)
	}
	if content != "hello" {
		t.Fatalf("content = %q", content)
	}
	if payload.Usage == nil || payload.Usage.PromptTokens != 2 || payload.Usage.CompletionTokens != 3 {
		t.Fatalf("usage = %+v", payload.Usage)
	}
}

func TestProtocolConverterResponsesStreamToChat(t *testing.T) {
	converter := NewProtocolConverter()
	input := strings.Join([]string{
		`event: response.created`,
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt","status":"in_progress"}}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hello"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt","status":"completed"}}`,
		``,
	}, "\n")
	var out strings.Builder

	_, err := converter.ConvertStream(StreamInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      "target-model",
		Reader:     strings.NewReader(input),
		Writer:     &out,
	})
	if err != nil {
		t.Fatalf("ConvertStream returned error: %v", err)
	}
	if !strings.Contains(out.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("expected chat chunks, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "data: [DONE]") {
		t.Fatalf("expected done marker, got: %s", out.String())
	}
}

func TestProtocolConverterAnthropicStreamToChat(t *testing.T) {
	converter := NewProtocolConverter()
	input := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[],"usage":{"input_tokens":2,"output_tokens":0}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")
	var out strings.Builder

	result, err := converter.ConvertStream(StreamInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolAnthropic,
		Model:      "target-claude",
		Reader:     strings.NewReader(input),
		Writer:     &out,
	})
	if err != nil {
		t.Fatalf("ConvertStream returned error: %v", err)
	}
	if !strings.Contains(out.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("expected chat chunks, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `"content":"hello"`) {
		t.Fatalf("expected text delta, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "data: [DONE]") {
		t.Fatalf("expected done marker, got: %s", out.String())
	}
	if result.Usage.InputTokens != 2 || result.Usage.OutputTokens != 3 || result.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %+v", result.Usage)
	}
}
