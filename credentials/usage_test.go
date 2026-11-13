package credentials_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_Usage_SendsFilters(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ai-usage",
		Body:   usageJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			query := r.URL.Query()
			if got := query.Get("days"); got != "7" {
				t.Fatalf("days = %q", got)
			}
			if got := query.Get("profile"); got != "work" {
				t.Fatalf("profile = %q", got)
			}
			if got := query.Get("org"); got != "team_example" {
				t.Fatalf("org = %q", got)
			}
		},
	})

	usage, err := svc.Usage(context.Background(), credentials.UsageRequest{
		Days:    7,
		Profile: "work",
		Org:     "team_example",
	})
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	if usage.Since.Format("2006-01-02") != "2026-10-01" {
		t.Fatalf("since = %v", usage.Since)
	}
	if usage.AccountID == nil || *usage.AccountID != "usr_example" {
		t.Fatalf("accountId = %v", usage.AccountID)
	}
	if usage.ProfileID != nil {
		t.Fatalf("profileId = %v, want nil", usage.ProfileID)
	}
	if len(usage.Sandboxes) != 1 || usage.Sandboxes[0].SandboxID == nil {
		t.Fatalf("sandboxes = %#v", usage.Sandboxes)
	}
	if len(usage.Usage) != 1 || usage.Usage[0].Credential != "openai-api-key" {
		t.Fatalf("usage = %#v", usage.Usage)
	}
	if usage.Totals.Requests != 2 || usage.Totals.InputTokens != 10 {
		t.Fatalf("totals = %#v", usage.Totals)
	}
	if usage.Dollars != 0.01 {
		t.Fatalf("dollars = %v", usage.Dollars)
	}
}

func TestService_Usage_OmitsFiltersByDefault(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ai-usage",
		Body:   usageJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			for _, key := range []string{"days", "profile", "org"} {
				if r.URL.Query().Has(key) {
					t.Fatalf("%s was sent, want omitted", key)
				}
			}
		},
	})

	if _, err := svc.Usage(context.Background(), credentials.UsageRequest{}); err != nil {
		t.Fatalf("Usage: %v", err)
	}
}

func TestService_SandboxUsage_SendsDays(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/ai-usage",
		Body:   sandboxUsageJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("days"); got != "30" {
				t.Fatalf("days = %q", got)
			}
		},
	})

	usage, err := svc.SandboxUsage(context.Background(), "bx_example", credentials.SandboxUsageRequest{Days: 30})
	if err != nil {
		t.Fatalf("SandboxUsage: %v", err)
	}
	if usage.SandboxID != "bx_example" || usage.Blocked || !usage.Proxied {
		t.Fatalf("usage = %#v", usage)
	}
	if usage.CredentialProfileID != nil {
		t.Fatalf("credentialProfileId = %v, want nil", usage.CredentialProfileID)
	}
	if usage.Since.Format("2006-01-02") != "2026-09-11" {
		t.Fatalf("since = %v", usage.Since)
	}
}
