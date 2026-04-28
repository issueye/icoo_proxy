package translation

import (
	"encoding/json"
	"icoo_proxy/internal/consts"
)

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

func (u TokenUsage) Normalize() TokenUsage {
	if u.TotalTokens == 0 {
		u.TotalTokens = u.InputTokens + u.OutputTokens
	}
	return u
}

func ExtractUsage(protocol consts.Protocol, body []byte) TokenUsage {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return TokenUsage{}
	}
	return ExtractUsageFromPayload(protocol, payload)
}

func ExtractUsageFromPayload(protocol consts.Protocol, payload map[string]interface{}) TokenUsage {
	if payload == nil {
		return TokenUsage{}
	}
	usageMap, _ := payload["usage"].(map[string]interface{})
	if usageMap == nil {
		return TokenUsage{}
	}

	switch protocol {
	case consts.ProtocolAnthropic:
		return TokenUsage{
			InputTokens:  intValue(usageMap["input_tokens"]),
			OutputTokens: intValue(usageMap["output_tokens"]),
		}.Normalize()
	case consts.ProtocolOpenAIChat:
		return TokenUsage{
			InputTokens:  intValue(usageMap["prompt_tokens"]),
			OutputTokens: intValue(usageMap["completion_tokens"]),
			TotalTokens:  intValue(usageMap["total_tokens"]),
		}.Normalize()
	case consts.ProtocolOpenAIResponses:
		return TokenUsage{
			InputTokens:  intValue(usageMap["input_tokens"]),
			OutputTokens: intValue(usageMap["output_tokens"]),
			TotalTokens:  intValue(usageMap["total_tokens"]),
		}.Normalize()
	default:
		return TokenUsage{}
	}
}
