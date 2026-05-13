package ai_llm_proxy

import (
	"io"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/domain"
)

type RequestInput struct {
	Downstream constants.Protocol
	Upstream   constants.Protocol
	Model      string
	Body       []byte
}

type ResponseInput struct {
	Downstream constants.Protocol
	Upstream   constants.Protocol
	Model      string
	Body       []byte
}

type StreamInput struct {
	Downstream constants.Protocol
	Upstream   constants.Protocol
	Model      string
	Reader     io.Reader
	Writer     io.Writer
}

type StreamResult struct {
	Usage domain.TokenUsage
}

type Converter interface {
	ConvertRequest(input RequestInput) ([]byte, error)
	ConvertResponse(input ResponseInput) ([]byte, error)
	ConvertStream(input StreamInput) (StreamResult, error)
	ExtractUsage(protocol constants.Protocol, body []byte) domain.TokenUsage
}
