package proxyhandler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/issueye/icoo_proxy/common/ai_llm_proxy"
	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/common/pluginipc"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/oauth"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/store"
	"github.com/issueye/icoo_proxy/plugins/grokbuild/internal/upstream"
)

const maxFailoverAttempts = 3

// Handler serves proxy.complete / proxy.stream for the grokbuild plugin.
type Handler struct {
	Store     *store.Store
	Upstream  *upstream.Client
	Converter ai_llm_proxy.Converter
	Refresh   *oauth.Refresher
}

func New(st *store.Store, up *upstream.Client, refresh *oauth.Refresher) *Handler {
	if refresh == nil {
		refresh = oauth.NewRefresher(nil)
	}
	return &Handler{
		Store:     st,
		Upstream:  up,
		Converter: ai_llm_proxy.NewProtocolConverter(),
		Refresh:   refresh,
	}
}

func (h *Handler) ensureToken(ctx context.Context, cred store.Credential) (string, error) {
	ts, err := h.Refresh.EnsureAccess(ctx, cred.ID, cred.AccessToken, cred.RefreshToken, cred.ExpiresAt)
	if err != nil {
		// Fall back to stored access token if refresh fails but token present.
		if strings.TrimSpace(cred.AccessToken) != "" {
			return cred.AccessToken, nil
		}
		return "", err
	}
	// Persist rotation when tokens changed.
	if ts.AccessToken != cred.AccessToken || ts.RefreshToken != cred.RefreshToken || !ts.ExpiresAt.Equal(cred.ExpiresAt) {
		_ = h.Store.ApplyTokens(cred.ID, ts.AccessToken, ts.RefreshToken, ts.ExpiresAt)
	}
	return ts.AccessToken, nil
}

func (h *Handler) Complete(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
	ingress := constants.Protocol(req.Ingress)
	upBody, err := h.Converter.ConvertRequest(ai_llm_proxy.RequestInput{
		Downstream: ingress,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      req.Model,
		Body:       req.Body,
	})
	if err != nil {
		return nil, pluginipc.RPCUnsupportedIngress(err)
	}

	usable, err := h.Store.ListUsable()
	if err != nil {
		return unauthorized(err.Error()), nil
	}
	if len(usable) == 0 {
		return unauthorized("no enabled grok credential; open the GrokBuild plugin page to add a token"), nil
	}

	// Prefer Pick order, then remaining for failover.
	first, err := h.Store.Pick()
	if err != nil {
		return unauthorized(err.Error()), nil
	}
	order := []store.Credential{first}
	for _, c := range usable {
		if c.ID != first.ID {
			order = append(order, c)
		}
	}
	if len(order) > maxFailoverAttempts {
		order = order[:maxFailoverAttempts]
	}

	var lastBody []byte
	var lastStatus int
	for _, cred := range order {
		token, tokErr := h.ensureToken(ctx, cred)
		if tokErr != nil {
			_ = h.Store.MarkFailure(cred.ID, 0, 0, tokErr.Error())
			lastStatus = http.StatusUnauthorized
			lastBody = []byte(`{"error":{"message":"` + jsonEscape(tokErr.Error()) + `"}}`)
			continue
		}
		resp, err := h.Upstream.PostResponses(ctx, token, req.Model, upBody, false)
		if err != nil {
			_ = h.Store.MarkFailure(cred.ID, 0, 0, err.Error())
			lastStatus = http.StatusBadGateway
			lastBody = []byte(`{"error":{"message":"` + jsonEscape(err.Error()) + `"}}`)
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			_ = h.Store.MarkFailure(cred.ID, resp.StatusCode, 0, readErr.Error())
			lastStatus = http.StatusBadGateway
			lastBody = []byte(`{"error":{"message":"read upstream body failed"}}`)
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = h.Store.MarkSuccess(cred.ID)
			down, convErr := h.Converter.ConvertResponse(ai_llm_proxy.ResponseInput{
				Downstream: ingress,
				Upstream:   constants.ProtocolOpenAIResponses,
				Model:      req.Model,
				Body:       raw,
			})
			if convErr != nil {
				return nil, convErr
			}
			usage := h.Converter.ExtractUsage(constants.ProtocolOpenAIResponses, raw)
			return pluginipc.OKJSON(down, &pluginipc.Usage{
				InputTokens:  int64(usage.InputTokens),
				OutputTokens: int64(usage.OutputTokens),
				TotalTokens:  int64(usage.TotalTokens),
			}), nil
		}

		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		_ = h.Store.MarkFailure(cred.ID, resp.StatusCode, retryAfter, truncate(string(raw), 200))
		lastStatus = resp.StatusCode
		lastBody = raw
		// Failover only for retriable statuses before any body is committed.
		if !canFailover(resp.StatusCode) {
			break
		}
	}

	if lastStatus == 0 {
		lastStatus = http.StatusBadGateway
	}
	if lastBody == nil {
		lastBody = []byte(`{"error":{"message":"upstream failed"}}`)
	}
	return pluginipc.JSONStatus(lastStatus, lastBody, nil), nil
}

// PrepareStream implements pluginipc.StreamHandler (prepare + run closure).
// Failover is only attempted before open succeeds; run uses the host-provided
// cancelable context so stream.cancel can abort upstream conversion.
func (h *Handler) PrepareStream(ctx context.Context, req pluginipc.ProxyRequest) (
	open *pluginipc.StreamOpenResult,
	errResp *pluginipc.ProxyResponse,
	run func(ctx context.Context, w *pluginipc.StreamWriter),
	err error,
) {
	ingress := constants.Protocol(req.Ingress)
	upBody, err := h.Converter.ConvertRequest(ai_llm_proxy.RequestInput{
		Downstream: ingress,
		Upstream:   constants.ProtocolOpenAIResponses,
		Model:      req.Model,
		Body:       forceStreamTrue(req.Body),
	})
	if err != nil {
		return nil, nil, nil, pluginipc.RPCUnsupportedIngress(err)
	}

	// For streams: pick credential at prepare time; failover only on open failure
	// before any chunk is written.
	usable, err := h.Store.ListUsable()
	if err != nil || len(usable) == 0 {
		msg := "no enabled grok credential"
		if err != nil {
			msg = err.Error()
		}
		return nil, unauthorized(msg), nil, nil
	}
	first, err := h.Store.Pick()
	if err != nil {
		return nil, unauthorized(err.Error()), nil, nil
	}
	order := []store.Credential{first}
	for _, c := range usable {
		if c.ID != first.ID {
			order = append(order, c)
		}
	}
	if len(order) > maxFailoverAttempts {
		order = order[:maxFailoverAttempts]
	}

	// Probe until we get a successful open (headers) or exhaust pool.
	var (
		chosen store.Credential
		resp   *http.Response
	)
	for _, cred := range order {
		token, tokErr := h.ensureToken(ctx, cred)
		if tokErr != nil {
			_ = h.Store.MarkFailure(cred.ID, 0, 0, tokErr.Error())
			continue
		}
		r, callErr := h.Upstream.PostResponses(ctx, token, req.Model, upBody, true)
		if callErr != nil {
			_ = h.Store.MarkFailure(cred.ID, 0, 0, callErr.Error())
			continue
		}
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			chosen = cred
			resp = r
			break
		}
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = r.Body.Close()
		retryAfter := parseRetryAfter(r.Header.Get("Retry-After"))
		_ = h.Store.MarkFailure(cred.ID, r.StatusCode, retryAfter, truncate(string(raw), 200))
		if !canFailover(r.StatusCode) {
			return nil, pluginipc.JSONStatus(r.StatusCode, raw, nil), nil, nil
		}
	}
	if resp == nil {
		return nil, pluginipc.JSONStatus(http.StatusBadGateway, []byte(`{"error":{"message":"all credentials failed for stream open"}}`), nil), nil, nil
	}

	streamID := pluginipc.NewStreamID("gb")
	open = pluginipc.SSEOpen(streamID)

	run = func(runCtx context.Context, w *pluginipc.StreamWriter) {
		defer resp.Body.Close()
		if runCtx == nil {
			runCtx = context.Background()
		}
		result, convErr := h.Converter.ConvertStream(ai_llm_proxy.StreamInput{
			Context:    runCtx,
			Downstream: ingress,
			Upstream:   constants.ProtocolOpenAIResponses,
			Model:      req.Model,
			Reader:     resp.Body,
			Writer:     w.AsWriter(),
		})
		if convErr != nil {
			_ = h.Store.MarkFailure(chosen.ID, 0, 0, convErr.Error())
			_ = w.Error(pluginipc.CodeInternalError, convErr.Error())
			return
		}
		_ = h.Store.MarkSuccess(chosen.ID)
		_ = w.End(&pluginipc.Usage{
			InputTokens:  int64(result.Usage.InputTokens),
			OutputTokens: int64(result.Usage.OutputTokens),
			TotalTokens:  int64(result.Usage.TotalTokens),
		})
	}
	return open, nil, run, nil
}

func canFailover(status int) bool {
	switch status {
	case 401, 402, 403, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func parseRetryAfter(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// unauthorized keeps the historical {"error":{"message":"..."}} body shape used by
// grokbuild clients (SDK Unauthorized adds type/code fields).
func unauthorized(msg string) *pluginipc.ProxyResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]string{"message": msg},
	})
	return pluginipc.JSONStatus(http.StatusUnauthorized, body, nil)
}

func forceStreamTrue(body []byte) []byte {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return body
	}
	m["stream"] = true
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}
