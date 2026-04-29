package openai_chat

// ResponseMessage 表示响应消息。
type ResponseMessage struct {
	Role       RoleType        `json:"role"`                 // 角色
	Content    any             `json:"content"`              // 内容 string | MediaContent
	Name       string          `json:"name"`                 // 名称
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"` // 工具调用
	ToolCallID string          `json:"tool_call_id"`         // 工具调用ID
	Reasoning  []ReasoningPart `json:"reasoning,omitempty"`  // 推理内容
}

// ResponseChoice 表示响应选择。
type ResponseChoice struct {
	Index        int             `json:"index"`         // 选择索引
	Message      ResponseMessage `json:"message"`       // 消息
	FinishReason string          `json:"finish_reason"` // 完成原因
}

// ResponsePromptTokensDetails 表示提示词 Token 数详情。
type ResponsePromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"` // 缓存 Token 数
	TextTokens   int `json:"text_tokens"`   // 文本 Token 数
	AudioTokens  int `json:"audio_tokens"`  // 音频 Token 数
	ImageTokens  int `json:"image_tokens"`  // 图片 Token 数
}

// ResponseCompletionTokensDetails 表示补全 Token 数详情。
type ResponseCompletionTokensDetails struct {
	TextTokens      int `json:"text_tokens"`      // 文本 Token 数
	AudioTokens     int `json:"audio_tokens"`     // 音频 Token 数
	ReasoningTokens int `json:"reasoning_tokens"` // 推理 Token 数
}

// Usage 表示使用情况。
type ResponseUsage struct {
	PromptTokens            int                             `json:"prompt_tokens"`             // 提示词 Token 数
	CompletionTokens        int                             `json:"completion_tokens"`         // 补全 Token 数
	TotalTokens             int                             `json:"total_tokens"`              // 总 Token 数
	PromptTokensDetails     ResponsePromptTokensDetails     `json:"prompt_tokens_details"`     // 提示词 Token 数详情
	CompletionTokensDetails ResponseCompletionTokensDetails `json:"completion_tokens_details"` // 补全 Token 数详情
}

// Response 表示响应。
type Response struct {
	ID                string           `json:"id"`                 // 响应ID
	Object            string           `json:"object"`             // 响应对象
	Created           int64            `json:"created"`            // 创建时间
	Model             string           `json:"model"`              // 模型
	Choices           []ResponseChoice `json:"choices"`            // 响应选择
	Usage             ResponseUsage    `json:"usage"`              // 使用情况
	SystemFingerprint string           `json:"system_fingerprint"` // 系统指纹
}
