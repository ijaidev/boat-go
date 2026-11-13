package webhooks_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/webhooks"
)

func TestService_List_ReturnsWebhooks(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/webhooks",
		Body:   `{"ok":true,"type":"webhook.list","webhooks":[` + webhookOne + `,` + webhookTwo + `]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Authorization"); got != "Bearer boat_test_key" {
				t.Fatalf("Authorization = %q", got)
			}
			if got := r.Header.Get("Accept"); got != "application/json" {
				t.Fatalf("Accept = %q", got)
			}
			if len(r.URL.Query()) != 0 {
				t.Fatalf("query = %v, want empty", r.URL.Query())
			}
		},
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != webhookID || got[0].URL != "https://example.com/hooks/ci" {
		t.Fatalf("first = %#v", got[0])
	}
	if got[0].Name == nil || *got[0].Name != "CI hooks" {
		t.Fatalf("first name = %v", got[0].Name)
	}
	if len(got[0].Events) != 2 || got[0].Events[0] != model.SandboxReady {
		t.Fatalf("first events = %v", got[0].Events)
	}
	if got[1].Name != nil {
		t.Fatalf("second name = %v, want nil", got[1].Name)
	}
}

func TestService_List_Empty(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/webhooks",
		Body:   `{"ok":true,"type":"webhook.list","webhooks":[]}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestService_List_MapsUnauthorized(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/webhooks",
		Status: http.StatusUnauthorized,
		Body:   `{"ok":false,"type":"error","status":401,"code":"unauthorized","message":"bad key","requestId":"req_list"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.List(context.Background())
	requireAPIError(t, err, http.StatusUnauthorized, "unauthorized")
	if core.RetryAfter(err) != 0 {
		t.Fatalf("RetryAfter = %v, want 0", core.RetryAfter(err))
	}
}

func TestService_Get_ReturnsWebhook(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/webhooks/" + webhookID,
		Body:   `{"ok":true,"type":"webhook.info","webhook":` + webhookOne + `}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Get(context.Background(), webhookID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != webhookID || got.URL != "https://example.com/hooks/ci" {
		t.Fatalf("webhook = %#v", got)
	}
}

func TestService_Get_RequiresID(t *testing.T) {
	t.Parallel()

	svc := webhooks.New(&testutil.Doer{})
	if _, err := svc.Get(context.Background(), ""); err == nil {
		t.Fatal("Get = nil, want id error")
	}
}

func TestService_Get_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/webhooks/" + webhookID,
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no webhook","requestId":"req_get"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Get(context.Background(), webhookID)
	requireAPIError(t, err, http.StatusNotFound, "not_found")
}
