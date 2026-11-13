package environments_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/environments"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_List_SendsOrgAndReturnsEnvironments(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/environments",
		Body:   `{"ok":true,"type":"environments.list","environments":[` + environmentJSON() + `]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			if got := r.Header.Get("Authorization"); got != "Bearer boat_test_key" {
				t.Fatalf("Authorization = %q", got)
			}
		},
	})

	got, err := svc.List(context.Background(), environments.ListRequest{Org: "team_example"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("environments = %d, want 1", len(got))
	}
	env := got[0]
	if env.Name != "base" || !env.IsDefault || env.LatestVersionNumber == nil || *env.LatestVersionNumber != 2 {
		t.Fatalf("environment = %#v", env)
	}
	if len(env.Versions) != 1 || env.Versions[0].VersionNumber != 2 || env.Versions[0].SandboxCount != 3 {
		t.Fatalf("versions = %#v", env.Versions)
	}
}

func TestService_List_OmitsEmptyOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/environments",
		Body:   `{"ok":true,"type":"environments.list","environments":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireNoQuery(t, r, "org")
		},
	})

	got, err := svc.List(context.Background(), environments.ListRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("environments = %d, want 0", len(got))
	}
}

func TestService_Create_SendsNameAndOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/environments",
		Body:   `{"ok":true,"type":"environments.list","environments":[` + environmentJSON() + `]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireBody(t, r, map[string]any{"name": "customer-demos"})
		},
	})

	got, err := svc.Create(context.Background(), environments.CreateRequest{
		Name: "customer-demos",
		Org:  "team_example",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(got) != 1 || got[0].Name != "base" {
		t.Fatalf("environments = %#v", got)
	}
}

func TestService_Update_SendsFullVersionBody(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/environments/" + envID,
		Body:   `{"success":true,"environment":` + environmentJSON() + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireBody(t, r, map[string]any{
				"name":                   "customer-demos",
				"isDefault":              true,
				"safeForThirdParties":    false,
				"passGithub":             true,
				"passSecrets":            false,
				"passSandboxCredentials": true,
				"passAgentsCredentials":  false,
				"credentialProfileId":    nil,
				"proxyAiCredentials":     true,
				"access": map[string]any{
					"restricted": true,
					"members":    []any{"a@example.com"},
				},
				"envContents": "TOKEN=test-secret\n",
				"secretFiles": []any{
					map[string]any{"path": ".env", "contents": "test-secret"},
				},
				"repositories": []any{
					map[string]any{
						"repositoryId":  "repo_example",
						"baseBranch":    "dev",
						"setupScript":   "npm ci",
						"setupBlocking": true,
					},
				},
			})
		},
	})

	got, err := svc.Update(context.Background(), envID, environments.UpdateRequest{
		Name:                   core.Ptr("customer-demos"),
		IsDefault:              core.Ptr(true),
		SafeForThirdParties:    core.Ptr(false),
		PassGithub:             core.Ptr(true),
		PassSecrets:            core.Ptr(false),
		PassSandboxCredentials: core.Ptr(true),
		PassAgentsCredentials:  core.Ptr(false),
		CredentialProfileID:    core.Null[string](),
		ProxyAiCredentials:     core.Ptr(true),
		Access: &model.ResourceAccessUpdate{
			Restricted: true,
			Members:    &[]string{"a@example.com"},
		},
		EnvContents: core.Ptr("TOKEN=test-secret\n"),
		SecretFiles: &[]model.SecretFile{
			{Path: ".env", Contents: "test-secret"},
		},
		Repositories: &[]environments.RepositorySelection{
			{
				RepositoryID:  "repo_example",
				BaseBranch:    core.Ptr("dev"),
				SetupScript:   core.Ptr("npm ci"),
				SetupBlocking: core.Ptr(true),
			},
		},
		Org: "team_example",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got == nil || got.ID.String() != envID {
		t.Fatalf("environment = %#v", got)
	}
}

func TestService_Update_OmitsUnsetFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  environments.UpdateRequest
	}{
		{name: "empty request", req: environments.UpdateRequest{}},
		{name: "nil credential profile", req: environments.UpdateRequest{CredentialProfileID: core.Null[string]()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t, testutil.Route{
				Method: http.MethodPut,
				Path:   "/environments/" + envID,
				Body:   `{"success":true,"environment":` + environmentJSON() + `}`,
				Assert: func(t *testing.T, r *http.Request) {
					t.Helper()
					body := readJSONBody(t, r)
					if tt.req.CredentialProfileID == nil {
						if len(body) != 0 {
							t.Fatalf("body = %#v, want omitted fields", body)
						}
						return
					}
					if len(body) != 1 {
						t.Fatalf("body = %#v, want only credentialProfileId", body)
					}
					value, ok := body["credentialProfileId"]
					if !ok || value != nil {
						t.Fatalf("credentialProfileId = %#v, want null", value)
					}
				},
			})

			if _, err := svc.Update(context.Background(), envID, tt.req); err != nil {
				t.Fatalf("Update: %v", err)
			}
		})
	}
}

func TestService_Delete_SendsOrgAndReturnsEnvironment(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/environments/" + envID,
		Body:   `{"success":true,"environment":` + environmentJSON() + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireEmptyBody(t, r)
		},
	})

	got, err := svc.Delete(context.Background(), envID, environments.WithOrg("team_example"))
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got == nil || got.Name != "base" {
		t.Fatalf("environment = %#v", got)
	}
}

func TestService_Delete_AllowsNullEnvironment(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/environments/" + envID,
		Body:   `{"success":true,"environment":null}`,
	})

	got, err := svc.Delete(context.Background(), envID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got != nil {
		t.Fatalf("environment = %#v, want nil", got)
	}
}

func TestService_Upgrade_SendsAgentIDsAndOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/environments/" + envID + "/upgrade",
		Body:   `{"success":true,"upgraded":2,"upgradedLive":1,"upgradedStopped":1,"failed":0,"latestVersionId":"` + versionID + `"}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireBody(t, r, map[string]any{"agentIds": []any{"bx_1", "bx_2"}})
		},
	})

	got, err := svc.Upgrade(context.Background(), envID, environments.UpgradeRequest{
		AgentIDs: []string{"bx_1", "bx_2"},
		Org:      "team_example",
	})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if got.Upgraded != 2 || got.Failed != 0 || got.UpgradedLive == nil || *got.UpgradedLive != 1 {
		t.Fatalf("upgrade = %#v", got)
	}
	if got.LatestVersionID == nil || *got.LatestVersionID != versionID {
		t.Fatalf("latest version = %v", got.LatestVersionID)
	}
}

func TestService_Upgrade_AllowsEmptySelector(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/environments/" + envID + "/upgrade",
		Body:   `{"success":true,"upgraded":0,"failed":0}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireBody(t, r, map[string]any{})
		},
	})

	if _, err := svc.Upgrade(context.Background(), envID, environments.UpgradeRequest{}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
}
