package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_Limits_SelectsWallet(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/limits",
		Body: `{
			"ok": true,
			"type": "limits.info",
			"canStart": true,
			"activeSandboxes": 1,
			"maxActiveSandboxes": 100,
			"billingStatus": "active",
			"teamId": "team_example",
			"wallet": {"balanceDollars": 52.4},
			"starts": {
				"unlimited": false,
				"minute": {"limit": 10, "used": 3, "remaining": 7}
			}
		}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("org"); got != "example-org" {
				t.Fatalf("org = %q, want example-org", got)
			}
			if got := query.Get("teamId"); got != "team_example" {
				t.Fatalf("teamId = %q, want team_example", got)
			}
			if got := r.Header.Get("X-Boat-Org"); got != "team_example" {
				t.Fatalf("X-Boat-Org = %q, want team_example", got)
			}
		},
	})

	got, err := svc.Limits(context.Background(), account.LimitsRequest{
		Org:    "example-org",
		TeamID: "team_example",
	}, core.WithOrg("team_example"))
	if err != nil {
		t.Fatalf("Limits: %v", err)
	}
	if !got.CanStart {
		t.Fatal("canStart = false, want true")
	}
	if got.ActiveSandboxes != 1 || got.MaxActiveSandboxes != 100 {
		t.Fatalf("sandboxes = %d/%d, want 1/100", got.ActiveSandboxes, got.MaxActiveSandboxes)
	}
	if got.TeamID == nil || *got.TeamID != "team_example" {
		t.Fatal("teamId mismatch")
	}
	if got.Starts == nil || got.Starts.Minute == nil || got.Starts.Minute.Remaining == nil {
		t.Fatal("start window missing")
	}
	if *got.Starts.Minute.Remaining != 7 {
		t.Fatalf("remaining starts = %d, want 7", *got.Starts.Minute.Remaining)
	}
}

func TestService_Limits_OmitsUnsetWallet(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/limits",
		Body:   `{"ok":true,"type":"limits.info","canStart":true,"activeSandboxes":0,"maxActiveSandboxes":1,"billingStatus":"active"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.RawQuery; got != "" {
				t.Fatalf("query = %q, want empty", got)
			}
		},
	})

	if _, err := svc.Limits(context.Background(), account.LimitsRequest{}); err != nil {
		t.Fatalf("Limits: %v", err)
	}
}
