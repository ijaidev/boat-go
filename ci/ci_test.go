package ci_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/ci"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

const jobID = "122395651084"

func newService(t *testing.T, routes ...testutil.Route) *ci.Service {
	t.Helper()
	srv := testutil.NewServer(t, routes...)
	return ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))
}

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestService_MapsAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		code   string
		call   func(context.Context, *ci.Service) error
	}{
		{
			name:   "overview bad request",
			method: http.MethodGet,
			path:   "/ci",
			status: http.StatusBadRequest,
			code:   "invalid_request",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Overview(ctx, ci.OverviewRequest{})
				return err
			},
		},
		{
			name:   "overview unauthorized",
			method: http.MethodGet,
			path:   "/ci",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Overview(ctx, ci.OverviewRequest{})
				return err
			},
		},
		{
			name:   "connect payment required",
			method: http.MethodPost,
			path:   "/ci/connect",
			status: http.StatusPaymentRequired,
			code:   "billing_required",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Connect(ctx, ci.ConnectRequest{})
				return err
			},
		},
		{
			name:   "public repos forbidden",
			method: http.MethodPost,
			path:   "/ci/public",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.PublicRepos(ctx, ci.PublicReposRequest{Account: "acme"})
				return err
			},
		},
		{
			name:   "job not found",
			method: http.MethodGet,
			path:   "/ci/jobs/" + jobID,
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Job(ctx, jobID)
				return err
			},
		},
		{
			name:   "connect conflict",
			method: http.MethodPost,
			path:   "/ci/connect",
			status: http.StatusConflict,
			code:   "already_connected",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Connect(ctx, ci.ConnectRequest{Account: "acme"})
				return err
			},
		},
		{
			name:   "job log rate limited",
			method: http.MethodGet,
			path:   "/ci/jobs/" + jobID + "/logs",
			status: http.StatusTooManyRequests,
			code:   "rate_limited",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.JobLog(ctx, jobID)
				return err
			},
		},
		{
			name:   "overview unavailable",
			method: http.MethodGet,
			path:   "/ci",
			status: http.StatusServiceUnavailable,
			code:   "unavailable",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Overview(ctx, ci.OverviewRequest{})
				return err
			},
		},
		{
			name:   "job internal error",
			method: http.MethodGet,
			path:   "/ci/jobs/" + jobID,
			status: http.StatusInternalServerError,
			code:   "internal",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.Job(ctx, jobID)
				return err
			},
		},
		{
			name:   "job log bad gateway",
			method: http.MethodGet,
			path:   "/ci/jobs/" + jobID + "/logs",
			status: http.StatusBadGateway,
			code:   "bad_gateway",
			call: func(ctx context.Context, svc *ci.Service) error {
				_, err := svc.JobLog(ctx, jobID)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t, testutil.Route{
				Method: tt.method,
				Path:   tt.path,
				Status: tt.status,
				Body: fmt.Sprintf(
					`{"ok":false,"type":"ci.error","status":%d,"code":%q,"message":"rejected","requestId":"req_1"}`,
					tt.status, tt.code,
				),
			})

			err := tt.call(context.Background(), svc)
			if !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %s", err, tt.code)
			}
			var apiErr *core.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T, want *core.APIError", err)
			}
			if apiErr.Status != tt.status {
				t.Fatalf("status = %d, want %d", apiErr.Status, tt.status)
			}
			if apiErr.RequestID != "req_1" {
				t.Fatalf("request id = %q, want req_1", apiErr.RequestID)
			}
		})
	}
}

func TestService_NilTransport(t *testing.T) {
	t.Parallel()

	svc := ci.New(nil)
	ctx := context.Background()
	if _, err := svc.Overview(ctx, ci.OverviewRequest{}); err == nil {
		t.Fatal("Overview = nil, want transport error")
	}
	if _, err := svc.Connect(ctx, ci.ConnectRequest{}); err == nil {
		t.Fatal("Connect = nil, want transport error")
	}
	if _, err := svc.Job(ctx, jobID); err == nil {
		t.Fatal("Job = nil, want transport error")
	}
	if _, err := svc.JobLog(ctx, jobID); err == nil {
		t.Fatal("JobLog = nil, want transport error")
	}
	if _, err := svc.PublicRepos(ctx, ci.PublicReposRequest{Account: "acme"}); err == nil {
		t.Fatal("PublicRepos = nil, want transport error")
	}
}
