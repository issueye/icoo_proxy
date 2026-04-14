package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAI request/response types

type openaiChatRequest struct {
	Model       string               `json:"model"`
	Messages    []openaiChatMessage  `json:"messages"`
	Temperature *float64             `json:"temperature,omitempty"`
	MaxTokens   *int                 `json:"max_tokens,omitempty"`
	Stream      bool                 `json:"stream,omitempty"`
	Tools       []openaiToolDef      `json:"tools,omitempty"`
	Stop        []string             `json:"stop,omitempty"`
}

type openaiChatMessage struct {
	Role       string          `json:"role"`
	Content    interface{}     `json:"content"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type openaiContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openaiImageURL `json:"image_url,omitempty"`
}

type openaiImageURL struct {
	URL string `json:"url"`
}

type openaiToolDef struct {
	Type     string          `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiFunctionCall `json:"function"`
}

type openaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiChatResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []openaiRespChoice  `json:"choices"`
	Usage   *openaiRespUsage    `json:"usage,omitempty"`
}

type openaiRespChoice struct {
	Index        int               `json:"index"`
	Message      *openaiChatMessage `json:"message,omitempty"`
	Delta        *openaiStreamDelta `json:"delta,omitempty"`
	FinishReason *string           `json:"finish_reason"`
}

type openaiStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiRespUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiStreamChunk struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openaiStreamChoice `json:"choices"`
}

type openaiStreamChoice struct {
	Index        int               `json:"index"`
	Delta        *openaiStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

type openaiModelsResponse struct {
	Object string           `json:"object"`
	Data   []openaiModelData `json:"data"`
}

type openaiModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIAdapter implements ProtocolAdapter for OpenAI Chat Completions API.
type OpenAIAdapter struct{}

func contentToString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func (a *OpenAIAdapter) ParseRequest(body []byte) (*InternalRequest, error) {
	var req openaiChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI request: %w", err)
	}

	internal := &InternalRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Stop:        req.Stop,
	}

	for _, msg := range req.Messages {
		im := InternalMessage{Role: msg.Role}

		// Tool result messages
		if msg.Role == "tool" {
			im.Content = []ContentBlock{{
				Type: "tool_result",
				ToolResult: &ToolResult{
					ToolUseID: msg.ToolCallID,
					Content:   contentToString(msg.Content),
				},
			}}
			internal.Messages = append(internal.Messages, im)
			continue
		}

		// Assistant messages with tool calls
		if len(msg.ToolCalls) > 0 {
			text := contentToString(msg.Content)
			if text != "" {
				im.Content = append(im.Content, ContentBlock{Type: "text", Text: text})
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				im.Content = append(im.Content, ContentBlock{
					Type: "tool_use",
					ToolUse: &ToolUse{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: args,
					},
				})
			}
			internal.Messages = append(internal.Messages, im)
			continue
		}

		// Regular content
		switch v := msg.Content.(type) {
		case string:
			im.Content = []ContentBlock{{Type: "text", Text: v}}
		case []interface{}:
			for _, part := range v {
				pm, _ := json.Marshal(part)
				var cp openaiContentPart
				json.Unmarshal(pm, &cp)
				switch cp.Type {
				case "text":
					im.Content = append(im.Content, ContentBlock{Type: "text", Text: cp.Text})
				case "image_url":
					im.Content = append(im.Content, ContentBlock{Type: "image", ImageURL: cp.ImageURL.URL})
				}
			}
		default:
			im.Content = []ContentBlock{{Type: "text", Text: contentToString(msg.Content)}}
		}

		if msg.Role == "system" && internal.System == "" {
			internal.System = contentToString(msg.Content)
		}
		internal.Messages = append(internal.Messages, im)
	}

	for _, t := range req.Tools {
		if t.Type == "function" {
			internal.Tools = append(internal.Tools, InternalTool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
	}

	return internal, nil
}

func (a *OpenAIAdapter) BuildRequest(req *InternalRequest) ([]byte, string, error) {
	chatReq := openaiChatRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Stop:        req.Stop,
	}

	for _, msg := range req.Messages {
		cm := openaiChatMessage{Role: msg.Role}

		// Tool result
		if msg.Role == "tool" {
			for _, block := range msg.Content {
				if block.Type == "tool_result" && block.ToolResult != nil {
					cm.Content = block.ToolResult.Content
					cm.ToolCallID = block.ToolResult.ToolUseID
					break
				}
			}
			chatReq.Messages = append(chatReq.Messages, cm)
			continue
		}

		// Check for tool use
		hasToolUse := false
		for _, block := range msg.Content {
			if block.Type == "tool_use" {
				hasToolUse = true
				break
			}
		}

		if hasToolUse {
			var textParts []string
			var toolCalls []openaiToolCall
			for _, block := range msg.Content {
				if block.Type == "text" && block.Text != "" {
					textParts = append(textParts, block.Text)
				}
				if block.Type == "tool_use" && block.ToolUse != nil {
					argsJSON, _ := json.Marshal(block.ToolUse.Arguments)
					toolCalls = append(toolCalls, openaiToolCall{
						ID:   block.ToolUse.ID,
						Type: "function",
						Function: openaiFunctionCall{
							Name:      block.ToolUse.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
			if len(textParts) > 0 {
				cm.Content = strings.Join(textParts, "")
			}
			cm.ToolCalls = toolCalls
			chatReq.Messages = append(chatReq.Messages, cm)
			continue
		}

		// Check for images
		hasImage := false
		for _, block := range msg.Content {
			if block.Type == "image" {
				hasImage = true
				break
			}
		}

		if hasImage {
			var parts []openaiContentPart
			for _, block := range msg.Content {
				switch block.Type {
				case "text":
					parts = append(parts, openaiContentPart{Type: "text", Text: block.Text})
				case "image":
					url := block.ImageURL
					if block.Data != "" {
						url = "data:" + block.MimeType + ";base64," + block.Data
					}
					parts = append(parts, openaiContentPart{
						Type:     "image_url",
						ImageURL: &openaiImageURL{URL: url},
					})
				}
			}
			cm.Content = parts
		} else {
			text := ""
			for _, block := range msg.Content {
				if block.Type == "text" {
					text += block.Text
				}
			}
			cm.Content = text
		}
		chatReq.Messages = append(chatReq.Messages, cm)
	}

	for _, t := range req.Tools {
		chatReq.Tools = append(chatReq.Tools, openaiToolDef{
			Type: "function",
			Function: openaiFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}
	return body, "/chat/completions", nil
}

func (a *OpenAIAdapter) ParseResponse(body []byte) (*InternalResponse, error) {
	var resp openaiChatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	internal := &InternalResponse{ID: resp.ID, Model: resp.Model}
	for _, choice := range resp.Choices {
		ic := InternalChoice{
			Index:        choice.Index,
			FinishReason: derefStr(choice.FinishReason),
		}
		if choice.Message != nil {
			im := &InternalMessage{Role: choice.Message.Role}
			text := contentToString(choice.Message.Content)
			if text != "" {
				im.Content = append(im.Content, ContentBlock{Type: "text", Text: text})
			}
			for _, tc := range choice.Message.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				im.Content = append(im.Content, ContentBlock{
					Type: "tool_use",
					ToolUse: &ToolUse{ID: tc.ID, Name: tc.Function.Name, Arguments: args},
				})
			}
			ic.Message = im
		}
		internal.Choices = append(internal.Choices, ic)
	}

	if resp.Usage != nil {
		internal.Usage = &InternalUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return internal, nil
}

func (a *OpenAIAdapter) BuildResponse(resp *InternalResponse) ([]byte, error) {
	chatResp := openaiChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
	}

	for _, choice := range resp.Choices {
		rc := openaiRespChoice{Index: choice.Index, FinishReason: &choice.FinishReason}
		if choice.Message != nil {
			cm := &openaiChatMessage{Role: choice.Message.Role}
			var textParts []string
			var toolCalls []openaiToolCall
			for _, block := range choice.Message.Content {
				switch block.Type {
				case "text":
					textParts = append(textParts, block.Text)
				case "tool_use":
					if block.ToolUse != nil {
						argsJSON, _ := json.Marshal(block.ToolUse.Arguments)
						toolCalls = append(toolCalls, openaiToolCall{
							ID: block.ToolUse.ID, Type: "function",
							Function: openaiFunctionCall{Name: block.ToolUse.Name, Arguments: string(argsJSON)},
						})
					}
				}
			}
			if len(textParts) > 0 {
				cm.Content = strings.Join(textParts, "")
			}
			cm.ToolCalls = toolCalls
			rc.Message = cm
		}
		chatResp.Choices = append(chatResp.Choices, rc)
	}

	if resp.Usage != nil {
		chatResp.Usage = &openaiRespUsage{
			PromptTokens: resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens: resp.Usage.TotalTokens,
		}
	}
	return json.Marshal(chatResp)
}

func (a *OpenAIAdapter) ParseStreamEvent(eventType, data string) (*InternalStreamChunk, error) {
	if data == "[DONE]" {
		return &InternalStreamChunk{StreamDone: true}, nil
	}
	var chunk openaiStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI stream event: %w", err)
	}
	internal := &InternalStreamChunk{ID: chunk.ID, Model: chunk.Model}
	for _, choice := range chunk.Choices {
		ic := InternalChoice{Index: choice.Index, FinishReason: derefStr(choice.FinishReason)}
		if choice.Delta != nil {
			delta := &InternalDelta{Role: choice.Delta.Role}
			if choice.Delta.Content != "" {
				delta.Content = []ContentBlock{{Type: "text", Text: choice.Delta.Content}}
			}
			for _, tc := range choice.Delta.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				delta.ToolUses = append(delta.ToolUses, ToolUse{ID: tc.ID, Name: tc.Function.Name, Arguments: args})
			}
			ic.Delta = delta
		}
		internal.Choices = append(internal.Choices, ic)
	}
	return internal, nil
}

func (a *OpenAIAdapter) BuildStreamEvent(chunk *InternalStreamChunk) (string, string, error) {
	if chunk.StreamDone {
		return "", "[DONE]", nil
	}
	sc := openaiStreamChunk{
		ID: chunk.ID, Object: "chat.completion.chunk",
		Created: time.Now().Unix(), Model: chunk.Model,
	}
	for _, choice := range chunk.Choices {
		sch := openaiStreamChoice{Index: choice.Index, FinishReason: &choice.FinishReason}
		if choice.Delta != nil {
			delta := &openaiStreamDelta{Role: choice.Delta.Role}
			for _, block := range choice.Delta.Content {
				if block.Type == "text" && block.Text != "" {
					delta.Content += block.Text
				}
			}
			for _, tu := range choice.Delta.ToolUses {
				argsJSON, _ := json.Marshal(tu.Arguments)
				delta.ToolCalls = append(delta.ToolCalls, openaiToolCall{
					ID: tu.ID, Type: "function",
					Function: openaiFunctionCall{Name: tu.Name, Arguments: string(argsJSON)},
				})
			}
			sch.Delta = delta
		}
		sc.Choices = append(sc.Choices, sch)
	}
	data, err := json.Marshal(sc)
	if err != nil {
		return "", "", err
	}
	return "", string(data), nil
}

func (a *OpenAIAdapter) StreamDone() string { return "[DONE]" }

func (a *OpenAIAdapter) BuildHTTPRequest(ctx context.Context, apiBase, apiKey, method, path string, body []byte) (*http.Request, error) {
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
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return req, nil
}

func (a *OpenAIAdapter) ListModelsRequest(ctx context.Context, apiBase, apiKey string) (*http.Request, error) {
	url := strings.TrimRight(apiBase, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return req, nil
}

func (a *OpenAIAdapter) ParseModelsResponse(body []byte) ([]ModelInfo, error) {
	var resp openaiModelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI models response: %w", err)
	}
	models := make([]ModelInfo, 0, len(resp.Data))
	for _, m := range resp.Data {
		models = append(models, ModelInfo{ID: m.ID, Name: m.ID, OwnedBy: m.OwnedBy})
	}
	return models, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
