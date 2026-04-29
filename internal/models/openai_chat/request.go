package openai_chat

// "text" | "image_url" | "input_audio" | "file" | "video_url"
type RequestMediaType string

const (
	RequestMediaTypeText     RequestMediaType = "text"
	RequestMediaTypeImage    RequestMediaType = "image"
	RequestMediaTypeVideo    RequestMediaType = "video"
	RequestMediaTypeAudio    RequestMediaType = "audio"
	RequestMediaTypeFile     RequestMediaType = "file"
	RequestMediaTypeVideoURL RequestMediaType = "video_url"
)

func (r RequestMediaType) ToString() string {
	return string(r)
}

// ReasoningPart 表示 reasoning 内容块。
type ReasoningPart struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      string `json:"data,omitempty"`
}

// RequestMessage 表示请求消息。
type RequestMessage struct {
	Role       RoleType        `json:"role"`                 // 角色
	Name       string          `json:"name"`                 // 工具名称
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"` // 工具调用列表
	ToolCallID string          `json:"tool_call_id"`         // 工具调用ID
	Reasoning  []ReasoningPart `json:"reasoning,omitempty"`  // 推理内容
	Content    any             `json:"content"`              // 内容 string | MediaContent
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"` // 是否包含使用统计
}

type RequestTool struct {
	Name        string         `json:"name"`        // 工具名称
	Description string         `json:"description"` // 工具描述
	Parameters  map[string]any `json:"parameters"`  // 工具参数
}

// ResponseFormat 表示响应格式。
type ResponseFormat struct {
	Type       string `json:"type"`        // 响应格式类型 "text" | "json_object" | "json_schema"
	JsonSchema any    `json:"json_schema"` // JSON模式 (仅当类型为"json_schema"时有效)
}

// RequestToolChoice 表示工具选择。
type RequestToolChoice struct {
	Type     string   `json:"type"`     // 工具选择类型
	Function Function `json:"function"` // 工具调用函数
}

// RequestAudio 表示音频。
type RequestAudio struct {
	Voice  string `json:"voice"`  // 语音类型
	Format string `json:"format"` // 语音格式 "mp3" | "wav"
}

// ReqeustBody 表示请求体。
type ReqeustBody struct {
	Model               string             `json:"model"`                 // 模型
	Messages            []RequestMessage   `json:"messages"`              // 消息列表
	Temperature         float64            `json:"temperature"`           // 温度参数
	TopP                float64            `json:"top_p"`                 // TopP参数
	N                   int                `json:"n"`                     // 生成数量
	Stream              bool               `json:"stream"`                // 是否流式输出
	StreamOptions       StreamOptions      `json:"stream_options"`        // 流式输出选项
	Stop                string             `json:"stop"`                  // 停止序列
	MaxTokens           int                `json:"max_tokens"`            // 最大令牌数
	MaxCompletionTokens int                `json:"max_completion_tokens"` // 最大完成令牌数
	PresencePenalty     float64            `json:"presence_penalty"`      // 孺在惩罚
	FrequencyPenalty    float64            `json:"frequency_penalty"`     // 频率惩罚
	LogitBias           map[string]float64 `json:"logit_bias"`            // Logit偏置
	User                string             `json:"user"`                  // 用户ID
	Tools               []RequestTool      `json:"tools"`                 // 工具列表
	ToolChoice          any                `json:"tool_choice"`           // 工具选择 string | RequestToolChoice
	ResponseFormat      ResponseFormat     `json:"response_format"`       // 响应格式
	Seed                int                `json:"seed"`                  // 随机种子
	ReasoningEffort     string             `json:"reasoning_effort"`      // 推理强度 (用于支持推理的模型) "low" | "medium" | "high"
	Modalities          []string           `json:"modalities"`            // 模态列表
	Audio               RequestAudio       `json:"audio"`                 // 音频参数
}
