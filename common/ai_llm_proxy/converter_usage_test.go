package ai_llm_proxy

import (
	"testing"

	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/domain"
)

func TestProtocolConverterExtractUsage(t *testing.T) {
	tests := []struct {
		name     string
		protocol constants.Protocol
		body     string
		want     domain.TokenUsage
	}{
		{
			name:     "anthropic",
			protocol: constants.ProtocolAnthropic,
			body:     `{"usage":{"input_tokens":11,"output_tokens":7}}`,
			want:     domain.TokenUsage{InputTokens: 11, OutputTokens: 7, TotalTokens: 18},
		},
		{
			name:     "openai chat",
			protocol: constants.ProtocolOpenAIChat,
			body:     `{"usage":{"prompt_tokens":13,"completion_tokens":5,"total_tokens":18}}`,
			want:     domain.TokenUsage{InputTokens: 13, OutputTokens: 5, TotalTokens: 18},
		},
		{
			name:     "openai responses",
			protocol: constants.ProtocolOpenAIResponses,
			body:     `{"usage":{"input_tokens":17,"output_tokens":3,"total_tokens":20}}`,
			want:     domain.TokenUsage{InputTokens: 17, OutputTokens: 3, TotalTokens: 20},
		},
		{
			name:     "nested response usage",
			protocol: constants.ProtocolOpenAIResponses,
			body:     `{"response":{"id":"resp_1","usage":{"input_tokens":19,"output_tokens":2,"total_tokens":21}}}`,
			want:     domain.TokenUsage{InputTokens: 19, OutputTokens: 2, TotalTokens: 21},
		},
		{
			name:     "missing usage",
			protocol: constants.ProtocolOpenAIChat,
			body:     `{"id":"chatcmpl_1"}`,
			want:     domain.TokenUsage{},
		},
		{
			name:     "unrecognized usage shape",
			protocol: constants.ProtocolOpenAIResponses,
			body:     `{"response":{"usage":[]}}`,
			want:     domain.TokenUsage{},
		},
		{
			name:     "hybrid object uses only chat fields",
			protocol: constants.ProtocolOpenAIChat,
			body:     `{"usage":{"input_tokens":100,"output_tokens":200,"prompt_tokens":23,"completion_tokens":4,"total_tokens":27}}`,
			want:     domain.TokenUsage{InputTokens: 23, OutputTokens: 4, TotalTokens: 27},
		},
		{
			name:     "hybrid object uses only responses fields",
			protocol: constants.ProtocolOpenAIResponses,
			body:     `{"usage":{"input_tokens":29,"output_tokens":6,"prompt_tokens":300,"completion_tokens":400,"total_tokens":35}}`,
			want:     domain.TokenUsage{InputTokens: 29, OutputTokens: 6, TotalTokens: 35},
		},
		{
			name:     "unknown protocol",
			protocol: constants.Protocol("unknown"),
			body:     `{"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`,
			want:     domain.TokenUsage{},
		},
	}

	converter := NewProtocolConverter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := converter.ExtractUsage(tt.protocol, []byte(tt.body))
			if got != tt.want {
				t.Fatalf("ExtractUsage() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
