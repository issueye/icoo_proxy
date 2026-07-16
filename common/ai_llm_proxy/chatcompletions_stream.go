package ai_llm_proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

// ChatEventToResponsesState tracks state for converting Chat Completions SSE
// chunks into Responses SSE events.
type ChatEventToResponsesState struct {
	ResponseID     string
	Model          string
	Created        int64
	SequenceNumber int

	CreatedSent   bool
	CompletedSent bool
	TextItemOpen  bool
	TextItemID    string

	OutputIndex  int
	ContentIndex int
	Usage        *ResponsesUsage
	FinishReason string

	ToolCalls     map[chatStreamToolKey]*chatStreamToolCall
	ToolCallOrder []chatStreamToolKey
}

type chatStreamToolKey struct {
	ChoiceIndex int
	ToolIndex   int
}

type chatStreamToolCall struct {
	ItemID            string
	CallID            string
	Name              string
	ArgumentFragments []string
	Arguments         string
	Emitted           bool
}

func NewChatEventToResponsesState() *ChatEventToResponsesState {
	return &ChatEventToResponsesState{
		ResponseID: generateResponsesID(),
		Created:    time.Now().Unix(),
		ToolCalls:  make(map[chatStreamToolKey]*chatStreamToolCall),
	}
}

// ChatChunkToResponsesEvents converts one Chat Completions chunk into zero or
// more Responses stream events. Tool calls are accumulated by choice and tool
// index so interleaved parallel calls can be emitted as ordered output items.
func ChatChunkToResponsesEvents(chunk *ChatCompletionsChunk, state *ChatEventToResponsesState) []ResponsesStreamEvent {
	if chunk.ID != "" {
		state.ResponseID = chunk.ID
	}
	if state.Model == "" && chunk.Model != "" {
		state.Model = chunk.Model
	}

	var events []ResponsesStreamEvent
	events = append(events, chatEnsureResponsesCreated(state)...)

	if chunk.Usage != nil {
		state.Usage = responsesUsageFromChatUsage(chunk.Usage)
	}

	flushToolCalls := false
	for _, choice := range chunk.Choices {
		if choice.Delta.Content != nil && *choice.Delta.Content != "" {
			events = append(events, chatEnsureTextOutputItem(state)...)
			events = append(events, chatMakeResponsesEvent(state, "response.output_text.delta", &ResponsesStreamEvent{
				OutputIndex:  state.OutputIndex,
				ContentIndex: state.ContentIndex,
				Delta:        *choice.Delta.Content,
				ItemID:       state.TextItemID,
			}))
		}
		for position, toolCall := range choice.Delta.ToolCalls {
			toolIndex := position
			if toolCall.Index != nil {
				toolIndex = *toolCall.Index
			}
			key := chatStreamToolKey{ChoiceIndex: choice.Index, ToolIndex: toolIndex}
			accumulated, ok := state.ToolCalls[key]
			if !ok {
				accumulated = &chatStreamToolCall{ItemID: generateItemID()}
				state.ToolCalls[key] = accumulated
				state.ToolCallOrder = append(state.ToolCallOrder, key)
			}
			if toolCall.ID != "" {
				accumulated.CallID = toolCall.ID
			}
			accumulated.Name += toolCall.Function.Name
			if toolCall.Function.Arguments != "" {
				accumulated.ArgumentFragments = append(accumulated.ArgumentFragments, toolCall.Function.Arguments)
				accumulated.Arguments += toolCall.Function.Arguments
			}
		}
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			state.FinishReason = *choice.FinishReason
			flushToolCalls = true
		}
	}
	if flushToolCalls {
		events = append(events, chatFlushToolCalls(state)...)
	}

	return events
}

// FinalizeChatResponsesStream emits terminal Responses events after [DONE] or
// EOF. It is idempotent.
func FinalizeChatResponsesStream(state *ChatEventToResponsesState) []ResponsesStreamEvent {
	if state.CompletedSent {
		return nil
	}
	state.CompletedSent = true

	var events []ResponsesStreamEvent
	events = append(events, chatEnsureResponsesCreated(state)...)
	events = append(events, chatFlushToolCalls(state)...)
	events = append(events, chatCloseTextOutputItem(state)...)

	status := chatFinishReasonToResponsesStatus(state.FinishReason)
	var details *ResponsesIncompleteDetails
	if status == "incomplete" {
		details = &ResponsesIncompleteDetails{Reason: "max_output_tokens"}
	}
	events = append(events, chatMakeResponsesEvent(state, "response.completed", &ResponsesStreamEvent{
		Response: &ResponsesResponse{
			ID:                state.ResponseID,
			Object:            "response",
			Model:             state.Model,
			Status:            status,
			Output:            []ResponsesOutput{},
			Usage:             state.Usage,
			IncompleteDetails: details,
		},
	}))
	return events
}

func chatFlushToolCalls(state *ChatEventToResponsesState) []ResponsesStreamEvent {
	if len(state.ToolCallOrder) == 0 {
		return nil
	}

	var events []ResponsesStreamEvent
	events = append(events, chatCloseTextOutputItem(state)...)
	for _, key := range state.ToolCallOrder {
		toolCall := state.ToolCalls[key]
		if toolCall == nil || toolCall.Emitted {
			continue
		}
		toolCall.Emitted = true
		if toolCall.CallID == "" {
			toolCall.CallID = generateItemID()
		}
		arguments := toolCall.Arguments
		if arguments == "" {
			arguments = "{}"
		}

		outputIndex := state.OutputIndex
		events = append(events, chatMakeResponsesEvent(state, "response.output_item.added", &ResponsesStreamEvent{
			OutputIndex: outputIndex,
			Item: &ResponsesOutput{
				Type:   "function_call",
				ID:     toolCall.ItemID,
				CallID: toolCall.CallID,
				Name:   toolCall.Name,
				Status: "in_progress",
			},
		}))
		for _, fragment := range toolCall.ArgumentFragments {
			events = append(events, chatMakeResponsesEvent(state, "response.function_call_arguments.delta", &ResponsesStreamEvent{
				OutputIndex: outputIndex,
				Delta:       fragment,
				ItemID:      toolCall.ItemID,
				CallID:      toolCall.CallID,
				Name:        toolCall.Name,
			}))
		}
		events = append(events, chatMakeResponsesEvent(state, "response.function_call_arguments.done", &ResponsesStreamEvent{
			OutputIndex: outputIndex,
			ItemID:      toolCall.ItemID,
			CallID:      toolCall.CallID,
			Name:        toolCall.Name,
			Arguments:   arguments,
		}))
		events = append(events, chatMakeResponsesEvent(state, "response.output_item.done", &ResponsesStreamEvent{
			OutputIndex: outputIndex,
			Item: &ResponsesOutput{
				Type:      "function_call",
				ID:        toolCall.ItemID,
				CallID:    toolCall.CallID,
				Name:      toolCall.Name,
				Arguments: arguments,
				Status:    "completed",
			},
		}))
		state.OutputIndex++
	}
	return events
}

func chatCloseTextOutputItem(state *ChatEventToResponsesState) []ResponsesStreamEvent {
	if !state.TextItemOpen {
		return nil
	}
	events := []ResponsesStreamEvent{
		chatMakeResponsesEvent(state, "response.output_text.done", &ResponsesStreamEvent{
			OutputIndex:  state.OutputIndex,
			ContentIndex: state.ContentIndex,
			ItemID:       state.TextItemID,
		}),
		chatMakeResponsesEvent(state, "response.output_item.done", &ResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Item: &ResponsesOutput{
				Type:   "message",
				ID:     state.TextItemID,
				Role:   "assistant",
				Status: "completed",
			},
		}),
	}
	state.TextItemOpen = false
	state.OutputIndex++
	return events
}

func chatEnsureResponsesCreated(state *ChatEventToResponsesState) []ResponsesStreamEvent {
	if state.CreatedSent {
		return nil
	}
	state.CreatedSent = true
	return []ResponsesStreamEvent{chatMakeResponsesEvent(state, "response.created", &ResponsesStreamEvent{
		Response: &ResponsesResponse{
			ID:     state.ResponseID,
			Object: "response",
			Model:  state.Model,
			Status: "in_progress",
			Output: []ResponsesOutput{},
		},
	})}
}

func chatEnsureTextOutputItem(state *ChatEventToResponsesState) []ResponsesStreamEvent {
	if state.TextItemOpen {
		return nil
	}
	state.TextItemOpen = true
	state.TextItemID = generateItemID()
	return []ResponsesStreamEvent{chatMakeResponsesEvent(state, "response.output_item.added", &ResponsesStreamEvent{
		OutputIndex: state.OutputIndex,
		Item: &ResponsesOutput{
			Type:   "message",
			ID:     state.TextItemID,
			Role:   "assistant",
			Status: "in_progress",
		},
	})}
}

func chatMakeResponsesEvent(state *ChatEventToResponsesState, eventType string, template *ResponsesStreamEvent) ResponsesStreamEvent {
	seq := state.SequenceNumber
	state.SequenceNumber++
	evt := *template
	evt.Type = eventType
	evt.SequenceNumber = seq
	return evt
}

func responsesUsageFromChatUsage(usage *ChatUsage) *ResponsesUsage {
	if usage == nil {
		return nil
	}
	out := &ResponsesUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}
	if out.TotalTokens == 0 {
		out.TotalTokens = out.InputTokens + out.OutputTokens
	}
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens > 0 {
		out.InputTokensDetails = &ResponsesInputTokensDetails{
			CachedTokens: usage.PromptTokensDetails.CachedTokens,
		}
	}
	return out
}

func scanChatCompletionsSSE(reader io.Reader, handle func(chunk *ChatCompletionsChunk) error, finalize func() error) error {
	bufReader := bufio.NewReader(reader)
	dataLines := make([]string, 0, 4)

	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if data == "" {
			return nil
		}
		if data == "[DONE]" {
			if err := finalize(); err != nil {
				return err
			}
			return errStopSSEScan
		}
		var chunk ChatCompletionsChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		return handle(&chunk)
	}

	for {
		line, err := bufReader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if line == "" {
				if err := flush(); err != nil {
					if errors.Is(err, errStopSSEScan) {
						return nil
					}
					return err
				}
			} else if !strings.HasPrefix(line, ":") && strings.HasPrefix(line, "data:") {
				value := strings.TrimPrefix(line, "data:")
				dataLines = append(dataLines, strings.TrimPrefix(value, " "))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			if errors.Is(err, errStopSSEScan) {
				return nil
			}
			return err
		}
	}
	if err := flush(); err != nil {
		if errors.Is(err, errStopSSEScan) {
			return nil
		}
		return err
	}
	return finalize()
}
