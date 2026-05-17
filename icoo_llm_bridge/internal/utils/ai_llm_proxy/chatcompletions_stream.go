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
}

func NewChatEventToResponsesState() *ChatEventToResponsesState {
	return &ChatEventToResponsesState{
		ResponseID: generateResponsesID(),
		Created:    time.Now().Unix(),
	}
}

// ChatChunkToResponsesEvents converts one Chat Completions chunk into zero or
// more Responses stream events. It intentionally handles text and usage first;
// streaming tool calls are left as a documented lossy fallback for this phase.
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
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			state.FinishReason = *choice.FinishReason
		}
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
	if state.TextItemOpen {
		events = append(events, chatMakeResponsesEvent(state, "response.output_text.done", &ResponsesStreamEvent{
			OutputIndex:  state.OutputIndex,
			ContentIndex: state.ContentIndex,
			ItemID:       state.TextItemID,
		}))
		events = append(events, chatMakeResponsesEvent(state, "response.output_item.done", &ResponsesStreamEvent{
			OutputIndex: state.OutputIndex,
			Item: &ResponsesOutput{
				Type:   "message",
				ID:     state.TextItemID,
				Status: "completed",
			},
		}))
		state.TextItemOpen = false
		state.OutputIndex++
	}

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
