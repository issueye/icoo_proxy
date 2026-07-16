package ai_llm_proxy

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/issueye/icoo_proxy/common/constants"
)

var matrixProtocols = []constants.Protocol{
	constants.ProtocolAnthropic,
	constants.ProtocolOpenAIChat,
	constants.ProtocolOpenAIResponses,
}

func TestProtocolConverterRequestMatrix(t *testing.T) {
	bodies := map[constants.Protocol][]byte{
		constants.ProtocolAnthropic:       []byte(`{"model":"source","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`),
		constants.ProtocolOpenAIChat:      []byte(`{"model":"source","messages":[{"role":"user","content":"hello"}]}`),
		constants.ProtocolOpenAIResponses: []byte(`{"model":"source","input":"hello"}`),
	}
	unsupported := map[string]string{
		matrixCell(constants.ProtocolAnthropic, constants.ProtocolOpenAIChat):       "request conversion from anthropic to openai-chat is not implemented",
		matrixCell(constants.ProtocolOpenAIResponses, constants.ProtocolOpenAIChat): "request conversion from openai-responses to openai-chat is not implemented",
	}
	converter := NewProtocolConverter()

	for _, downstream := range matrixProtocols {
		for _, upstream := range matrixProtocols {
			name := fmt.Sprintf("%s_to_%s", downstream, upstream)
			t.Run(name, func(t *testing.T) {
				out, err := converter.ConvertRequest(RequestInput{
					Downstream: downstream,
					Upstream:   upstream,
					Model:      "target",
					Body:       bodies[downstream],
				})
				if wantErr, ok := unsupported[matrixCell(downstream, upstream)]; ok {
					if err == nil || err.Error() != wantErr {
						t.Fatalf("ConvertRequest() error = %v, want %q", err, wantErr)
					}
					return
				}
				if err != nil {
					t.Fatalf("ConvertRequest() error = %v", err)
				}
				assertJSONObject(t, out)
			})
		}
	}
}

func TestProtocolConverterNonStreamResponseMatrix(t *testing.T) {
	bodies := map[constants.Protocol][]byte{
		constants.ProtocolAnthropic:       []byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[{"type":"text","text":"hello"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`),
		constants.ProtocolOpenAIChat:      []byte(`{"id":"chatcmpl_1","object":"chat.completion","model":"gpt","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`),
		constants.ProtocolOpenAIResponses: []byte(`{"id":"resp_1","object":"response","model":"gpt","status":"completed","output":[{"id":"item_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`),
	}
	converter := NewProtocolConverter()

	for _, upstream := range matrixProtocols {
		for _, downstream := range matrixProtocols {
			name := fmt.Sprintf("%s_to_%s", upstream, downstream)
			t.Run(name, func(t *testing.T) {
				out, err := converter.ConvertResponse(ResponseInput{
					Downstream: downstream,
					Upstream:   upstream,
					Model:      "target",
					Body:       bodies[upstream],
				})
				if err != nil {
					t.Fatalf("ConvertResponse() error = %v", err)
				}
				assertJSONObject(t, out)
			})
		}
	}
}

func TestProtocolConverterStreamMatrix(t *testing.T) {
	streams := map[constants.Protocol]string{
		constants.ProtocolAnthropic: strings.Join([]string{
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			``,
		}, "\n"),
		constants.ProtocolOpenAIChat: strings.Join([]string{
			`data: [DONE]`,
			``,
		}, "\n"),
		constants.ProtocolOpenAIResponses: strings.Join([]string{
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt","status":"completed"}}`,
			``,
		}, "\n"),
	}
	converter := NewProtocolConverter()

	for _, upstream := range matrixProtocols {
		for _, downstream := range matrixProtocols {
			name := fmt.Sprintf("%s_to_%s", upstream, downstream)
			t.Run(name, func(t *testing.T) {
				var out strings.Builder
				_, err := converter.ConvertStream(StreamInput{
					Downstream: downstream,
					Upstream:   upstream,
					Model:      "target",
					Reader:     strings.NewReader(streams[upstream]),
					Writer:     &out,
				})
				if err != nil {
					t.Fatalf("ConvertStream() error = %v", err)
				}
				if out.Len() == 0 {
					t.Fatal("ConvertStream() wrote no output")
				}
			})
		}
	}
}

func matrixCell(from, to constants.Protocol) string {
	return string(from) + "->" + string(to)
}

func assertJSONObject(t *testing.T, body []byte) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("output is not a JSON object: %v; body = %s", err, body)
	}
}
