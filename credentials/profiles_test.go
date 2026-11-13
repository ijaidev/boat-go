package credentials_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_ListProfiles_ReturnsProfiles(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/credential-profiles",
		Body:   `{"ok": true, "type": "credential_profile.list", "profiles": [` + profileJSON + `]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Authorization"); got != "Bearer boat_test_key" {
				t.Fatal("Authorization header does not carry the test key")
			}
			if got := r.URL.Query().Get("org"); got != "" {
				t.Fatalf("org = %q, want empty", got)
			}
		},
	})

	profiles, err := svc.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if profile.ID != "prof_example" || profile.Name != "work" {
		t.Fatalf("profile = %#v", profile)
	}
	if profile.Owner.Kind != model.CredentialProfileOwnerKindPersonal || !profile.Owner.AiProxyEnabled {
		t.Fatalf("owner = %#v", profile.Owner)
	}
	if !profile.CanManage || len(profile.Connected) != 1 || profile.Connected[0] != model.OpenaiAPIKey {
		t.Fatalf("profile = %#v", profile)
	}
	if profile.Accounts == nil {
		t.Fatal("accounts = nil, want accounts")
	}
	accounts := (*profile.Accounts)["openai-api-key"]
	if len(accounts) != 1 || !accounts[0].Active {
		t.Fatalf("accounts = %#v", accounts)
	}
	if len(profile.CustomApis) != 1 || profile.CustomApis[0] != "STRIPE_KEY" {
		t.Fatalf("customApis = %v", profile.CustomApis)
	}
}

func TestService_CreateProfile_SendsBody(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/credential-profiles",
		Status: http.StatusCreated,
		Body:   `{"ok": true, "type": "credential_profile.created", "success": true, "profile": ` + profileJSON + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var body struct {
				Name     string `json:"name"`
				Org      string `json:"org"`
				CopyFrom string `json:"copyFrom"`
			}
			decodeBody(t, r, &body)
			if body.Name != "work" {
				t.Fatalf("name = %q", body.Name)
			}
			if body.Org != "team_example" {
				t.Fatalf("org = %q", body.Org)
			}
			if body.CopyFrom != "default" {
				t.Fatalf("copyFrom = %q", body.CopyFrom)
			}
		},
	})

	profile, err := svc.CreateProfile(context.Background(), credentials.CreateProfileRequest{
		Name:     "work",
		Org:      "team_example",
		CopyFrom: "default",
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if profile.ID != "prof_example" || profile.Name != "work" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestService_CreateProfile_OmitsOptionalFields(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/credential-profiles",
		Status: http.StatusCreated,
		Body:   `{"ok": true, "type": "credential_profile.created", "success": true, "profile": ` + profileJSON + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var body map[string]any
			decodeBody(t, r, &body)
			if _, ok := body["org"]; ok {
				t.Fatal("org was sent, want omitted")
			}
			if _, ok := body["copyFrom"]; ok {
				t.Fatal("copyFrom was sent, want omitted")
			}
		},
	})

	if _, err := svc.CreateProfile(context.Background(), credentials.CreateProfileRequest{Name: "work"}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
}

func TestService_UpdateProfile_SendsBody(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/credential-profiles/work",
		Body:   `{"ok": true, "type": "credential_profile.updated", "success": true, "profile": ` + profileJSON + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var body struct {
				Name      string                      `json:"name"`
				IsDefault *bool                       `json:"isDefault"`
				Access    *model.ResourceAccessUpdate `json:"access"`
			}
			decodeBody(t, r, &body)
			if body.Name != "renamed" {
				t.Fatalf("name = %q", body.Name)
			}
			if body.IsDefault == nil || !*body.IsDefault {
				t.Fatalf("isDefault = %v, want true", body.IsDefault)
			}
			if body.Access == nil || !body.Access.Restricted {
				t.Fatalf("access = %#v", body.Access)
			}
			if body.Access.Members == nil || len(*body.Access.Members) != 1 || (*body.Access.Members)[0] != "member@example.com" {
				t.Fatalf("access members = %v", body.Access.Members)
			}
		},
	})

	profile, err := svc.UpdateProfile(context.Background(), "work", credentials.UpdateProfileRequest{
		Name:      "renamed",
		IsDefault: core.Ptr(true),
		Access: &model.ResourceAccessUpdate{
			Restricted: true,
			Members:    &[]string{"member@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if profile.ID != "prof_example" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestService_DeleteProfile_Deletes(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/credential-profiles/default",
		Body:   `{"ok": true, "type": "credential_profile.deleted", "success": true}`,
	})

	if err := svc.DeleteProfile(context.Background(), "default"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
}
