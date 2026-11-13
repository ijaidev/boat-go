package environments_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/environments"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_AddRepo_SendsRepositoryAndOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/environments/" + envID + "/repos",
		Body:   `{"success":true,"versionId":"` + versionID + `","versionNumber":5}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireBody(t, r, map[string]any{
				"repositoryId": "repo_example",
				"baseBranch":   "dev",
			})
		},
	})

	got, err := svc.AddRepo(context.Background(), envID, environments.AddRepoRequest{
		RepositoryID: "repo_example",
		BaseBranch:   core.Ptr("dev"),
		Org:          "team_example",
	})
	if err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	if got.VersionNumber == nil || *got.VersionNumber != 5 {
		t.Fatalf("version number = %v", got.VersionNumber)
	}
}

func TestService_RemoveRepo_SendsPathAndOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/environments/" + envID + "/repos/repo_example",
		Body:   `{"success":true,"versionId":"` + versionID + `","versionNumber":6}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireEmptyBody(t, r)
		},
	})

	got, err := svc.RemoveRepo(context.Background(), envID, "repo_example", environments.WithOrg("team_example"))
	if err != nil {
		t.Fatalf("RemoveRepo: %v", err)
	}
	if got.VersionNumber == nil || *got.VersionNumber != 6 {
		t.Fatalf("version number = %v", got.VersionNumber)
	}
}

func TestService_ListRepos_SendsFilters(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/repos",
		Body: `{"ok":true,"type":"repos.list","environmentId":"env_123","installations":[
			{"type":"Organization","accountLogin":"example","repositories":[
				{"databaseId":"repo_example","name":"web","fullName":"example/web","private":true}]}
		],"selectedRepositories":[
			{"databaseId":"repo_example","name":"web","fullName":"example/web","baseBranch":"dev","setupScript":"npm ci","setupBlocking":true}
		],"pageInfo":{"hasMore":true,"limit":25,"nextCursor":"cur_2"}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "sync", "true")
			requireQuery(t, r, "limit", "25")
			requireQuery(t, r, "cursor", "cur_1")
			requireQuery(t, r, "sort", "asc")
			requireQuery(t, r, "q", "web")
			requireQuery(t, r, "selected", "false")
		},
	})

	got, err := svc.ListRepos(context.Background(), environments.ListReposRequest{
		Sync:     core.Ptr(true),
		Limit:    25,
		Cursor:   "cur_1",
		Sort:     "asc",
		Query:    "web",
		Selected: core.Ptr(false),
	})
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if got.EnvironmentID != "env_123" || len(got.Installations) != 1 {
		t.Fatalf("repos = %#v", got)
	}
	if len(got.SelectedRepositories) != 1 || *got.SelectedRepositories[0].BaseBranch != "dev" {
		t.Fatalf("selected = %#v", got.SelectedRepositories)
	}
	if got.PageInfo == nil || !got.PageInfo.HasMore || got.PageInfo.NextCursor == nil || *got.PageInfo.NextCursor != "cur_2" {
		t.Fatalf("page info = %#v", got.PageInfo)
	}
}

func TestService_ListRepos_OmitsUnsetFilters(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/repos",
		Body:   `{"ok":true,"type":"repos.list","environmentId":"env_123","installations":[],"selectedRepositories":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.RawQuery; got != "" {
				t.Fatalf("query = %q, want empty", got)
			}
		},
	})

	if _, err := svc.ListRepos(context.Background(), environments.ListReposRequest{}); err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
}

func TestService_SelectRepo_SendsBody(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/repos",
		Body: `{"ok":true,"type":"repos.updated","success":true,"environmentId":"env_123","selectedRepositories":[
			{"databaseId":"repo_example","name":"web","fullName":"example/web","baseBranch":"dev"}
		]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireBody(t, r, map[string]any{
				"repositoryId": "repo_example",
				"baseBranch":   "dev",
			})
		},
	})

	got, err := svc.SelectRepo(context.Background(), environments.SelectRepoRequest{
		RepositoryID: "repo_example",
		BaseBranch:   core.Ptr("dev"),
	})
	if err != nil {
		t.Fatalf("SelectRepo: %v", err)
	}
	if !got.Success || got.EnvironmentID != "env_123" || len(got.SelectedRepositories) != 1 {
		t.Fatalf("selection = %#v", got)
	}
}

func TestService_Repos_RequireIDs(t *testing.T) {
	t.Parallel()

	svc := environments.New(&testutil.Doer{})
	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "add missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.AddRepo(ctx, "", environments.AddRepoRequest{RepositoryID: "repo_example"})
				return err
			},
		},
		{
			name: "add missing repository",
			call: func(ctx context.Context) error {
				_, err := svc.AddRepo(ctx, envID, environments.AddRepoRequest{})
				return err
			},
		},
		{
			name: "remove missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.RemoveRepo(ctx, "", "repo_example")
				return err
			},
		},
		{
			name: "remove missing repository",
			call: func(ctx context.Context) error {
				_, err := svc.RemoveRepo(ctx, envID, "")
				return err
			},
		},
		{
			name: "select missing repository",
			call: func(ctx context.Context) error {
				_, err := svc.SelectRepo(ctx, environments.SelectRepoRequest{})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.call(context.Background()); err == nil {
				t.Fatal("call = nil, want error")
			}
		})
	}
}
