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

// ResponseContent 响应内容
type ResponseContent struct {
	Type string `json:"type"` // 内容类型
	Text string `json:"text"` // 文本内容
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
	ID         string          `json:"id"`          // 响应 ID
	Type       string          `json:"type"`        // 响应类型
	Role       string          `json:"role"`        // 响应角色
	Content    ResponseContent `json:"content"`     // 响应内容
	Model      string          `json:"model"`       // 模型名称
	StopReason StopReason      `json:"stop_reason"` // 停止原因 "end_turn" | "max_tokens" | "stop_sequence" | "tool_use"
	Usage      ResponseUsage   `json:"usage"`       // 响应使用情况
}
