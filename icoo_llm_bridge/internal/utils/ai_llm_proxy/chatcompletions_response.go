package ai_llm_proxy

import (
	"encoding/json"
	"fmt"
)

// ChatCompletionsToResponsesResponse converts a non-streaming Chat
// Completions response into a Responses API response. It preserves the first
// choice's text, reasoning content, tool calls, finish reason, and usage.
func ChatCompletionsToResponsesResponse(resp *ChatCompletionsResponse, fallbackModel string) *ResponsesResponse {
	id := resp.ID
	if id == "" {
		id = generateResponsesID()
	}

	model := resp.Model
	if model == "" {
		model = fallbackModel
	}

	out := &ResponsesResponse{
		ID:     id,
		Object: "response",
		Model:  model,
		Status: "completed",
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		out.Output = chatChoiceToResponsesOutput(choice)
		out.Status = chatFinishReasonToResponsesStatus(choice.FinishReason)
		if out.Status == "incomplete" {
			out.IncompleteDetails = &ResponsesIncompleteDetails{Reason: "max_output_tokens"}
		}
	}
	if len(out.Output) == 0 {
		out.Output = []ResponsesOutput{{
			Type:    "message",
			ID:      generateItemID(),
			Role:    "assistant",
			Content: []ResponsesContentPart{{Type: "output_text", Text: ""}},
			Status:  "completed",
		}}
	}

	if resp.Usage != nil {
		out.Usage = responsesUsageFromChatUsage(resp.Usage)
	}

	return out
}

func chatChoiceToResponsesOutput(choice ChatChoice) []ResponsesOutput {
	var out []ResponsesOutput

	if choice.Message.ReasoningContent != "" {
		out = append(out, ResponsesOutput{
			Type: "reasoning",
			ID:   generateItemID(),
			Summary: []ResponsesSummary{{
				Type: "summary_text",
				Text: choice.Message.ReasoningContent,
			}},
			Status: "completed",
		})
	}

	text := chatResponseContentText(choice.Message.Content)
	if text != "" || len(choice.Message.ToolCalls) == 0 {
		out = append(out, ResponsesOutput{
			Type: "message",
			ID:   generateItemID(),
			Role: "assistant",
			Content: []ResponsesContentPart{{
				Type: "output_text",
				Text: text,
			}},
			Status: "completed",
		})
	}

	for _, toolCall := range choice.Message.ToolCalls {
		out = append(out, chatToolCallToResponsesOutput(toolCall))
	}
	if choice.Message.FunctionCall != nil {
		out = append(out, ResponsesOutput{
			Type:      "function_call",
			ID:        generateItemID(),
			CallID:    generateItemID(),
			Name:      choice.Message.FunctionCall.Name,
			Arguments: choice.Message.FunctionCall.Arguments,
			Status:    "completed",
		})
	}

	return out
}

func chatResponseContentText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	text, err := parseAssistantContent(raw)
	if err == nil {
		return text
	}
	return ""
}

func chatToolCallToResponsesOutput(toolCall ChatToolCall) ResponsesOutput {
	callID := toolCall.ID
	if callID == "" {
		callID = generateItemID()
	}
	args := toolCall.Function.Arguments
	if args == "" {
		args = "{}"
	}
	return ResponsesOutput{
		Type:      "function_call",
		ID:        generateItemID(),
		CallID:    callID,
		Name:      toolCall.Function.Name,
		Arguments: args,
		Status:    "completed",
	}
}

func chatFinishReasonToResponsesStatus(finishReason string) string {
	switch finishReason {
	case "length":
		return "incomplete"
	default:
		return "completed"
	}
}

// TransformChatCompletionsResponseJSONToResponses converts a Chat Completions
// response JSON body into an OpenAI Responses response JSON body.
func TransformChatCompletionsResponseJSONToResponses(body []byte, fallbackModel string) ([]byte, error) {
	var resp ChatCompletionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal chat completions response: %w", err)
	}
	out := ChatCompletionsToResponsesResponse(&resp, fallbackModel)
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal responses response: %w", err)
	}
	return encoded, nil
}

// TransformChatCompletionsResponseJSONToAnthropic converts a Chat Completions
// response JSON body into an Anthropic Messages response JSON body.
func TransformChatCompletionsResponseJSONToAnthropic(body []byte, fallbackModel string) ([]byte, error) {
	responsesBody, err := TransformChatCompletionsResponseJSONToResponses(body, fallbackModel)
	if err != nil {
		return nil, err
	}
	return TransformResponsesResponseJSONToAnthropic(responsesBody, fallbackModel)
}
