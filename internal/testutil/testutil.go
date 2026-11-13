// Package testutil provides small deterministic helpers for SDK tests.
package testutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/transport"
)

// Route describes one expected request and its response.
type Route struct {
	Method string
	Path   string
	Status int
	Header http.Header
	Body   string
	Assert func(*testing.T, *http.Request)
}

// NewServer starts an httptest server that serves routes and fails the test on an unexpected
// request. It closes the server with t.Cleanup.
func NewServer(t *testing.T, routes ...Route) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, route := range routes {
			if route.Method != r.Method || route.Path != r.URL.Path {
				continue
			}
			if route.Assert != nil {
				route.Assert(t, r)
			}
			for key, values := range route.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			if w.Header().Get("Content-Type") == "" && route.Body != "" {
				w.Header().Set("Content-Type", "application/json")
			}
			status := route.Status
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(route.Body))
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Transport returns a transport pointed at a test server with short retry delays.
func Transport(t *testing.T, baseURL, apiKey string) *transport.Client {
	t.Helper()
	client, err := transport.New(core.Config{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		RetryMin:   time.Millisecond,
		RetryMax:   2 * time.Millisecond,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("transport.New: %v", err)
	}
	return client
}

// Doer is a fake core.Doer for tests that focus on service logic rather than the wire format.
type Doer struct {
	DoFunc  func(context.Context, *core.Request, any, ...core.RequestOption) error
	RawFunc func(context.Context, *core.Request, ...core.RequestOption) (*http.Response, error)

	Calls []*core.Request
}

// Do implements core.Doer.
func (d *Doer) Do(ctx context.Context, req *core.Request, out any, opts ...core.RequestOption) error {
	d.Calls = append(d.Calls, req)
	if d.DoFunc == nil {
		return errors.New("testutil: unexpected Do")
	}
	return d.DoFunc(ctx, req, out, opts...)
}

// DoRaw implements core.Doer.
func (d *Doer) DoRaw(ctx context.Context, req *core.Request, opts ...core.RequestOption) (*http.Response, error) {
	d.Calls = append(d.Calls, req)
	if d.RawFunc == nil {
		return nil, errors.New("testutil: unexpected DoRaw")
	}
	return d.RawFunc(ctx, req, opts...)
}
