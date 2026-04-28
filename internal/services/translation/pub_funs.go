package translation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"icoo_proxy/internal/consts"
	"io"
	"strings"
)

func StringValue(raw interface{}, fallback string) string {
	if value, ok := raw.(string); ok && value != "" {
		return value
	}
	return fallback
}

// rewriteResponsesRequest 改写 OpenAI Responses 请求模型，并补齐默认 reasoning 配置。
func RewriteResponsesRequest(body []byte, model string) ([]byte, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid json body")
	}
	payload["model"] = model
	ApplyDefaultResponsesReasoning(consts.DefaultResponsesReasoningEffort, payload)
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite request body")
	}
	return rewritten, nil
}

// ApplyDefaultResponsesReasoning 为 OpenAI Responses 请求添加默认 reasoning 配置。
func ApplyDefaultResponsesReasoning(defaultEffort string, payload map[string]any) {
	if payload == nil {
		return
	}
	raw, ok := payload["reasoning"]
	if !ok || raw == nil {
		payload["reasoning"] = map[string]any{"effort": defaultEffort}
		return
	}
	reasoning, ok := raw.(map[string]any)
	if !ok {
		return
	}
	if strings.TrimSpace(StringValue(reasoning["effort"], "")) == "" {
		reasoning["effort"] = defaultEffort
	}
	payload["reasoning"] = reasoning
}

func AggregateResponsesStreamToJSON(body io.Reader) ([]byte, error) {
	aggregate := &responsesStreamAggregate{
		outputItems:    make(map[int]map[string]interface{}),
		outputTexts:    make(map[int]string),
		outputItemSeen: make(map[int]bool),
	}
	reader := bufio.NewReader(body)
	for {
		event, err := readSSEEvent(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data := strings.TrimSpace(event.Data)
		if data == "" || data == "[DONE]" {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return nil, fmt.Errorf("decode upstream stream event: %w", err)
		}
		eventType := strings.TrimSpace(event.Name)
		if eventType == "" {
			eventType = stringValue(payload["type"], "")
		}
		switch eventType {
		case "response.created", "response.in_progress", "response.completed", "response.failed":
			response := objectValue(payload["response"])
			if response != nil {
				aggregate.response = cloneMap(response)
			}
		case "response.output_item.done":
			item := objectValue(payload["item"])
			if item == nil {
				continue
			}
			outputIndex := intValue(payload["output_index"])
			aggregate.outputItems[outputIndex] = cloneMap(item)
			aggregate.outputItemSeen[outputIndex] = true
		case "response.output_text.done":
			outputIndex := intValue(payload["output_index"])
			text := stringValue(payload["text"], "")
			if strings.TrimSpace(text) != "" || text != "" {
				aggregate.outputTexts[outputIndex] = text
			}
		case "error":
			errObj := objectValue(payload["error"])
			if errObj != nil {
				return nil, fmt.Errorf(stringValue(errObj["message"], "upstream stream returned error"))
			}
			return nil, fmt.Errorf("upstream stream returned error")
		}
	}

	response := aggregate.response
	if response == nil {
		response = map[string]interface{}{
			"id":     "resp_proxy_stream",
			"object": "response",
			"status": "completed",
		}
	} else {
		response = cloneMap(response)
	}

	if len(aggregate.outputItemSeen) > 0 {
		response["output"] = aggregate.orderedOutputItems()
	} else if len(aggregate.outputTexts) > 0 {
		response["output"] = aggregate.syntheticMessageOutput()
	}

	if text := strings.TrimSpace(extractResponsesOutputText(response["output"])); text != "" {
		response["output_text"] = text
	} else if text := aggregate.combinedOutputText(); text != "" {
		response["output_text"] = text
	}
	return json.Marshal(response)
}

// rewriteModel 将请求体中的 model 改写为路由解析后的目标模型。
func RewriteModel(body []byte, model string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid json body")
	}
	payload["model"] = model
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite request body")
	}
	return rewritten, nil
}
