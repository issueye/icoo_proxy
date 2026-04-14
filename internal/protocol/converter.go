package protocol

import (
	"context"
	"fmt"
	"net/http"
)

// ProtocolAdapter defines the interface that all AI protocol adapters must implement.
type ProtocolAdapter interface {
	// ParseRequest parses a provider-specific request body into internal format.
	ParseRequest(body []byte) (*InternalRequest, error)

	// BuildRequest builds a provider-specific request body from internal format.
	// Returns the request body bytes and the URL path suffix.
	BuildRequest(req *InternalRequest) ([]byte, string, error)

	// ParseResponse parses a provider-specific response body into internal format.
	ParseResponse(body []byte) (*InternalResponse, error)

	// BuildResponse builds a provider-specific response body from internal format.
	BuildResponse(resp *InternalResponse) ([]byte, error)

	// ParseStreamEvent parses a single SSE event from the provider into internal format.
	ParseStreamEvent(eventType, data string) (*InternalStreamChunk, error)

	// BuildStreamEvent builds a single SSE event in the provider's format from internal chunk.
	BuildStreamEvent(chunk *InternalStreamChunk) (eventType, data string, err error)

	// StreamDone returns the stream completion marker for this protocol.
	StreamDone() string

	// BuildHTTPRequest constructs the full HTTP request to the provider.
	BuildHTTPRequest(ctx context.Context, apiBase, apiKey string, method, path string, body []byte) (*http.Request, error)

	// ListModelsRequest constructs the HTTP request for listing models.
	ListModelsRequest(ctx context.Context, apiBase, apiKey string) (*http.Request, error)

	// ParseModelsResponse parses the provider's model list response.
	ParseModelsResponse(body []byte) ([]ModelInfo, error)
}

// adapterRegistry maps provider type names to adapter factory functions.
// This is populated via RegisterAdapter to avoid import cycles.
var adapterRegistry = map[string]func() ProtocolAdapter{}

// RegisterAdapter registers a protocol adapter factory for a provider type.
// Called from init() in each adapter package.
func RegisterAdapter(providerType string, factory func() ProtocolAdapter) {
	adapterRegistry[providerType] = factory
}

// GetAdapter returns the appropriate protocol adapter for the given provider type.
func GetAdapter(providerType string) (ProtocolAdapter, error) {
	factory, ok := adapterRegistry[providerType]
	if ok {
		return factory(), nil
	}
	// Default to OpenAI-compatible for unknown types
	if defaultFactory, ok := adapterRegistry["openai"]; ok {
		return defaultFactory(), nil
	}
	return nil, &ErrUnsupportedType{Type: providerType}
}

// GetProviderTypes returns all registered provider types.
func GetProviderTypes() []string {
	types := make([]string, 0, len(adapterRegistry))
	for t := range adapterRegistry {
		types = append(types, t)
	}
	return types
}

// ErrUnsupportedType is returned when an unsupported provider type is used.
type ErrUnsupportedType struct {
	Type string
}

func (e *ErrUnsupportedType) Error() string {
	return fmt.Sprintf("unsupported provider type: %s", e.Type)
}

// RegisterDefaults registers all built-in protocol adapters.
func RegisterDefaults() {
	RegisterAdapter("openai", func() ProtocolAdapter { return &OpenAIAdapter{} })
	RegisterAdapter("anthropic", func() ProtocolAdapter { return &AnthropicAdapter{} })
	RegisterAdapter("gemini", func() ProtocolAdapter { return &GeminiAdapter{} })
}
