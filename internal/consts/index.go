package consts

type Protocol string

type Vendor string

const (
	ProtocolAnthropic       Protocol = "anthropic"
	ProtocolOpenAIChat      Protocol = "openai-chat"
	ProtocolOpenAIResponses Protocol = "openai-responses"
)

const (
	VendorOpenAI    Vendor = "openai"
	VendorDeepSeek  Vendor = "deepseek"
	VendorAnthropic Vendor = "anthropic"
)

func (p Protocol) ToString() string {
	return string(p)
}

func (v Vendor) ToString() string {
	return string(v)
}

const DefaultResponsesReasoningEffort = "medium" // 默认 reasoning 配置
