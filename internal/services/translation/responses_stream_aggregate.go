package translation

import (
	"fmt"
	"sort"
	"strings"
)

type responsesStreamAggregate struct {
	response       map[string]interface{}
	outputItems    map[int]map[string]interface{}
	outputTexts    map[int]string
	outputItemSeen map[int]bool
}

func (a *responsesStreamAggregate) orderedOutputItems() []map[string]interface{} {
	indexes := make([]int, 0, len(a.outputItemSeen))
	for index := range a.outputItemSeen {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	items := make([]map[string]interface{}, 0, len(indexes))
	for _, index := range indexes {
		item := cloneMap(a.outputItems[index])
		if item == nil {
			item = map[string]interface{}{}
		}
		content, _ := item["content"].([]interface{})
		if stringValue(item["type"], "") == "message" && len(content) == 0 {
			if text := a.outputTexts[index]; text != "" {
				item["content"] = []interface{}{
					map[string]interface{}{
						"type": "output_text",
						"text": text,
					},
				}
			}
		}
		items = append(items, item)
	}
	return items
}

func (a *responsesStreamAggregate) syntheticMessageOutput() []map[string]interface{} {
	indexes := make([]int, 0, len(a.outputTexts))
	for index := range a.outputTexts {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	items := make([]map[string]interface{}, 0, len(indexes))
	for _, index := range indexes {
		items = append(items, map[string]interface{}{
			"id":   fmt.Sprintf("msg_proxy_%d", index),
			"type": "message",
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type": "output_text",
					"text": a.outputTexts[index],
				},
			},
		})
	}
	return items
}

func (a *responsesStreamAggregate) combinedOutputText() string {
	if len(a.outputTexts) == 0 {
		return ""
	}
	indexes := make([]int, 0, len(a.outputTexts))
	for index := range a.outputTexts {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	parts := make([]string, 0, len(indexes))
	for _, index := range indexes {
		if text := strings.TrimSpace(a.outputTexts[index]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
