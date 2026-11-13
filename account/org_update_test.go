package account_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_UpdateOrg_SendsOnlySetFields(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/orgs/acme",
		Body:   `{"ok":true,"type":"org.updated","success":true,"name":"Acme Robotics","newSandboxesPrivate":true}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body = readBody(t, r)
		},
	})
	svc := account.New(testutil.Transport(t, srv.URL, testAPIKey))

	got, err := svc.UpdateOrg(context.Background(), "acme", account.UpdateOrgRequest{
		Name:                core.Ptr("Acme Robotics"),
		NewSandboxesPrivate: core.Ptr(true),
	})
	if err != nil {
		t.Fatalf("UpdateOrg: %v", err)
	}
	if !got.Success || got.Name == nil || *got.Name != "Acme Robotics" {
		t.Fatalf("result = %#v", got)
	}
	if got.NewSandboxesPrivate == nil || !*got.NewSandboxesPrivate {
		t.Fatalf("newSandboxesPrivate = %#v", got.NewSandboxesPrivate)
	}
	if body["name"] != "Acme Robotics" || body["newSandboxesPrivate"] != true {
		t.Fatalf("body = %#v", body)
	}
}

func TestService_UpdateOrg_RequiresAField(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	svc := account.New(d)
	if _, err := svc.UpdateOrg(context.Background(), "acme", account.UpdateOrgRequest{}); err == nil {
		t.Fatal("UpdateOrg = nil, want validation error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("Do calls = %d, want 0", len(d.Calls))
	}
}

func TestService_UpdateOrg_NoRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/orgs/acme",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"sandbox.error","status":503,"code":"unavailable","message":"busy"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
		},
	})
	svc := account.New(testutil.Transport(t, srv.URL, testAPIKey))

	_, err := svc.UpdateOrg(context.Background(), "acme", account.UpdateOrgRequest{Name: core.Ptr("new")},
		core.WithIdempotencyKey("key-1"))
	if err == nil {
		t.Fatal("UpdateOrg = nil, want error")
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusServiceUnavailable {
		t.Fatalf("error = %v, want 503", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}
