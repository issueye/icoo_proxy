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

// Gemini request/response types

type geminiRequest struct {
	Contents         []geminiContent       `json:"contents"`
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []geminiToolDeclaration `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string                    `json:"text,omitempty"`
	InlineData   *geminiInlineData         `json:"inlineData,omitempty"`
	FunctionCall *geminiFunctionCall       `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiToolDeclaration struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Gemini response types

type geminiResponse struct {
	Candidates     []geminiCandidate    `json:"candidates"`
	UsageMetadata  *geminiUsageMetadata `json:"usageMetadata,omitempty"`
	ModelVersion   string               `json:"modelVersion,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	Index         int           `json:"index,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// Gemini stream response (array of response chunks)
type geminiStreamChunk struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// Gemini models response
type geminiModelsResponse struct {
	Models []geminiModelData `json:"models"`
}

type geminiModelData struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// GeminiAdapter implements ProtocolAdapter for Google Gemini API.
type GeminiAdapter struct{}

func (a *GeminiAdapter) ParseRequest(body []byte) (*InternalRequest, error) {
	var req geminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini request: %w", err)
	}

	internal := &InternalRequest{Stream: false}
	if req.GenerationConfig != nil {
		internal.Temperature = req.GenerationConfig.Temperature
		internal.MaxTokens = req.GenerationConfig.MaxOutputTokens
		internal.Stop = req.GenerationConfig.StopSequences
	}

	// Extract system instruction
	if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) > 0 {
		internal.System = req.SystemInstruction.Parts[0].Text
	}

	// Extract model from context (Gemini puts model in URL, not body)
	// We'll handle this at the handler level

	for _, content := range req.Contents {
		role := "user"
		if content.Role == "model" {
			role = "assistant"
		}

		im := InternalMessage{Role: role}
		for _, part := range content.Parts {
			if part.Text != "" {
				im.Content = append(im.Content, ContentBlock{Type: "text", Text: part.Text})
			}
			if part.InlineData != nil {
				im.Content = append(im.Content, ContentBlock{
					Type:     "image",
					MimeType: part.InlineData.MimeType,
					Data:     part.InlineData.Data,
				})
			}
			if part.FunctionCall != nil {
				im.Content = append(im.Content, ContentBlock{
					Type: "tool_use",
					ToolUse: &ToolUse{
						ID:        fmt.Sprintf("call_%s", part.FunctionCall.Name),
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					},
				})
			}
			if part.FunctionResponse != nil {
				im.Role = "tool"
				resultContent := ""
				if b, err := json.Marshal(part.FunctionResponse.Response); err == nil {
					resultContent = string(b)
				}
				im.Content = append(im.Content, ContentBlock{
					Type: "tool_result",
					ToolResult: &ToolResult{
						ToolUseID: fmt.Sprintf("call_%s", part.FunctionResponse.Name),
						Content:   resultContent,
					},
				})
			}
		}
		internal.Messages = append(internal.Messages, im)
	}

	for _, tool := range req.Tools {
		for _, decl := range tool.FunctionDeclarations {
			internal.Tools = append(internal.Tools, InternalTool{
				Name:        decl.Name,
				Description: decl.Description,
				Parameters:  decl.Parameters,
			})
		}
	}

	return internal, nil
}

func (a *GeminiAdapter) BuildRequest(req *InternalRequest) ([]byte, string, error) {
	gr := geminiRequest{}

	// System instruction
	if req.System != "" {
		gr.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	// Generation config
	if req.Temperature != nil || req.MaxTokens != nil || len(req.Stop) > 0 {
		gr.GenerationConfig = &geminiGenerationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
			StopSequences:   req.Stop,
		}
	}

	// Messages
	for _, msg := range req.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		// Tool results need special handling - use functionResponse part
		if msg.Role == "tool" {
			for _, block := range msg.Content {
				if block.Type == "tool_result" && block.ToolResult != nil {
					var resp interface{}
					json.Unmarshal([]byte(block.ToolResult.Content), &resp)
					gr.Contents = append(gr.Contents, geminiContent{
						Role: "user", // Gemini expects function responses in user turn
						Parts: []geminiPart{{
							FunctionResponse: &geminiFunctionResponse{
								Name:     strings.TrimPrefix(block.ToolResult.ToolUseID, "call_"),
								Response: resp,
							},
						}},
					})
				}
			}
			continue
		}

		var parts []geminiPart
		var toolCalls []geminiPart

		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				parts = append(parts, geminiPart{Text: block.Text})
			case "image":
				if block.Data != "" {
					parts = append(parts, geminiPart{
						InlineData: &geminiInlineData{
							MimeType: block.MimeType,
							Data:     block.Data,
						},
					})
				} else if block.ImageURL != "" {
					parts = append(parts, geminiPart{Text: block.ImageURL})
				}
			case "tool_use":
				if block.ToolUse != nil {
					toolCalls = append(toolCalls, geminiPart{
						FunctionCall: &geminiFunctionCall{
							Name: block.ToolUse.Name,
							Args: block.ToolUse.Arguments,
						},
					})
				}
			}
		}

		if len(toolCalls) > 0 {
			parts = append(parts, toolCalls...)
		}

		if len(parts) > 0 {
			gr.Contents = append(gr.Contents, geminiContent{Role: role, Parts: parts})
		}
	}

	// Tools
	for _, t := range req.Tools {
		gr.Tools = append(gr.Tools, geminiToolDeclaration{
			FunctionDeclarations: []geminiFunctionDecl{{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			}},
		})
	}

	body, err := json.Marshal(gr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	// Build URL path: /models/{model}:generateContent or streamGenerateContent
	action := "generateContent"
	if req.Stream {
		action = "streamGenerateContent"
	}
	path := fmt.Sprintf("/models/%s:%s", req.Model, action)

	return body, path, nil
}

func (a *GeminiAdapter) ParseResponse(body []byte) (*InternalResponse, error) {
	var resp geminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	internal := &InternalResponse{}

	for _, candidate := range resp.Candidates {
		im := &InternalMessage{Role: "assistant"}
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				im.Content = append(im.Content, ContentBlock{Type: "text", Text: part.Text})
			}
			if part.FunctionCall != nil {
				im.Content = append(im.Content, ContentBlock{
					Type: "tool_use",
					ToolUse: &ToolUse{
						ID:        fmt.Sprintf("call_%s", part.FunctionCall.Name),
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					},
				})
			}
		}

		finishReason := candidate.FinishReason
		if finishReason == "STOP" {
			finishReason = "stop"
		}

		internal.Choices = append(internal.Choices, InternalChoice{
			Index:        candidate.Index,
			Message:      im,
			FinishReason: strings.ToLower(finishReason),
		})
	}

	if resp.UsageMetadata != nil {
		internal.Usage = &InternalUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return internal, nil
}

func (a *GeminiAdapter) BuildResponse(resp *InternalResponse) ([]byte, error) {
	gr := geminiResponse{}

	for _, choice := range resp.Choices {
		if choice.Message == nil {
			continue
		}
		var parts []geminiPart
		for _, block := range choice.Message.Content {
			switch block.Type {
			case "text":
				parts = append(parts, geminiPart{Text: block.Text})
			case "tool_use":
				if block.ToolUse != nil {
					parts = append(parts, geminiPart{
						FunctionCall: &geminiFunctionCall{
							Name: block.ToolUse.Name,
							Args: block.ToolUse.Arguments,
						},
					})
				}
			}
		}

		finishReason := strings.ToUpper(choice.FinishReason)
		if finishReason == "STOP" || finishReason == "" {
			finishReason = "STOP"
		}

		gr.Candidates = append(gr.Candidates, geminiCandidate{
			Content:      geminiContent{Role: "model", Parts: parts},
			FinishReason: finishReason,
			Index:        choice.Index,
		})
	}

	return json.Marshal(gr)
}

func (a *GeminiAdapter) ParseStreamEvent(eventType, data string) (*InternalStreamChunk, error) {
	// Gemini stream format: each line is a JSON object (no event type prefix)
	var chunk geminiStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini stream chunk: %w", err)
	}

	internal := &InternalStreamChunk{}

	for _, candidate := range chunk.Candidates {
		delta := &InternalDelta{}
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				delta.Content = append(delta.Content, ContentBlock{Type: "text", Text: part.Text})
			}
			if part.FunctionCall != nil {
				delta.ToolUses = append(delta.ToolUses, ToolUse{
					ID:        fmt.Sprintf("call_%s", part.FunctionCall.Name),
					Name:      part.FunctionCall.Name,
					Arguments: part.FunctionCall.Args,
				})
			}
		}

		finishReason := ""
		if candidate.FinishReason != "" {
			finishReason = strings.ToLower(candidate.FinishReason)
			if finishReason == "stop" {
				finishReason = "stop"
			}
			if finishReason == "STOP" {
				finishReason = "stop"
			}
		}

		internal.Choices = append(internal.Choices, InternalChoice{
			Index:        candidate.Index,
			Delta:        delta,
			FinishReason: finishReason,
		})
	}

	// Check if stream is done (last chunk has finishReason)
	if len(chunk.Candidates) > 0 && chunk.Candidates[0].FinishReason == "STOP" {
		// Stream may still continue with other data
	}

	return internal, nil
}

func (a *GeminiAdapter) BuildStreamEvent(chunk *InternalStreamChunk) (string, string, error) {
	if chunk.StreamDone {
		return "", "", nil // Gemini doesn't have explicit done marker
	}

	gr := geminiStreamChunk{}
	for _, choice := range chunk.Choices {
		var parts []geminiPart
		if choice.Delta != nil {
			for _, block := range choice.Delta.Content {
				if block.Type == "text" && block.Text != "" {
					parts = append(parts, geminiPart{Text: block.Text})
				}
			}
			for _, tu := range choice.Delta.ToolUses {
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{Name: tu.Name, Args: tu.Arguments},
				})
			}
		}

		finishReason := ""
		if choice.FinishReason != "" {
			finishReason = strings.ToUpper(choice.FinishReason)
		}

		gr.Candidates = append(gr.Candidates, geminiCandidate{
			Content:      geminiContent{Role: "model", Parts: parts},
			FinishReason: finishReason,
			Index:        choice.Index,
		})
	}

	data, err := json.Marshal(gr)
	if err != nil {
		return "", "", err
	}
	return "", string(data), nil
}

func (a *GeminiAdapter) StreamDone() string { return "" }

func (a *GeminiAdapter) BuildHTTPRequest(ctx context.Context, apiBase, apiKey, method, path string, body []byte) (*http.Request, error) {
	url := strings.TrimRight(apiBase, "/") + path
	if apiKey != "" {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		url += sep + "key=" + apiKey
	}
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *GeminiAdapter) ListModelsRequest(ctx context.Context, apiBase, apiKey string) (*http.Request, error) {
	url := strings.TrimRight(apiBase, "/") + "/models"
	if apiKey != "" {
		url += "?key=" + apiKey
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (a *GeminiAdapter) ParseModelsResponse(body []byte) ([]ModelInfo, error) {
	var resp geminiModelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini models response: %w", err)
	}
	models := make([]ModelInfo, 0, len(resp.Models))
	for _, m := range resp.Models {
		// Gemini model names are like "models/gemini-pro", extract just the name part
		name := m.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		displayName := m.DisplayName
		if displayName == "" {
			displayName = name
		}
		models = append(models, ModelInfo{ID: name, Name: displayName})
	}
	return models, nil
}
