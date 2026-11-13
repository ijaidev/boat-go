package environments_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/environments"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_MapsAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		code   string
		call   func(context.Context, *environments.Service) error
	}{
		{
			name:   "create invalid name",
			method: http.MethodPost,
			path:   "/environments",
			status: http.StatusBadRequest,
			code:   "invalid_name",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Create(ctx, environments.CreateRequest{Name: "bad name"})
				return err
			},
		},
		{
			name:   "list unauthorized",
			method: http.MethodGet,
			path:   "/environments",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.List(ctx, environments.ListRequest{})
				return err
			},
		},
		{
			name:   "create billing required",
			method: http.MethodPost,
			path:   "/environments",
			status: http.StatusPaymentRequired,
			code:   "billing_required",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Create(ctx, environments.CreateRequest{Name: "base"})
				return err
			},
		},
		{
			name:   "create forbidden",
			method: http.MethodPost,
			path:   "/environments",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Create(ctx, environments.CreateRequest{Name: "base"})
				return err
			},
		},
		{
			name:   "update not found",
			method: http.MethodPut,
			path:   "/environments/" + envID,
			status: http.StatusNotFound,
			code:   "environment_not_found",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Update(ctx, envID, environments.UpdateRequest{})
				return err
			},
		},
		{
			name:   "create conflict",
			method: http.MethodPost,
			path:   "/environments",
			status: http.StatusConflict,
			code:   "environment_exists",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Create(ctx, environments.CreateRequest{Name: "base"})
				return err
			},
		},
		{
			name:   "upgrade rate limited",
			method: http.MethodPost,
			path:   "/environments/" + envID + "/upgrade",
			status: http.StatusTooManyRequests,
			code:   "rate_limited",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Upgrade(ctx, envID, environments.UpgradeRequest{})
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
					`{"ok":false,"type":"sandbox.error","status":%d,"code":%q,"message":"rejected","requestId":"req_1"}`,
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
			if apiErr.Status != tt.status || apiErr.RequestID != "req_1" {
				t.Fatalf("api error = %#v", apiErr)
			}
		})
	}
}

func TestService_Mutations_NoRetryWithCallerIdempotencyKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		call   func(context.Context, *environments.Service) error
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/environments",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Create(ctx, environments.CreateRequest{Name: "base"})
				return err
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/environments/" + envID,
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Update(ctx, envID, environments.UpdateRequest{})
				return err
			},
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/environments/" + envID,
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Delete(ctx, envID)
				return err
			},
		},
		{
			name:   "upgrade",
			method: http.MethodPost,
			path:   "/environments/" + envID + "/upgrade",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.Upgrade(ctx, envID, environments.UpgradeRequest{})
				return err
			},
		},
		{
			name:   "set var",
			method: http.MethodPut,
			path:   "/environments/" + envID + "/vars/KEY",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.SetVar(ctx, envID, "KEY", "value")
				return err
			},
		},
		{
			name:   "delete var",
			method: http.MethodDelete,
			path:   "/environments/" + envID + "/vars/KEY",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.DeleteVar(ctx, envID, "KEY")
				return err
			},
		},
		{
			name:   "set secret file",
			method: http.MethodPut,
			path:   "/environments/" + envID + "/secret-files",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.SetSecretFile(ctx, envID, environments.SecretFileRequest{Path: ".env"})
				return err
			},
		},
		{
			name:   "delete secret file",
			method: http.MethodDelete,
			path:   "/environments/" + envID + "/secret-files",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.DeleteSecretFile(ctx, envID, ".env")
				return err
			},
		},
		{
			name:   "add repo",
			method: http.MethodPost,
			path:   "/environments/" + envID + "/repos",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.AddRepo(ctx, envID, environments.AddRepoRequest{RepositoryID: "repo_example"})
				return err
			},
		},
		{
			name:   "remove repo",
			method: http.MethodDelete,
			path:   "/environments/" + envID + "/repos/repo_example",
			call: func(ctx context.Context, svc *environments.Service) error {
				_, err := svc.RemoveRepo(ctx, envID, "repo_example")
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			svc := newService(t, testutil.Route{
				Method: tt.method,
				Path:   tt.path,
				Status: http.StatusServiceUnavailable,
				Body:   `{"ok":false,"type":"sandbox.error","status":503,"code":"unavailable","message":"busy"}`,
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					calls.Add(1)
				},
			})

			err := tt.call(context.Background(), svc)
			if err == nil {
				t.Fatal("call = nil, want error")
			}
			if calls.Load() != 1 {
				t.Fatalf("calls = %d, want 1", calls.Load())
			}
		})
	}
}

func TestService_List_RetriesTransientErrors(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/environments",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"sandbox.error","status":503,"code":"unavailable","message":"busy"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
		},
	})

	if _, err := svc.List(context.Background(), environments.ListRequest{}); err == nil {
		t.Fatal("List = nil, want error")
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestService_RequiresIdentifiers(t *testing.T) {
	t.Parallel()

	svc := environments.New(&testutil.Doer{})
	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "create missing name",
			call: func(ctx context.Context) error {
				_, err := svc.Create(ctx, environments.CreateRequest{})
				return err
			},
		},
		{
			name: "update missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.Update(ctx, "", environments.UpdateRequest{})
				return err
			},
		},
		{
			name: "delete missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.Delete(ctx, "")
				return err
			},
		},
		{
			name: "upgrade missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.Upgrade(ctx, "", environments.UpgradeRequest{})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.call(context.Background()); err == nil {
				t.Fatal("call = nil, want error")
			}
		})
	}
}

func TestService_NilTransport(t *testing.T) {
	t.Parallel()

	svc := environments.New(nil)
	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "list",
			call: func(ctx context.Context) error {
				_, err := svc.List(ctx, environments.ListRequest{})
				return err
			},
		},
		{
			name: "create",
			call: func(ctx context.Context) error {
				_, err := svc.Create(ctx, environments.CreateRequest{Name: "base"})
				return err
			},
		},
		{
			name: "update",
			call: func(ctx context.Context) error {
				_, err := svc.Update(ctx, envID, environments.UpdateRequest{})
				return err
			},
		},
		{
			name: "delete",
			call: func(ctx context.Context) error {
				_, err := svc.Delete(ctx, envID)
				return err
			},
		},
		{
			name: "upgrade",
			call: func(ctx context.Context) error {
				_, err := svc.Upgrade(ctx, envID, environments.UpgradeRequest{})
				return err
			},
		},
		{
			name: "set var",
			call: func(ctx context.Context) error {
				_, err := svc.SetVar(ctx, envID, "KEY", "value")
				return err
			},
		},
		{
			name: "delete var",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteVar(ctx, envID, "KEY")
				return err
			},
		},
		{
			name: "set secret file",
			call: func(ctx context.Context) error {
				_, err := svc.SetSecretFile(ctx, envID, environments.SecretFileRequest{Path: ".env"})
				return err
			},
		},
		{
			name: "delete secret file",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteSecretFile(ctx, envID, ".env")
				return err
			},
		},
		{
			name: "add repo",
			call: func(ctx context.Context) error {
				_, err := svc.AddRepo(ctx, envID, environments.AddRepoRequest{RepositoryID: "repo_example"})
				return err
			},
		},
		{
			name: "remove repo",
			call: func(ctx context.Context) error {
				_, err := svc.RemoveRepo(ctx, envID, "repo_example")
				return err
			},
		},
		{
			name: "list repos",
			call: func(ctx context.Context) error {
				_, err := svc.ListRepos(ctx, environments.ListReposRequest{})
				return err
			},
		},
		{
			name: "select repo",
			call: func(ctx context.Context) error {
				_, err := svc.SelectRepo(ctx, environments.SelectRepoRequest{RepositoryID: "repo_example"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.call(context.Background()); err == nil {
				t.Fatal("call = nil, want transport error")
			}
		})
	}
}
