package openai_responses

// InputContent 输入内容
type InputContent struct {
	Type     InputType `json:"type"`                // 内容类型
	Text     string    `json:"text,omitempty"`      // 文本内容 input_text
	ImageUrl string    `json:"image_url,omitempty"` // 图片 URL input_image
	Detail   string    `json:"detail,omitempty"`    // 详细内容 input_image
	FileID   string    `json:"file_id,omitempty"`   // 文件 ID input_image | input_file
	FileData string    `json:"file_data,omitempty"` // 文件数据 input_file
	FileName string    `json:"file_name,omitempty"` // 文件名 input_file
}

// InputMessage 输入消息
type InputMessage struct {
	Type    string       `json:"type" default:"message"` // 内容类型 "message"
	Role    RoleType     `json:"role"`                   // 角色
	Content InputContent `json:"content"`                // 内容
}

// RequestReasoning 请求原因参数
type RequestReasoning struct {
	Effort  string `json:"effort"`  // 原因努力 "low" | "medium" | "high"
	Summary string `json:"summary"` // 摘要
}

// RequestBody 请求体
type RequestBody struct {
	Model              string           `json:"model"`                // 模型名称
	Input              any              `json:"input"`                // 输入内容 string | []InputMessage
	Instruction        string           `json:"instruction"`          // 指令
	MaxOutputTokens    int              `json:"max_output_tokens"`    // 最大输出 token 数
	Temperature        float64          `json:"temperature"`          // 温度参数
	TopP               float64          `json:"top_p"`                // Top-P 参数
	Stream             bool             `json:"stream"`               // 是否流式输出
	Tools              []Tool           `json:"tools"`                // 工具列表
	ToolChoice         any              `json:"tool_choice"`          // 工具选择参数
	Reasoning          RequestReasoning `json:"reasoning"`            // 原因参数
	PreviousResponseID string           `json:"previous_response_id"` // 上一个响应 ID
	Truncation         TruncationType   `json:"truncation"`           // 截断参数 "auto" | "disabled"
}
