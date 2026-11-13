package ci_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/ci"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_PublicRepos_SendsSettings(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/ci/public",
		Body: `{"ok":true,"type":"ci.public","allowPublicRepos":true,
			"publicRepoUsers":[{"login":"alice","avatarUrl":"https://avatars.example/alice"},
				{"login":"bob","avatarUrl":null}]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body := decodeBody(t, r)
			if body["account"] != "acme" {
				t.Fatalf("account = %v", body["account"])
			}
			if body["allow"] != false {
				t.Fatalf("allow = %v, want false", body["allow"])
			}
			if body["dryRun"] != true {
				t.Fatalf("dryRun = %v, want true", body["dryRun"])
			}
			if body["org"] != "acme-org" {
				t.Fatalf("org = %v", body["org"])
			}
			users, ok := body["users"].([]any)
			if !ok || len(users) != 2 || users[0] != "alice" || users[1] != "bob" {
				t.Fatalf("users = %v", body["users"])
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.PublicRepos(context.Background(), ci.PublicReposRequest{
		Account: "acme",
		Allow:   core.Ptr(false),
		Users:   &[]string{"alice", "bob"},
		DryRun:  core.Ptr(true),
		Org:     "acme-org",
	})
	if err != nil {
		t.Fatalf("PublicRepos: %v", err)
	}
	if !got.Ok || got.Type != model.CiPublic {
		t.Fatalf("envelope = %#v", got)
	}
	if !got.AllowPublicRepos {
		t.Fatal("allowPublicRepos = false, want true")
	}
	if len(got.PublicRepoUsers) != 2 {
		t.Fatalf("users = %#v", got.PublicRepoUsers)
	}
	if got.PublicRepoUsers[0].Login == nil || *got.PublicRepoUsers[0].Login != "alice" {
		t.Fatalf("first login = %v", got.PublicRepoUsers[0].Login)
	}
	if got.PublicRepoUsers[1].AvatarURL != nil {
		t.Fatalf("second avatar = %v, want nil", got.PublicRepoUsers[1].AvatarURL)
	}
}

func TestService_PublicRepos_OmitsUnsetOptionalFields(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/ci/public",
		Body:   `{"ok":true,"type":"ci.public","allowPublicRepos":false,"publicRepoUsers":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body := decodeBody(t, r)
			if len(body) != 1 || body["account"] != "acme" {
				t.Fatalf("body = %v, want only account", body)
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if _, err := svc.PublicRepos(context.Background(), ci.PublicReposRequest{Account: "acme"}); err != nil {
		t.Fatalf("PublicRepos: %v", err)
	}
}

func TestService_PublicRepos_EmptyUsersClearsList(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/ci/public",
		Body:   `{"ok":true,"type":"ci.public","allowPublicRepos":true,"publicRepoUsers":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body := decodeBody(t, r)
			users, ok := body["users"].([]any)
			if !ok || len(users) != 0 {
				t.Fatalf("users = %v, want an empty list", body["users"])
			}
		},
	})
	svc := ci.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	empty := []string{}
	if _, err := svc.PublicRepos(context.Background(), ci.PublicReposRequest{
		Account: "acme",
		Users:   &empty,
	}); err != nil {
		t.Fatalf("PublicRepos: %v", err)
	}
}

func TestService_PublicRepos_RequiresAccount(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	if _, err := ci.New(d).PublicRepos(context.Background(), ci.PublicReposRequest{}); err == nil {
		t.Fatal("PublicRepos = nil, want account error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(d.Calls))
	}
}
