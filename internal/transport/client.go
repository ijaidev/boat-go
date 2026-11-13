// Package transport implements core.Doer for the Boat public API.
package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ijaidev/boat-go/core"
)

const (
	defaultBaseURL   = "https://boat.dev/api/v1"
	defaultUserAgent = "boat-go/dev"
	maxJSONBody      = 32 << 20 // 32 MiB
	defaultRetries   = 3
	defaultRetryMin  = 200 * time.Millisecond
	defaultRetryMax  = 5 * time.Second
)

// Client is the Boat HTTP transport. It is safe for concurrent use.
type Client struct {
	baseURL     *url.URL
	apiKey      string
	org         string
	httpClient  core.HTTPClient
	tokenSource core.TokenSource
	userAgent   string
	maxRetries  int
	retryMin    time.Duration
	retryMax    time.Duration
}

// New builds a transport from cfg.
func New(cfg core.Config) (*Client, error) {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("boat: parse base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("boat: base URL must use http or https")
	}
	if u.Host == "" {
		return nil, errors.New("boat: base URL has no host")
	}
	if cfg.APIKey == "" && cfg.TokenSource == nil {
		return nil, errors.New("boat: APIKey or TokenSource is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	retries := cfg.MaxRetries
	switch {
	case retries == 0:
		retries = defaultRetries
	case retries < 0:
		retries = 0
	}

	retryMin := cfg.RetryMin
	if retryMin <= 0 {
		retryMin = defaultRetryMin
	}
	retryMax := cfg.RetryMax
	if retryMax <= 0 {
		retryMax = defaultRetryMax
	}
	if retryMax < retryMin {
		retryMax = retryMin
	}

	ua := cfg.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	return &Client{
		baseURL:     u,
		apiKey:      cfg.APIKey,
		org:         cfg.Org,
		httpClient:  httpClient,
		tokenSource: cfg.TokenSource,
		userAgent:   ua,
		maxRetries:  retries,
		retryMin:    retryMin,
		retryMax:    retryMax,
	}, nil
}

func (c *Client) token(ctx context.Context) (string, error) {
	if c.tokenSource != nil {
		token, err := c.tokenSource.Token(ctx)
		if err != nil {
			return "", fmt.Errorf("boat: token source: %w", err)
		}
		return token, nil
	}
	return c.apiKey, nil
}

func (c *Client) requestURL(req *core.Request) string {
	u := *c.baseURL
	u.Path = strings.TrimSuffix(c.baseURL.Path, "/") + "/" + strings.TrimPrefix(req.Path, "/")
	if req.Query != nil {
		u.RawQuery = req.Query.Encode()
	} else {
		u.RawQuery = ""
	}
	return u.String()
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
