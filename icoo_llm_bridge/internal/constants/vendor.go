package constants

type Vendor string

const (
	VendorOpenAI    Vendor = "openai"
	VendorDeepSeek  Vendor = "deepseek"
	VendorGLM       Vendor = "glm"
	VendorAnthropic Vendor = "anthropic"
	VendorCustom    Vendor = "custom"
)

func (v Vendor) String() string {
	return string(v)
}
