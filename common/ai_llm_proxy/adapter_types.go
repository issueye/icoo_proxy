package ai_llm_proxy

import (
	"context"
	"io"

	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/domain"
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
	Context    context.Context
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
