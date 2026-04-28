package utils

import (
	"encoding/json"
	"strings"
)

// redactJSONBody 对 JSON 字符串中的敏感字段进行递归脱敏。
func RedactJSONBody(body []byte) string {
	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return string(body)
	}
	redacted := RedactJSONValue(payload)
	data, err := json.Marshal(redacted)
	if err != nil {
		return string(body)
	}
	return string(data)
}

// redactJSONValue 递归遍历 JSON 值并替换敏感字段内容。
func RedactJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			if IsSensitiveName(key) {
				result[key] = "<redacted>"
				continue
			}
			result[key] = RedactJSONValue(item)
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			result = append(result, RedactJSONValue(item))
		}
		return result
	default:
		return value
	}
}

// isSensitiveName 判断字段名是否可能包含密钥、令牌、密码等敏感信息。
func IsSensitiveName(name string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", ".", "").Replace(name))
	for _, marker := range []string{"authorization", "apikey", "token", "secret", "password", "credential"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return normalized == "key"
}
