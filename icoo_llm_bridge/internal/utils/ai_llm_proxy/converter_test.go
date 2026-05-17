package ai_llm_proxy

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

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

func TestProtocolConverterResponsesResponseToChatEmptyAssistantContent(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"id":"resp_1","object":"response","model":"gpt-5.5","status":"completed","output":[]}`)

	out, err := converter.ConvertResponse(ResponseInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      "gpt-5.5",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}
	var payload ChatCompletionsResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	var content string
	if err := json.Unmarshal(payload.Choices[0].Message.Content, &content); err != nil {
		t.Fatalf("unmarshal empty message content: %v", err)
	}
	if content != "" {
		t.Fatalf("content = %q", content)
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

func TestProtocolConverterChatResponseToResponses(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"id":"chatcmpl_1","object":"chat.completion","model":"gpt","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"length"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`)

	out, err := converter.ConvertResponse(ResponseInput{
		Downstream: constants.ProtocolOpenAIResponses,
		Upstream:   constants.ProtocolOpenAIChat,
		Model:      "target-model",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}
	var payload ResponsesResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload.Object != "response" {
		t.Fatalf("object = %v, body = %s", payload.Object, string(out))
	}
	if payload.Model != "gpt" {
		t.Fatalf("model = %q", payload.Model)
	}
	if payload.Status != "incomplete" || payload.IncompleteDetails == nil || payload.IncompleteDetails.Reason != "max_output_tokens" {
		t.Fatalf("status/details = %q/%+v", payload.Status, payload.IncompleteDetails)
	}
	if len(payload.Output) != 1 || len(payload.Output[0].Content) != 1 || payload.Output[0].Content[0].Text != "hello" {
		t.Fatalf("output = %+v", payload.Output)
	}
	if payload.Usage == nil || payload.Usage.InputTokens != 2 || payload.Usage.OutputTokens != 3 || payload.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %+v", payload.Usage)
	}
}

func TestProtocolConverterChatResponseToAnthropic(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"id":"chatcmpl_1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`)

	out, err := converter.ConvertResponse(ResponseInput{
		Downstream: constants.ProtocolAnthropic,
		Upstream:   constants.ProtocolOpenAIChat,
		Model:      "target-model",
		Body:       body,
	})
	if err != nil {
		t.Fatalf("ConvertResponse returned error: %v", err)
	}
	var payload AnthropicResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload.Type != "message" || payload.Role != "assistant" {
		t.Fatalf("type/role = %q/%q", payload.Type, payload.Role)
	}
	if payload.Model != "target-model" {
		t.Fatalf("model = %q", payload.Model)
	}
	if payload.StopReason != "end_turn" {
		t.Fatalf("stop_reason = %q", payload.StopReason)
	}
	if len(payload.Content) != 1 || payload.Content[0].Type != "text" || payload.Content[0].Text != "hello" {
		t.Fatalf("content = %+v", payload.Content)
	}
	if payload.Usage.InputTokens != 2 || payload.Usage.OutputTokens != 3 {
		t.Fatalf("usage = %+v", payload.Usage)
	}
}

func TestProtocolConverterUnsupportedRequestDirections(t *testing.T) {
	converter := NewProtocolConverter()
	body := []byte(`{"model":"m"}`)

	for _, tc := range []struct {
		name       string
		downstream constants.Protocol
		upstream   constants.Protocol
	}{
		{
			name:       "anthropic to chat",
			downstream: constants.ProtocolAnthropic,
			upstream:   constants.ProtocolOpenAIChat,
		},
		{
			name:       "responses to chat",
			downstream: constants.ProtocolOpenAIResponses,
			upstream:   constants.ProtocolOpenAIChat,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := converter.ConvertRequest(RequestInput{
				Downstream: tc.downstream,
				Upstream:   tc.upstream,
				Body:       body,
			})
			if err == nil || !strings.Contains(err.Error(), "not implemented") {
				t.Fatalf("expected not implemented error, got %v", err)
			}
		})
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

func TestProtocolConverterResponsesStreamToChatStopsAfterTerminalEventWithoutEOF(t *testing.T) {
	converter := NewProtocolConverter()
	reader, writer := io.Pipe()
	done := make(chan struct{})
	var out strings.Builder

	go func() {
		defer close(done)
		_, err := io.WriteString(writer, strings.Join([]string{
			`event: response.created`,
			`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt","status":"in_progress"}}`,
			``,
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hello"}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt","status":"completed"}}`,
			``,
			``,
		}, "\n"))
		if err != nil {
			t.Errorf("write stream: %v", err)
			return
		}
		<-time.After(2 * time.Second)
		_ = writer.Close()
	}()

	errCh := make(chan error, 1)
	go func() {
		_, err := converter.ConvertStream(StreamInput{
			Downstream: constants.ProtocolOpenAIChat,
			Upstream:   constants.ProtocolOpenAIResponses,
			Model:      "target-model",
			Reader:     reader,
			Writer:     &out,
		})
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ConvertStream returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("ConvertStream did not stop after terminal event")
	}

	_ = writer.Close()
	<-done

	if !strings.Contains(out.String(), `"content":"hello"`) {
		t.Fatalf("expected text delta, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "data: [DONE]") {
		t.Fatalf("expected done marker, got: %s", out.String())
	}
}

func TestProtocolConverterResponsesStreamToChatLargeDataLine(t *testing.T) {
	converter := NewProtocolConverter()
	largeText := strings.Repeat("x", 70*1024)
	eventBody, err := json.Marshal(ResponsesStreamEvent{
		Type:        "response.output_text.delta",
		OutputIndex: 0,
		Delta:       largeText,
	})
	if err != nil {
		t.Fatalf("marshal stream event: %v", err)
	}
	input := strings.Join([]string{
		`event: response.output_text.delta`,
		`data: ` + string(eventBody),
		``,
	}, "\n")
	var out strings.Builder

	_, err = converter.ConvertStream(StreamInput{
		Downstream: constants.ProtocolOpenAIChat,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      "target-model",
		Reader:     strings.NewReader(input),
		Writer:     &out,
	})
	if err != nil {
		t.Fatalf("ConvertStream returned error: %v", err)
	}
	if !strings.Contains(out.String(), largeText) {
		t.Fatalf("expected large text delta in output")
	}
	if !strings.Contains(out.String(), "data: [DONE]") {
		t.Fatalf("expected done marker")
	}
}

func TestProtocolConverterChatStreamToResponses(t *testing.T) {
	converter := NewProtocolConverter()
	input := strings.Join([]string{
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		``,
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")
	var out strings.Builder

	result, err := converter.ConvertStream(StreamInput{
		Downstream: constants.ProtocolOpenAIResponses,
		Upstream:   constants.ProtocolOpenAIChat,
		Model:      "target-model",
		Reader:     strings.NewReader(input),
		Writer:     &out,
	})
	if err != nil {
		t.Fatalf("ConvertStream returned error: %v", err)
	}
	if !strings.Contains(out.String(), `event: response.created`) {
		t.Fatalf("expected response.created, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `event: response.output_text.delta`) || !strings.Contains(out.String(), `"delta":"hello"`) {
		t.Fatalf("expected text delta, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `event: response.completed`) || !strings.Contains(out.String(), `"input_tokens":2`) {
		t.Fatalf("expected completed event with usage, got: %s", out.String())
	}
	if result.Usage.InputTokens != 2 || result.Usage.OutputTokens != 3 || result.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %+v", result.Usage)
	}
}

func TestProtocolConverterChatStreamToAnthropic(t *testing.T) {
	converter := NewProtocolConverter()
	input := strings.Join([]string{
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		``,
		`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")
	var out strings.Builder

	result, err := converter.ConvertStream(StreamInput{
		Downstream: constants.ProtocolAnthropic,
		Upstream:   constants.ProtocolOpenAIChat,
		Model:      "target-model",
		Reader:     strings.NewReader(input),
		Writer:     &out,
	})
	if err != nil {
		t.Fatalf("ConvertStream returned error: %v", err)
	}
	if !strings.Contains(out.String(), `event: message_start`) {
		t.Fatalf("expected message_start, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `"type":"text_delta"`) || !strings.Contains(out.String(), `"text":"hello"`) {
		t.Fatalf("expected text delta, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `event: message_delta`) || !strings.Contains(out.String(), `"input_tokens":2`) {
		t.Fatalf("expected message_delta with usage, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `event: message_stop`) {
		t.Fatalf("expected message_stop, got: %s", out.String())
	}
	if result.Usage.InputTokens != 2 || result.Usage.OutputTokens != 3 || result.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %+v", result.Usage)
	}
}

func TestProtocolConverterChatStreamStopsAfterDoneWithoutEOF(t *testing.T) {
	converter := NewProtocolConverter()
	reader, writer := io.Pipe()
	done := make(chan struct{})
	var out strings.Builder

	go func() {
		defer close(done)
		_, err := io.WriteString(writer, strings.Join([]string{
			`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`,
			``,
			`data: {"id":"chatcmpl_1","object":"chat.completion.chunk","model":"gpt","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			``,
			`data: [DONE]`,
			``,
			``,
		}, "\n"))
		if err != nil {
			t.Errorf("write stream: %v", err)
			return
		}
		<-time.After(2 * time.Second)
		_ = writer.Close()
	}()

	errCh := make(chan error, 1)
	go func() {
		_, err := converter.ConvertStream(StreamInput{
			Downstream: constants.ProtocolOpenAIResponses,
			Upstream:   constants.ProtocolOpenAIChat,
			Model:      "target-model",
			Reader:     reader,
			Writer:     &out,
		})
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ConvertStream returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("ConvertStream did not stop after [DONE]")
	}

	_ = writer.Close()
	<-done

	if !strings.Contains(out.String(), `event: response.completed`) {
		t.Fatalf("expected completed event, got: %s", out.String())
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
