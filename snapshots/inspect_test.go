package snapshots_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/snapshots"
)

func asAPIError(err error, target **core.APIError) bool {
	return errors.As(err, target)
}

func TestService_Tree_ReturnsEntries(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/" + snapshotID + "/tree",
		Body: `{"ok":true,"type":"snapshot.tree","snapshotId":"` + snapshotID + `","sandboxId":"bx_example",
			"generation":3,"treeAvailable":true,"truncated":false,"fileCount":2,"totalSizeBytes":1024,
			"entries":[{"path":"src","kind":"dir"},{"path":"src/main.ts","kind":"file","size":1024}]}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	tree, err := svc.Tree(context.Background(), snapshotID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if !tree.TreeAvailable || tree.Truncated {
		t.Fatalf("tree flags = %#v", tree)
	}
	if len(tree.Entries) != 2 {
		t.Fatalf("entries = %#v", tree.Entries)
	}
	if tree.Entries[1].Kind != model.File {
		t.Fatalf("entry kind = %q", tree.Entries[1].Kind)
	}
	if tree.Entries[1].Size == nil || *tree.Entries[1].Size != 1024 {
		t.Fatalf("entry size = %v", tree.Entries[1].Size)
	}
}

func TestService_Tree_LegacySnapshotUnavailable(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/" + snapshotID + "/tree",
		Body: `{"ok":true,"type":"snapshot.tree","snapshotId":"` + snapshotID + `","sandboxId":"bx_example",
			"generation":0,"treeAvailable":false,"truncated":false,"fileCount":0,"totalSizeBytes":0,
			"entries":[],"reason":"legacy_snapshot"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	tree, err := svc.Tree(context.Background(), snapshotID)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if tree.TreeAvailable {
		t.Fatal("TreeAvailable = true, want false")
	}
	if tree.Reason == nil || *tree.Reason != "legacy_snapshot" {
		t.Fatalf("reason = %v", tree.Reason)
	}
}

func TestService_Tree_RequiresSnapshotID(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.Tree(context.Background(), ""); err == nil {
		t.Fatal("Tree = nil, want id error")
	}
}

func TestService_Tree_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots/" + snapshotID + "/tree",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no snapshot"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Tree(context.Background(), snapshotID)
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
}
