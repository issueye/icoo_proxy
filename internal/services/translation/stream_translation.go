package translation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"icoo_proxy/internal/pkg/utils"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type sseEvent struct {
	Name string
	Data string
}

type anthropicStreamBlock struct {
	Index        int
	Key          string
	Type         string
	ItemID       string
	OutputIndex  int
	ContentIndex int
	ToolCallID   string
	ToolName     string
	HasDelta     bool
	Stopped      bool
}

type anthropicStreamState struct {
	logger         *slog.Logger
	w              http.ResponseWriter
	flusher        http.Flusher
	requestID      string
	model          string
	messageID      string
	messageStarted bool
	messageStopped bool
	nextBlockIndex int
	sawToolUse     bool
	inputTokens    int
	outputTokens   int
	textBlocks     map[string]*anthropicStreamBlock
	toolBlocks     map[string]*anthropicStreamBlock
}

type chatCompletionChunkDelta struct {
	Role      string                   `json:"role,omitempty"`
	Content   string                   `json:"content,omitempty"`
	ToolCalls []map[string]interface{} `json:"tool_calls,omitempty"`
}

type chatCompletionStreamState struct {
	logger            *slog.Logger
	w                 http.ResponseWriter
	flusher           http.Flusher
	requestID         string
	model             string
	responseID        string
	created           int64
	streamStarted     bool
	streamStopped     bool
	sentAssistantRole bool
	sawToolUse        bool
	inputTokens       int
	outputTokens      int
	toolCalls         map[string]*chatToolCallState
	toolCallOrder     []string
	textDeltaKeys     map[string]bool
}

type chatToolCallState struct {
	Index      int
	Key        string
	CallID     string
	Name       string
	Arguments  string
	Started    bool
	OutputDone bool
}

type anthropicChatBlockState struct {
	Index        int
	Type         string
	ToolCall     *chatToolCallState
	TextSent     bool
	InputSent    bool
	PendingInput string
}

type anthropicChatStreamState struct {
	logger            *slog.Logger
	w                 http.ResponseWriter
	flusher           http.Flusher
	requestID         string
	model             string
	responseID        string
	created           int64
	streamStopped     bool
	sentAssistantRole bool
	sawToolUse        bool
	inputTokens       int
	outputTokens      int
	finishReason      string
	toolCalls         map[string]*chatToolCallState
	toolCallOrder     []string
	blocks            map[int]*anthropicChatBlockState
}

func TranslateResponsesStreamToAnthropic(w http.ResponseWriter, body io.Reader, model, requestID string, logger *slog.Logger) (TokenUsage, error) {
	state := &anthropicStreamState{
		w:          w,
		logger:     logger,
		requestID:  requestID,
		model:      model,
		textBlocks: make(map[string]*anthropicStreamBlock),
		toolBlocks: make(map[string]*anthropicStreamBlock),
	}
	if flusher, ok := w.(http.Flusher); ok {
		state.flusher = flusher
	}

	reader := bufio.NewReader(body)
	for {
		event, err := readSSEEvent(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = state.emitErrorEvent(err.Error())
			return state.tokenUsage(), err
		}
		if strings.TrimSpace(event.Data) == "" {
			continue
		}
		state.logUpstreamEvent(event)
		if strings.TrimSpace(event.Data) == "[DONE]" {
			if err := state.finish(nil); err != nil {
				return state.tokenUsage(), err
			}
			return state.tokenUsage(), nil
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
			_ = state.emitErrorEvent("failed to decode upstream stream event")
			return state.tokenUsage(), fmt.Errorf("decode upstream stream event: %w", err)
		}
		eventType := strings.TrimSpace(event.Name)
		if eventType == "" {
			eventType = stringValue(payload["type"], "")
		}
		if eventType == "" {
			continue
		}

		if err := state.handleResponsesEvent(eventType, payload); err != nil {
			_ = state.emitErrorEvent(err.Error())
			return state.tokenUsage(), err
		}
		if state.messageStopped {
			return state.tokenUsage(), nil
		}
	}

	if err := state.finish(nil); err != nil {
		return state.tokenUsage(), err
	}
	return state.tokenUsage(), nil
}

func TranslateResponsesStreamToChat(w http.ResponseWriter, body io.Reader, model, requestID string, logger *slog.Logger) (TokenUsage, error) {
	state := &chatCompletionStreamState{
		w:             w,
		logger:        logger,
		requestID:     requestID,
		model:         model,
		created:       time.Now().Unix(),
		toolCalls:     make(map[string]*chatToolCallState),
		textDeltaKeys: make(map[string]bool),
	}
	if flusher, ok := w.(http.Flusher); ok {
		state.flusher = flusher
	}

	reader := bufio.NewReader(body)
	for {
		event, err := readSSEEvent(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return state.tokenUsage(), err
		}
		if strings.TrimSpace(event.Data) == "" {
			continue
		}
		state.logUpstreamEvent(event)
		if strings.TrimSpace(event.Data) == "[DONE]" {
			return state.tokenUsage(), state.finish(nil)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
			return state.tokenUsage(), fmt.Errorf("decode upstream stream event: %w", err)
		}
		eventType := strings.TrimSpace(event.Name)
		if eventType == "" {
			eventType = stringValue(payload["type"], "")
		}
		if eventType == "" {
			continue
		}
		if err := state.handleResponsesEvent(eventType, payload); err != nil {
			return state.tokenUsage(), err
		}
		if state.streamStopped {
			return state.tokenUsage(), nil
		}
	}

	return state.tokenUsage(), state.finish(nil)
}

func TranslateAnthropicStreamToChat(w http.ResponseWriter, body io.Reader, model, requestID string, logger *slog.Logger) (TokenUsage, error) {
	state := &anthropicChatStreamState{
		w:         w,
		logger:    logger,
		requestID: requestID,
		model:     model,
		created:   time.Now().Unix(),
		toolCalls: make(map[string]*chatToolCallState),
		blocks:    make(map[int]*anthropicChatBlockState),
	}
	if flusher, ok := w.(http.Flusher); ok {
		state.flusher = flusher
	}

	reader := bufio.NewReader(body)
	for {
		event, err := readSSEEvent(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return state.tokenUsage(), err
		}
		if strings.TrimSpace(event.Data) == "" {
			continue
		}
		state.logUpstreamEvent(event)
		if strings.TrimSpace(event.Data) == "[DONE]" {
			return state.tokenUsage(), state.finish(nil)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
			return state.tokenUsage(), fmt.Errorf("decode upstream stream event: %w", err)
		}
		eventType := strings.TrimSpace(event.Name)
		if eventType == "" {
			eventType = stringValue(payload["type"], "")
		}
		if eventType == "" {
			continue
		}
		if err := state.handleAnthropicEvent(eventType, payload); err != nil {
			return state.tokenUsage(), err
		}
		if state.streamStopped {
			return state.tokenUsage(), nil
		}
	}

	return state.tokenUsage(), state.finish(nil)
}

func readSSEEvent(reader *bufio.Reader) (sseEvent, error) {
	var (
		name      string
		dataLines []string
	)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return sseEvent{}, err
		}
		if len(line) == 0 && err == io.EOF {
			if len(dataLines) == 0 && name == "" {
				return sseEvent{}, io.EOF
			}
			return sseEvent{Name: name, Data: strings.Join(dataLines, "\n")}, nil
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			if len(dataLines) == 0 && name == "" {
				if err == io.EOF {
					return sseEvent{}, io.EOF
				}
				continue
			}
			return sseEvent{Name: name, Data: strings.Join(dataLines, "\n")}, nil
		}
		if strings.HasPrefix(trimmed, ":") {
			if err == io.EOF {
				return sseEvent{Name: name, Data: strings.Join(dataLines, "\n")}, nil
			}
			continue
		}
		if strings.HasPrefix(trimmed, "event:") {
			name = strings.TrimSpace(trimmed[len("event:"):])
		}
		if strings.HasPrefix(trimmed, "data:") {
			value := trimmed[len("data:"):]
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
			dataLines = append(dataLines, value)
		}
		if err == io.EOF {
			return sseEvent{Name: name, Data: strings.Join(dataLines, "\n")}, nil
		}
	}
}

func (s *anthropicStreamState) handleResponsesEvent(eventType string, payload map[string]interface{}) error {
	switch eventType {
	case "response.created":
		response := objectValue(payload["response"])
		s.messageID = firstNonEmpty(
			stringValue(response["id"], ""),
			stringValue(payload["response_id"], ""),
			s.messageID,
			"msg_proxy_stream",
		)
		usage := mapUsageToAnthropic(response["usage"])
		s.inputTokens = intValue(usage["input_tokens"])
		if s.inputTokens == 0 {
			s.inputTokens = intValue(objectValue(response["usage"])["input_tokens"])
		}
		return s.ensureMessageStart()
	case "response.output_text.delta":
		if err := s.ensureMessageStart(); err != nil {
			return err
		}
		block, err := s.ensureTextBlock(payload)
		if err != nil {
			return err
		}
		text := stringValue(payload["delta"], "")
		if text == "" {
			return nil
		}
		block.HasDelta = true
		return s.emitEvent("content_block_delta", map[string]interface{}{
			"type":  "content_block_delta",
			"index": block.Index,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": text,
			},
		})
	case "response.output_text.done":
		if err := s.ensureMessageStart(); err != nil {
			return err
		}
		block, err := s.ensureTextBlock(payload)
		if err != nil {
			return err
		}
		text := stringValue(payload["text"], "")
		if text != "" && !block.HasDelta {
			block.HasDelta = true
			if err := s.emitEvent("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": block.Index,
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": text,
				},
			}); err != nil {
				return err
			}
		}
		return s.stopBlock(block)
	case "response.output_item.added":
		item := objectValue(payload["item"])
		if stringValue(item["type"], "") != "function_call" {
			return nil
		}
		if err := s.ensureMessageStart(); err != nil {
			return err
		}
		_, err := s.ensureToolBlock(payload, item)
		return err
	case "response.function_call_arguments.delta":
		if err := s.ensureMessageStart(); err != nil {
			return err
		}
		block, err := s.ensureToolBlock(payload, nil)
		if err != nil {
			return err
		}
		delta := stringValue(payload["delta"], "")
		if delta == "" {
			return nil
		}
		block.HasDelta = true
		s.sawToolUse = true
		return s.emitEvent("content_block_delta", map[string]interface{}{
			"type":  "content_block_delta",
			"index": block.Index,
			"delta": map[string]interface{}{
				"type":         "input_json_delta",
				"partial_json": delta,
			},
		})
	case "response.function_call_arguments.done":
		if err := s.ensureMessageStart(); err != nil {
			return err
		}
		block, err := s.ensureToolBlock(payload, map[string]interface{}{
			"type":    "function_call",
			"id":      firstNonEmpty(stringValue(payload["item_id"], ""), stringValue(payload["id"], "")),
			"call_id": stringValue(payload["call_id"], ""),
			"name":    stringValue(payload["name"], ""),
		})
		if err != nil {
			return err
		}
		arguments := stringValue(payload["arguments"], "")
		if arguments != "" && !block.HasDelta {
			block.HasDelta = true
			if err := s.emitEvent("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": block.Index,
				"delta": map[string]interface{}{
					"type":         "input_json_delta",
					"partial_json": arguments,
				},
			}); err != nil {
				return err
			}
		}
		return s.stopBlock(block)
	case "response.output_item.done":
		item := objectValue(payload["item"])
		if stringValue(item["type"], "") != "function_call" {
			return nil
		}
		if err := s.ensureMessageStart(); err != nil {
			return err
		}
		block, err := s.ensureToolBlock(payload, item)
		if err != nil {
			return err
		}
		arguments := stringValue(item["arguments"], "")
		if arguments != "" && !block.HasDelta {
			block.HasDelta = true
			if err := s.emitEvent("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": block.Index,
				"delta": map[string]interface{}{
					"type":         "input_json_delta",
					"partial_json": arguments,
				},
			}); err != nil {
				return err
			}
		}
		return s.stopBlock(block)
	case "response.completed":
		response := objectValue(payload["response"])
		return s.finish(response)
	case "response.failed":
		response := objectValue(payload["response"])
		message := "upstream response failed"
		if errObj := objectValue(response["error"]); errObj != nil {
			message = stringValue(errObj["message"], message)
		}
		if err := s.emitErrorEvent(message); err != nil {
			return err
		}
		s.messageStopped = true
		return nil
	case "error":
		errObj := objectValue(payload["error"])
		message := stringValue(errObj["message"], "upstream stream returned error")
		if err := s.emitEvent("error", map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"type":    firstNonEmpty(stringValue(errObj["type"], ""), "api_error"),
				"message": message,
			},
		}); err != nil {
			return err
		}
		s.messageStopped = true
		return nil
	default:
		return nil
	}
}

func (s *chatCompletionStreamState) handleResponsesEvent(eventType string, payload map[string]interface{}) error {
	switch eventType {
	case "response.created":
		response := objectValue(payload["response"])
		s.responseID = firstNonEmpty(
			stringValue(response["id"], ""),
			stringValue(payload["response_id"], ""),
			s.responseID,
			"chatcmpl-proxy-stream",
		)
		usage := mapUsageToChat(response["usage"])
		s.inputTokens = intValue(usage["prompt_tokens"])
		s.outputTokens = intValue(usage["completion_tokens"])
		return nil
	case "response.output_text.delta":
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		text := stringValue(payload["delta"], "")
		if text == "" {
			return nil
		}
		s.markTextDeltaSeen(payload)
		return s.emitChunk(chatCompletionChunkDelta{Content: text}, "")
	case "response.output_text.done":
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		text := stringValue(payload["text"], "")
		if text == "" {
			return nil
		}
		if s.hasTextDelta(payload) {
			return nil
		}
		return s.emitChunk(chatCompletionChunkDelta{Content: text}, "")
	case "response.output_item.added":
		item := objectValue(payload["item"])
		if stringValue(item["type"], "") != "function_call" {
			return nil
		}
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		toolCall := s.ensureToolCall(payload, item)
		s.sawToolUse = true
		return s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallStartDelta(toolCall)}}, "")
	case "response.function_call_arguments.delta":
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		toolCall := s.ensureToolCall(payload, nil)
		delta := stringValue(payload["delta"], "")
		if delta == "" {
			return nil
		}
		toolCall.Arguments += delta
		s.sawToolUse = true
		return s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallArgumentsDelta(toolCall, delta)}}, "")
	case "response.function_call_arguments.done":
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		toolCall := s.ensureToolCall(payload, map[string]interface{}{
			"type":    "function_call",
			"id":      firstNonEmpty(stringValue(payload["item_id"], ""), stringValue(payload["id"], "")),
			"call_id": stringValue(payload["call_id"], ""),
			"name":    stringValue(payload["name"], ""),
		})
		arguments := stringValue(payload["arguments"], "")
		if arguments == "" || toolCall.Arguments != "" {
			return nil
		}
		toolCall.Arguments = arguments
		s.sawToolUse = true
		return s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallArgumentsDelta(toolCall, arguments)}}, "")
	case "response.output_item.done":
		item := objectValue(payload["item"])
		if stringValue(item["type"], "") != "function_call" {
			return nil
		}
		toolCall := s.ensureToolCall(payload, item)
		if toolCall.OutputDone {
			return nil
		}
		arguments := stringValue(item["arguments"], "")
		if arguments != "" && toolCall.Arguments == "" {
			toolCall.Arguments = arguments
			if err := s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallArgumentsDelta(toolCall, arguments)}}, ""); err != nil {
				return err
			}
		}
		toolCall.OutputDone = true
		return nil
	case "response.completed":
		response := objectValue(payload["response"])
		return s.finish(response)
	case "response.failed":
		response := objectValue(payload["response"])
		message := "upstream response failed"
		if errObj := objectValue(response["error"]); errObj != nil {
			message = stringValue(errObj["message"], message)
		}
		return s.emitError(message)
	case "error":
		errObj := objectValue(payload["error"])
		return s.emitError(stringValue(errObj["message"], "upstream stream returned error"))
	default:
		return nil
	}
}

func (s *anthropicChatStreamState) handleAnthropicEvent(eventType string, payload map[string]interface{}) error {
	switch eventType {
	case "message_start":
		message := objectValue(payload["message"])
		s.responseID = firstNonEmpty(
			stringValue(message["id"], ""),
			stringValue(payload["message_id"], ""),
			s.responseID,
			"chatcmpl-proxy-stream",
		)
		usage := objectValue(message["usage"])
		if value := intValue(usage["input_tokens"]); value > 0 {
			s.inputTokens = value
		}
		if value := intValue(usage["output_tokens"]); value > 0 {
			s.outputTokens = value
		}
		return nil
	case "content_block_start":
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		return s.handleAnthropicBlockStart(payload)
	case "content_block_delta":
		if err := s.ensureAssistantRoleChunk(); err != nil {
			return err
		}
		return s.handleAnthropicBlockDelta(payload)
	case "content_block_stop":
		return s.handleAnthropicBlockStop(payload)
	case "message_delta":
		return s.handleAnthropicMessageDelta(payload)
	case "message_stop":
		return s.finish(nil)
	case "error":
		errObj := objectValue(payload["error"])
		return s.emitError(stringValue(errObj["message"], "upstream stream returned error"))
	default:
		return nil
	}
}

func (s *anthropicStreamState) ensureMessageStart() error {
	if s.messageStarted {
		return nil
	}
	if s.messageID == "" {
		s.messageID = "msg_proxy_stream"
	}
	s.messageStarted = true
	return s.emitEvent("message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            s.messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         s.model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  s.inputTokens,
				"output_tokens": s.outputTokens,
			},
		},
	})
}

func (s *chatCompletionStreamState) ensureAssistantRoleChunk() error {
	if s.sentAssistantRole {
		return nil
	}
	s.sentAssistantRole = true
	return s.emitChunk(chatCompletionChunkDelta{Role: "assistant"}, "")
}

func (s *anthropicChatStreamState) ensureAssistantRoleChunk() error {
	if s.sentAssistantRole {
		return nil
	}
	s.sentAssistantRole = true
	return s.emitChunk(chatCompletionChunkDelta{Role: "assistant"}, "")
}

func (s *chatCompletionStreamState) markTextDeltaSeen(payload map[string]interface{}) {
	key := s.textDeltaKey(payload)
	if key == "" {
		return
	}
	s.textDeltaKeys[key] = true
}

func (s *chatCompletionStreamState) hasTextDelta(payload map[string]interface{}) bool {
	key := s.textDeltaKey(payload)
	if key == "" {
		return false
	}
	return s.textDeltaKeys[key]
}

func (s *chatCompletionStreamState) textDeltaKey(payload map[string]interface{}) string {
	itemID := firstNonEmpty(stringValue(payload["item_id"], ""), stringValue(payload["id"], ""))
	outputIndex := intValue(payload["output_index"])
	contentIndex := intValue(payload["content_index"])
	return fmt.Sprintf("%s:%d:%d", itemID, outputIndex, contentIndex)
}

func (s *anthropicStreamState) ensureTextBlock(payload map[string]interface{}) (*anthropicStreamBlock, error) {
	itemID := firstNonEmpty(stringValue(payload["item_id"], ""), stringValue(payload["id"], ""))
	outputIndex := intValue(payload["output_index"])
	contentIndex := intValue(payload["content_index"])
	key := fmt.Sprintf("%s:%d:%d", itemID, outputIndex, contentIndex)
	if block, ok := s.textBlocks[key]; ok {
		return block, nil
	}
	block := &anthropicStreamBlock{
		Index:        s.nextBlockIndex,
		Key:          key,
		Type:         "text",
		ItemID:       itemID,
		OutputIndex:  outputIndex,
		ContentIndex: contentIndex,
	}
	s.nextBlockIndex++
	s.textBlocks[key] = block
	if err := s.emitEvent("content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": block.Index,
		"content_block": map[string]interface{}{
			"type": "text",
			"text": "",
		},
	}); err != nil {
		return nil, err
	}
	return block, nil
}

func (s *anthropicStreamState) ensureToolBlock(payload map[string]interface{}, item map[string]interface{}) (*anthropicStreamBlock, error) {
	itemID := firstNonEmpty(
		stringValue(payload["item_id"], ""),
		stringValue(objectValue(item)["id"], ""),
		stringValue(payload["id"], ""),
	)
	outputIndex := intValue(payload["output_index"])
	key := firstNonEmpty(itemID, strconv.Itoa(outputIndex))
	if block, ok := s.toolBlocks[key]; ok {
		if block.ToolName == "" {
			block.ToolName = stringValue(objectValue(item)["name"], "")
		}
		if block.ToolCallID == "" {
			block.ToolCallID = firstNonEmpty(
				stringValue(objectValue(item)["call_id"], ""),
				stringValue(payload["call_id"], ""),
				itemID,
			)
		}
		return block, nil
	}
	callID := firstNonEmpty(
		stringValue(objectValue(item)["call_id"], ""),
		stringValue(payload["call_id"], ""),
		itemID,
	)
	name := firstNonEmpty(
		stringValue(objectValue(item)["name"], ""),
		stringValue(payload["name"], ""),
	)
	block := &anthropicStreamBlock{
		Index:       s.nextBlockIndex,
		Key:         key,
		Type:        "tool_use",
		ItemID:      itemID,
		OutputIndex: outputIndex,
		ToolCallID:  callID,
		ToolName:    name,
	}
	s.nextBlockIndex++
	s.toolBlocks[key] = block
	s.sawToolUse = true
	if err := s.emitEvent("content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": block.Index,
		"content_block": map[string]interface{}{
			"type":  "tool_use",
			"id":    firstNonEmpty(block.ToolCallID, block.ItemID, fmt.Sprintf("tool_%d", block.Index)),
			"name":  block.ToolName,
			"input": map[string]interface{}{},
		},
	}); err != nil {
		return nil, err
	}
	return block, nil
}

func (s *chatCompletionStreamState) ensureToolCall(payload map[string]interface{}, item map[string]interface{}) *chatToolCallState {
	itemMap := objectValue(item)
	itemID := firstNonEmpty(
		stringValue(payload["item_id"], ""),
		stringValue(itemMap["id"], ""),
		stringValue(payload["id"], ""),
	)
	outputIndex := intValue(payload["output_index"])
	key := firstNonEmpty(itemID, strconv.Itoa(outputIndex))
	if toolCall, ok := s.toolCalls[key]; ok {
		if toolCall.CallID == "" {
			toolCall.CallID = firstNonEmpty(stringValue(itemMap["call_id"], ""), stringValue(payload["call_id"], ""), itemID)
		}
		if toolCall.Name == "" {
			toolCall.Name = firstNonEmpty(stringValue(itemMap["name"], ""), stringValue(payload["name"], ""))
		}
		return toolCall
	}
	toolCall := &chatToolCallState{
		Index:  len(s.toolCallOrder),
		Key:    key,
		CallID: firstNonEmpty(stringValue(itemMap["call_id"], ""), stringValue(payload["call_id"], ""), itemID),
		Name:   firstNonEmpty(stringValue(itemMap["name"], ""), stringValue(payload["name"], "")),
	}
	s.toolCalls[key] = toolCall
	s.toolCallOrder = append(s.toolCallOrder, key)
	return toolCall
}

func (s *anthropicChatStreamState) handleAnthropicBlockStart(payload map[string]interface{}) error {
	index := intValue(payload["index"])
	contentBlock := objectValue(payload["content_block"])
	block := s.ensureBlock(index)
	block.Type = firstNonEmpty(stringValue(contentBlock["type"], ""), block.Type, "text")

	switch block.Type {
	case "tool_use":
		toolCall := s.ensureToolCallForBlock(block, contentBlock)
		s.sawToolUse = true
		if err := s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallStartDelta(toolCall)}}, ""); err != nil {
			return err
		}
		if rawInput, ok := contentBlock["input"]; ok {
			if data, err := json.Marshal(rawInput); err == nil {
				block.PendingInput = string(data)
			}
		}
		return nil
	case "text":
		text := stringValue(contentBlock["text"], "")
		if text == "" {
			return nil
		}
		block.TextSent = true
		return s.emitChunk(chatCompletionChunkDelta{Content: text}, "")
	default:
		return nil
	}
}

func (s *anthropicChatStreamState) handleAnthropicBlockDelta(payload map[string]interface{}) error {
	block := s.ensureBlock(intValue(payload["index"]))
	delta := objectValue(payload["delta"])
	switch stringValue(delta["type"], "") {
	case "text_delta":
		text := stringValue(delta["text"], "")
		if text == "" {
			return nil
		}
		block.Type = firstNonEmpty(block.Type, "text")
		block.TextSent = true
		return s.emitChunk(chatCompletionChunkDelta{Content: text}, "")
	case "input_json_delta":
		toolCall := s.ensureToolCallForBlock(block, nil)
		partial := stringValue(delta["partial_json"], "")
		if partial == "" {
			return nil
		}
		toolCall.Arguments += partial
		block.InputSent = true
		s.sawToolUse = true
		return s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallArgumentsDelta(toolCall, partial)}}, "")
	default:
		return nil
	}
}

func (s *anthropicChatStreamState) handleAnthropicBlockStop(payload map[string]interface{}) error {
	block, ok := s.blocks[intValue(payload["index"])]
	if !ok || block == nil || block.Type != "tool_use" || block.ToolCall == nil {
		return nil
	}
	if block.InputSent || block.ToolCall.Arguments != "" || block.PendingInput == "" {
		return nil
	}
	block.ToolCall.Arguments = block.PendingInput
	block.InputSent = true
	return s.emitChunk(chatCompletionChunkDelta{ToolCalls: []map[string]interface{}{toolCallArgumentsDelta(block.ToolCall, block.PendingInput)}}, "")
}

func (s *anthropicChatStreamState) handleAnthropicMessageDelta(payload map[string]interface{}) error {
	delta := objectValue(payload["delta"])
	if stopReason := stringValue(delta["stop_reason"], ""); stopReason != "" {
		s.finishReason = stopReason
	}
	usage := objectValue(payload["usage"])
	if _, ok := usage["input_tokens"]; ok {
		s.inputTokens = intValue(usage["input_tokens"])
	}
	if _, ok := usage["output_tokens"]; ok {
		s.outputTokens = intValue(usage["output_tokens"])
	}
	return nil
}

func (s *anthropicChatStreamState) ensureBlock(index int) *anthropicChatBlockState {
	if block, ok := s.blocks[index]; ok {
		return block
	}
	block := &anthropicChatBlockState{Index: index}
	s.blocks[index] = block
	return block
}

func (s *anthropicChatStreamState) ensureToolCallForBlock(block *anthropicChatBlockState, contentBlock map[string]interface{}) *chatToolCallState {
	if block == nil {
		block = s.ensureBlock(0)
	}
	if block.ToolCall != nil {
		if block.ToolCall.CallID == "" {
			block.ToolCall.CallID = firstNonEmpty(stringValue(contentBlock["id"], ""), block.ToolCall.Key, fmt.Sprintf("call_%d", block.ToolCall.Index))
		}
		if block.ToolCall.Name == "" {
			block.ToolCall.Name = stringValue(contentBlock["name"], "")
		}
		return block.ToolCall
	}
	key := fmt.Sprintf("anthropic:%d", block.Index)
	toolCall := &chatToolCallState{
		Index:  len(s.toolCallOrder),
		Key:    key,
		CallID: firstNonEmpty(stringValue(contentBlock["id"], ""), key, fmt.Sprintf("call_%d", block.Index)),
		Name:   stringValue(contentBlock["name"], ""),
	}
	block.Type = "tool_use"
	block.ToolCall = toolCall
	s.toolCalls[key] = toolCall
	s.toolCallOrder = append(s.toolCallOrder, key)
	return toolCall
}

func (s *anthropicStreamState) stopBlock(block *anthropicStreamBlock) error {
	if block == nil || block.Stopped {
		return nil
	}
	block.Stopped = true
	return s.emitEvent("content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": block.Index,
	})
}

func (s *anthropicStreamState) finish(response map[string]interface{}) error {
	if s.messageStopped {
		return nil
	}
	if response != nil {
		s.messageID = firstNonEmpty(stringValue(response["id"], ""), s.messageID)
		usage := mapUsageToAnthropic(response["usage"])
		if value := intValue(usage["input_tokens"]); value > 0 {
			s.inputTokens = value
		}
		if value := intValue(usage["output_tokens"]); value > 0 {
			s.outputTokens = value
		}
	}
	if err := s.ensureMessageStart(); err != nil {
		return err
	}
	for _, block := range s.sortedBlocks() {
		if err := s.stopBlock(block); err != nil {
			return err
		}
	}
	stopReason := "end_turn"
	if response != nil {
		stopReason = mapResponsesStatusToAnthropicStopReason(response)
	}
	if s.sawToolUse {
		stopReason = "tool_use"
	}
	if err := s.emitEvent("message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]interface{}{
			"input_tokens":  s.inputTokens,
			"output_tokens": s.outputTokens,
		},
	}); err != nil {
		return err
	}
	if err := s.emitEvent("message_stop", map[string]interface{}{
		"type": "message_stop",
	}); err != nil {
		return err
	}
	s.messageStopped = true
	return nil
}

func (s *chatCompletionStreamState) finish(response map[string]interface{}) error {
	if s.streamStopped {
		return nil
	}
	if response != nil {
		s.responseID = firstNonEmpty(stringValue(response["id"], ""), s.responseID)
		usage := mapUsageToChat(response["usage"])
		if value := intValue(usage["prompt_tokens"]); value > 0 {
			s.inputTokens = value
		}
		if value := intValue(usage["completion_tokens"]); value > 0 {
			s.outputTokens = value
		}
	}
	finishReason := "stop"
	if response != nil {
		finishReason = mapResponsesStatusToFinishReason(response)
	}
	if s.sawToolUse {
		finishReason = "tool_calls"
	}
	if err := s.ensureAssistantRoleChunk(); err != nil {
		return err
	}
	if err := s.emitChunk(chatCompletionChunkDelta{}, finishReason); err != nil {
		return err
	}
	if _, err := fmt.Fprint(s.w, "data: [DONE]\n\n"); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.streamStopped = true
	return nil
}

func (s *anthropicChatStreamState) finish(response map[string]interface{}) error {
	if s.streamStopped {
		return nil
	}
	if response != nil {
		s.responseID = firstNonEmpty(stringValue(response["id"], ""), s.responseID)
		usage := objectValue(response["usage"])
		if value := intValue(usage["input_tokens"]); value > 0 {
			s.inputTokens = value
		}
		if value := intValue(usage["output_tokens"]); value > 0 {
			s.outputTokens = value
		}
	}
	finishReason := "stop"
	switch s.finishReason {
	case "max_tokens":
		finishReason = "length"
	case "tool_use":
		finishReason = "tool_calls"
	case "end_turn", "stop_sequence", "":
		finishReason = "stop"
	default:
		finishReason = "stop"
	}
	if s.sawToolUse {
		finishReason = "tool_calls"
	}
	if err := s.ensureAssistantRoleChunk(); err != nil {
		return err
	}
	if err := s.emitChunk(chatCompletionChunkDelta{}, finishReason); err != nil {
		return err
	}
	if _, err := fmt.Fprint(s.w, "data: [DONE]\n\n"); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.streamStopped = true
	return nil
}

func (s *chatCompletionStreamState) emitError(message string) error {
	if s.streamStopped {
		return nil
	}
	payload := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "api_error",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.logDownstreamEvent("error", data)
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.streamStopped = true
	return nil
}

func (s *anthropicChatStreamState) emitError(message string) error {
	if s.streamStopped {
		return nil
	}
	payload := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "api_error",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.logDownstreamEvent("error", data)
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.streamStopped = true
	return nil
}

func (s *anthropicStreamState) sortedBlocks() []*anthropicStreamBlock {
	items := make([]*anthropicStreamBlock, 0, len(s.textBlocks)+len(s.toolBlocks))
	for _, block := range s.textBlocks {
		items = append(items, block)
	}
	for _, block := range s.toolBlocks {
		items = append(items, block)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}

func toolCallStartDelta(toolCall *chatToolCallState) map[string]interface{} {
	return map[string]interface{}{
		"index": toolCall.Index,
		"id":    firstNonEmpty(toolCall.CallID, toolCall.Key, fmt.Sprintf("call_%d", toolCall.Index)),
		"type":  "function",
		"function": map[string]interface{}{
			"name":      toolCall.Name,
			"arguments": "",
		},
	}
}

func toolCallArgumentsDelta(toolCall *chatToolCallState, delta string) map[string]interface{} {
	return map[string]interface{}{
		"index": toolCall.Index,
		"function": map[string]interface{}{
			"arguments": delta,
		},
	}
}

func (s *chatCompletionStreamState) emitChunk(delta chatCompletionChunkDelta, finishReason string) error {
	if s.responseID == "" {
		s.responseID = "chatcmpl-proxy-stream"
	}
	if s.created == 0 {
		s.created = time.Now().Unix()
	}
	payload := map[string]interface{}{
		"id":      s.responseID,
		"object":  "chat.completion.chunk",
		"created": s.created,
		"model":   s.model,
		"choices": []map[string]interface{}{{
			"index": 0,
			"delta": map[string]interface{}{},
		}},
	}
	choice := payload["choices"].([]map[string]interface{})[0]
	deltaMap := choice["delta"].(map[string]interface{})
	if delta.Role != "" {
		deltaMap["role"] = delta.Role
	}
	if delta.Content != "" {
		deltaMap["content"] = delta.Content
	}
	if len(delta.ToolCalls) > 0 {
		deltaMap["tool_calls"] = delta.ToolCalls
	}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	} else {
		choice["finish_reason"] = nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.logDownstreamEvent("chat.completion.chunk", data)
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *anthropicChatStreamState) emitChunk(delta chatCompletionChunkDelta, finishReason string) error {
	if s.responseID == "" {
		s.responseID = "chatcmpl-proxy-stream"
	}
	if s.created == 0 {
		s.created = time.Now().Unix()
	}
	payload := map[string]interface{}{
		"id":      s.responseID,
		"object":  "chat.completion.chunk",
		"created": s.created,
		"model":   s.model,
		"choices": []map[string]interface{}{{
			"index": 0,
			"delta": map[string]interface{}{},
		}},
	}
	choice := payload["choices"].([]map[string]interface{})[0]
	deltaMap := choice["delta"].(map[string]interface{})
	if delta.Role != "" {
		deltaMap["role"] = delta.Role
	}
	if delta.Content != "" {
		deltaMap["content"] = delta.Content
	}
	if len(delta.ToolCalls) > 0 {
		deltaMap["tool_calls"] = delta.ToolCalls
	}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	} else {
		choice["finish_reason"] = nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.logDownstreamEvent("chat.completion.chunk", data)
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *anthropicStreamState) emitErrorEvent(message string) error {
	return s.emitEvent("error", map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    "api_error",
			"message": message,
		},
	})
}

func (s *anthropicStreamState) emitEvent(eventName string, payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.logDownstreamEvent(eventName, data)
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventName, data); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *anthropicStreamState) logUpstreamEvent(event sseEvent) {
	s.logChain("conversion.stream.upstream_event",
		"request_id", s.requestID,
		"event", firstNonEmpty(event.Name, "<data-only>"),
		"data", utils.RedactJSONBody([]byte(event.Data)),
	)
}

func (s *chatCompletionStreamState) logUpstreamEvent(event sseEvent) {
	s.logChain("conversion.stream.upstream_event",
		"request_id", s.requestID,
		"event", firstNonEmpty(event.Name, "<data-only>"),
		"data", utils.RedactJSONBody([]byte(event.Data)),
	)
}

func (s *anthropicChatStreamState) logUpstreamEvent(event sseEvent) {
	s.logChain("conversion.stream.upstream_event",
		"request_id", s.requestID,
		"event", firstNonEmpty(event.Name, "<data-only>"),
		"data", utils.RedactJSONBody([]byte(event.Data)),
	)
}

// logChain 写入结构化链路日志；未配置日志器时直接忽略。
func (s *anthropicStreamState) logChain(event string, attrs ...interface{}) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Info(event, attrs...)
}

func (s *chatCompletionStreamState) logChain(event string, attrs ...interface{}) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Info(event, attrs...)
}

func (s *anthropicChatStreamState) logChain(event string, attrs ...interface{}) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Info(event, attrs...)
}

func (s *anthropicStreamState) logDownstreamEvent(eventName string, payload []byte) {
	s.logChain("conversion.stream.downstream_event",
		"request_id", s.requestID,
		"event", eventName,
		"data", utils.RedactJSONBody(payload),
	)
}

func (s *chatCompletionStreamState) logDownstreamEvent(eventName string, payload []byte) {
	s.logChain("conversion.stream.downstream_event",
		"request_id", s.requestID,
		"event", eventName,
		"data", utils.RedactJSONBody(payload),
	)
}

func (s *anthropicChatStreamState) logDownstreamEvent(eventName string, payload []byte) {
	s.logChain("conversion.stream.downstream_event",
		"request_id", s.requestID,
		"event", eventName,
		"data", utils.RedactJSONBody(payload),
	)
}

func (s *anthropicStreamState) tokenUsage() TokenUsage {
	return TokenUsage{
		InputTokens:  s.inputTokens,
		OutputTokens: s.outputTokens,
	}.Normalize()
}

func (s *chatCompletionStreamState) tokenUsage() TokenUsage {
	return TokenUsage{
		InputTokens:  s.inputTokens,
		OutputTokens: s.outputTokens,
	}.Normalize()
}

func (s *anthropicChatStreamState) tokenUsage() TokenUsage {
	return TokenUsage{
		InputTokens:  s.inputTokens,
		OutputTokens: s.outputTokens,
	}.Normalize()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
