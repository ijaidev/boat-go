package credentials_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_SetKey_SendsKeyAndKeepCurrent(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/credential-profiles/work/keys/openai-api-key",
		Body:   `{"ok": true, "type": "credential_profile.updated", "success": true, "profile": ` + profileJSON + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var body struct {
				Value       string `json:"value"`
				KeepCurrent *bool  `json:"keepCurrent"`
			}
			decodeBody(t, r, &body)
			if body.Value != "test_key_value" {
				t.Fatal("key value was not forwarded")
			}
			if body.KeepCurrent == nil || !*body.KeepCurrent {
				t.Fatalf("keepCurrent = %v, want true", body.KeepCurrent)
			}
		},
	})

	profile, err := svc.SetKey(context.Background(), "work", model.OpenaiAPIKey, credentials.SetKeyRequest{
		Value:       "test_key_value",
		KeepCurrent: core.Ptr(true),
	})
	if err != nil {
		t.Fatalf("SetKey: %v", err)
	}
	if profile.ID != "prof_example" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestService_SetKey_OmitsKeepCurrentByDefault(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/credential-profiles/work/keys/openai-api-key",
		Body:   `{"ok": true, "type": "credential_profile.updated", "success": true, "profile": ` + profileJSON + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var body map[string]any
			decodeBody(t, r, &body)
			if _, ok := body["keepCurrent"]; ok {
				t.Fatal("keepCurrent was sent, want omitted")
			}
		},
	})

	if _, err := svc.SetKey(context.Background(), "work", model.OpenaiAPIKey, credentials.SetKeyRequest{
		Value: "test_key_value",
	}); err != nil {
		t.Fatalf("SetKey: %v", err)
	}
}

func TestService_DeleteKey_ReturnsUpdatedProfile(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/credential-profiles/work/keys/claude-subscription",
		Body:   `{"ok": true, "type": "credential_profile.updated", "success": true, "profile": ` + profileJSON + `}`,
	})

	profile, err := svc.DeleteKey(context.Background(), "work", model.ClaudeSubscription)
	if err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if profile.ID != "prof_example" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestService_SetKey_NoRetryWithCallerIdempotencyKey(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/credential-profiles/work/keys/openai-api-key",
		Status: http.StatusServiceUnavailable,
		Body:   `{"ok": false, "type": "credential_profile.error", "status": 503, "code": "unavailable", "message": "try later"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			calls.Add(1)
		},
	})
	svc := credentials.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.SetKey(context.Background(), "work", model.OpenaiAPIKey, credentials.SetKeyRequest{
		Value: "test_key_value",
	}, core.WithIdempotencyKey("key-1"))
	if err == nil {
		t.Fatal("SetKey = nil, want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}
