package snapshots_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/snapshots"
)

const shareJSON = `{"id":"shr_example","codePrefix":"bsh_example","status":"active",
	"createdAt":"2026-11-13T00:00:00Z","lastUsedAt":null,"revokedAt":null,"useCount":0}`

func TestService_CreateShare_ReturnsSecretCode(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/saved-snapshots/base/shares",
		Status: http.StatusCreated,
		Body: `{"ok":true,"type":"saved_snapshot.share_created","name":"base",
			"share":` + shareJSON + `,"shareCode":"bsh_test_secret_code"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	created, err := svc.CreateShare(context.Background(), "base")
	if err != nil {
		t.Fatalf("CreateShare: %v", err)
	}
	if created.ShareCode != "bsh_test_secret_code" {
		t.Fatalf("share code = %q", created.ShareCode)
	}
	if created.Share.ID != "shr_example" || created.Name != "base" {
		t.Fatalf("response = %#v", created)
	}
}

func TestService_CreateShare_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/saved-snapshots/missing/shares",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no saved snapshot"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.CreateShare(context.Background(), "missing")
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
}

func TestService_ListShares_ReturnsShares(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/saved-snapshots/base/shares",
		Body:   `{"ok":true,"type":"saved_snapshot.share_list","name":"base","shares":[` + shareJSON + `]}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.ListShares(context.Background(), "base")
	if err != nil {
		t.Fatalf("ListShares: %v", err)
	}
	if len(got) != 1 || got[0].Status != model.NamedSnapshotShareStatusActive {
		t.Fatalf("shares = %#v", got)
	}
}

func TestService_RevokeShare_SendsShareID(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/saved-snapshots/base/shares/shr_example",
		Body: `{"ok":true,"type":"saved_snapshot.share_revoked","name":"base","share":
			{"id":"shr_example","codePrefix":"bsh_example","status":"revoked",
			 "createdAt":"2026-11-13T00:00:00Z","lastUsedAt":null,"revokedAt":"2026-11-13T01:00:00Z","useCount":0}}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	revoked, err := svc.RevokeShare(context.Background(), "base", "shr_example")
	if err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}
	if revoked.Share.Status != model.NamedSnapshotShareStatusRevoked {
		t.Fatalf("status = %q", revoked.Share.Status)
	}
}

func TestService_Shares_RequireNames(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.CreateShare(context.Background(), ""); err == nil {
		t.Fatal("CreateShare without name = nil, want error")
	}
	if _, err := svc.ListShares(context.Background(), ""); err == nil {
		t.Fatal("ListShares without name = nil, want error")
	}
	if _, err := svc.RevokeShare(context.Background(), "", "shr_example"); err == nil {
		t.Fatal("RevokeShare without name = nil, want error")
	}
	if _, err := svc.RevokeShare(context.Background(), "base", ""); err == nil {
		t.Fatal("RevokeShare without share id = nil, want error")
	}
}
