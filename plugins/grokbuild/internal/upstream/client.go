package upstream

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL          = "https://cli-chat-proxy.grok.com/v1"
	DefaultClientVersion    = "0.2.93"
	DefaultClientIdentifier = "icoo-grokbuild-plugin"
	DefaultTokenAuth        = "xai-grok-cli"
	DefaultUserAgent        = "icoo-grokbuild-plugin/0.1.0"
)

// Client talks to the Grok Build Responses backend.
type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          16,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 60 * time.Second,
			},
		},
	}
}

// SetHTTPClient replaces the underlying HTTP client (e.g. after proxy change).
func (c *Client) SetHTTPClient(cli *http.Client) {
	if c == nil || cli == nil {
		return
	}
	c.http = cli
}

// PostResponses posts a JSON body to /responses.
func (c *Client) PostResponses(ctx context.Context, accessToken, model string, body []byte, stream bool) (*http.Response, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("upstream: missing access token")
	}
	url := c.baseURL + "/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-XAI-Token-Auth", DefaultTokenAuth)
	req.Header.Set("x-grok-client-version", DefaultClientVersion)
	req.Header.Set("x-grok-client-identifier", DefaultClientIdentifier)
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	if model != "" {
		req.Header.Set("x-grok-model-override", model)
	}
	return c.http.Do(req)
}

// ListModels returns raw models payload when available.
func (c *Client) ListModels(ctx context.Context, accessToken string) (int, []byte, error) {
	return c.getJSON(ctx, "/models", accessToken)
}

// GetBilling returns monthly billing JSON when available.
func (c *Client) GetBilling(ctx context.Context, accessToken string) (int, []byte, error) {
	return c.getJSON(ctx, "/billing", accessToken)
}

// GetBillingCredits returns weekly credits JSON (?format=credits).
func (c *Client) GetBillingCredits(ctx context.Context, accessToken string) (int, []byte, error) {
	return c.getJSON(ctx, "/billing?format=credits", accessToken)
}

func (c *Client) getJSON(ctx context.Context, path, accessToken string) (int, []byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-XAI-Token-Auth", DefaultTokenAuth)
	req.Header.Set("x-grok-client-version", DefaultClientVersion)
	req.Header.Set("x-grok-client-identifier", DefaultClientIdentifier)
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return resp.StatusCode, raw, err
}
