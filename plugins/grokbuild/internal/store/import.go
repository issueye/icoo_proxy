package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ParseImport accepts several Grok CLI / community JSON shapes.
//
// Supported:
//  1. Single object with access_token / refresh_token
//  2. { "credentials": [ ... ] }
//  3. Grok auth.json style nested under "accounts" or "tokens"
//  4. Array of credential objects
func ParseImport(raw []byte, defaultLabel string) ([]Credential, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty import payload")
	}
	defaultLabel = strings.TrimSpace(defaultLabel)
	if defaultLabel == "" {
		defaultLabel = "imported"
	}

	// Array first.
	if raw[0] == '[' {
		var arr []map[string]any
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("parse array: %w", err)
		}
		out := make([]Credential, 0, len(arr))
		for i, m := range arr {
			c, ok := mapToCredential(m, fmt.Sprintf("%s-%d", defaultLabel, i+1))
			if ok {
				out = append(out, c)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no credentials found in array")
		}
		return out, nil
	}

	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse object: %w", err)
	}

	// Direct credential object.
	if c, ok := mapToCredential(root, defaultLabel); ok {
		return []Credential{c}, nil
	}

	// credentials: []
	if v, ok := root["credentials"]; ok {
		b, _ := json.Marshal(v)
		return ParseImport(b, defaultLabel)
	}

	// accounts: { key: { accessToken, ... } } or accounts: []
	if v, ok := root["accounts"]; ok {
		return parseAccounts(v, defaultLabel)
	}

	// Common CLI auth.json: { "accessToken": "...", "refreshToken": "..." }
	// already handled by mapToCredential; try nested "auth" / "token".
	for _, key := range []string{"auth", "token", "session", "current"} {
		if v, ok := root[key]; ok {
			if m, ok := v.(map[string]any); ok {
				if c, ok := mapToCredential(m, defaultLabel); ok {
					return []Credential{c}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unrecognized credential JSON shape")
}

func parseAccounts(v any, defaultLabel string) ([]Credential, error) {
	switch t := v.(type) {
	case []any:
		out := make([]Credential, 0, len(t))
		for i, item := range t {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if c, ok := mapToCredential(m, fmt.Sprintf("%s-%d", defaultLabel, i+1)); ok {
				out = append(out, c)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no credentials in accounts array")
		}
		return out, nil
	case map[string]any:
		out := make([]Credential, 0, len(t))
		for key, item := range t {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			label := key
			if c, ok := mapToCredential(m, label); ok {
				out = append(out, c)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no credentials in accounts map")
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported accounts type")
	}
}

func mapToCredential(m map[string]any, defaultLabel string) (Credential, bool) {
	access := firstString(m, "access_token", "accessToken", "token", "api_key", "apiKey")
	if access == "" {
		return Credential{}, false
	}
	refresh := firstString(m, "refresh_token", "refreshToken")
	label := firstString(m, "label", "name", "email", "account")
	if label == "" {
		label = defaultLabel
	}
	email := firstString(m, "email")
	id := firstString(m, "id")
	priority := firstInt(m, "priority")
	enabled := true
	if v, ok := m["enabled"]; ok {
		if b, ok := v.(bool); ok {
			enabled = b
		}
	}
	var exp time.Time
	if s := firstString(m, "expires_at", "expiresAt", "expiry"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			exp = t
		}
	}
	return Credential{
		ID:           id,
		Label:        label,
		Email:        email,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp,
		Enabled:      enabled,
		Priority:     priority,
	}, true
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			}
		}
	}
	return 0
}
