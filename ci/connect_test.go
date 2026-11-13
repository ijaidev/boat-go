package ci_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/ci"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Connect_InstallsApp(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/ci/connect",
		Body:   `{"ok":true,"type":"ci.connect","installUrl":"https://github.com/apps/boat/installations/new?state=example"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("content type = %q", got)
			}
			if body := decodeBody(t, r); len(body) != 0 {
				t.Fatalf("body = %v, want no fields", body)
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Connect(context.Background(), ci.ConnectRequest{})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !got.Ok || got.Type != model.CiConnect {
		t.Fatalf("envelope = %#v", got)
	}
	if got.InstallURL != "https://github.com/apps/boat/installations/new?state=example" {
		t.Fatalf("install URL = %q", got.InstallURL)
	}
}

func TestService_Connect_AuthorizesExistingAccount(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/ci/connect",
		Body:   `{"ok":true,"type":"ci.connect","installUrl":"https://github.com/apps/boat/installations/new?state=example"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body := decodeBody(t, r)
			if len(body) != 3 {
				t.Fatalf("body = %v, want org, account and orgApp", body)
			}
			if body["org"] != "acme" {
				t.Fatalf("org = %v", body["org"])
			}
			if body["account"] != "acme" {
				t.Fatalf("account = %v", body["account"])
			}
			if body["orgApp"] != true {
				t.Fatalf("orgApp = %v", body["orgApp"])
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Connect(context.Background(), ci.ConnectRequest{
		Org:     "acme",
		Account: "acme",
		OrgApp:  core.Ptr(true),
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
}

func TestService_Connect_NoRetryWithCallerIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/ci/connect",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"ci.error","status":503,"code":"unavailable","message":"try later"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Connect(context.Background(), ci.ConnectRequest{},
		core.WithIdempotencyKey("key-1"))
	if err == nil {
		t.Fatal("Connect = nil, want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestService_Connect_SetsNoRetry(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{DoFunc: func(_ context.Context, req *core.Request, _ any, _ ...core.RequestOption) error {
		if !req.NoRetry {
			t.Fatal("NoRetry = false, want true")
		}
		if req.Method != http.MethodPost || req.Path != "/ci/connect" {
			t.Fatalf("request = %s %s", req.Method, req.Path)
		}
		return nil
	}}
	if _, err := ci.New(d).Connect(context.Background(), ci.ConnectRequest{}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
}
