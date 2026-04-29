package anthropic

// StopReason 停止原因
type StopReason string

const (
	StopReasonEndTurn      StopReason = "end_turn"
	StopReasonMaxTokens    StopReason = "max_tokens"
	StopReasonStopSequence StopReason = "stop_sequence"
	StopReasonToolUse      StopReason = "tool_use"
)

func (s StopReason) ToString() string {
	return string(s)
}

// ResponseContentBlock 表示 Anthropic 响应内容块。
type ResponseContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      string `json:"data,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`
}

// ResponseUsage 响应使用情况
type ResponseUsage struct {
	InputTokens              int `json:"input_tokens"`                // 输入令牌数
	OutputTokens             int `json:"output_tokens"`               // 输出令牌数
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"` // 缓存创建输入令牌数
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`     // 缓存读取输入令牌数
}

// ResponseBody 响应体
type ResponseBody struct {
	ID         string                 `json:"id"`          // 响应 ID
	Type       string                 `json:"type"`        // 响应类型
	Role       string                 `json:"role"`        // 响应角色
	Content    []ResponseContentBlock `json:"content"`     // 响应内容块
	Model      string                 `json:"model"`       // 模型名称
	StopReason StopReason             `json:"stop_reason"` // 停止原因 "end_turn" | "max_tokens" | "stop_sequence" | "tool_use"
	Usage      ResponseUsage          `json:"usage"`       // 响应使用情况
}
