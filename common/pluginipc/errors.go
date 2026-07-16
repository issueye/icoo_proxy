package pluginipc

import (
	"errors"
	"fmt"
)

// JSON-RPC 2.0 and icoo-specific application error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	CodeUnauthorized       = -32001
	CodeUnsupportedIngress = -32002
	CodeUpstreamError      = -32003
	CodeStreamNotFound     = -32004
	CodeShuttingDown       = -32005
	CodeTooManyStreams     = -32006
	CodeFrameTooLarge      = -32007
)

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return "pluginipc: <nil error>"
	}
	return fmt.Sprintf("pluginipc rpc error %d: %s", e.Code, e.Message)
}

func NewRPCError(code int, message string, data any) *RPCError {
	return &RPCError{Code: code, Message: message, Data: data}
}

var (
	ErrClosed            = errors.New("pluginipc: connection closed")
	ErrFrameTooLarge     = errors.New("pluginipc: frame too large")
	ErrProtocol          = errors.New("pluginipc: protocol error")
	ErrUnauthorized      = NewRPCError(CodeUnauthorized, "unauthorized", nil)
	ErrTooManyStreams    = NewRPCError(CodeTooManyStreams, "too many streams", nil)
	ErrStreamNotFound    = NewRPCError(CodeStreamNotFound, "stream not found", nil)
	ErrUnsupportedIngress = NewRPCError(CodeUnsupportedIngress, "unsupported ingress", nil)
)

// HTTPStatus maps an RPC error code to a suggested host HTTP status.
func HTTPStatus(code int) int {
	switch code {
	case CodeInvalidParams, CodeUnsupportedIngress:
		return 400
	case CodeFrameTooLarge:
		return 413
	case CodeShuttingDown, CodeTooManyStreams:
		return 503
	default:
		return 502
	}
}
