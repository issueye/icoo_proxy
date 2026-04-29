package translation

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

type chatStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string `json:"role,omitempty"`
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func TestTranslateResponsesStreamToChat_TextDeltaAndDone(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("response.created", map[string]any{
			"type": "response.created",
			"response": map[string]any{
				"id": "resp_123",
				"usage": map[string]any{
					"input_tokens":  10,
					"output_tokens": 2,
					"total_tokens":  12,
				},
			},
		}),
		sseJSONEvent("response.output_text.delta", map[string]any{
			"type":          "response.output_text.delta",
			"item_id":       "msg_1",
			"output_index":  0,
			"content_index": 0,
			"delta":         "Hel",
		}),
		sseJSONEvent("response.output_text.done", map[string]any{
			"type":          "response.output_text.done",
			"item_id":       "msg_1",
			"output_index":  0,
			"content_index": 0,
			"text":          "Hello",
		}),
		sseJSONEvent("response.completed", map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"id":     "resp_123",
				"status": "completed",
				"usage": map[string]any{
					"input_tokens":  10,
					"output_tokens": 5,
					"total_tokens":  15,
				},
			},
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateResponsesStreamToChat(recorder, strings.NewReader(body), "gpt-5.4", "req-1", slog.Default())
	if err != nil {
		t.Fatalf("TranslateResponsesStreamToChat returned error: %v", err)
	}
	if usage.InputTokens != 10 || usage.OutputTokens != 5 || usage.TotalTokens != 15 {
		t.Fatalf("expected usage 10/5/15, got %#v", usage)
	}

	frames := parseSSEDataFrames(t, recorder.Body.String())
	if len(frames) != 4 {
		t.Fatalf("expected 4 SSE data frames, got %d: %v", len(frames), frames)
	}

	roleChunk := decodeChatChunk(t, frames[0])
	if got := roleChunk.Choices[0].Delta.Role; got != "assistant" {
		t.Fatalf("expected first chunk role assistant, got %q", got)
	}

	textChunk := decodeChatChunk(t, frames[1])
	if got := textChunk.Choices[0].Delta.Content; got != "Hel" {
		t.Fatalf("expected text delta Hel, got %q", got)
	}

	finishChunk := decodeChatChunk(t, frames[2])
	if finishChunk.Choices[0].FinishReason == nil || *finishChunk.Choices[0].FinishReason != "stop" {
		t.Fatalf("expected finish_reason stop, got %#v", finishChunk.Choices[0].FinishReason)
	}
	if got := finishChunk.Model; got != "gpt-5.4" {
		t.Fatalf("expected model gpt-5.4, got %q", got)
	}
	if got := finishChunk.ID; got != "resp_123" {
		t.Fatalf("expected response id resp_123, got %q", got)
	}
	if got := finishChunk.Choices[0].Delta.Content; got != "" {
		t.Fatalf("expected finish chunk without duplicated content, got %q", got)
	}

	if frames[3] != "[DONE]" {
		t.Fatalf("expected final [DONE] frame, got %q", frames[3])
	}
}

func TestTranslateResponsesStreamToChat_ToolCallStream(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("response.created", map[string]any{
			"type": "response.created",
			"response": map[string]any{
				"id": "resp_tool",
			},
		}),
		sseJSONEvent("response.output_item.added", map[string]any{
			"type":         "response.output_item.added",
			"item_id":      "fc_1",
			"output_index": 0,
			"item": map[string]any{
				"id":      "fc_1",
				"type":    "function_call",
				"call_id": "call_1",
				"name":    "weather",
			},
		}),
		sseJSONEvent("response.function_call_arguments.delta", map[string]any{
			"type":         "response.function_call_arguments.delta",
			"item_id":      "fc_1",
			"output_index": 0,
			"delta":        "{\"city\":",
		}),
		sseJSONEvent("response.function_call_arguments.delta", map[string]any{
			"type":         "response.function_call_arguments.delta",
			"item_id":      "fc_1",
			"output_index": 0,
			"delta":        "\"Paris\"}",
		}),
		sseJSONEvent("response.completed", map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"id":     "resp_tool",
				"status": "completed",
			},
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateResponsesStreamToChat(recorder, strings.NewReader(body), "gpt-5.4", "req-2", slog.Default())
	if err != nil {
		t.Fatalf("TranslateResponsesStreamToChat returned error: %v", err)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0 {
		t.Fatalf("expected zero usage for stream without usage payload, got %#v", usage)
	}

	frames := parseSSEDataFrames(t, recorder.Body.String())
	if len(frames) != 6 {
		t.Fatalf("expected 6 SSE data frames, got %d: %v", len(frames), frames)
	}

	startChunk := decodeChatChunk(t, frames[1])
	if len(startChunk.Choices[0].Delta.ToolCalls) != 1 {
		t.Fatalf("expected one tool call in start chunk")
	}
	startTool := startChunk.Choices[0].Delta.ToolCalls[0]
	if startTool.ID != "call_1" {
		t.Fatalf("expected tool call id call_1, got %q", startTool.ID)
	}
	if startTool.Type != "function" {
		t.Fatalf("expected tool call type function, got %q", startTool.Type)
	}
	if startTool.Function.Name != "weather" {
		t.Fatalf("expected tool name weather, got %q", startTool.Function.Name)
	}
	if startTool.Function.Arguments != "" {
		t.Fatalf("expected empty initial arguments, got %q", startTool.Function.Arguments)
	}

	argChunk1 := decodeChatChunk(t, frames[2])
	if got := argChunk1.Choices[0].Delta.ToolCalls[0].Function.Arguments; got != "{\"city\":" {
		t.Fatalf("expected first arg delta, got %q", got)
	}
	argChunk2 := decodeChatChunk(t, frames[3])
	if got := argChunk2.Choices[0].Delta.ToolCalls[0].Function.Arguments; got != "\"Paris\"}" {
		t.Fatalf("expected second arg delta, got %q", got)
	}

	finishChunk := decodeChatChunk(t, frames[4])
	if finishChunk.Choices[0].FinishReason == nil || *finishChunk.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("expected finish_reason tool_calls, got %#v", finishChunk.Choices[0].FinishReason)
	}
	if frames[5] != "[DONE]" {
		t.Fatalf("expected final [DONE] frame, got %q", frames[5])
	}
}

func TestTranslateResponsesStreamToChat_ErrorEvent(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("error", map[string]any{
			"type": "error",
			"error": map[string]any{
				"message": "upstream bad",
			},
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateResponsesStreamToChat(recorder, strings.NewReader(body), "gpt-5.4", "req-3", slog.Default())
	if err != nil {
		t.Fatalf("TranslateResponsesStreamToChat returned error: %v", err)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0 {
		t.Fatalf("expected zero usage for error event, got %#v", usage)
	}

	frames := parseSSEDataFrames(t, recorder.Body.String())
	if len(frames) != 1 {
		t.Fatalf("expected 1 SSE data frame, got %d: %v", len(frames), frames)
	}

	var payload map[string]map[string]string
	if err := json.Unmarshal([]byte(frames[0]), &payload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if got := payload["error"]["message"]; got != "upstream bad" {
		t.Fatalf("expected error message upstream bad, got %q", got)
	}
}

func TestTranslateResponsesStreamToAnthropic_ReturnsUsage(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("response.created", map[string]any{
			"type": "response.created",
			"response": map[string]any{
				"id": "resp_anthropic",
				"usage": map[string]any{
					"input_tokens":  7,
					"output_tokens": 1,
					"total_tokens":  8,
				},
			},
		}),
		sseJSONEvent("response.output_text.delta", map[string]any{
			"type":          "response.output_text.delta",
			"item_id":       "msg_1",
			"output_index":  0,
			"content_index": 0,
			"delta":         "Hi",
		}),
		sseJSONEvent("response.completed", map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"id":     "resp_anthropic",
				"status": "completed",
				"usage": map[string]any{
					"input_tokens":  7,
					"output_tokens": 4,
					"total_tokens":  11,
				},
			},
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateResponsesStreamToAnthropic(recorder, strings.NewReader(body), "claude-sonnet", "req-4", slog.Default())
	if err != nil {
		t.Fatalf("TranslateResponsesStreamToAnthropic returned error: %v", err)
	}
	if usage.InputTokens != 7 || usage.OutputTokens != 4 || usage.TotalTokens != 11 {
		t.Fatalf("expected usage 7/4/11, got %#v", usage)
	}
	if !strings.Contains(recorder.Body.String(), "event: message_start") {
		t.Fatalf("expected anthropic stream output, got %q", recorder.Body.String())
	}
}

func TestTranslateAnthropicStreamToChat_TextDeltaAndDone(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("message_start", map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id": "msg_123",
				"usage": map[string]any{
					"input_tokens":  9,
					"output_tokens": 1,
				},
			},
		}),
		sseJSONEvent("content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]any{
				"type": "text",
				"text": "",
			},
		}),
		sseJSONEvent("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{
				"type": "text_delta",
				"text": "Hello",
			},
		}),
		sseJSONEvent("message_delta", map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason": "end_turn",
			},
			"usage": map[string]any{
				"input_tokens":  9,
				"output_tokens": 5,
			},
		}),
		sseJSONEvent("message_stop", map[string]any{
			"type": "message_stop",
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateAnthropicStreamToChat(recorder, strings.NewReader(body), "gpt-5.4", "req-a1", slog.Default())
	if err != nil {
		t.Fatalf("TranslateAnthropicStreamToChat returned error: %v", err)
	}
	if usage.InputTokens != 9 || usage.OutputTokens != 5 || usage.TotalTokens != 14 {
		t.Fatalf("expected usage 9/5/14, got %#v", usage)
	}

	frames := parseSSEDataFrames(t, recorder.Body.String())
	if len(frames) != 4 {
		t.Fatalf("expected 4 SSE data frames, got %d: %v", len(frames), frames)
	}

	roleChunk := decodeChatChunk(t, frames[0])
	if got := roleChunk.Choices[0].Delta.Role; got != "assistant" {
		t.Fatalf("expected first chunk role assistant, got %q", got)
	}

	textChunk := decodeChatChunk(t, frames[1])
	if got := textChunk.Choices[0].Delta.Content; got != "Hello" {
		t.Fatalf("expected text delta Hello, got %q", got)
	}

	finishChunk := decodeChatChunk(t, frames[2])
	if finishChunk.Choices[0].FinishReason == nil || *finishChunk.Choices[0].FinishReason != "stop" {
		t.Fatalf("expected finish_reason stop, got %#v", finishChunk.Choices[0].FinishReason)
	}
	if got := finishChunk.ID; got != "msg_123" {
		t.Fatalf("expected response id msg_123, got %q", got)
	}
	if frames[3] != "[DONE]" {
		t.Fatalf("expected final [DONE] frame, got %q", frames[3])
	}
}

func TestTranslateAnthropicStreamToChat_ToolUseStream(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("message_start", map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id": "msg_tool",
			},
		}),
		sseJSONEvent("content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "toolu_1",
				"name": "weather",
			},
		}),
		sseJSONEvent("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{
				"type":         "input_json_delta",
				"partial_json": "{\"city\":",
			},
		}),
		sseJSONEvent("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{
				"type":         "input_json_delta",
				"partial_json": "\"Paris\"}",
			},
		}),
		sseJSONEvent("message_delta", map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason": "tool_use",
			},
		}),
		sseJSONEvent("message_stop", map[string]any{
			"type": "message_stop",
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateAnthropicStreamToChat(recorder, strings.NewReader(body), "gpt-5.4", "req-a2", slog.Default())
	if err != nil {
		t.Fatalf("TranslateAnthropicStreamToChat returned error: %v", err)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0 {
		t.Fatalf("expected zero usage for stream without usage payload, got %#v", usage)
	}

	frames := parseSSEDataFrames(t, recorder.Body.String())
	if len(frames) != 6 {
		t.Fatalf("expected 6 SSE data frames, got %d: %v", len(frames), frames)
	}

	startChunk := decodeChatChunk(t, frames[1])
	if len(startChunk.Choices[0].Delta.ToolCalls) != 1 {
		t.Fatalf("expected one tool call in start chunk")
	}
	startTool := startChunk.Choices[0].Delta.ToolCalls[0]
	if startTool.ID != "toolu_1" {
		t.Fatalf("expected tool call id toolu_1, got %q", startTool.ID)
	}
	if startTool.Type != "function" {
		t.Fatalf("expected tool call type function, got %q", startTool.Type)
	}
	if startTool.Function.Name != "weather" {
		t.Fatalf("expected tool name weather, got %q", startTool.Function.Name)
	}

	argChunk1 := decodeChatChunk(t, frames[2])
	if got := argChunk1.Choices[0].Delta.ToolCalls[0].Function.Arguments; got != "{\"city\":" {
		t.Fatalf("expected first arg delta, got %q", got)
	}
	argChunk2 := decodeChatChunk(t, frames[3])
	if got := argChunk2.Choices[0].Delta.ToolCalls[0].Function.Arguments; got != "\"Paris\"}" {
		t.Fatalf("expected second arg delta, got %q", got)
	}

	finishChunk := decodeChatChunk(t, frames[4])
	if finishChunk.Choices[0].FinishReason == nil || *finishChunk.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("expected finish_reason tool_calls, got %#v", finishChunk.Choices[0].FinishReason)
	}
	if frames[5] != "[DONE]" {
		t.Fatalf("expected final [DONE] frame, got %q", frames[5])
	}
}

func TestTranslateAnthropicStreamToChat_ErrorEvent(t *testing.T) {
	body := buildSSEStream(
		sseJSONEvent("error", map[string]any{
			"type": "error",
			"error": map[string]any{
				"message": "anthropic upstream bad",
			},
		}),
	)

	recorder := httptest.NewRecorder()
	usage, err := TranslateAnthropicStreamToChat(recorder, strings.NewReader(body), "gpt-5.4", "req-a3", slog.Default())
	if err != nil {
		t.Fatalf("TranslateAnthropicStreamToChat returned error: %v", err)
	}
	if usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0 {
		t.Fatalf("expected zero usage for error event, got %#v", usage)
	}

	frames := parseSSEDataFrames(t, recorder.Body.String())
	if len(frames) != 1 {
		t.Fatalf("expected 1 SSE data frame, got %d: %v", len(frames), frames)
	}

	var payload map[string]map[string]string
	if err := json.Unmarshal([]byte(frames[0]), &payload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if got := payload["error"]["message"]; got != "anthropic upstream bad" {
		t.Fatalf("expected error message anthropic upstream bad, got %q", got)
	}
}

func buildSSEStream(events ...string) string {
	return strings.Join(events, "")
}

func sseJSONEvent(eventName string, payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshal sse payload: %v", err))
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, data)
}

func parseSSEDataFrames(t *testing.T, raw string) []string {
	t.Helper()
	parts := strings.Split(raw, "\n\n")
	frames := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		for _, line := range strings.Split(part, "\n") {
			if strings.HasPrefix(line, "data: ") {
				frames = append(frames, strings.TrimPrefix(line, "data: "))
			}
		}
	}
	return frames
}

func decodeChatChunk(t *testing.T, raw string) chatStreamChunk {
	t.Helper()
	var chunk chatStreamChunk
	if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
		t.Fatalf("failed to decode chat chunk: %v; raw=%s", err, raw)
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("expected exactly one choice, got %d", len(chunk.Choices))
	}
	return chunk
}
