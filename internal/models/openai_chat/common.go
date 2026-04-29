package openai_chat

// "system" | "user" | "assistant" | "tool" | "developer"
type RoleType string

const (
	RoleTypeUser      RoleType = "user"
	RoleTypeAssistant RoleType = "assistant"
	RoleTypeSystem    RoleType = "system"
	RoleTypeTool      RoleType = "tool"
	RoleTypeDeveloper RoleType = "developer"
)

func (r RoleType) ToString() string {
	return string(r)
}

// RequestInputAudio 表示输入音频。
type InputAudio struct {
	Data   string `json:"data"`   // 输入音频URL
	Format string `json:"format"` // 输入音频格式 "mp3" | "wav"
}

type VideoURL struct {
	Url string `json:"url"` // 视频URL
}

// File 表示文件。
type File struct {
	FileName string `json:"file_name"` // 文件名
	FileData string `json:"file_data"` // 文件数据
	FileID   string `json:"file_id"`   // 文件ID
}

type ImageURL struct {
	Url    string `json:"url"`    // 图片URL
	Detail string `json:"detail"` // 图片详情 "low" | "high" | "auto"
}

// MediaContent 表示媒体内容。
type MediaContent struct {
	Type       RequestMediaType `json:"type"`        // 媒体类型
	Text       string           `json:"text"`        // 文本内容
	ImageURL   ImageURL         `json:"image_url"`   // 图片URL
	InputAudio InputAudio       `json:"input_audio"` // 输入音频URL
	VideoURL   VideoURL         `json:"video_url"`   // 视频URL
	File       File             `json:"file"`        // 文件
}

// Function 表示函数。
type Function struct {
	Name      string `json:"name"`      // 工具名称
	Arguments string `json:"arguments"` // 工具调用参数
}

// ToolCall 表示工具调用。
type ToolCall struct {
	ID       string   `json:"id"`       // 工具调用ID
	Type     string   `json:"type"`     // 工具调用类型
	Function Function `json:"function"` // 工具调用函数
}
