package openai_responses

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

// ResponseOutputContent 表示输出内容。
type ResponseOutputContent struct {
	Type string `json:"type"`           // 内容类型
	Text string `json:"text,omitempty"` // 内容文本
}

// ResponseOutput 表示输出。
type ResponseOutput struct {
	Type      OutputType              `json:"type"`                // 输出类型
	ID        string                  `json:"id,omitempty"`        // 输出ID
	Status    ResponseStatus          `json:"status,omitempty"`    // 状态码
	Role      RoleType                `json:"role,omitempty"`      // 角色
	Content   []ResponseOutputContent `json:"content,omitempty"`   // 输出内容
	Reasoning []ReasoningPart         `json:"reasoning,omitempty"` // 推理内容
	CallID    string                  `json:"call_id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	Arguments string                  `json:"arguments,omitempty"`
	Output    any                     `json:"output,omitempty"`
}

// ResponseBody 表示响应体。
type ResponseBody struct {
	ID        string           `json:"id"`         // 响应ID
	Object    string           `json:"object"`     // 响应对象
	CreatedAt int64            `json:"created_at"` // 创建时间
	Status    ResponseStatus   `json:"status"`     // 状态码
	Model     string           `json:"model"`      // 模型
	Output    []ResponseOutput `json:"output"`     // 输出
	Usage     ResponseUsage    `json:"usage"`      // 使用情况
}
