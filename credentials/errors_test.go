package credentials_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_MapsAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		status     int
		code       string
		header     http.Header
		retryAfter time.Duration
		call       func(context.Context, *credentials.Service) error
	}{
		{
			name:   "list profiles unauthorized",
			method: http.MethodGet,
			path:   "/credential-profiles",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.ListProfiles(ctx)
				return err
			},
		},
		{
			name:   "create profile bad request",
			method: http.MethodPost,
			path:   "/credential-profiles",
			status: http.StatusBadRequest,
			code:   "invalid_profile",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.CreateProfile(ctx, credentials.CreateProfileRequest{Name: "work"})
				return err
			},
		},
		{
			name:   "update profile forbidden",
			method: http.MethodPut,
			path:   "/credential-profiles/work",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.UpdateProfile(ctx, "work", credentials.UpdateProfileRequest{Name: "renamed"})
				return err
			},
		},
		{
			name:   "delete profile is default",
			method: http.MethodDelete,
			path:   "/credential-profiles/work",
			status: http.StatusConflict,
			code:   "is_default",
			call: func(ctx context.Context, svc *credentials.Service) error {
				return svc.DeleteProfile(ctx, "work")
			},
		},
		{
			name:   "set key not an api key",
			method: http.MethodPut,
			path:   "/credential-profiles/work/keys/openai-api-key",
			status: http.StatusBadRequest,
			code:   "not_an_api_key",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetKey(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name:   "delete key profile not found",
			method: http.MethodDelete,
			path:   "/credential-profiles/work/keys/openai-api-key",
			status: http.StatusNotFound,
			code:   "credential_profile_not_found",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteKey(ctx, "work", model.OpenaiAPIKey)
				return err
			},
		},
		{
			name:   "list accounts forbidden",
			method: http.MethodGet,
			path:   "/credential-profiles/work/accounts/openai-api-key",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.ListAccounts(ctx, "work", model.OpenaiAPIKey)
				return err
			},
		},
		{
			name:   "add account not found",
			method: http.MethodPost,
			path:   "/credential-profiles/work/accounts/openai-api-key",
			status: http.StatusNotFound,
			code:   "credential_not_found",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.AddAccount(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name:   "switch account not found",
			method: http.MethodPut,
			path:   "/credential-profiles/work/accounts/openai-api-key/acct_1/active",
			status: http.StatusNotFound,
			code:   "account_not_found",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SwitchAccount(ctx, "work", model.OpenaiAPIKey, "acct_1")
				return err
			},
		},
		{
			name:   "delete account not found",
			method: http.MethodDelete,
			path:   "/credential-profiles/work/accounts/openai-api-key/acct_1",
			status: http.StatusNotFound,
			code:   "account_not_found",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteAccount(ctx, "work", model.OpenaiAPIKey, "acct_1")
				return err
			},
		},
		{
			name:   "set custom api rate limited",
			method: http.MethodPut,
			path:   "/credential-profiles/work/custom-apis/STRIPE_KEY",
			status: http.StatusTooManyRequests,
			code:   "rate_limited",
			header: http.Header{"Retry-After": {"1"}},
			// The mutation is never retried, but the server hint is still carried.
			retryAfter: time.Second,
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "work", "STRIPE_KEY", credentials.SetCustomAPIRequest{
					Value:   "test_custom_api_value",
					BaseURL: "https://api.example.com/v1",
				})
				return err
			},
		},
		{
			name:   "delete custom api not found",
			method: http.MethodDelete,
			path:   "/credential-profiles/work/custom-apis/STRIPE_KEY",
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.DeleteCustomAPI(ctx, "work", "STRIPE_KEY")
				return err
			},
		},
		{
			name:   "get proxy unauthorized",
			method: http.MethodGet,
			path:   "/ai-proxy",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.GetProxy(ctx, credentials.GetProxyRequest{})
				return err
			},
		},
		{
			name:   "set proxy forbidden",
			method: http.MethodPut,
			path:   "/ai-proxy",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetProxy(ctx, credentials.SetProxyRequest{Enabled: true})
				return err
			},
		},
		{
			name:   "usage unauthorized",
			method: http.MethodGet,
			path:   "/ai-usage",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.Usage(ctx, credentials.UsageRequest{})
				return err
			},
		},
		{
			name:   "sandbox usage not found",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/ai-usage",
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SandboxUsage(ctx, "bx_example", credentials.SandboxUsageRequest{})
				return err
			},
		},
		{
			name:   "set sandbox access forbidden",
			method: http.MethodPut,
			path:   "/sandboxes/bx_example/ai-access",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetSandboxAccess(ctx, "bx_example", credentials.SetSandboxAccessRequest{Blocked: true})
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
				Header: tt.header,
				Body: fmt.Sprintf(
					`{"ok": false, "type": "credential.error", "status": %d, "code": %q, "message": "rejected", "requestId": "req_1"}`,
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
			if apiErr.Message != "rejected" {
				t.Fatalf("message = %q", apiErr.Message)
			}
			if tt.retryAfter != 0 && apiErr.RetryAfter != tt.retryAfter {
				t.Fatalf("retry after = %v, want %v", apiErr.RetryAfter, tt.retryAfter)
			}
		})
	}
}

func TestService_ErrorsDoNotContainSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		call   func(context.Context, *credentials.Service) error
	}{
		{
			name:   "set key",
			method: http.MethodPut,
			path:   "/credential-profiles/work/keys/openai-api-key",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetKey(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name:   "add account",
			method: http.MethodPost,
			path:   "/credential-profiles/work/accounts/openai-api-key",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.AddAccount(ctx, "work", model.OpenaiAPIKey, credentials.SetKeyRequest{Value: "test_key_value"})
				return err
			},
		},
		{
			name:   "set custom api",
			method: http.MethodPut,
			path:   "/credential-profiles/work/custom-apis/STRIPE_KEY",
			call: func(ctx context.Context, svc *credentials.Service) error {
				_, err := svc.SetCustomAPI(ctx, "work", "STRIPE_KEY", credentials.SetCustomAPIRequest{
					Value:   "test_custom_api_value",
					BaseURL: "https://api.example.com/v1",
				})
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
				Status: http.StatusBadRequest,
				Body:   `{"ok": false, "type": "credential.error", "status": 400, "code": "invalid_key", "message": "rejected"}`,
			})

			err := tt.call(context.Background(), svc)
			if err == nil {
				t.Fatal("call = nil, want error")
			}
			msg := err.Error()
			for _, secret := range []string{"test_key_value", "test_custom_api_value"} {
				if strings.Contains(msg, secret) {
					t.Fatal("error text contains a secret value")
				}
			}
		})
	}
}
