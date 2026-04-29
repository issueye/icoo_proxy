package anthropic

import "encoding/json"

// RequestSource 表示内容来源。
type RequestSource struct {
	Type      string `json:"type,omitempty"`       // 来源类型 base64 url
	MediaType string `json:"media_type,omitempty"` // 媒体类型 image/jpeg image/png
	Data      string `json:"data,omitempty"`       // base64 编码的媒体数据
	URL       string `json:"url,omitempty"`        // 媒体 URL
}

// RequestContent 表示聊天内容
type RequestContent struct {
	Type      ContentType     `json:"type"`                  // 内容类型
	Text      string          `json:"text,omitempty"`        // 文本内容
	Source    RequestSource   `json:"source,omitempty"`      // 来源信息
	ID        string          `json:"id,omitempty"`          // 块 ID 或 tool_use ID
	Name      string          `json:"name,omitempty"`        // 工具名称
	Input     json.RawMessage `json:"input,omitempty"`       // tool_use 输入参数
	ToolUseID string          `json:"tool_use_id,omitempty"` // 关联的 tool_use ID
	Content   string          `json:"content,omitempty"`     // tool_result 内容
}

// RequestMessage 表示单条 Anthropic 消息。
type RequestMessage struct {
	Role    string         `json:"role"`    // 消息角色 "user" | "assistant"
	Content RequestContent `json:"content"` // 消息内容，可能为字符串或块数组
}

// RequestToolChoice 表示工具选择策略。
type RequestToolChoice struct {
	Type string `json:"type"` // 工具选择类型 "auto" | "any" | "tool"
	Name string `json:"name"` // 工具名称
}

// RequestThinking 表示 thinking 配置。
type RequestThinking struct {
	Type         string `json:"type"`          // thinking 类型 "enabled" | "disabled"
	BudgetTokens int    `json:"budget_tokens"` // 总 token 数
}

type RequestToolInputSchema struct {
	Type       string         `json:"type"`                 // 输入参数类型
	Properties map[string]any `json:"properties,omitempty"` // 输入参数属性
	Required   []string       `json:"required,omitempty"`   // 必填参数
}

// RequestTool 表示 Anthropic 工具定义。
type RequestTool struct {
	Name        string                 `json:"name"`                   // 工具名称
	Description string                 `json:"description,omitempty"`  // 工具说明
	InputSchema RequestToolInputSchema `json:"input_schema,omitempty"` // 输入参数 Schema
}

// RequestMessagesRequest 表示 Anthropic Messages 请求体。
type RequestMessagesRequest struct {
	Model         string            `json:"model"`                    // 目标模型名称
	Messages      []RequestMessage  `json:"messages"`                 // 消息列表
	System        any               `json:"system,omitempty"`         // 系统提示，可能为字符串或块数组
	MaxTokens     int               `json:"max_tokens"`               // 最大输出 token 数
	Temperature   *float64          `json:"temperature,omitempty"`    // 采样温度
	TopP          *float64          `json:"top_p,omitempty"`          // nucleus sampling 参数
	TopK          int               `json:"top_k,omitempty"`          // top-k 采样参数
	Stream        bool              `json:"stream,omitempty"`         // 是否启用流式输出
	StopSequences []string          `json:"stop_sequences,omitempty"` // 自定义停止序列
	Tools         []RequestTool     `json:"tools,omitempty"`          // 可用工具定义列表
	ToolChoice    RequestToolChoice `json:"tool_choice,omitempty"`    // 工具选择策略
	Thinking      *RequestThinking  `json:"thinking,omitempty"`       // thinking 配置
}
