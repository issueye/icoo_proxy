package ai_llm_proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/issueye/icoo_proxy/common/constants"
)

func TestChatChunkToResponsesEventsAccumulatesParallelToolCalls(t *testing.T) {
	state := NewChatEventToResponsesState()
	state.Model = "gpt-test"
	text := "checking"
	toolCallsFinish := "tool_calls"
	zero, one := 0, 1

	var events []ResponsesStreamEvent
	events = append(events, ChatChunkToResponsesEvents(&ChatCompletionsChunk{
		ID: "chatcmpl_tools",
		Choices: []ChatChunkChoice{{
			Index: 0,
			Delta: ChatDelta{Content: &text},
		}},
	}, state)...)
	events = append(events, ChatChunkToResponsesEvents(&ChatCompletionsChunk{
		Choices: []ChatChunkChoice{{
			Index: 0,
			Delta: ChatDelta{ToolCalls: []ChatToolCall{
				{Index: &zero, ID: "call_weather", Type: "function", Function: ChatFunctionCall{Name: "get_weather", Arguments: `{"city":"`}},
				{Index: &one, ID: "call_time", Type: "function", Function: ChatFunctionCall{Name: "get_time", Arguments: `{"zone":"`}},
			}},
		}},
	}, state)...)
	events = append(events, ChatChunkToResponsesEvents(&ChatCompletionsChunk{
		Choices: []ChatChunkChoice{{
			Index: 0,
			Delta: ChatDelta{ToolCalls: []ChatToolCall{
				{Index: &zero, Function: ChatFunctionCall{Arguments: `Paris"}`}},
				{Index: &one, Function: ChatFunctionCall{Arguments: `UTC"}`}},
			}},
			FinishReason: &toolCallsFinish,
		}},
	}, state)...)
	events = append(events, FinalizeChatResponsesStream(state)...)

	assertToolCallResponseEvents(t, events, []expectedStreamToolCall{
		{outputIndex: 1, callID: "call_weather", name: "get_weather", fragments: []string{`{"city":"`, `Paris"}`}, arguments: `{"city":"Paris"}`},
		{outputIndex: 2, callID: "call_time", name: "get_time", fragments: []string{`{"zone":"`, `UTC"}`}, arguments: `{"zone":"UTC"}`},
	})

	textDone := responseEventIndex(events, "response.output_text.done", 0)
	firstToolAdded := responseEventIndex(events, "response.output_item.added", 1)
	if textDone < 0 || firstToolAdded < 0 || textDone >= firstToolAdded {
		t.Fatalf("text item must close before the first tool item: text done=%d, tool added=%d", textDone, firstToolAdded)
	}
	if got := events[len(events)-1]; got.Type != "response.completed" || got.Response == nil || got.Response.Status != "completed" {
		t.Fatalf("last event = %#v, want completed response", got)
	}
}

func TestConvertChatToolCallStreamToAnthropicToolUseBlocks(t *testing.T) {
	input := strings.Join([]string{
		`data: {"id":"chatcmpl_tools","object":"chat.completion.chunk","model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_weather","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\""}},{"index":1,"id":"call_time","type":"function","function":{"name":"get_time","arguments":"{\"zone\":\""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl_tools","object":"chat.completion.chunk","model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":"","arguments":"Paris\"}"}},{"index":1,"function":{"name":"","arguments":"UTC\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":12,"completion_tokens":8,"total_tokens":20}}`,
		`data: [DONE]`,
		"",
	}, "\n\n")

	var output bytes.Buffer
	result, err := NewProtocolConverter().ConvertStream(StreamInput{
		Context:    context.Background(),
		Downstream: constants.ProtocolAnthropic,
		Upstream:   constants.ProtocolOpenAIChat,
		Model:      "gpt-test",
		Reader:     strings.NewReader(input),
		Writer:     &output,
	})
	if err != nil {
		t.Fatalf("ConvertStream returned error: %v", err)
	}
	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 8 || result.Usage.TotalTokens != 20 {
		t.Fatalf("usage = %#v, want 12/8/20", result.Usage)
	}

	var events []AnthropicStreamEvent
	if err := scanSSE(strings.NewReader(output.String()), func(_ string, data []byte) error {
		var event AnthropicStreamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("parse Anthropic SSE: %v\n%s", err, output.String())
	}

	var starts []AnthropicContentBlock
	var deltas []string
	var stopReason string
	for _, event := range events {
		if event.Type == "content_block_start" && event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			starts = append(starts, *event.ContentBlock)
		}
		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Type == "input_json_delta" {
			deltas = append(deltas, event.Delta.PartialJSON)
		}
		if event.Type == "message_delta" && event.Delta != nil {
			stopReason = event.Delta.StopReason
		}
	}
	if len(starts) != 2 {
		t.Fatalf("tool_use starts = %#v, want two", starts)
	}
	if starts[0].ID != "call_weather" || starts[0].Name != "get_weather" || starts[1].ID != "call_time" || starts[1].Name != "get_time" {
		t.Fatalf("tool_use starts = %#v", starts)
	}
	wantDeltas := []string{`{"city":"`, `Paris"}`, `{"zone":"`, `UTC"}`}
	if strings.Join(deltas, "|") != strings.Join(wantDeltas, "|") {
		t.Fatalf("argument deltas = %#v, want %#v", deltas, wantDeltas)
	}
	if stopReason != "tool_use" {
		t.Fatalf("stop reason = %q, want tool_use", stopReason)
	}
}

type expectedStreamToolCall struct {
	outputIndex int
	callID      string
	name        string
	fragments   []string
	arguments   string
}

func assertToolCallResponseEvents(t *testing.T, events []ResponsesStreamEvent, expected []expectedStreamToolCall) {
	t.Helper()
	for _, want := range expected {
		var added, done, itemDone *ResponsesStreamEvent
		var fragments []string
		for i := range events {
			event := &events[i]
			if event.OutputIndex != want.outputIndex {
				continue
			}
			switch event.Type {
			case "response.output_item.added":
				if event.Item != nil && event.Item.Type == "function_call" {
					added = event
				}
			case "response.function_call_arguments.delta":
				fragments = append(fragments, event.Delta)
			case "response.function_call_arguments.done":
				done = event
			case "response.output_item.done":
				if event.Item != nil && event.Item.Type == "function_call" {
					itemDone = event
				}
			}
		}
		if added == nil || added.Item.CallID != want.callID || added.Item.Name != want.name {
			t.Fatalf("output %d added event = %#v", want.outputIndex, added)
		}
		if strings.Join(fragments, "|") != strings.Join(want.fragments, "|") {
			t.Fatalf("output %d fragments = %#v, want %#v", want.outputIndex, fragments, want.fragments)
		}
		if done == nil || done.CallID != want.callID || done.Name != want.name || done.Arguments != want.arguments {
			t.Fatalf("output %d arguments done = %#v", want.outputIndex, done)
		}
		if itemDone == nil || itemDone.Item.CallID != want.callID || itemDone.Item.Name != want.name || itemDone.Item.Arguments != want.arguments || itemDone.Item.Status != "completed" {
			t.Fatalf("output %d item done = %#v", want.outputIndex, itemDone)
		}
	}
}

func responseEventIndex(events []ResponsesStreamEvent, eventType string, outputIndex int) int {
	for i, event := range events {
		if event.Type == eventType && event.OutputIndex == outputIndex {
			return i
		}
	}
	return -1
}
