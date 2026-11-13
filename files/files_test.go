package files_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/files"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func newService(t *testing.T, routes ...testutil.Route) *files.Service {
	t.Helper()
	srv := testutil.NewServer(t, routes...)
	return files.New(testutil.Transport(t, srv.URL, "boat_test_key"))
}

func TestService_MapsAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		code   string
		call   func(context.Context, *files.Service) error
	}{
		{
			name:   "read invalid path",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/files",
			status: http.StatusBadRequest,
			code:   "invalid_path",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Read(ctx, "bx_example", "../etc/passwd", files.EncodingUTF8)
				return err
			},
		},
		{
			name:   "list unauthorized",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/files/list",
			status: http.StatusUnauthorized,
			code:   "unauthorized",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.List(ctx, "bx_example", files.ListRequest{})
				return err
			},
		},
		{
			name:   "host billing required",
			method: http.MethodPost,
			path:   "/sandboxes/bx_example/host",
			status: http.StatusPaymentRequired,
			code:   "billing_required",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Host(ctx, "bx_example", files.HostRequest{Port: 3000})
				return err
			},
		},
		{
			name:   "write forbidden",
			method: http.MethodPut,
			path:   "/sandboxes/bx_example/files",
			status: http.StatusForbidden,
			code:   "forbidden",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Write(ctx, "bx_example", files.WriteRequest{Path: "/home/user/x", Content: "x"})
				return err
			},
		},
		{
			name:   "info not found",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/files/info",
			status: http.StatusNotFound,
			code:   "not_found",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Info(ctx, "bx_example", "/home/user/missing")
				return err
			},
		},
		{
			name:   "move destination exists",
			method: http.MethodPost,
			path:   "/sandboxes/bx_example/files/move",
			status: http.StatusConflict,
			code:   "destination_exists",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Move(ctx, "bx_example", files.MoveRequest{
					Source:      "/home/user/a",
					Destination: "/home/user/b",
				})
				return err
			},
		},
		{
			name:   "write too large",
			method: http.MethodPut,
			path:   "/sandboxes/bx_example/files",
			status: http.StatusRequestEntityTooLarge,
			code:   "file_too_large",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Write(ctx, "bx_example", files.WriteRequest{Path: "/home/user/big", Content: "x"})
				return err
			},
		},
		{
			name:   "search rate limited",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/files/search",
			status: http.StatusTooManyRequests,
			code:   "rate_limited",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Search(ctx, "bx_example", files.SearchRequest{Pattern: "*.go"})
				return err
			},
		},
		{
			name:   "find internal error",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/files/find",
			status: http.StatusInternalServerError,
			code:   "internal",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Find(ctx, "bx_example", files.FindRequest{Pattern: "needle"})
				return err
			},
		},
		{
			name:   "download url bad gateway",
			method: http.MethodPost,
			path:   "/sandboxes/bx_example/files/download-url",
			status: http.StatusBadGateway,
			code:   "bad_gateway",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.DownloadURL(ctx, "bx_example", files.URLRequest{Path: "/home/user/x"})
				return err
			},
		},
		{
			name:   "artifact unavailable",
			method: http.MethodGet,
			path:   "/sandboxes/bx_example/artifacts",
			status: http.StatusServiceUnavailable,
			code:   "unavailable",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Artifact(ctx, "bx_example", "/home/user/out.bin")
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

	svc := files.New(nil)
	if _, err := svc.Info(context.Background(), "bx_example", "/home/user/x"); err == nil {
		t.Fatal("Info = nil, want transport error")
	}
	if _, err := svc.Artifact(context.Background(), "bx_example", "/home/user/x"); err == nil {
		t.Fatal("Artifact = nil, want transport error")
	}
}

func TestService_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *files.Service) error
	}{
		{
			name: "read",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Read(ctx, "", "/home/user/a", files.EncodingUTF8)
				return err
			},
		},
		{
			name: "write",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Write(ctx, "", files.WriteRequest{Path: "/home/user/a"})
				return err
			},
		},
		{
			name: "delete",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Delete(ctx, "", "/home/user/a", files.DeleteOptions{})
				return err
			},
		},
		{
			name: "list",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.List(ctx, "", files.ListRequest{})
				return err
			},
		},
		{
			name: "info",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Info(ctx, "", "/home/user/a")
				return err
			},
		},
		{
			name: "search",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Search(ctx, "", files.SearchRequest{Pattern: "*.go"})
				return err
			},
		},
		{
			name: "find",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Find(ctx, "", files.FindRequest{Pattern: "needle"})
				return err
			},
		},
		{
			name: "mkdir",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Mkdir(ctx, "", files.MkdirRequest{Path: "/home/user/new"})
				return err
			},
		},
		{
			name: "move",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Move(ctx, "", files.MoveRequest{Source: "/home/user/a", Destination: "/home/user/b"})
				return err
			},
		},
		{
			name: "chmod",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Chmod(ctx, "", files.ChmodRequest{Path: "/home/user/a", Mode: "0755"})
				return err
			},
		},
		{
			name: "replace",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Replace(ctx, "", files.ReplaceRequest{Paths: []string{"/home/user/a"}})
				return err
			},
		},
		{
			name: "upload url",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.UploadURL(ctx, "", files.URLRequest{Path: "/home/user/a"})
				return err
			},
		},
		{
			name: "download url",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.DownloadURL(ctx, "", files.URLRequest{Path: "/home/user/a"})
				return err
			},
		},
		{
			name: "artifact",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Artifact(ctx, "", "/home/user/a")
				return err
			},
		},
		{
			name: "host",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Host(ctx, "", files.HostRequest{Port: 3000})
				return err
			},
		},
		{
			name: "desktop",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.Desktop(ctx, "", files.DesktopRequest{})
				return err
			},
		},
		{
			name: "ssh key",
			call: func(ctx context.Context, svc *files.Service) error {
				_, err := svc.SSHKey(ctx, "", files.SSHKeyRequest{Key: "ssh-ed25519 AAAAexample user@example.com"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if err := tt.call(context.Background(), files.New(d)); err == nil {
				t.Fatal("call = nil, want sandbox id error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("calls = %d, want 0", len(d.Calls))
			}
		})
	}
}
