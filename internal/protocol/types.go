package protocol

// InternalRequest is the unified internal representation of an AI chat request.
// All protocol adapters convert to/from this format.
type InternalRequest struct {
	Model       string          `json:"model"`
	Messages    []InternalMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []InternalTool  `json:"tools,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

// InternalMessage represents a single message in the conversation.
type InternalMessage struct {
	Role    string          `json:"role"` // system, user, assistant, tool
	Content []ContentBlock  `json:"content"`
}

// ContentBlock represents a single content block within a message.
type ContentBlock struct {
	Type     string       `json:"type"` // text, image, tool_use, tool_result
	Text     string       `json:"text,omitempty"`
	ImageURL string       `json:"image_url,omitempty"`
	MimeType string       `json:"mime_type,omitempty"`
	Data     string       `json:"data,omitempty"` // base64 image data
	ToolUse  *ToolUse     `json:"tool_use,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolUse represents a tool call in the response.
type ToolUse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// InternalTool represents a tool definition.
type InternalTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// InternalResponse is the unified internal representation of an AI chat response.
type InternalResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []InternalChoice   `json:"choices"`
	Usage   *InternalUsage     `json:"usage,omitempty"`
}

// InternalChoice represents a single choice in the response.
type InternalChoice struct {
	Index        int             `json:"index"`
	Message      *InternalMessage `json:"message,omitempty"`
	Delta        *InternalDelta  `json:"delta,omitempty"`
	FinishReason string          `json:"finish_reason,omitempty"`
}

// InternalDelta represents a streaming chunk delta.
type InternalDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   []ContentBlock  `json:"content,omitempty"`
	ToolUses  []ToolUse       `json:"tool_uses,omitempty"`
}

// InternalUsage represents token usage information.
type InternalUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// InternalStreamChunk represents a single SSE streaming chunk.
type InternalStreamChunk struct {
	ID           string          `json:"id"`
	Model        string          `json:"model"`
	Choices      []InternalChoice `json:"choices"`
	Usage        *InternalUsage  `json:"usage,omitempty"`
	StreamDone   bool            `json:"stream_done,omitempty"`
}

// ModelInfo represents a model's basic information.
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	OwnedBy  string `json:"owned_by,omitempty"`
}
