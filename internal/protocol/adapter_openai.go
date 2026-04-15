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
	Model       string              `json:"model"`
	Messages    []openaiChatMessage `json:"messages"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Tools       []openaiToolDef     `json:"tools,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
}

type openaiResponsesRequest struct {
	Model           string                   `json:"model"`
	Input           interface{}              `json:"input"`
	Instructions    string                   `json:"instructions,omitempty"`
	Temperature     *float64                 `json:"temperature,omitempty"`
	MaxOutputTokens *int                     `json:"max_output_tokens,omitempty"`
	Stream          bool                     `json:"stream,omitempty"`
	Tools           []openaiResponsesToolDef `json:"tools,omitempty"`
}

type openaiChatMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
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
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiResponsesToolDef struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Function    *openaiFunction        `json:"function,omitempty"`
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
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openaiRespChoice `json:"choices"`
	Usage   *openaiRespUsage   `json:"usage,omitempty"`
}

type openaiResponsesResponse struct {
	ID        string                      `json:"id"`
	Object    string                      `json:"object"`
	CreatedAt int64                       `json:"created_at"`
	Status    string                      `json:"status"`
	Model     string                      `json:"model"`
	Output    []openaiResponsesOutputItem `json:"output"`
	Usage     *openaiResponsesUsage       `json:"usage,omitempty"`
}

type openaiResponsesOutputItem struct {
	ID        string                       `json:"id,omitempty"`
	Type      string                       `json:"type"`
	Status    string                       `json:"status,omitempty"`
	Role      string                       `json:"role,omitempty"`
	Content   []openaiResponsesContentPart `json:"content,omitempty"`
	Name      string                       `json:"name,omitempty"`
	CallID    string                       `json:"call_id,omitempty"`
	Arguments string                       `json:"arguments,omitempty"`
}

type openaiResponsesContentPart struct {
	Type        string        `json:"type"`
	Text        string        `json:"text,omitempty"`
	Annotations []interface{} `json:"annotations,omitempty"`
}

type openaiResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type openaiRespChoice struct {
	Index        int                `json:"index"`
	Message      *openaiChatMessage `json:"message,omitempty"`
	Delta        *openaiStreamDelta `json:"delta,omitempty"`
	FinishReason *string            `json:"finish_reason"`
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
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openaiStreamChoice `json:"choices"`
}

type openaiStreamChoice struct {
	Index        int                `json:"index"`
	Delta        *openaiStreamDelta `json:"delta"`
	FinishReason *string            `json:"finish_reason"`
}

type openaiModelsResponse struct {
	Object string            `json:"object"`
	Data   []openaiModelData `json:"data"`
}

type openaiModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIAdapter implements ProtocolAdapter for OpenAI Chat Completions API.
type OpenAIAdapter struct{}
type OpenAIResponsesAdapter struct{ OpenAIAdapter }

type OpenAIResponsesStreamState struct {
	ResponseID   string
	MessageID    string
	Created      bool
	MessageAdded bool
	TextStarted  bool
	TextDone     bool
	Completed    bool
}

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

func (a *OpenAIAdapter) ParseResponsesRequest(body []byte) (*InternalRequest, error) {
	var req openaiResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI responses request: %w", err)
	}

	internal := &InternalRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxOutputTokens,
		Stream:      req.Stream,
	}
	if strings.TrimSpace(req.Instructions) != "" {
		internal.System = strings.TrimSpace(req.Instructions)
	}

	internal.Messages = append(internal.Messages, parseResponsesInput(req.Input)...)
	for _, tool := range req.Tools {
		if internalTool, ok := convertResponsesTool(tool); ok {
			internal.Tools = append(internal.Tools, internalTool)
		}
	}

	return internal, nil
}

func (a *OpenAIAdapter) BuildResponsesRequest(req *InternalRequest) ([]byte, string, error) {
	responsesReq := openaiResponsesRequest{
		Model:           req.Model,
		Instructions:    req.System,
		Temperature:     req.Temperature,
		MaxOutputTokens: req.MaxTokens,
		Stream:          req.Stream,
	}

	input := make([]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case "tool":
			for _, block := range msg.Content {
				if block.Type == "tool_result" && block.ToolResult != nil {
					input = append(input, map[string]interface{}{
						"type":    "function_call_output",
						"call_id": block.ToolResult.ToolUseID,
						"output":  block.ToolResult.Content,
					})
				}
			}
		case "system":
			if responsesReq.Instructions == "" {
				var parts []string
				for _, block := range msg.Content {
					if block.Type == "text" && block.Text != "" {
						parts = append(parts, block.Text)
					}
				}
				responsesReq.Instructions = strings.Join(parts, "\n")
			}
		default:
			content := make([]map[string]interface{}, 0, len(msg.Content))
			for _, block := range msg.Content {
				switch block.Type {
				case "text":
					if block.Text != "" {
						content = append(content, map[string]interface{}{
							"type": "input_text",
							"text": block.Text,
						})
					}
				case "image":
					url := block.ImageURL
					if block.Data != "" {
						url = "data:" + block.MimeType + ";base64," + block.Data
					}
					if url != "" {
						content = append(content, map[string]interface{}{
							"type":      "input_image",
							"image_url": url,
						})
					}
				case "tool_use":
					if block.ToolUse != nil {
						argsJSON, _ := json.Marshal(block.ToolUse.Arguments)
						input = append(input, map[string]interface{}{
							"type":      "function_call",
							"call_id":   block.ToolUse.ID,
							"name":      block.ToolUse.Name,
							"arguments": string(argsJSON),
						})
					}
				}
			}
			if len(content) > 0 {
				input = append(input, map[string]interface{}{
					"type":    "message",
					"role":    msg.Role,
					"content": content,
				})
			}
		}
	}
	if len(input) == 1 {
		if message, ok := input[0].(map[string]interface{}); ok && message["role"] == "user" {
			if content, ok := message["content"].([]map[string]interface{}); ok && len(content) == 1 && content[0]["type"] == "input_text" {
				responsesReq.Input = content[0]["text"]
			}
		}
	}
	if responsesReq.Input == nil {
		responsesReq.Input = input
	}

	for _, tool := range req.Tools {
		responsesReq.Tools = append(responsesReq.Tools, openaiResponsesToolDef{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	body, err := json.Marshal(responsesReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal OpenAI responses request: %w", err)
	}
	return body, "/responses", nil
}

func (a *OpenAIAdapter) BuildRequest(req *InternalRequest) ([]byte, string, error) {
	chatReq := openaiChatRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Stop:        req.Stop,
	}

	hasSystemMessage := false
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			hasSystemMessage = true
			break
		}
	}
	if req.System != "" && !hasSystemMessage {
		chatReq.Messages = append(chatReq.Messages, openaiChatMessage{
			Role:    "system",
			Content: req.System,
		})
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
					Type:    "tool_use",
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
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return json.Marshal(chatResp)
}

func (a *OpenAIAdapter) BuildResponsesResponse(resp *InternalResponse) ([]byte, error) {
	responsesResp := openaiResponsesResponse{
		ID:        resp.ID,
		Object:    "response",
		CreatedAt: time.Now().Unix(),
		Status:    "completed",
		Model:     resp.Model,
		Output:    buildResponsesOutputItems(resp),
	}
	if responsesResp.ID == "" {
		responsesResp.ID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	if resp.Usage != nil {
		responsesResp.Usage = &openaiResponsesUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}
	return json.Marshal(responsesResp)
}

func (a *OpenAIAdapter) ParseResponsesResponse(body []byte) (*InternalResponse, error) {
	var resp openaiResponsesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI responses response: %w", err)
	}

	internal := &InternalResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Choices: []InternalChoice{{
			Index:   0,
			Message: &InternalMessage{Role: "assistant"},
		}},
	}

	message := internal.Choices[0].Message
	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			if item.Role != "" {
				message.Role = item.Role
			}
			for _, part := range item.Content {
				switch part.Type {
				case "output_text", "text", "input_text":
					if part.Text != "" {
						message.Content = append(message.Content, ContentBlock{
							Type: "text",
							Text: part.Text,
						})
					}
				}
			}
		case "function_call":
			var args map[string]interface{}
			if item.Arguments != "" {
				_ = json.Unmarshal([]byte(item.Arguments), &args)
			}
			message.Content = append(message.Content, ContentBlock{
				Type: "tool_use",
				ToolUse: &ToolUse{
					ID:        item.CallID,
					Name:      item.Name,
					Arguments: args,
				},
			})
		}
	}

	if resp.Usage != nil {
		internal.Usage = &InternalUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return internal, nil
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

func (a *OpenAIAdapter) BuildResponsesStreamEvents(chunk *InternalStreamChunk, state *OpenAIResponsesStreamState) ([]string, error) {
	if state == nil {
		state = &OpenAIResponsesStreamState{}
	}
	if state.ResponseID == "" {
		state.ResponseID = chunk.ID
		if state.ResponseID == "" {
			state.ResponseID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
		}
	}
	if state.MessageID == "" {
		state.MessageID = state.ResponseID + "_msg_0"
	}

	events := make([]map[string]interface{}, 0, 8)
	if !state.Created {
		events = append(events, map[string]interface{}{
			"type": "response.created",
			"response": map[string]interface{}{
				"id":         state.ResponseID,
				"object":     "response",
				"created_at": time.Now().Unix(),
				"status":     "in_progress",
				"model":      chunk.Model,
				"output":     []interface{}{},
			},
		})
		state.Created = true
	}

	for _, choice := range chunk.Choices {
		if choice.Delta != nil {
			textDelta := ""
			for _, block := range choice.Delta.Content {
				if block.Type == "text" && block.Text != "" {
					textDelta += block.Text
				}
			}
			if textDelta != "" {
				if !state.MessageAdded {
					events = append(events, map[string]interface{}{
						"type":         "response.output_item.added",
						"output_index": 0,
						"item": openaiResponsesOutputItem{
							ID:      state.MessageID,
							Type:    "message",
							Status:  "in_progress",
							Role:    "assistant",
							Content: []openaiResponsesContentPart{},
						},
					})
					state.MessageAdded = true
				}
				if !state.TextStarted {
					events = append(events, map[string]interface{}{
						"type":          "response.content_part.added",
						"item_id":       state.MessageID,
						"output_index":  0,
						"content_index": 0,
						"part": openaiResponsesContentPart{
							Type:        "output_text",
							Text:        "",
							Annotations: []interface{}{},
						},
					})
					state.TextStarted = true
				}
				events = append(events, map[string]interface{}{
					"type":          "response.output_text.delta",
					"item_id":       state.MessageID,
					"output_index":  0,
					"content_index": 0,
					"delta":         textDelta,
				})
			}
			for idx, toolUse := range choice.Delta.ToolUses {
				argsJSON, _ := json.Marshal(toolUse.Arguments)
				events = append(events, map[string]interface{}{
					"type":         "response.output_item.added",
					"output_index": idx + 1,
					"item": openaiResponsesOutputItem{
						ID:        "fc_" + toolUse.ID,
						Type:      "function_call",
						Status:    "in_progress",
						CallID:    toolUse.ID,
						Name:      toolUse.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		if choice.FinishReason != "" && state.TextStarted && !state.TextDone {
			events = append(events, map[string]interface{}{
				"type":          "response.output_text.done",
				"item_id":       state.MessageID,
				"output_index":  0,
				"content_index": 0,
				"text":          "",
			})
			state.TextDone = true
		}
	}

	if chunk.StreamDone && !state.Completed {
		if state.TextStarted && !state.TextDone {
			events = append(events, map[string]interface{}{
				"type":          "response.output_text.done",
				"item_id":       state.MessageID,
				"output_index":  0,
				"content_index": 0,
				"text":          "",
			})
			state.TextDone = true
		}

		responsePayload := map[string]interface{}{
			"id":         state.ResponseID,
			"object":     "response",
			"created_at": time.Now().Unix(),
			"status":     "completed",
			"model":      chunk.Model,
		}
		if chunk.Usage != nil {
			responsePayload["usage"] = openaiResponsesUsage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}
		}
		events = append(events, map[string]interface{}{
			"type":     "response.completed",
			"response": responsePayload,
		})
		state.Completed = true
	}

	payloads := make([]string, 0, len(events))
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, string(data))
	}
	return payloads, nil
}

func (a *OpenAIAdapter) ParseResponsesStreamEvent(eventType, data string) (*InternalStreamChunk, error) {
	var event struct {
		Type         string                     `json:"type"`
		Delta        string                     `json:"delta,omitempty"`
		Text         string                     `json:"text,omitempty"`
		OutputIndex  int                        `json:"output_index,omitempty"`
		ContentIndex int                        `json:"content_index,omitempty"`
		Item         *openaiResponsesOutputItem `json:"item,omitempty"`
		Response     *struct {
			ID    string                `json:"id"`
			Model string                `json:"model"`
			Usage *openaiResponsesUsage `json:"usage,omitempty"`
		} `json:"response,omitempty"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI responses stream event: %w", err)
	}

	switch event.Type {
	case "response.output_text.delta":
		return &InternalStreamChunk{
			Choices: []InternalChoice{{
				Index: 0,
				Delta: &InternalDelta{
					Content: []ContentBlock{{Type: "text", Text: event.Delta}},
				},
			}},
		}, nil
	case "response.output_item.added":
		if event.Item != nil && event.Item.Type == "function_call" {
			var args map[string]interface{}
			if event.Item.Arguments != "" {
				_ = json.Unmarshal([]byte(event.Item.Arguments), &args)
			}
			return &InternalStreamChunk{
				Choices: []InternalChoice{{
					Index: 0,
					Delta: &InternalDelta{
						ToolUses: []ToolUse{{
							ID:        event.Item.CallID,
							Name:      event.Item.Name,
							Arguments: args,
						}},
					},
				}},
			}, nil
		}
		return &InternalStreamChunk{}, nil
	case "response.completed":
		chunk := &InternalStreamChunk{StreamDone: true}
		if event.Response != nil {
			chunk.ID = event.Response.ID
			chunk.Model = event.Response.Model
			if event.Response.Usage != nil {
				chunk.Usage = &InternalUsage{
					PromptTokens:     event.Response.Usage.InputTokens,
					CompletionTokens: event.Response.Usage.OutputTokens,
					TotalTokens:      event.Response.Usage.TotalTokens,
				}
			}
		}
		return chunk, nil
	default:
		return &InternalStreamChunk{}, nil
	}
}

func (a *OpenAIResponsesAdapter) ParseRequest(body []byte) (*InternalRequest, error) {
	return a.OpenAIAdapter.ParseResponsesRequest(body)
}

func (a *OpenAIResponsesAdapter) BuildRequest(req *InternalRequest) ([]byte, string, error) {
	return a.OpenAIAdapter.BuildResponsesRequest(req)
}

func (a *OpenAIResponsesAdapter) ParseResponse(body []byte) (*InternalResponse, error) {
	return a.OpenAIAdapter.ParseResponsesResponse(body)
}

func (a *OpenAIResponsesAdapter) BuildResponse(resp *InternalResponse) ([]byte, error) {
	return a.OpenAIAdapter.BuildResponsesResponse(resp)
}

func (a *OpenAIResponsesAdapter) ParseStreamEvent(eventType, data string) (*InternalStreamChunk, error) {
	return a.OpenAIAdapter.ParseResponsesStreamEvent(eventType, data)
}

func (a *OpenAIResponsesAdapter) BuildStreamEvent(chunk *InternalStreamChunk) (string, string, error) {
	events, err := a.OpenAIAdapter.BuildResponsesStreamEvents(chunk, &OpenAIResponsesStreamState{})
	if err != nil {
		return "", "", err
	}
	if len(events) == 0 {
		return "", "", nil
	}
	return "", events[0], nil
}

func (a *OpenAIResponsesAdapter) StreamDone() string { return "[DONE]" }

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

func parseResponsesInput(raw interface{}) []InternalMessage {
	switch v := raw.(type) {
	case nil:
		return nil
	case string:
		return []InternalMessage{{
			Role:    "user",
			Content: []ContentBlock{{Type: "text", Text: v}},
		}}
	case []interface{}:
		return parseResponsesInputArray(v)
	case map[string]interface{}:
		if msg := parseResponsesInputItem(v); msg != nil {
			return []InternalMessage{*msg}
		}
		return nil
	default:
		return []InternalMessage{{
			Role:    "user",
			Content: []ContentBlock{{Type: "text", Text: contentToString(v)}},
		}}
	}
}

func parseResponsesInputArray(items []interface{}) []InternalMessage {
	messages := make([]InternalMessage, 0, len(items))
	pendingUserBlocks := make([]ContentBlock, 0)

	flushPendingUser := func() {
		if len(pendingUserBlocks) == 0 {
			return
		}
		messages = append(messages, InternalMessage{
			Role:    "user",
			Content: pendingUserBlocks,
		})
		pendingUserBlocks = nil
	}

	for _, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			itemType := getStringValue(itemMap["type"])
			if getStringValue(itemMap["role"]) != "" || itemType == "message" || itemType == "function_call_output" || itemType == "function_call" {
				flushPendingUser()
				if msg := parseResponsesInputItem(itemMap); msg != nil {
					messages = append(messages, *msg)
				}
				continue
			}
		}
		pendingUserBlocks = append(pendingUserBlocks, parseResponsesContentBlocks(item)...)
	}

	flushPendingUser()
	return messages
}

func parseResponsesInputItem(item map[string]interface{}) *InternalMessage {
	itemType := getStringValue(item["type"])
	switch itemType {
	case "function_call_output":
		output := contentToString(item["output"])
		if output == "" {
			output = getStringValue(item["text"])
		}
		return &InternalMessage{
			Role: "tool",
			Content: []ContentBlock{{
				Type: "tool_result",
				ToolResult: &ToolResult{
					ToolUseID: getStringValue(item["call_id"]),
					Content:   output,
				},
			}},
		}
	case "function_call":
		var arguments map[string]interface{}
		switch v := item["arguments"].(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &arguments)
		case map[string]interface{}:
			arguments = v
		}
		return &InternalMessage{
			Role: "assistant",
			Content: []ContentBlock{{
				Type: "tool_use",
				ToolUse: &ToolUse{
					ID:        getStringValue(item["call_id"]),
					Name:      getStringValue(item["name"]),
					Arguments: arguments,
				},
			}},
		}
	default:
		role := getStringValue(item["role"])
		if role == "" {
			role = "user"
		}
		content := parseResponsesMessageContent(item["content"])
		if len(content) == 0 {
			content = parseResponsesContentBlocks(item)
		}
		if len(content) == 0 {
			return nil
		}
		return &InternalMessage{Role: role, Content: content}
	}
}

func parseResponsesMessageContent(raw interface{}) []ContentBlock {
	switch v := raw.(type) {
	case nil:
		return nil
	case string:
		return []ContentBlock{{Type: "text", Text: v}}
	case []interface{}:
		var blocks []ContentBlock
		for _, item := range v {
			blocks = append(blocks, parseResponsesContentBlocks(item)...)
		}
		return blocks
	default:
		return parseResponsesContentBlocks(v)
	}
}

func parseResponsesContentBlocks(raw interface{}) []ContentBlock {
	switch v := raw.(type) {
	case nil:
		return nil
	case string:
		return []ContentBlock{{Type: "text", Text: v}}
	case map[string]interface{}:
		partType := getStringValue(v["type"])
		switch partType {
		case "input_text", "output_text", "text":
			text := getStringValue(v["text"])
			if text != "" {
				return []ContentBlock{{Type: "text", Text: text}}
			}
		case "input_image", "image", "image_url":
			imageURL := getStringValue(v["image_url"])
			if imageURL == "" {
				if nested, ok := v["image_url"].(map[string]interface{}); ok {
					imageURL = getStringValue(nested["url"])
				}
			}
			if imageURL != "" {
				return []ContentBlock{{Type: "image", ImageURL: imageURL}}
			}
		}
		if text := getStringValue(v["text"]); text != "" {
			return []ContentBlock{{Type: "text", Text: text}}
		}
		return nil
	default:
		return []ContentBlock{{Type: "text", Text: contentToString(v)}}
	}
}

func convertResponsesTool(tool openaiResponsesToolDef) (InternalTool, bool) {
	switch tool.Type {
	case "", "function":
		if tool.Function != nil {
			return InternalTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			}, tool.Function.Name != ""
		}
		return InternalTool{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		}, tool.Name != ""
	default:
		return InternalTool{}, false
	}
}

func buildResponsesOutputItems(resp *InternalResponse) []openaiResponsesOutputItem {
	items := make([]openaiResponsesOutputItem, 0, len(resp.Choices))
	baseID := resp.ID
	if baseID == "" {
		baseID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}

	for idx, choice := range resp.Choices {
		if choice.Message == nil {
			continue
		}

		messageParts := make([]openaiResponsesContentPart, 0, len(choice.Message.Content))
		for _, block := range choice.Message.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					messageParts = append(messageParts, openaiResponsesContentPart{
						Type:        "output_text",
						Text:        block.Text,
						Annotations: []interface{}{},
					})
				}
			case "tool_use":
				if block.ToolUse == nil {
					continue
				}
				argsJSON, _ := json.Marshal(block.ToolUse.Arguments)
				items = append(items, openaiResponsesOutputItem{
					ID:        fmt.Sprintf("fc_%s_%d", baseID, idx),
					Type:      "function_call",
					Status:    "completed",
					CallID:    block.ToolUse.ID,
					Name:      block.ToolUse.Name,
					Arguments: string(argsJSON),
				})
			}
		}

		if len(messageParts) > 0 {
			role := choice.Message.Role
			if role == "" {
				role = "assistant"
			}
			items = append(items, openaiResponsesOutputItem{
				ID:      fmt.Sprintf("msg_%s_%d", baseID, idx),
				Type:    "message",
				Status:  "completed",
				Role:    role,
				Content: messageParts,
			})
		}
	}

	return items
}

func getStringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}
