// Package core defines the small kernel shared by every Boat SDK package.
//
// It contains request and response primitives, the transport seam, pagination and streaming
// interfaces, and the public API error type. Domain behavior belongs in the service packages.
package core

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Doer executes Boat API requests. The SDK transport implements it; tests may fake it.
type Doer interface {
	// Do executes req and decodes the JSON envelope payload into out.
	Do(ctx context.Context, req *Request, out any, opts ...RequestOption) error
	// DoRaw executes req and returns the response. The caller owns and must close the body.
	DoRaw(ctx context.Context, req *Request, opts ...RequestOption) (*http.Response, error)
}

// Request is a transport-neutral Boat API request.
//
// JSON is marshalled once per attempt and is replayable, so it may be retried. Body is a one-shot
// reader: a request with Body is never retried.
type Request struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	JSON   any
	Body   io.Reader

	// NoRetry prevents the transport from retrying this request, even when it carries an
	// Idempotency-Key. Use it for non-idempotent operations such as command execution.
	NoRetry bool
}

// SetHeader sets a request header, creating the header map when needed.
func (r *Request) SetHeader(key, value string) {
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(key, value)
}

// AddQuery adds a query value, creating the query map when needed.
func (r *Request) AddQuery(key, value string) {
	if r.Query == nil {
		r.Query = make(url.Values)
	}
	r.Query.Add(key, value)
}

// Config controls the shared transport.
//
// MaxRetries 0 means the default (3 attempts). A negative value disables retries. RetryMin and
// RetryMax default to 200ms and 5s. BaseURL defaults to https://boat.dev/api/v1.
type Config struct {
	BaseURL     string
	APIKey      string
	Org         string
	HTTPClient  HTTPClient
	TokenSource TokenSource
	UserAgent   string
	MaxRetries  int
	RetryMin    time.Duration
	RetryMax    time.Duration
}

// HTTPClient is the subset of *http.Client used by the transport.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// TokenSource supplies a bearer token for a request. It is called for every attempt, so it may
// refresh credentials without rebuilding the client.
type TokenSource interface {
	Token(context.Context) (string, error)
}
