package ai_llm_proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/domain"
)

type protocolConverter struct{}

func NewPassthroughConverter() Converter {
	return NewProtocolConverter()
}

func NewProtocolConverter() Converter {
	return &protocolConverter{}
}

func (c *protocolConverter) ConvertRequest(input RequestInput) ([]byte, error) {
	if input.Downstream == "" || input.Upstream == "" {
		return nil, fmt.Errorf("protocols are required")
	}
	var (
		out []byte
		err error
	)
	switch {
	case input.Downstream == input.Upstream:
		out = input.Body
	case input.Downstream == constants.ProtocolAnthropic && input.Upstream == constants.ProtocolOpenAIResponses:
		out, err = TransformAnthropicRequestJSONToResponses(input.Body)
	case input.Downstream == constants.ProtocolOpenAIChat && input.Upstream == constants.ProtocolOpenAIResponses:
		out, err = TransformChatCompletionsRequestJSONToResponses(input.Body)
	case input.Downstream == constants.ProtocolOpenAIChat && input.Upstream == constants.ProtocolAnthropic:
		out, err = TransformChatCompletionsRequestJSONToAnthropic(input.Body)
	case input.Downstream == constants.ProtocolOpenAIResponses && input.Upstream == constants.ProtocolAnthropic:
		out, err = TransformResponsesRequestJSONToAnthropic(input.Body)
	default:
		return nil, fmt.Errorf("request conversion from %s to %s is not implemented", input.Downstream, input.Upstream)
	}
	if err != nil {
		return nil, err
	}
	return rewriteJSONModel(out, input.Model)
}

func (c *protocolConverter) ConvertResponse(input ResponseInput) ([]byte, error) {
	if input.Downstream == "" || input.Upstream == "" {
		return nil, fmt.Errorf("protocols are required")
	}
	switch {
	case input.Downstream == input.Upstream:
		return input.Body, nil
	case input.Upstream == constants.ProtocolAnthropic && input.Downstream == constants.ProtocolOpenAIResponses:
		return TransformAnthropicResponseJSONToResponses(input.Body)
	case input.Upstream == constants.ProtocolAnthropic && input.Downstream == constants.ProtocolOpenAIChat:
		return TransformAnthropicResponseJSONToChatCompletions(input.Body, input.Model)
	case input.Upstream == constants.ProtocolOpenAIResponses && input.Downstream == constants.ProtocolAnthropic:
		return TransformResponsesResponseJSONToAnthropic(input.Body, input.Model)
	case input.Upstream == constants.ProtocolOpenAIResponses && input.Downstream == constants.ProtocolOpenAIChat:
		return TransformResponsesResponseJSONToChatCompletions(input.Body, input.Model)
	default:
		return nil, fmt.Errorf("response conversion from %s to %s is not implemented", input.Upstream, input.Downstream)
	}
}

func (c *protocolConverter) ConvertStream(input StreamInput) (StreamResult, error) {
	if input.Reader == nil || input.Writer == nil {
		return StreamResult{}, fmt.Errorf("stream reader and writer are required")
	}
	if input.Downstream == input.Upstream {
		_, err := io.Copy(input.Writer, input.Reader)
		return StreamResult{}, err
	}
	switch {
	case input.Upstream == constants.ProtocolOpenAIResponses && input.Downstream == constants.ProtocolAnthropic:
		return c.convertResponsesStreamToAnthropic(input)
	case input.Upstream == constants.ProtocolOpenAIResponses && input.Downstream == constants.ProtocolOpenAIChat:
		return c.convertResponsesStreamToChat(input)
	case input.Upstream == constants.ProtocolAnthropic && input.Downstream == constants.ProtocolOpenAIResponses:
		return c.convertAnthropicStreamToResponses(input)
	case input.Upstream == constants.ProtocolAnthropic && input.Downstream == constants.ProtocolOpenAIChat:
		return c.convertAnthropicStreamToChat(input)
	default:
		return StreamResult{}, fmt.Errorf("stream conversion from %s to %s is not implemented", input.Upstream, input.Downstream)
	}
}

func (c *protocolConverter) ExtractUsage(protocol constants.Protocol, body []byte) domain.TokenUsage {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.TokenUsage{}
	}
	usage, _ := payload["usage"].(map[string]any)
	if usage == nil {
		return domain.TokenUsage{}
	}
	result := domain.TokenUsage{
		InputTokens:  intFromJSON(usage["input_tokens"]) + intFromJSON(usage["prompt_tokens"]),
		OutputTokens: intFromJSON(usage["output_tokens"]) + intFromJSON(usage["completion_tokens"]),
		TotalTokens:  intFromJSON(usage["total_tokens"]),
	}
	return result.Normalize()
}

func rewriteJSONModel(body []byte, model string) ([]byte, error) {
	if model == "" {
		return body, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid json body: %w", err)
	}
	payload["model"] = model
	return json.Marshal(payload)
}

func intFromJSON(raw any) int {
	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func (c *protocolConverter) convertResponsesStreamToAnthropic(input StreamInput) (StreamResult, error) {
	state := NewResponsesEventToAnthropicState()
	state.Model = input.Model
	err := scanSSE(input.Reader, func(eventName string, data []byte) error {
		var event ResponsesStreamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		if event.Type == "" {
			event.Type = eventName
		}
		for _, out := range ResponsesEventToAnthropicEvents(&event, state) {
			text, err := ResponsesAnthropicEventToSSE(out)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(input.Writer, text); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return StreamResult{}, err
	}
	for _, out := range FinalizeResponsesAnthropicStream(state) {
		text, err := ResponsesAnthropicEventToSSE(out)
		if err != nil {
			return StreamResult{}, err
		}
		if _, err := io.WriteString(input.Writer, text); err != nil {
			return StreamResult{}, err
		}
	}
	return StreamResult{Usage: domain.TokenUsage{
		InputTokens:  state.InputTokens,
		OutputTokens: state.OutputTokens,
		TotalTokens:  state.InputTokens + state.OutputTokens,
	}}, nil
}

func (c *protocolConverter) convertResponsesStreamToChat(input StreamInput) (StreamResult, error) {
	state := NewResponsesEventToChatState()
	state.Model = input.Model
	err := scanSSE(input.Reader, func(eventName string, data []byte) error {
		var event ResponsesStreamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		if event.Type == "" {
			event.Type = eventName
		}
		for _, out := range ResponsesEventToChatChunks(&event, state) {
			text, err := ChatChunkToSSE(out)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(input.Writer, text); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return StreamResult{}, err
	}
	for _, out := range FinalizeResponsesChatStream(state) {
		text, err := ChatChunkToSSE(out)
		if err != nil {
			return StreamResult{}, err
		}
		if _, err := io.WriteString(input.Writer, text); err != nil {
			return StreamResult{}, err
		}
	}
	_, _ = io.WriteString(input.Writer, "data: [DONE]\n\n")
	usage := domain.TokenUsage{}
	if state.Usage != nil {
		usage.InputTokens = state.Usage.PromptTokens
		usage.OutputTokens = state.Usage.CompletionTokens
		usage.TotalTokens = state.Usage.TotalTokens
	}
	return StreamResult{Usage: usage.Normalize()}, nil
}

func (c *protocolConverter) convertAnthropicStreamToResponses(input StreamInput) (StreamResult, error) {
	state := NewAnthropicEventToResponsesState()
	state.Model = input.Model
	err := scanSSE(input.Reader, func(eventName string, data []byte) error {
		var event AnthropicStreamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		if event.Type == "" {
			event.Type = eventName
		}
		for _, out := range AnthropicEventToResponsesEvents(&event, state) {
			text, err := ResponsesEventToSSE(out)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(input.Writer, text); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return StreamResult{}, err
	}
	for _, out := range FinalizeAnthropicResponsesStream(state) {
		text, err := ResponsesEventToSSE(out)
		if err != nil {
			return StreamResult{}, err
		}
		if _, err := io.WriteString(input.Writer, text); err != nil {
			return StreamResult{}, err
		}
	}
	return StreamResult{Usage: domain.TokenUsage{
		InputTokens:  state.InputTokens,
		OutputTokens: state.OutputTokens,
		TotalTokens:  state.InputTokens + state.OutputTokens,
	}}, nil
}

func (c *protocolConverter) convertAnthropicStreamToChat(input StreamInput) (StreamResult, error) {
	anthropicState := NewAnthropicEventToResponsesState()
	anthropicState.Model = input.Model
	chatState := NewResponsesEventToChatState()
	chatState.Model = input.Model

	writeResponsesEvents := func(events []ResponsesStreamEvent) error {
		for i := range events {
			for _, chunk := range ResponsesEventToChatChunks(&events[i], chatState) {
				text, err := ChatChunkToSSE(chunk)
				if err != nil {
					return err
				}
				if _, err := io.WriteString(input.Writer, text); err != nil {
					return err
				}
			}
		}
		return nil
	}

	err := scanSSE(input.Reader, func(eventName string, data []byte) error {
		var event AnthropicStreamEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		if event.Type == "" {
			event.Type = eventName
		}
		return writeResponsesEvents(AnthropicEventToResponsesEvents(&event, anthropicState))
	})
	if err != nil {
		return StreamResult{}, err
	}
	if err := writeResponsesEvents(FinalizeAnthropicResponsesStream(anthropicState)); err != nil {
		return StreamResult{}, err
	}
	for _, chunk := range FinalizeResponsesChatStream(chatState) {
		text, err := ChatChunkToSSE(chunk)
		if err != nil {
			return StreamResult{}, err
		}
		if _, err := io.WriteString(input.Writer, text); err != nil {
			return StreamResult{}, err
		}
	}
	_, _ = io.WriteString(input.Writer, "data: [DONE]\n\n")

	return StreamResult{Usage: domain.TokenUsage{
		InputTokens:  anthropicState.InputTokens,
		OutputTokens: anthropicState.OutputTokens,
		TotalTokens:  anthropicState.InputTokens + anthropicState.OutputTokens,
	}}, nil
}

func scanSSE(reader io.Reader, handle func(eventName string, data []byte) error) error {
	bufReader := bufio.NewReader(reader)
	eventName := ""
	dataLines := make([]string, 0, 4)
	flush := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		name := eventName
		eventName = ""
		if data == "" || data == "[DONE]" {
			return nil
		}
		return handle(name, []byte(data))
	}
	for {
		line, err := bufReader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if line == "" {
				if err := flush(); err != nil {
					return err
				}
			} else if !strings.HasPrefix(line, ":") {
				if strings.HasPrefix(line, "event:") {
					eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				} else if strings.HasPrefix(line, "data:") {
					value := strings.TrimPrefix(line, "data:")
					dataLines = append(dataLines, strings.TrimPrefix(value, " "))
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return flush()
}
