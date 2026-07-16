// Package oauth implements a minimal xAI auth.x.ai OAuth client for SuperGrok /
// Grok Build (device-code login + refresh_token grant).
//
// Adapted from community grokbuild-proxy patterns; not affiliated with xAI.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	Issuer                  = "https://auth.x.ai"
	DefaultClientID         = "b1a00492-073a-47ea-816f-4c329264a828"
	DefaultScope            = "openid profile email offline_access grok-cli:access api:access"
	DefaultTokenEndpoint    = Issuer + "/oauth2/token"
	DefaultDeviceAuthURL    = Issuer + "/oauth2/device/code"
	DefaultDiscoveryURL     = Issuer + "/.well-known/openid-configuration"
	DefaultRefreshSkew      = 180 * time.Second
	DefaultHTTPTimeout      = 30 * time.Second
)

// TokenSet is the OAuth token bundle.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scope        string
}

// Expired reports whether the access token should be refreshed.
func (t TokenSet) Expired(now time.Time, skew time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	if skew < 0 {
		skew = 0
	}
	return !now.Before(t.ExpiresAt.Add(-skew))
}

// DeviceCode is an RFC 8628 device authorization response.
type DeviceCode struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int
	Interval                int
}

// Client talks to auth.x.ai.
type Client struct {
	HTTP       *http.Client
	ClientID   string
	Scope      string
	TokenURL   string
	DeviceURL  string
}

func NewClient() *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: DefaultHTTPTimeout},
		ClientID: DefaultClientID,
		Scope:    DefaultScope,
		TokenURL: DefaultTokenEndpoint,
		DeviceURL: DefaultDeviceAuthURL,
	}
}

// Discover fills TokenURL / DeviceURL from OIDC discovery when possible.
func (c *Client) Discover(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultDiscoveryURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oauth discovery: status %d", resp.StatusCode)
	}
	var d struct {
		TokenEndpoint               string `json:"token_endpoint"`
		DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	}
	if err := json.Unmarshal(body, &d); err != nil {
		return err
	}
	if ep := strings.TrimSpace(d.TokenEndpoint); ep != "" && isTrustedAuthURL(ep) {
		c.TokenURL = ep
	}
	if ep := strings.TrimSpace(d.DeviceAuthorizationEndpoint); ep != "" && isTrustedAuthURL(ep) {
		c.DeviceURL = ep
	}
	return nil
}

// RequestDeviceCode starts device authorization.
func (c *Client) RequestDeviceCode(ctx context.Context) (*DeviceCode, error) {
	form := url.Values{
		"client_id": {c.clientID()},
		"scope":     {c.scope()},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.deviceURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.doNoRedirect(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code: status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var raw struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.DeviceCode == "" || raw.UserCode == "" {
		return nil, fmt.Errorf("device code: missing fields")
	}
	if raw.Interval <= 0 {
		raw.Interval = 5
	}
	return &DeviceCode{
		DeviceCode:              raw.DeviceCode,
		UserCode:                raw.UserCode,
		VerificationURI:         raw.VerificationURI,
		VerificationURIComplete: raw.VerificationURIComplete,
		ExpiresIn:               raw.ExpiresIn,
		Interval:                raw.Interval,
	}, nil
}

// ExchangeDeviceCode polls/completes the device grant. Callers should retry on
// authorization_pending / slow_down.
func (c *Client) ExchangeDeviceCode(ctx context.Context, deviceCode string) (*TokenSet, error) {
	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":   {c.clientID()},
		"device_code": {strings.TrimSpace(deviceCode)},
	}
	return c.postToken(ctx, form, "")
}

// Refresh exchanges a refresh_token.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*TokenSet, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh_token required")
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientID()},
		"refresh_token": {refreshToken},
	}
	return c.postToken(ctx, form, refreshToken)
}

// PollDevice blocks until the user completes device login or ctx/expiry ends.
func (c *Client) PollDevice(ctx context.Context, dc *DeviceCode) (*TokenSet, error) {
	if dc == nil {
		return nil, fmt.Errorf("nil device code")
	}
	interval := time.Duration(dc.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	if dc.ExpiresIn <= 0 {
		deadline = time.Now().Add(15 * time.Minute)
	}
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("device login expired")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		ts, err := c.ExchangeDeviceCode(ctx, dc.DeviceCode)
		if err == nil {
			return ts, nil
		}
		msg := err.Error()
		// authorization_pending / slow_down are expected while waiting.
		if strings.Contains(msg, "authorization_pending") || strings.Contains(msg, "slow_down") {
			if strings.Contains(msg, "slow_down") {
				interval += 2 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(interval):
			}
			continue
		}
		// access_denied / expired_token fail hard
		return nil, err
	}
}

func (c *Client) postToken(ctx context.Context, form url.Values, fallbackRefresh string) (*TokenSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.doNoRedirect(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		// include oauth error body for poll classification
		return nil, fmt.Errorf("token exchange status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Error != "" {
		return nil, fmt.Errorf("%s: %s", payload.Error, payload.ErrorDesc)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return nil, fmt.Errorf("missing access_token")
	}
	refresh := strings.TrimSpace(payload.RefreshToken)
	if refresh == "" {
		refresh = strings.TrimSpace(fallbackRefresh)
	}
	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return &TokenSet{
		AccessToken:  strings.TrimSpace(payload.AccessToken),
		RefreshToken: refresh,
		ExpiresAt:    time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
		Scope:        strings.TrimSpace(payload.Scope),
	}, nil
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: DefaultHTTPTimeout}
}

func (c *Client) doNoRedirect(req *http.Request) (*http.Response, error) {
	base := c.http()
	client := *base
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client.Do(req)
}

func (c *Client) clientID() string {
	if c != nil && strings.TrimSpace(c.ClientID) != "" {
		return strings.TrimSpace(c.ClientID)
	}
	return DefaultClientID
}

func (c *Client) scope() string {
	if c != nil && strings.TrimSpace(c.Scope) != "" {
		return strings.TrimSpace(c.Scope)
	}
	return DefaultScope
}

func (c *Client) tokenURL() string {
	if c != nil && strings.TrimSpace(c.TokenURL) != "" {
		return strings.TrimSpace(c.TokenURL)
	}
	return DefaultTokenEndpoint
}

func (c *Client) deviceURL() string {
	if c != nil && strings.TrimSpace(c.DeviceURL) != "" {
		return strings.TrimSpace(c.DeviceURL)
	}
	return DefaultDeviceAuthURL
}

func isTrustedAuthURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "auth.x.ai" || host == "accounts.x.ai"
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
