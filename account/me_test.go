package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Me_ReturnsCurrentUser(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/me",
		Body: `{
			"ok": true,
			"type": "user.info",
			"user": {
				"id": "acct_example",
				"login": "example-user",
				"email": null,
				"displayName": "Example User",
				"accountStatus": "active",
				"zeroDataRetention": true,
				"zeroDataRetentionEnabledAt": "2026-11-01T00:00:00Z"
			}
		}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
				t.Fatalf("Authorization = %q, want bearer test key", got)
			}
			if got := r.URL.RawQuery; got != "" {
				t.Fatalf("query = %q, want empty", got)
			}
		},
	})

	got, err := svc.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if got.Type != model.UserInfo {
		t.Fatalf("type = %q, want user.info", got.Type)
	}
	if got.User.ID == nil || *got.User.ID != "acct_example" {
		t.Fatal("user id mismatch")
	}
	if got.User.Login == nil || *got.User.Login != "example-user" {
		t.Fatal("login mismatch")
	}
	if got.User.DisplayName == nil || *got.User.DisplayName != "Example User" {
		t.Fatal("display name mismatch")
	}
	if got.User.Email != nil {
		t.Fatal("email = non-nil, want null")
	}
	if got.User.AccountStatus == nil || *got.User.AccountStatus != model.MeResponseUserAccountStatusActive {
		t.Fatal("account status mismatch")
	}
	if got.User.ZeroDataRetention == nil || !*got.User.ZeroDataRetention {
		t.Fatal("zeroDataRetention = false, want true")
	}
	if got.User.ZeroDataRetentionEnabledAt == nil {
		t.Fatal("zeroDataRetentionEnabledAt = nil, want a time")
	}
}
