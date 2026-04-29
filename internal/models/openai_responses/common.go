package openai_responses

type TruncationType string

const (
	TruncationTypeAuto     TruncationType = "auto"
	TruncationTypeDisabled TruncationType = "disabled"
)

func (t TruncationType) ToString() string {
	return string(t)
}

// user、assistant、system 或 developer
type RoleType string

const (
	RoleTypeUser      RoleType = "user"
	RoleTypeAssistant RoleType = "assistant"
	RoleTypeSystem    RoleType = "system"
	RoleTypeDeveloper RoleType = "developer"
)

func (r RoleType) ToString() string {
	return string(r)
}

// input_text | input_image | input_file
type InputType string

const (
	InputTypeText  InputType = "input_text"
	InputTypeImage InputType = "input_image"
	InputTypeFile  InputType = "input_file"
)

func (i InputType) ToString() string {
	return string(i)
}

// message | function_call
type OutputType string

const (
	OutputTypeMessage      OutputType = "message"
	OutputTypeFunctionCall OutputType = "function_call"
)

func (o OutputType) ToString() string {
	return string(o)
}

// web_search_preview | function
type ToolFunctionType string

const (
	ToolFunctionTypeWebSearchPreview ToolFunctionType = "web_search_preview"
	ToolFunctionTypeFunction         ToolFunctionType = "function"
)

func (t ToolFunctionType) ToString() string {
	return string(t)
}

// "completed" | "failed" | "in_progress" | "incomplete"
type ResponseStatus string

const (
	ResponseStatusCompleted  ResponseStatus = "completed"
	ResponseStatusFailed     ResponseStatus = "failed"
	ResponseStatusInProgress ResponseStatus = "in_progress"
	ResponseStatusIncomplete ResponseStatus = "incomplete"
)

func (s ResponseStatus) ToString() string {
	return string(s)
}

type ToolParameter struct {
	Type       string         `json:"type"`       // 参数类型 "object"
	Name       string         `json:"name"`       // 参数名称
	Properties map[string]any `json:"properties"` // 参数属性
	Required   []string       `json:"required"`   // 是否必填
}

type Tool struct {
	Type        ToolFunctionType `json:"type"`                  // 工具类型
	Name        string           `json:"name"`                  // 工具名称
	Description string           `json:"description"`           // 工具描述
	Parameters  []ToolParameter  `json:"parameters"`            // 工具参数
	Strict      bool             `json:"strict" default:"true"` // 是否严格模式 默认 true
}
