package ai_llm_proxy

import (
	"encoding/json"
	"fmt"
)

// TransformResponsesRequestJSONToAnthropic converts an OpenAI Responses request
// JSON body into an Anthropic Messages request JSON body.
func TransformResponsesRequestJSONToAnthropic(body []byte) ([]byte, error) {
	var req ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshal responses request: %w", err)
	}
	out, err := ResponsesToAnthropicRequest(&req)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}
	return encoded, nil
}

// TransformAnthropicRequestJSONToResponses converts an Anthropic Messages
// request JSON body into an OpenAI Responses request JSON body.
func TransformAnthropicRequestJSONToResponses(body []byte) ([]byte, error) {
	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshal anthropic request: %w", err)
	}
	out, err := AnthropicToResponses(&req)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal responses request: %w", err)
	}
	return encoded, nil
}

// TransformChatCompletionsRequestJSONToResponses converts a Chat Completions
// request JSON body into an OpenAI Responses request JSON body.
func TransformChatCompletionsRequestJSONToResponses(body []byte) ([]byte, error) {
	var req ChatCompletionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshal chat completions request: %w", err)
	}
	out, err := ChatCompletionsToResponses(&req)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal responses request: %w", err)
	}
	return encoded, nil
}

// TransformChatCompletionsRequestJSONToAnthropic converts a Chat Completions
// request JSON body into an Anthropic Messages request JSON body.
func TransformChatCompletionsRequestJSONToAnthropic(body []byte) ([]byte, error) {
	responsesBody, err := TransformChatCompletionsRequestJSONToResponses(body)
	if err != nil {
		return nil, err
	}
	return TransformResponsesRequestJSONToAnthropic(responsesBody)
}

// TransformAnthropicResponseJSONToResponses converts an Anthropic Messages
// response JSON body into an OpenAI Responses response JSON body.
func TransformAnthropicResponseJSONToResponses(body []byte) ([]byte, error) {
	var resp AnthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal anthropic response: %w", err)
	}
	out := AnthropicToResponsesResponse(&resp)
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal responses response: %w", err)
	}
	return encoded, nil
}

// TransformAnthropicResponseJSONToChatCompletions converts an Anthropic
// Messages response JSON body into a Chat Completions response JSON body.
// If the payload does not include a model, fallbackModel is used.
func TransformAnthropicResponseJSONToChatCompletions(body []byte, fallbackModel string) ([]byte, error) {
	responsesBody, err := TransformAnthropicResponseJSONToResponses(body)
	if err != nil {
		return nil, err
	}
	return TransformResponsesResponseJSONToChatCompletions(responsesBody, fallbackModel)
}

// TransformResponsesResponseJSONToAnthropic converts an OpenAI Responses
// response JSON body into an Anthropic Messages response JSON body.
// If the payload does not include a model, fallbackModel is used.
func TransformResponsesResponseJSONToAnthropic(body []byte, fallbackModel string) ([]byte, error) {
	var resp ResponsesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal responses response: %w", err)
	}
	if resp.Model == "" {
		resp.Model = fallbackModel
	}
	out := ResponsesToAnthropic(&resp, resp.Model)
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic response: %w", err)
	}
	return encoded, nil
}

// TransformResponsesResponseJSONToChatCompletions converts an OpenAI Responses
// response JSON body into a Chat Completions response JSON body.
// If the payload does not include a model, fallbackModel is used.
func TransformResponsesResponseJSONToChatCompletions(body []byte, fallbackModel string) ([]byte, error) {
	var resp ResponsesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal responses response: %w", err)
	}
	model := resp.Model
	if model == "" {
		model = fallbackModel
	}
	out := ResponsesToChatCompletions(&resp, model)
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completions response: %w", err)
	}
	return encoded, nil
}
