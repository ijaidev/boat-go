package telemetry_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/telemetry"
)

const (
	testEndpoint    = "https://otlp.example.com"
	testHeaderName  = "dd-api-key"
	testHeaderValue = "test_header_secret"
)

const telemetryConfigJSON = `{
	"endpoint": "https://otlp.example.com",
	"headers": ["dd-api-key"],
	"lastExportAt": "2026-10-09T12:00:00Z",
	"lastError": {"message": "collector rejected", "status": 502, "at": "2026-10-09T12:01:00Z"},
	"createdAt": "2026-10-09T11:00:00Z"
}`

func headers(values map[string]string) *map[string]string {
	return &values
}

func TestService_Get_ReturnsConfig(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/telemetry",
		Body:   `{"ok":true,"type":"telemetry.info","telemetry":` + telemetryConfigJSON + `}`,
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg == nil {
		t.Fatal("Get = nil config, want config")
	}
	if cfg.Endpoint != testEndpoint {
		t.Fatalf("endpoint = %q, want %q", cfg.Endpoint, testEndpoint)
	}
	if len(cfg.Headers) != 1 || cfg.Headers[0] != testHeaderName {
		t.Fatalf("headers = %#v, want names only", cfg.Headers)
	}
	if cfg.LastExportAt == nil {
		t.Fatal("lastExportAt = nil, want a time")
	}
	if cfg.LastError == nil || cfg.LastError.Status == nil || *cfg.LastError.Status != http.StatusBadGateway {
		t.Fatalf("lastError = %#v", cfg.LastError)
	}
}

func TestService_Get_ReturnsNilWhenUnset(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/telemetry",
		Body:   `{"ok":true,"type":"telemetry.info","telemetry":null}`,
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg != nil {
		t.Fatalf("Get = %#v, want nil config", cfg)
	}
}

func TestService_Get_MapsErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		code   string
	}{
		{status: http.StatusUnauthorized, code: "unauthorized"},
		{status: http.StatusForbidden, code: "forbidden"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodGet,
				Path:   "/telemetry",
				Status: tt.status,
				Body:   fmt.Sprintf(`{"ok":false,"type":"error","status":%d,"code":%q,"message":"failed"}`, tt.status, tt.code),
			})
			svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			_, err := svc.Get(context.Background())
			if !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestService_Set_SendsBodyAndPreservesTestResult(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var idempotencyKey string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/telemetry",
		Body: `{"ok":true,"type":"telemetry.updated","telemetry":` + telemetryConfigJSON + `,
			"test":{"ok":true,"status":202,"message":null}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			idempotencyKey = r.Header.Get("Idempotency-Key")
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Set(context.Background(), model.TelemetrySetRequest{
		Endpoint: testEndpoint,
		Headers:  headers(map[string]string{testHeaderName: testHeaderValue}),
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if idempotencyKey != "" {
		t.Fatalf("Idempotency-Key = %q, want none by default", idempotencyKey)
	}
	if body["endpoint"] != testEndpoint {
		t.Fatalf("body endpoint = %v", body["endpoint"])
	}
	sent, ok := body["headers"].(map[string]any)
	if !ok || sent[testHeaderName] != testHeaderValue {
		t.Fatalf("body headers = %#v", body["headers"])
	}
	if got.Telemetry == nil || got.Telemetry.Endpoint != testEndpoint {
		t.Fatalf("telemetry = %#v", got.Telemetry)
	}
	if len(got.Telemetry.Headers) != 1 || got.Telemetry.Headers[0] != testHeaderName {
		t.Fatalf("response headers = %#v, want names only", got.Telemetry.Headers)
	}
	if !got.Test.Ok {
		t.Fatalf("test = %#v, want ok", got.Test)
	}
	if got.Test.Status == nil || *got.Test.Status != http.StatusAccepted {
		t.Fatalf("test status = %v, want 202", got.Test.Status)
	}
	if got.Test.Message != nil {
		t.Fatalf("test message = %q, want nil", *got.Test.Message)
	}
	if strings.Contains(fmt.Sprintf("%+v", got), testHeaderValue) {
		t.Fatal("Set response contains a header value, want redacted")
	}
}

func TestService_Set_OmitsNilHeaders(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/telemetry",
		Body: `{"ok":true,"type":"telemetry.updated","telemetry":` + telemetryConfigJSON + `,
			"test":{"ok":true,"status":202,"message":null}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.Set(context.Background(), model.TelemetrySetRequest{Endpoint: testEndpoint}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if value, ok := body["headers"]; ok {
		t.Fatalf("headers = %v, want omitted", value)
	}
}

func TestService_Set_ValidatesRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     model.TelemetrySetRequest
		wantErr string
	}{
		{
			name:    "empty endpoint",
			req:     model.TelemetrySetRequest{},
			wantErr: "endpoint is required",
		},
		{
			name: "too many headers",
			req: model.TelemetrySetRequest{
				Endpoint: testEndpoint,
				Headers:  headers(map[string]string{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1", "f": "1", "g": "1", "h": "1", "i": "1", "j": "1", "k": "1", "l": "1", "m": "1", "n": "1", "o": "1", "p": "1", "q": "1", "r": "1", "s": "1", "t": "1", "u": "1"}),
			},
			wantErr: "at most 20 headers",
		},
		{
			name: "empty header name",
			req: model.TelemetrySetRequest{
				Endpoint: testEndpoint,
				Headers:  headers(map[string]string{"": "value"}),
			},
			wantErr: "header name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &testutil.Doer{}
			svc := telemetry.New(d)

			_, err := svc.Set(context.Background(), tt.req)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
			if len(d.Calls) != 0 {
				t.Fatalf("calls = %d, want no request", len(d.Calls))
			}
		})
	}
}

func TestService_Set_RedactsHeaderValuesFromValidationError(t *testing.T) {
	t.Parallel()

	secret := strings.Repeat("test_secret_value", 600)
	d := &testutil.Doer{}
	svc := telemetry.New(d)

	_, err := svc.Set(context.Background(), model.TelemetrySetRequest{
		Endpoint: testEndpoint,
		Headers:  headers(map[string]string{testHeaderName: secret}),
	})
	if err == nil {
		t.Fatal("Set = nil, want validation error")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "test_secret_value") {
		t.Fatalf("error leaks header value: %v", err)
	}
	if !strings.Contains(err.Error(), testHeaderName) {
		t.Fatalf("error = %v, want the header name", err)
	}
	if len(d.Calls) != 0 {
		t.Fatalf("calls = %d, want no request", len(d.Calls))
	}
}

func TestService_Set_MapsErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		code   string
	}{
		{status: http.StatusBadRequest, code: "invalid_endpoint"},
		{status: http.StatusUnauthorized, code: "unauthorized"},
		{status: http.StatusPaymentRequired, code: "credit_required"},
		{status: http.StatusForbidden, code: "forbidden"},
		{status: http.StatusConflict, code: "conflict"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPut,
				Path:   "/telemetry",
				Status: tt.status,
				Body:   fmt.Sprintf(`{"ok":false,"type":"error","status":%d,"code":%q,"message":"failed"}`, tt.status, tt.code),
			})
			svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			_, err := svc.Set(context.Background(), model.TelemetrySetRequest{Endpoint: testEndpoint})
			if !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestService_Set_NotRetriedWithoutIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/telemetry",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"error","status":503,"code":"unavailable","message":"down"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
		},
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Set(context.Background(), model.TelemetrySetRequest{Endpoint: testEndpoint})
	if !core.IsCode(err, "unavailable") {
		t.Fatalf("error = %v, want unavailable", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestService_Set_RetriesWithCallerIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var key string
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/telemetry",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"error","status":503,"code":"unavailable","message":"down"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
			key = r.Header.Get("Idempotency-Key")
		},
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Set(context.Background(), model.TelemetrySetRequest{Endpoint: testEndpoint},
		core.WithIdempotencyKey("key-1"))
	if !core.IsCode(err, "unavailable") {
		t.Fatalf("error = %v, want unavailable", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
	if key != "key-1" {
		t.Fatalf("Idempotency-Key = %q, want key-1", key)
	}
}

func TestService_Delete_RemovesExport(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/telemetry",
		Body:   `{"ok":true,"type":"telemetry.deleted","telemetry":null}`,
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if err := svc.Delete(context.Background()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestService_Delete_MapsErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		code   string
	}{
		{status: http.StatusUnauthorized, code: "unauthorized"},
		{status: http.StatusForbidden, code: "forbidden"},
		{status: http.StatusNotFound, code: "not_found"},
		{status: http.StatusConflict, code: "conflict"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodDelete,
				Path:   "/telemetry",
				Status: tt.status,
				Body:   fmt.Sprintf(`{"ok":false,"type":"error","status":%d,"code":%q,"message":"failed"}`, tt.status, tt.code),
			})
			svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			err := svc.Delete(context.Background())
			if !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestService_Delete_NoRetryEvenWithIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/telemetry",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"error","status":503,"code":"unavailable","message":"down"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
		},
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	err := svc.Delete(context.Background(), core.WithIdempotencyKey("key-1"))
	if !core.IsCode(err, "unavailable") {
		t.Fatalf("error = %v, want unavailable", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestService_Delete_MapsAPIErrorStatus(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/telemetry",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no export","requestId":"req_1"}`,
	})
	svc := telemetry.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	err := svc.Delete(context.Background())
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *core.APIError", err)
	}
	if apiErr.Status != http.StatusNotFound || apiErr.RequestID != "req_1" {
		t.Fatalf("api error = %#v", apiErr)
	}
}
