// Package netx builds HTTP clients that honor plugin proxy settings.
package netx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// NewHTTPClient returns an *http.Client with optional proxyURL.
// Empty proxyURL falls back to environment proxies (HTTP_PROXY/HTTPS_PROXY/ALL_PROXY).
// Supports http://, https://, socks5://, socks5h://.
func NewHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	tr, err := NewTransport(proxyURL)
	if err != nil {
		return nil, err
	}
	return &http.Client{Timeout: timeout, Transport: tr}, nil
}

// NewTransport builds a shared transport for OAuth / upstream clients.
func NewTransport(proxyURL string) (*http.Transport, error) {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return tr, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid proxy url: missing scheme or host")
	}

	switch strings.ToLower(u.Scheme) {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if u.User != nil {
			pass, _ := u.User.Password()
			auth = &proxy.Auth{User: u.User.Username(), Password: pass}
		}
		// socks5h: resolve DNS via proxy (default for x/net SOCKS5).
		dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("socks5 dialer: %w", err)
		}
		tr.Proxy = nil // dial through SOCKS instead
		if cd, ok := dialer.(proxy.ContextDialer); ok {
			tr.DialContext = cd.DialContext
		} else {
			tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
		// HTTP/2 over SOCKS can be flaky; keep HTTP/1.1 preferred.
		tr.ForceAttemptHTTP2 = false
	case "http", "https":
		tr.Proxy = http.ProxyURL(u)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q (use http, https, socks5)", u.Scheme)
	}
	return tr, nil
}

// EffectiveProxyDescription returns a human-readable active proxy source.
func EffectiveProxyDescription(explicit string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v + " (from env " + k + ")"
		}
	}
	return "(none — direct)"
}
