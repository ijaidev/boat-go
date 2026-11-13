package credentials_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/credentials"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_SetCustomAPI_SendsValueAndBaseURL(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/credential-profiles/work/custom-apis/STRIPE_KEY",
		Body:   `{"ok": true, "type": "credential_profile.updated", "success": true, "profile": ` + profileJSON + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			var body struct {
				Value   string `json:"value"`
				BaseURL string `json:"baseUrl"`
			}
			decodeBody(t, r, &body)
			if body.Value != "test_custom_api_value" {
				t.Fatal("custom API value was not forwarded")
			}
			if body.BaseURL != "https://api.example.com/v1" {
				t.Fatal("custom API base URL was not forwarded")
			}
		},
	})

	profile, err := svc.SetCustomAPI(context.Background(), "work", "STRIPE_KEY", credentials.SetCustomAPIRequest{
		Value:   "test_custom_api_value",
		BaseURL: "https://api.example.com/v1",
	})
	if err != nil {
		t.Fatalf("SetCustomAPI: %v", err)
	}
	if profile.ID != "prof_example" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestService_DeleteCustomAPI_ReturnsUpdatedProfile(t *testing.T) {
	t.Parallel()

	withoutCustomAPI := `{
		"id": "prof_example",
		"name": "work",
		"isDefault": false,
		"owner": {"id": "usr_example", "kind": "personal", "name": "example", "aiProxyEnabled": true},
		"canManage": true,
		"connected": [],
		"customApis": [],
		"createdAt": "2026-10-01T12:00:00Z",
		"updatedAt": "2026-10-09T08:00:00Z"
	}`

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/credential-profiles/work/custom-apis/STRIPE_KEY",
		Body:   `{"ok": true, "type": "credential_profile.updated", "success": true, "profile": ` + withoutCustomAPI + `}`,
	})

	profile, err := svc.DeleteCustomAPI(context.Background(), "work", "STRIPE_KEY")
	if err != nil {
		t.Fatalf("DeleteCustomAPI: %v", err)
	}
	if len(profile.CustomApis) != 0 {
		t.Fatalf("customApis = %v, want empty", profile.CustomApis)
	}
}
