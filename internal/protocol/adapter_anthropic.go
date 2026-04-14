package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Anthropic request/response types

type anthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature *float64            `json:"temperature,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Tools       []anthropicToolDef  `json:"tools,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content interface{}        `json:"content"` // string or []anthropicContentBlock
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Source *anthropicSource `json:"source,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input interface{}     `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"` // string for tool_result
	IsError   bool        `json:"is_error,omitempty"`
}

type anthropicSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Anthropic response types

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Anthropic SSE stream types

type anthropicStreamEvent struct {
	Type         string                   `json:"type"`
	Message      *anthropicResponse       `json:"message,omitempty"`
	Index        int                      `json:"index,omitempty"`
	ContentBlock *anthropicContentBlock   `json:"content_block,omitempty"`
	Delta        *anthropicStreamDelta    `json:"delta,omitempty"`
}

type anthropicStreamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// AnthropicModelsResponse - Anthropic doesn't have a standard models endpoint,
// but we define a basic structure for compatibility.
type anthropicModelsResponse struct {
	Data []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"data"`
}

// AnthropicAdapter implements ProtocolAdapter for Anthropic Messages API.
type AnthropicAdapter struct{}

func (a *AnthropicAdapter) ParseRequest(body []byte) (*InternalRequest, error) {
	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic request: %w", err)
	}

	internal := &InternalRequest{
		Model:       req.Model,
		System:      req.System,
		Temperature: req.Temperature,
		Stream:      req.Stream,
		Stop:        req.StopSequences,
	}
	if req.MaxTokens > 0 {
		internal.MaxTokens = &req.MaxTokens
	}

	for _, msg := range req.Messages {
		im := InternalMessage{Role: msg.Role}
		switch v := msg.Content.(type) {
		case string:
			im.Content = []ContentBlock{{Type: "text", Text: v}}
		case []interface{}:
			for _, item := range v {
				b, _ := json.Marshal(item)
				var block anthropicContentBlock
				json.Unmarshal(b, &block)
				switch block.Type {
				case "text":
					im.Content = append(im.Content, ContentBlock{Type: "text", Text: block.Text})
				case "image":
					if block.Source != nil {
						im.Content = append(im.Content, ContentBlock{
							Type:     "image",
							MimeType: block.Source.MediaType,
							Data:     block.Source.Data,
						})
					}
				case "tool_use":
					var args map[string]interface{}
					if m, ok := block.Input.(map[string]interface{}); ok {
						args = m
					}
					im.Content = append(im.Content, ContentBlock{
						Type: "tool_use",
						ToolUse: &ToolUse{ID: block.ID, Name: block.Name, Arguments: args},
					})
				case "tool_result":
					resultContent := ""
					switch rc := block.Content.(type) {
					case string:
						resultContent = rc
					default:
						b, _ := json.Marshal(rc)
						resultContent = string(b)
					}
					im.Role = "tool"
					im.Content = append(im.Content, ContentBlock{
						Type: "tool_result",
						ToolResult: &ToolResult{
							ToolUseID: block.ToolUseID,
							Content:   resultContent,
							IsError:   block.IsError,
						},
					})
				}
			}
		}
		internal.Messages = append(internal.Messages, im)
	}

	for _, t := range req.Tools {
		internal.Tools = append(internal.Tools, InternalTool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		})
	}

	return internal, nil
}

func (a *AnthropicAdapter) BuildRequest(req *InternalRequest) ([]byte, string, error) {
	ar := anthropicRequest{
		Model:       req.Model,
		System:      req.System,
		Temperature: req.Temperature,
		Stream:      req.Stream,
		StopSequences: req.Stop,
	}
	if req.MaxTokens != nil {
		ar.MaxTokens = *req.MaxTokens
	} else {
		ar.MaxTokens = 4096 // Anthropic requires max_tokens
	}

	for _, msg := range req.Messages {
		// Tool result messages
		if msg.Role == "tool" {
			var blocks []anthropicContentBlock
			for _, block := range msg.Content {
				if block.Type == "tool_result" && block.ToolResult != nil {
					blocks = append(blocks, anthropicContentBlock{
						Type:     "tool_result",
						ToolUseID: block.ToolResult.ToolUseID,
						Content:  block.ToolResult.Content,
						IsError:  block.ToolResult.IsError,
					})
				}
			}
			ar.Messages = append(ar.Messages, anthropicMessage{Role: "user", Content: blocks})
			continue
		}

		var blocks []anthropicContentBlock
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				blocks = append(blocks, anthropicContentBlock{Type: "text", Text: block.Text})
			case "image":
				mimeType := block.MimeType
				if mimeType == "" {
					mimeType = "image/png"
				}
				blocks = append(blocks, anthropicContentBlock{
					Type: "image",
					Source: &anthropicSource{
						Type:      "base64",
						MediaType: mimeType,
						Data:      block.Data,
					},
				})
			case "tool_use":
				if block.ToolUse != nil {
					toolIdx := 0
					for i, b := range blocks {
						if b.Type == "tool_use" {
							toolIdx = i + 1
						}
					}
					blocks = append(blocks, anthropicContentBlock{
						Type:  "tool_use",
						ID:    block.ToolUse.ID,
						Name:  block.ToolUse.Name,
						Input: block.ToolUse.Arguments,
					})
					_ = toolIdx
				}
			}
		}

		if len(blocks) > 0 {
			ar.Messages = append(ar.Messages, anthropicMessage{Role: msg.Role, Content: blocks})
		} else {
			// Simple text message
			ar.Messages = append(ar.Messages, anthropicMessage{Role: msg.Role, Content: ""})
		}
	}

	for _, t := range req.Tools {
		ar.Tools = append(ar.Tools, anthropicToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	body, err := json.Marshal(ar)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal Anthropic request: %w", err)
	}
	return body, "/messages", nil
}

func (a *AnthropicAdapter) ParseResponse(body []byte) (*InternalResponse, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	internal := &InternalResponse{ID: resp.ID, Model: resp.Model}
	im := &InternalMessage{Role: "assistant"}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			im.Content = append(im.Content, ContentBlock{Type: "text", Text: block.Text})
		case "tool_use":
			var args map[string]interface{}
			if m, ok := block.Input.(map[string]interface{}); ok {
				args = m
			}
			im.Content = append(im.Content, ContentBlock{
				Type: "tool_use",
				ToolUse: &ToolUse{ID: block.ID, Name: block.Name, Arguments: args},
			})
		}
	}

	finishReason := resp.StopReason
	if finishReason == "end_turn" {
		finishReason = "stop"
	}

	internal.Choices = append(internal.Choices, InternalChoice{
		Index:        0,
		Message:      im,
		FinishReason: finishReason,
	})
	internal.Usage = &InternalUsage{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}

	return internal, nil
}

func (a *AnthropicAdapter) BuildResponse(resp *InternalResponse) ([]byte, error) {
	ar := anthropicResponse{
		ID:    resp.ID,
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		for _, block := range resp.Choices[0].Message.Content {
			switch block.Type {
			case "text":
				ar.Content = append(ar.Content, anthropicContentBlock{Type: "text", Text: block.Text})
			case "tool_use":
				if block.ToolUse != nil {
					ar.Content = append(ar.Content, anthropicContentBlock{
						Type:  "tool_use",
						ID:    block.ToolUse.ID,
						Name:  block.ToolUse.Name,
						Input: block.ToolUse.Arguments,
					})
				}
			}
		}
		stopReason := resp.Choices[0].FinishReason
		if stopReason == "stop" {
			stopReason = "end_turn"
		}
		ar.StopReason = stopReason
	}

	if resp.Usage != nil {
		ar.Usage = anthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	}

	return json.Marshal(ar)
}

func (a *AnthropicAdapter) ParseStreamEvent(eventType, data string) (*InternalStreamChunk, error) {
	var event anthropicStreamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic stream event: %w", err)
	}

	switch event.Type {
	case "message_stop":
		return &InternalStreamChunk{StreamDone: true}, nil

	case "content_block_delta":
		if event.Delta != nil && event.Delta.Type == "text_delta" {
			return &InternalStreamChunk{
				Choices: []InternalChoice{{
					Index: 0,
					Delta: &InternalDelta{
						Content: []ContentBlock{{Type: "text", Text: event.Delta.Text}},
					},
				}},
			}, nil
		}
		// input_json_delta for tool use
		if event.Delta != nil && event.Delta.Type == "input_json_delta" {
			return &InternalStreamChunk{
				Choices: []InternalChoice{{
					Index: event.Index,
				}},
			}, nil
		}
		return &InternalStreamChunk{}, nil

	case "message_delta":
		if event.Delta != nil && event.Delta.StopReason != "" {
			stopReason := event.Delta.StopReason
			if stopReason == "end_turn" {
				stopReason = "stop"
			}
			return &InternalStreamChunk{
				Choices: []InternalChoice{{
					Index:        0,
					FinishReason: stopReason,
				}},
			}, nil
		}
		return &InternalStreamChunk{}, nil

	case "content_block_start":
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			var args map[string]interface{}
			json.Unmarshal([]byte(event.Delta.PartialJSON), &args)
			return &InternalStreamChunk{
				Choices: []InternalChoice{{
					Index: event.Index,
					Delta: &InternalDelta{
						ToolUses: []ToolUse{{
							ID:   event.ContentBlock.ID,
							Name: event.ContentBlock.Name,
						}},
					},
				}},
			}, nil
		}
		return &InternalStreamChunk{}, nil

	default:
		return &InternalStreamChunk{}, nil
	}
}

func (a *AnthropicAdapter) BuildStreamEvent(chunk *InternalStreamChunk) (string, string, error) {
	if chunk.StreamDone {
		return "message_stop", "{}", nil
	}

	// Build Anthropic SSE events from internal chunks
	if len(chunk.Choices) > 0 {
		choice := chunk.Choices[0]
		if choice.Delta != nil {
			for _, block := range choice.Delta.Content {
				if block.Type == "text" && block.Text != "" {
					deltaData, _ := json.Marshal(anthropicStreamEvent{
						Type:  "content_block_delta",
						Index: 0,
						Delta: &anthropicStreamDelta{Type: "text_delta", Text: block.Text},
					})
					return "content_block_delta", string(deltaData), nil
				}
			}
		}
		if choice.FinishReason != "" {
			stopReason := choice.FinishReason
			if stopReason == "stop" {
				stopReason = "end_turn"
			}
			deltaData, _ := json.Marshal(anthropicStreamEvent{
				Type:  "message_delta",
				Delta: &anthropicStreamDelta{StopReason: stopReason},
			})
			return "message_delta", string(deltaData), nil
		}
	}

	return "", "", nil
}

func (a *AnthropicAdapter) StreamDone() string { return "message_stop" }

func (a *AnthropicAdapter) BuildHTTPRequest(ctx context.Context, apiBase, apiKey, method, path string, body []byte) (*http.Request, error) {
	url := strings.TrimRight(apiBase, "/") + path
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	return req, nil
}

func (a *AnthropicAdapter) ListModelsRequest(ctx context.Context, apiBase, apiKey string) (*http.Request, error) {
	// Anthropic doesn't have a standard models listing endpoint.
	// We'll return a request to a known endpoint that can validate connectivity.
	url := strings.TrimRight(apiBase, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	return req, nil
}

func (a *AnthropicAdapter) ParseModelsResponse(body []byte) ([]ModelInfo, error) {
	// Anthropic may return models in a different format or not at all
	var resp anthropicModelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// If parsing fails, return empty list (Anthropic may not support this endpoint)
		return []ModelInfo{}, nil
	}
	models := make([]ModelInfo, 0)
	for _, m := range resp.Data {
		models = append(models, ModelInfo{ID: m.ID, Name: m.ID})
	}
	return models, nil
}

// Helper to convert stop reasons
func anthropicFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return reason
	}
}

// init registers the Anthropic adapter
func init() {
	// This would cause a duplicate init issue, so we register differently
}
