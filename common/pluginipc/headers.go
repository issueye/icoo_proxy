package pluginipc

import "strings"

// hop-by-hop and sensitive headers must never be forwarded to plugins.
var deniedHeaders = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"connection":          {},
	"keep-alive":          {},
	"transfer-encoding":   {},
	"te":                  {},
	"trailer":             {},
	"upgrade":             {},
	"host":                {},
	"content-length":      {},
}

var allowedHeaders = map[string]struct{}{
	"content-type":               {},
	"accept":                     {},
	"user-agent":                 {},
	"anthropic-version":          {},
	"anthropic-beta":             {},
	"x-claude-code-session-id":   {},
	"x-session-id":               {},
	"x-grok-conv-id":             {},
	"openai-organization":        {},
	"openai-project":             {},
}

// FilterHeaders keeps only allowlisted headers and drops denylisted ones.
// Keys are normalized to canonical lower-case form used by plugins.
func FilterHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		lk := strings.ToLower(strings.TrimSpace(k))
		if _, deny := deniedHeaders[lk]; deny {
			continue
		}
		if _, allow := allowedHeaders[lk]; !allow {
			continue
		}
		out[lk] = v
	}
	return out
}

// EnsureAnthropicVersion injects default anthropic-version when missing for anthropic ingress.
func EnsureAnthropicVersion(ingress string, headers map[string]string) map[string]string {
	if !strings.EqualFold(ingress, "anthropic") {
		return headers
	}
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["anthropic-version"]; !ok {
		headers["anthropic-version"] = "2023-06-01"
	}
	return headers
}
