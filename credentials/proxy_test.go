package credentials_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_GetProxy_SendsOrgQuery(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ai-proxy",
		Body:   proxyJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("org"); got != "team_example" {
				t.Fatalf("org = %q", got)
			}
		},
	})

	settings, err := svc.GetProxy(context.Background(), credentials.GetProxyRequest{Org: "team_example"})
	if err != nil {
		t.Fatalf("GetProxy: %v", err)
	}
	if settings.AccountID != "usr_example" || !settings.Enabled || settings.Org {
		t.Fatalf("settings = %#v", settings)
	}
	if settings.Pricing.DailyDollars != 1 || settings.Pricing.DollarsPer1000Requests != 1 {
		t.Fatalf("pricing = %#v", settings.Pricing)
	}
	if settings.ProxyURL != "https://proxy.example.com" {
		t.Fatalf("proxyURL = %q", settings.ProxyURL)
	}
	if settings.EnabledAt == nil {
		t.Fatal("enabledAt = nil, want a time")
	}
}

func TestService_GetProxy_OmitsOrgByDefault(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/ai-proxy",
		Body:   proxyJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if r.URL.Query().Has("org") {
				t.Fatalf("org = %q, want omitted", r.URL.Query().Get("org"))
			}
		},
	})

	if _, err := svc.GetProxy(context.Background(), credentials.GetProxyRequest{}); err != nil {
		t.Fatalf("GetProxy: %v", err)
	}
}

func TestService_SetProxy_SendsEnabledAndOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/ai-proxy",
		Body:   proxyJSON,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("org"); got != "team_example" {
				t.Fatalf("org = %q", got)
			}
			var body map[string]any
			decodeBody(t, r, &body)
			if enabled, ok := body["enabled"].(bool); !ok || enabled {
				t.Fatalf("enabled = %v, want false", body["enabled"])
			}
			if _, ok := body["org"]; ok {
				t.Fatal("org was sent in the body, want query only")
			}
		},
	})

	settings, err := svc.SetProxy(context.Background(), credentials.SetProxyRequest{
		Enabled: false,
		Org:     "team_example",
	})
	if err != nil {
		t.Fatalf("SetProxy: %v", err)
	}
	if settings.AccountID != "usr_example" {
		t.Fatalf("settings = %#v", settings)
	}
}
