package anthropic

type RoleType string

const (
	RoleTypeUser      RoleType = "user"
	RoleTypeAssistant RoleType = "assistant"
)

func (r RoleType) ToString() string {
	return string(r)
}

// "text" | "image" | "tool_use" | "tool_result"
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolResult ContentType = "tool_result"
)

func (c ContentType) ToString() string {
	return string(c)
}
