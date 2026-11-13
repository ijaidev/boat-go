package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Orgs_ReturnsWallets(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/orgs",
		Body: `{
			"ok": true,
			"type": "org.list",
			"activeWalletChosen": true,
			"orgs": [
				{
					"id": "acct_example",
					"name": "Personal",
					"type": "personal",
					"role": "owner",
					"memberCount": 1,
					"active": false
				},
				{
					"id": "team_example",
					"name": "example-org",
					"type": "org",
					"role": "member",
					"memberCount": 3,
					"subscriptionStatus": "active",
					"active": true
				}
			]
		}`,
	})

	got, err := svc.Orgs(context.Background())
	if err != nil {
		t.Fatalf("Orgs: %v", err)
	}
	if !got.ActiveWalletChosen {
		t.Fatal("activeWalletChosen = false, want true")
	}
	if len(got.Orgs) != 2 {
		t.Fatalf("orgs = %d, want 2", len(got.Orgs))
	}
	personal, team := got.Orgs[0], got.Orgs[1]
	if personal.Type != model.OrgTypePersonal || personal.Role != model.OrgRoleOwner {
		t.Fatalf("personal type/role = %q/%q", personal.Type, personal.Role)
	}
	if personal.SubscriptionStatus != nil {
		t.Fatal("personal subscription status = non-nil, want absent")
	}
	if personal.Active {
		t.Fatal("personal active = true, want false")
	}
	if team.ID != "team_example" || team.Name != "example-org" {
		t.Fatal("organization id/name mismatch")
	}
	if team.Type != model.OrgTypeOrg || team.Role != model.OrgRoleMember || team.MemberCount != 3 {
		t.Fatalf("organization = %#v", team)
	}
	if team.SubscriptionStatus == nil || *team.SubscriptionStatus != "active" {
		t.Fatal("subscription status mismatch")
	}
	if !team.Active {
		t.Fatal("organization active = false, want true")
	}
}

func TestService_SetActiveOrg_SendsOrg(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/orgs/active",
		Body: `{
			"ok": true,
			"type": "org.active_updated",
			"active": {"id": "team_example", "name": "acme", "type": "org"}
		}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			body = readBody(t, r)
		},
	})

	got, err := svc.SetActiveOrg(context.Background(), "acme")
	if err != nil {
		t.Fatalf("SetActiveOrg: %v", err)
	}
	if len(body) != 1 || body["org"] != "acme" {
		t.Fatalf("body = %v, want only org=acme", body)
	}
	if got.Active.ID != "team_example" || got.Active.Name != "acme" {
		t.Fatal("active organization mismatch")
	}
	if got.Active.Type != model.ActiveOrgResponseActiveTypeOrg {
		t.Fatalf("active type = %q, want org", got.Active.Type)
	}
}

func TestService_SetActiveOrg_SendsNullForPersonal(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/orgs/active",
		Body: `{
			"ok": true,
			"type": "org.active_updated",
			"active": {"id": "acct_example", "name": "Personal", "type": "personal"}
		}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body = readBody(t, r)
		},
	})

	got, err := svc.SetActiveOrg(context.Background(), "")
	if err != nil {
		t.Fatalf("SetActiveOrg: %v", err)
	}
	value, ok := body["org"]
	if !ok || value != nil {
		t.Fatalf("body = %v, want an explicit null org", body)
	}
	if got.Active.Type != model.ActiveOrgResponseActiveTypePersonal {
		t.Fatalf("active type = %q, want personal", got.Active.Type)
	}
}

func TestService_SetActiveOrg_MapsAmbiguousName(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/orgs/active",
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"account.error","status":409,"code":"ambiguous_org","message":"pass the id"}`,
	})

	_, err := svc.SetActiveOrg(context.Background(), "acme")
	if !core.IsCode(err, "ambiguous_org") {
		t.Fatalf("error = %v, want ambiguous_org", err)
	}
}
