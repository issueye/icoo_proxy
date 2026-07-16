package ai_llm_proxy

import (
	"encoding/json"
	"testing"
)

func TestChatCompletionsToResponsesPreservesStreamIntent(t *testing.T) {
	tests := []struct {
		name              string
		streamField       string
		wantStream        bool
		wantStreamEncoded bool
	}{
		{
			name: "omitted",
		},
		{
			name:        "false",
			streamField: `,"stream":false`,
		},
		{
			name:              "true",
			streamField:       `,"stream":true`,
			wantStream:        true,
			wantStreamEncoded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"model":"chat-model","messages":[{"role":"user","content":"hello"}]` + tt.streamField + `}`)

			converted, err := TransformChatCompletionsRequestJSONToResponses(body)
			if err != nil {
				t.Fatalf("TransformChatCompletionsRequestJSONToResponses returned error: %v", err)
			}

			var request ResponsesRequest
			if err := json.Unmarshal(converted, &request); err != nil {
				t.Fatalf("unmarshal converted request: %v", err)
			}
			if request.Stream != tt.wantStream {
				t.Fatalf("Stream = %v, want %v; body = %s", request.Stream, tt.wantStream, converted)
			}

			var payload map[string]json.RawMessage
			if err := json.Unmarshal(converted, &payload); err != nil {
				t.Fatalf("unmarshal converted payload: %v", err)
			}
			_, streamEncoded := payload["stream"]
			if streamEncoded != tt.wantStreamEncoded {
				t.Fatalf("stream field encoded = %v, want %v; body = %s", streamEncoded, tt.wantStreamEncoded, converted)
			}
		})
	}
}
