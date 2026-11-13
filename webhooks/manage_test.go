package webhooks_test

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/webhooks"
)

func TestService_Create_SendsRequestAndReturnsSecret(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/webhooks",
		Status: http.StatusCreated,
		Body:   `{"ok":true,"type":"webhook.created","webhook":` + webhookOne + `,"secret":"` + testSecret + `"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
			body := decodeBody(t, r)
			want := map[string]any{
				"name":   "CI hooks",
				"url":    "https://example.com/hooks/ci",
				"events": []any{"sandbox.ready", "sandbox.error"},
			}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body = %#v, want %#v", body, want)
			}
		},
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Create(context.Background(), webhooks.CreateRequest{
		Name:   core.Some("CI hooks"),
		URL:    "https://example.com/hooks/ci",
		Events: []model.WebhookEventType{model.SandboxReady, model.SandboxError},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Secret != testSecret {
		t.Fatalf("Secret = %q", got.Secret)
	}
	if got.Webhook.ID != webhookID || len(got.Webhook.Events) != 2 {
		t.Fatalf("Webhook = %#v", got.Webhook)
	}
}

func TestService_Create_NameStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nameValue *core.Optional[string]
		check     func(*testing.T, map[string]any)
	}{
		{
			name: "omitted",
			check: func(t *testing.T, body map[string]any) {
				t.Helper()
				if _, ok := body["name"]; ok {
					t.Fatalf("body = %#v, want no name", body)
				}
			},
		},
		{
			name:      "explicit null",
			nameValue: core.Null[string](),
			check: func(t *testing.T, body map[string]any) {
				t.Helper()
				v, ok := body["name"]
				if !ok || v != nil {
					t.Fatalf("body = %#v, want name null", body)
				}
			},
		},
		{
			name:      "value",
			nameValue: core.Some("hooks"),
			check: func(t *testing.T, body map[string]any) {
				t.Helper()
				if body["name"] != "hooks" {
					t.Fatalf("body = %#v, want name hooks", body)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPost,
				Path:   "/webhooks",
				Status: http.StatusCreated,
				Body:   `{"ok":true,"type":"webhook.created","webhook":` + webhookOne + `,"secret":"` + testSecret + `"}`,
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					tt.check(t, decodeBody(t, r))
				},
			})
			svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			if _, err := svc.Create(context.Background(), webhooks.CreateRequest{
				Name:   tt.nameValue,
				URL:    "https://example.com/hooks/ci",
				Events: []model.WebhookEventType{model.SandboxReady},
			}); err != nil {
				t.Fatalf("Create: %v", err)
			}
		})
	}
}

func TestService_Create_MapsConflict(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/webhooks",
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"error","status":409,"code":"conflict","message":"endpoint already registered","requestId":"req_create"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Create(context.Background(), webhooks.CreateRequest{
		URL:    "https://example.com/hooks/ci",
		Events: []model.WebhookEventType{model.SandboxReady},
	})
	requireAPIError(t, err, http.StatusConflict, "conflict")
}

func TestService_Create_IsNotRetriedWithoutIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/webhooks",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok":false,"type":"error","status":503,"code":"unavailable","message":"try later"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
			if got := r.Header.Get("Idempotency-Key"); got != "" {
				t.Fatalf("Idempotency-Key = %q, want empty", got)
			}
		},
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Create(context.Background(), webhooks.CreateRequest{
		URL:    "https://example.com/hooks/ci",
		Events: []model.WebhookEventType{model.SandboxReady},
	})
	requireAPIError(t, err, http.StatusServiceUnavailable, "unavailable")
	if got := calls.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestService_Update_SendsPatch(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/webhooks/" + webhookID,
		Body:   `{"ok":true,"type":"webhook.updated","webhook":` + webhookOne + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body := decodeBody(t, r)
			want := map[string]any{
				"name":   nil,
				"url":    "https://example.com/hooks/new",
				"events": []any{"sandbox.archived"},
			}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body = %#v, want %#v", body, want)
			}
		},
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Update(context.Background(), webhookID, webhooks.UpdateRequest{
		Name:   core.Null[string](),
		URL:    core.Ptr("https://example.com/hooks/new"),
		Events: []model.WebhookEventType{model.SandboxArchived},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != webhookID {
		t.Fatalf("webhook = %#v", got)
	}
}

func TestService_Update_OmitsUnsetName(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/webhooks/" + webhookID,
		Body:   `{"ok":true,"type":"webhook.updated","webhook":` + webhookOne + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body := decodeBody(t, r)
			want := map[string]any{"url": "https://example.com/hooks/new"}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body = %#v, want %#v", body, want)
			}
		},
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Update(context.Background(), webhookID, webhooks.UpdateRequest{
		URL: core.Ptr("https://example.com/hooks/new"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestService_Update_MapsBadRequest(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/webhooks/" + webhookID,
		Status: http.StatusBadRequest,
		Body:   `{"ok":false,"type":"error","status":400,"code":"invalid_request","message":"empty update","requestId":"req_update"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Update(context.Background(), webhookID, webhooks.UpdateRequest{})
	requireAPIError(t, err, http.StatusBadRequest, "invalid_request")
}

func TestService_Delete_ReturnsDeletedID(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/webhooks/" + webhookID,
		Body:   `{"ok":true,"type":"webhook.deleted","id":"` + webhookID + `"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Delete(context.Background(), webhookID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got != webhookID {
		t.Fatalf("id = %q, want %q", got, webhookID)
	}
}

func TestService_Delete_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/webhooks/" + webhookID,
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no webhook"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Delete(context.Background(), webhookID)
	requireAPIError(t, err, http.StatusNotFound, "not_found")
}

func TestService_Rotate_ReturnsNewSecret(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/webhooks/" + webhookID + "/rotate",
		Body:   `{"ok":true,"type":"webhook.rotated","webhook":` + webhookOne + `,"secret":"` + testSecret + `"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if r.ContentLength != 0 {
				t.Fatalf("ContentLength = %d, want 0", r.ContentLength)
			}
		},
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Rotate(context.Background(), webhookID)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if got.Secret != testSecret || got.Webhook.ID != webhookID {
		t.Fatalf("result = %#v", got)
	}
}

func TestService_Rotate_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/webhooks/" + webhookID + "/rotate",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no webhook"}`,
	})
	svc := webhooks.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Rotate(context.Background(), webhookID)
	requireAPIError(t, err, http.StatusNotFound, "not_found")
}

func TestService_MutationsRequireID(t *testing.T) {
	t.Parallel()

	svc := webhooks.New(&testutil.Doer{})
	tests := []struct {
		name string
		call func() error
	}{
		{name: "update", call: func() error {
			_, err := svc.Update(context.Background(), "", webhooks.UpdateRequest{})
			return err
		}},
		{name: "delete", call: func() error {
			_, err := svc.Delete(context.Background(), "")
			return err
		}},
		{name: "rotate", call: func() error {
			_, err := svc.Rotate(context.Background(), "")
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.call(); err == nil {
				t.Fatal("error = nil, want id error")
			}
		})
	}
}

func TestSecretResult_RedactsSecretInFormatting(t *testing.T) {
	t.Parallel()

	result := webhooks.SecretResult{
		Webhook: model.Webhook{ID: webhookID},
		Secret:  testSecret,
	}
	if result.Secret != testSecret {
		t.Fatalf("Secret = %q", result.Secret)
	}
	for _, formatted := range []string{
		fmt.Sprint(result),
		fmt.Sprintf("%v", result),
		fmt.Sprintf("%+v", result),
		fmt.Sprintf("%#v", result),
	} {
		if strings.Contains(formatted, testSecret) {
			t.Fatalf("formatted secret leaked: %s", formatted)
		}
		if !strings.Contains(formatted, "[REDACTED]") {
			t.Fatalf("formatted secret is not marked redacted: %s", formatted)
		}
	}
}
