package constants

type Vendor string

const (
	VendorOpenAI    Vendor = "openai"
	VendorDeepSeek  Vendor = "deepseek"
	VendorGLM       Vendor = "glm"
	VendorAnthropic Vendor = "anthropic"
	VendorCustom    Vendor = "custom"
	// VendorPlugin routes traffic through a process plugin over IPC
	// (plugin_id on the provider). Not an HTTP upstream.
	VendorPlugin Vendor = "plugin"
)

func (v Vendor) String() string {
	return string(v)
}
