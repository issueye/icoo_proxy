package constants

type Protocol string

const (
	ProtocolAnthropic       Protocol = "anthropic"
	ProtocolOpenAIChat      Protocol = "openai-chat"
	ProtocolOpenAIResponses Protocol = "openai-responses"
)

func (p Protocol) String() string {
	return string(p)
}

func ParseProtocol(raw string) (Protocol, bool) {
	switch Protocol(raw) {
	case ProtocolAnthropic, ProtocolOpenAIChat, ProtocolOpenAIResponses:
		return Protocol(raw), true
	default:
		return "", false
	}
}
