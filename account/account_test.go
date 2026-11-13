package account_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

const testAPIKey = "boat_test_key"

func newService(t *testing.T, routes ...testutil.Route) *account.Service {
	t.Helper()
	srv := testutil.NewServer(t, routes...)
	return account.New(testutil.Transport(t, srv.URL, testAPIKey))
}

func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
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
		call   func(context.Context, *account.Service) error
	}{
		{
			name:   "me unauthorized",
			method: http.MethodGet,
			path:   "/me",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.Me(ctx)
				return err
			},
		},
		{
			name:   "orgs unauthorized",
			method: http.MethodGet,
			path:   "/orgs",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.Orgs(ctx)
				return err
			},
		},
		{
			name:   "set active org not a member",
			method: http.MethodPatch,
			path:   "/orgs/active",
			status: http.StatusForbidden,
			code:   "not_org_member",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.SetActiveOrg(ctx, "acme")
				return err
			},
		},
		{
			name:   "limits unauthorized",
			method: http.MethodGet,
			path:   "/limits",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.Limits(ctx, account.LimitsRequest{})
				return err
			},
		},
		{
			name:   "api keys forbidden",
			method: http.MethodGet,
			path:   "/api-keys",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.APIKeys(ctx)
				return err
			},
		},
		{
			name:   "create scoped key invalid",
			method: http.MethodPost,
			path:   "/api-keys/scoped",
			status: http.StatusBadRequest,
			code:   "invalid_scope",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.CreateScopedAPIKey(ctx, account.CreateScopedAPIKeyRequest{Preset: "read-only"})
				return err
			},
		},
		{
			name:   "rotate key expired",
			method: http.MethodPost,
			path:   "/api-keys/sak_example/rotate",
			status: http.StatusConflict,
			code:   "api_key_expired",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.RotateAPIKey(ctx, "sak_example")
				return err
			},
		},
		{
			name:   "revoke key not found",
			method: http.MethodDelete,
			path:   "/api-keys/sak_example",
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.RevokeAPIKey(ctx, "sak_example")
				return err
			},
		},
		{
			name:   "key usage not found",
			method: http.MethodGet,
			path:   "/api-keys/sak_example/usage",
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.APIKeyUsage(ctx, "sak_example")
				return err
			},
		},
		{
			name:   "data retention forbidden",
			method: http.MethodGet,
			path:   "/account/data-retention",
			status: http.StatusForbidden,
			code:   "api_key_forbidden",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.DataRetention(ctx)
				return err
			},
		},
		{
			name:   "update data retention forbidden",
			method: http.MethodPatch,
			path:   "/account/data-retention",
			status: http.StatusForbidden,
			code:   "session_required",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.UpdateDataRetention(ctx, account.UpdateDataRetentionRequest{
					Enabled:      core.Ptr(false),
					Confirmation: account.DataRetentionConfirmation,
				})
				return err
			},
		},
		{
			name:   "update data retention conflict",
			method: http.MethodPatch,
			path:   "/account/data-retention",
			status: http.StatusConflict,
			code:   "retention_conflict",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.UpdateDataRetention(ctx, account.UpdateDataRetentionRequest{
					Enabled: core.Ptr(true),
				})
				return err
			},
		},
		{
			name:   "deletion operation not found",
			method: http.MethodGet,
			path:   "/deletion-operations/bdop_example",
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.DeletionOperation(ctx, "bdop_example")
				return err
			},
		},
		{
			name:   "secrets unauthorized",
			method: http.MethodGet,
			path:   "/secrets",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.Secrets(ctx)
				return err
			},
		},
		{
			name:   "update secrets unauthorized",
			method: http.MethodPost,
			path:   "/secrets",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.UpdateSecrets(ctx, account.UpdateSecretsRequest{})
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
					`{"ok":false,"type":"account.error","status":%d,"code":%q,"message":"rejected","requestId":"req_example"}`,
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
			if apiErr.RequestID != "req_example" {
				t.Fatalf("request id = %q, want req_example", apiErr.RequestID)
			}
		})
	}
}

func TestService_NilTransport(t *testing.T) {
	t.Parallel()

	svc := account.New(nil)
	if _, err := svc.Me(context.Background()); err == nil {
		t.Fatal("Me = nil, want transport error")
	}
	if _, err := svc.DataRetention(context.Background()); err == nil {
		t.Fatal("DataRetention = nil, want transport error")
	}
}
