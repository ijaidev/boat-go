package transport_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/transport"
)

func newClient(t *testing.T, handler http.Handler, edit func(*core.Config)) (*transport.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := core.Config{
		BaseURL:    srv.URL,
		APIKey:     "boat_test_key",
		RetryMin:   time.Millisecond,
		RetryMax:   2 * time.Millisecond,
		MaxRetries: 2,
	}
	if edit != nil {
		edit(&cfg)
	}
	client, err := transport.New(cfg)
	if err != nil {
		t.Fatalf("transport.New: %v", err)
	}
	return client, srv
}

func TestClient_Do_SendsAuthOrgAndQuery(t *testing.T) {
	t.Parallel()

	var gotAuth, gotOrg, gotUA, gotPath, gotQuery string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotOrg = r.Header.Get("X-Boat-Org")
		gotUA = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `{"ok":true,"type":"test","value":42}`)
	}), func(cfg *core.Config) {
		cfg.Org = "org_a"
	})

	var out struct {
		Value int `json:"value"`
	}
	err := client.Do(context.Background(), &core.Request{
		Method: http.MethodGet,
		Path:   "/sandboxes",
		Query:  url.Values{"limit": {"10"}},
	}, &out)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Value != 42 {
		t.Fatalf("value = %d, want 42", out.Value)
	}
	if gotAuth != "Bearer boat_test_key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotOrg != "org_a" {
		t.Fatalf("X-Boat-Org = %q, want org_a", gotOrg)
	}
	if gotUA != "boat-go/dev" {
		t.Fatalf("User-Agent = %q, want boat-go/dev", gotUA)
	}
	if gotPath != "/sandboxes" {
		t.Fatalf("path = %q, want /sandboxes", gotPath)
	}
	if gotQuery != "limit=10" {
		t.Fatalf("query = %q, want limit=10", gotQuery)
	}
}

func TestClient_Do_JoinsBasePath(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)

	client, err := transport.New(core.Config{BaseURL: srv.URL + "/api/v1/", APIKey: "boat_test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/sandboxes"}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotPath != "/api/v1/sandboxes" {
		t.Fatalf("path = %q, want /api/v1/sandboxes", gotPath)
	}
}

func TestClient_Do_JSONBody(t *testing.T) {
	t.Parallel()

	var gotBody, gotContentType string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		gotContentType = r.Header.Get("Content-Type")
		_, _ = io.WriteString(w, `{"ok":true,"sandbox":{"id":"bx_example","name":"n","state":"ready","desktopAvailable":false,"snapshotAvailable":false}}`)
	}), nil)

	var out struct {
		Sandbox struct {
			ID string `json:"id"`
		} `json:"sandbox"`
	}
	err := client.Do(context.Background(), &core.Request{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		JSON:   map[string]any{"ttlSeconds": 60},
	}, &out)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody != `{"ttlSeconds":60}` {
		t.Fatalf("body = %s", gotBody)
	}
	if out.Sandbox.ID != "bx_example" {
		t.Fatalf("sandbox id = %q", out.Sandbox.ID)
	}
}

func TestClient_Do_MapsAPIError(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{
			"ok": false, "type": "sandbox.error", "status": 409,
			"code": "sandbox_not_ready", "message": "sandbox is not ready",
			"requestId": "req_123",
			"error": {"code": "sandbox_not_ready", "message": "sandbox is not ready", "status": 409,
				"details": {"retryable": true, "reason": "restoring"}},
			"retryAfterSeconds": 2
		}`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{Method: http.MethodPost, Path: "/sandboxes/bx_1/commands"}, nil)
	if err == nil {
		t.Fatal("Do = nil, want error")
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *core.APIError", err)
	}
	if apiErr.Status != 409 || apiErr.Code != "sandbox_not_ready" || apiErr.RequestID != "req_123" {
		t.Fatalf("apiErr = %#v", apiErr)
	}
	if apiErr.RetryAfter != 2*time.Second {
		t.Fatalf("RetryAfter = %v, want 2s", apiErr.RetryAfter)
	}
	if !core.Retryable(err) {
		t.Fatal("Retryable = false, want true")
	}
	if !core.IsCode(err, "sandbox_not_ready") {
		t.Fatal("IsCode = false, want true")
	}
	if strings.Contains(err.Error(), "restoring") {
		t.Fatalf("error leaks details: %v", err)
	}
}

func TestClient_Do_RetriesGetOn503(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"ok":false,"type":"sandbox.error","status":503,"code":"no_ready_machine","message":"busy"}`)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true,"value":"ok"}`)
	}), nil)

	var out struct {
		Value string `json:"value"`
	}
	if err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/limits"}, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
	if out.Value != "ok" {
		t.Fatalf("value = %q", out.Value)
	}
}

func TestClient_Do_DoesNotRetryPostWithoutIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"ok":false,"type":"sandbox.error","status":503,"code":"no_ready_machine","message":"busy"}`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{Method: http.MethodPost, Path: "/sandboxes/bx_1/stop"}, nil)
	if err == nil {
		t.Fatal("Do = nil, want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestClient_Do_RetriesPostWithIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"ok":false,"type":"sandbox.error","status":503,"code":"no_ready_machine","message":"busy"}`)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{Method: http.MethodPost, Path: "/sandboxes"}, nil,
		core.WithIdempotencyKey("key-1"))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestClient_Do_DoesNotRetryOneShotBody(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"ok":false,"type":"sandbox.error","status":503,"code":"no_ready_machine","message":"busy"}`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{
		Method: http.MethodPost,
		Path:   "/sandboxes",
		Body:   strings.NewReader("hello"),
	}, nil, core.WithIdempotencyKey("key-1"))
	if err == nil {
		t.Fatal("Do = nil, want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestClient_DoRaw_ReturnsBodyAndStatus(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "raw bytes")
	}), nil)

	resp, err := client.DoRaw(context.Background(), &core.Request{Method: http.MethodGet, Path: "/artifacts"})
	if err != nil {
		t.Fatalf("DoRaw: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated || string(body) != "raw bytes" {
		t.Fatalf("resp = %d %q", resp.StatusCode, body)
	}
}

func TestClient_DoRaw_RetriesGetOn503(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}), nil)

	resp, err := client.DoRaw(context.Background(), &core.Request{Method: http.MethodGet, Path: "/artifacts"})
	if err != nil {
		t.Fatalf("DoRaw: %v", err)
	}
	defer resp.Body.Close()
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestClient_Do_ContextCanceled(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}), nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := client.Do(ctx, &core.Request{Method: http.MethodGet, Path: "/sandboxes"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("calls = %d, want 0", calls.Load())
	}
}

func TestClient_Do_OrgOptionOverridesAndClears(t *testing.T) {
	t.Parallel()

	var got []string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("X-Boat-Org"))
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), func(cfg *core.Config) {
		cfg.Org = "org_a"
	})

	_ = client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/me"}, nil, core.WithOrg("org_b"))
	_ = client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/me"}, nil, core.WithOrg(""))

	if len(got) != 2 || got[0] != "org_b" || got[1] != "" {
		t.Fatalf("org headers = %q, want [org_b ]", got)
	}
}

func TestClient_Do_AuthorizationOverride(t *testing.T) {
	t.Parallel()

	var got string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{
		Method: http.MethodGet,
		Path:   "/me",
		Header: http.Header{"Authorization": {"Bearer custom"}},
	}, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got != "Bearer custom" {
		t.Fatalf("Authorization = %q", got)
	}
}

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

func TestClient_Do_UsesTokenSource(t *testing.T) {
	t.Parallel()

	var got string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), func(cfg *core.Config) {
		cfg.APIKey = ""
		cfg.TokenSource = staticToken("boat_from_source")
	})

	if err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/me"}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got != "Bearer boat_from_source" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestNew_ValidatesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  core.Config
	}{
		{name: "no auth", cfg: core.Config{BaseURL: "https://example.com"}},
		{name: "bad scheme", cfg: core.Config{BaseURL: "ftp://example.com", APIKey: "k"}},
		{name: "no host", cfg: core.Config{BaseURL: "https:///api", APIKey: "k"}},
		{name: "bad url", cfg: core.Config{BaseURL: "://", APIKey: "k"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := transport.New(tt.cfg); err == nil {
				t.Fatal("New = nil error, want error")
			}
		})
	}
}

func TestClient_Do_NonJSONErrorDoesNotLeakBody(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "sensitive raw body")
	}), nil)

	err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/sandboxes"}, nil)
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want *core.APIError", err)
	}
	if apiErr.Status != 502 || apiErr.Code != "http_502" {
		t.Fatalf("apiErr = %#v", apiErr)
	}
	if strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("error leaks body: %v", err)
	}
}

func TestClient_Do_DoesNotSleepOnLongRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"ok":false,"type":"sandbox.error","status":429,"code":"rate_limited","message":"slow down"}`)
	}), nil)

	start := time.Now()
	err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/sandboxes"}, nil)
	if err == nil {
		t.Fatal("Do = nil, want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Do waited %v, want no long sleep", elapsed)
	}
	if got := core.RetryAfter(err); got != time.Minute {
		t.Fatalf("RetryAfter = %v, want 1m", got)
	}
}

func TestClient_Do_InvalidSuccessJSON(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `not json`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/me"}, nil)
	if err == nil {
		t.Fatal("Do = nil, want error")
	}
}

func TestClient_Do_EnvelopeOKFalse(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ok":false,"type":"sandbox.error","status":400,"code":"invalid_json","message":"bad"}`)
	}), nil)

	err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/me"}, nil)
	if !core.IsCode(err, "invalid_json") {
		t.Fatalf("error = %v, want invalid_json", err)
	}
}

func TestClient_Do_QueryEncoding(t *testing.T) {
	t.Parallel()

	var got string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("label")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), nil)

	if err := client.Do(context.Background(), &core.Request{
		Method: http.MethodGet,
		Path:   "/sandboxes",
		Query:  url.Values{"label": {"env=prod"}},
	}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got != "env=prod" {
		t.Fatalf("label = %q", got)
	}
}

func TestClient_Do_RequestOptionHeader(t *testing.T) {
	t.Parallel()

	var got string
	client, _ := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Custom")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}), nil)

	if err := client.Do(context.Background(), &core.Request{Method: http.MethodGet, Path: "/me"}, nil,
		core.WithHeader("X-Custom", "v")); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got != "v" {
		t.Fatalf("X-Custom = %q", got)
	}
}
