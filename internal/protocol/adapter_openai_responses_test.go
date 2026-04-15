package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIAdapterParseResponsesRequest(t *testing.T) {
	adapter := &OpenAIAdapter{}
	body := []byte(`{
		"model":"gpt-4.1",
		"instructions":"You are concise.",
		"input":[
			{
				"type":"message",
				"role":"user",
				"content":[
					{"type":"input_text","text":"hello"},
					{"type":"input_image","image_url":"https://example.com/image.png"}
				]
			}
		],
		"tools":[
			{
				"type":"function",
				"name":"lookup_weather",
				"description":"Lookup weather",
				"parameters":{"type":"object"}
			}
		]
	}`)

	req, err := adapter.ParseResponsesRequest(body)
	if err != nil {
		t.Fatalf("ParseResponsesRequest() error = %v", err)
	}
	if req.Model != "gpt-4.1" {
		t.Fatalf("Model = %q", req.Model)
	}
	if req.System != "You are concise." {
		t.Fatalf("System = %q", req.System)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("len(Messages) = %d", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("Role = %q", req.Messages[0].Role)
	}
	if len(req.Messages[0].Content) != 2 {
		t.Fatalf("len(Content) = %d", len(req.Messages[0].Content))
	}
	if req.Messages[0].Content[0].Text != "hello" {
		t.Fatalf("Text = %q", req.Messages[0].Content[0].Text)
	}
	if req.Messages[0].Content[1].ImageURL != "https://example.com/image.png" {
		t.Fatalf("ImageURL = %q", req.Messages[0].Content[1].ImageURL)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "lookup_weather" {
		t.Fatalf("unexpected tools: %+v", req.Tools)
	}
}

func TestOpenAIAdapterBuildResponsesResponse(t *testing.T) {
	adapter := &OpenAIAdapter{}
	resp := &InternalResponse{
		ID:    "resp_test",
		Model: "gpt-4.1",
		Choices: []InternalChoice{
			{
				Index: 0,
				Message: &InternalMessage{
					Role: "assistant",
					Content: []ContentBlock{
						{Type: "text", Text: "hello world"},
						{
							Type: "tool_use",
							ToolUse: &ToolUse{
								ID:   "call_123",
								Name: "lookup_weather",
								Arguments: map[string]interface{}{
									"city": "Shanghai",
								},
							},
						},
					},
				},
			},
		},
		Usage: &InternalUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	body, err := adapter.BuildResponsesResponse(resp)
	if err != nil {
		t.Fatalf("BuildResponsesResponse() error = %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["object"] != "response" {
		t.Fatalf("object = %v", payload["object"])
	}
	output, ok := payload["output"].([]interface{})
	if !ok || len(output) != 2 {
		t.Fatalf("unexpected output: %#v", payload["output"])
	}
	usage, ok := payload["usage"].(map[string]interface{})
	if !ok || usage["total_tokens"].(float64) != 15 {
		t.Fatalf("unexpected usage: %#v", payload["usage"])
	}
}

func TestOpenAIAdapterBuildResponsesRequest(t *testing.T) {
	adapter := &OpenAIAdapter{}
	temp := 0.3
	maxTokens := 512
	body, path, err := adapter.BuildResponsesRequest(&InternalRequest{
		Model:       "gpt-4.1",
		System:      "You are concise.",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Messages: []InternalMessage{
			{
				Role: "user",
				Content: []ContentBlock{
					{Type: "text", Text: "hello"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildResponsesRequest() error = %v", err)
	}
	if path != "/responses" {
		t.Fatalf("path = %q", path)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["instructions"] != "You are concise." {
		t.Fatalf("instructions = %v", payload["instructions"])
	}
	if payload["input"] != "hello" {
		t.Fatalf("input = %v", payload["input"])
	}
}

func TestOpenAIAdapterBuildResponsesStreamEvents(t *testing.T) {
	adapter := &OpenAIAdapter{}
	state := &OpenAIResponsesStreamState{}

	events, err := adapter.BuildResponsesStreamEvents(&InternalStreamChunk{
		ID:    "resp_stream",
		Model: "gpt-4.1",
		Choices: []InternalChoice{
			{
				Index: 0,
				Delta: &InternalDelta{
					Role: "assistant",
					Content: []ContentBlock{
						{Type: "text", Text: "hel"},
					},
				},
			},
		},
	}, state)
	if err != nil {
		t.Fatalf("BuildResponsesStreamEvents() error = %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("len(events) = %d", len(events))
	}
	if !strings.Contains(events[0], `"type":"response.created"`) {
		t.Fatalf("expected response.created event, got %s", events[0])
	}
	if !strings.Contains(strings.Join(events, "\n"), `"type":"response.output_text.delta"`) {
		t.Fatalf("expected response.output_text.delta event, got %v", events)
	}

	doneEvents, err := adapter.BuildResponsesStreamEvents(&InternalStreamChunk{
		ID:         "resp_stream",
		Model:      "gpt-4.1",
		StreamDone: true,
		Usage: &InternalUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, state)
	if err != nil {
		t.Fatalf("BuildResponsesStreamEvents(done) error = %v", err)
	}
	if !strings.Contains(strings.Join(doneEvents, "\n"), `"type":"response.completed"`) {
		t.Fatalf("expected response.completed event, got %v", doneEvents)
	}
}
