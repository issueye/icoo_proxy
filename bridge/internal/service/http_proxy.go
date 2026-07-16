package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func parseProviderProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("proxy_url is invalid: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("proxy_url must include scheme and host")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5":
		return parsed, nil
	default:
		return nil, fmt.Errorf("proxy_url scheme must be http, https, or socks5")
	}
}

func newProxiedHTTPClient(timeout time.Duration, rawProxyURL string) (*http.Client, error) {
	return newProxiedHTTPClientWithResponseHeaderTimeout(timeout, 0, rawProxyURL)
}

func newProxiedHTTPClientWithResponseHeaderTimeout(timeout time.Duration, responseHeaderTimeout time.Duration, rawProxyURL string) (*http.Client, error) {
	proxyURL, err := parseProviderProxyURL(rawProxyURL)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: timeout}
	if proxyURL == nil && responseHeaderTimeout <= 0 {
		return client, nil
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	if responseHeaderTimeout > 0 {
		transport.ResponseHeaderTimeout = responseHeaderTimeout
	}
	client.Transport = transport
	return client, nil
}
