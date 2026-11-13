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

const (
	snapshotID = "11111111-1111-4111-8111-111111111111"
	sandboxID  = "bx_example"
)

func TestService_List_SendsQueryAndReadsPage(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots",
		Body: `{"ok":true,"type":"snapshot.list","snapshots":[
			{"id":"11111111-1111-4111-8111-111111111111","sandboxId":"bx_example","status":"completed",
			 "kind":"incremental","generation":3,"createdAt":"2026-06-24T06:24:00Z",
			 "completedAt":"2026-06-24T06:24:50Z","sizeBytes":18874368,"fileCount":6781}],
			"pageInfo":{"nextCursor":"cur_2","hasMore":true,"limit":1}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			q := r.URL.Query()
			if got := q.Get("limit"); got != "1" {
				t.Fatalf("limit = %q", got)
			}
			if got := q.Get("cursor"); got != "cur_1" {
				t.Fatalf("cursor = %q", got)
			}
			if got := q.Get("sort"); got != "asc" {
				t.Fatalf("sort = %q", got)
			}
		},
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.List(context.Background(), snapshots.ListRequest{Limit: 1, Cursor: "cur_1", Sort: "asc"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 || page.NextCursor != "cur_2" || !page.HasMore {
		t.Fatalf("page = %#v", page)
	}
	if page.Items[0].Status != model.SnapshotSummaryStatusCompleted {
		t.Fatalf("status = %q", page.Items[0].Status)
	}
	if page.Items[0].Generation == nil || *page.Items[0].Generation != 3 {
		t.Fatalf("generation = %v", page.Items[0].Generation)
	}
}

func TestService_List_EmptyPage(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots",
		Body:   `{"ok":true,"type":"snapshot.list","snapshots":[]}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.List(context.Background(), snapshots.ListRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 0 || page.HasMore || page.NextCursor != "" {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_List_MapsAPIError(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/snapshots",
		Status: http.StatusUnauthorized,
		Body:   `{"ok":false,"type":"error","status":401,"code":"unauthorized","message":"bad key","requestId":"req_1"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.List(context.Background(), snapshots.ListRequest{})
	if !core.IsCode(err, "unauthorized") {
		t.Fatalf("error = %v, want unauthorized", err)
	}
	var apiErr *core.APIError
	if !asAPIError(err, &apiErr) || apiErr.RequestID != "req_1" {
		t.Fatalf("request id not preserved: %#v", apiErr)
	}
}

func TestService_ListForSandbox_UsesSandboxPath(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/snapshots",
		Body:   `{"ok":true,"type":"snapshot.list","snapshots":[{"id":"11111111-1111-4111-8111-111111111111","sandboxId":"bx_example","status":"completed","createdAt":"2026-06-24T06:24:00Z"}]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Fatalf("limit = %q", got)
			}
		},
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.ListForSandbox(context.Background(), sandboxID, snapshots.ListRequest{Limit: 2})
	if err != nil {
		t.Fatalf("ListForSandbox: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].SandboxID != sandboxID {
		t.Fatalf("page = %#v", page)
	}
}

func TestService_ListForSandbox_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.ListForSandbox(context.Background(), "", snapshots.ListRequest{}); err == nil {
		t.Fatal("ListForSandbox = nil, want id error")
	}
}

func TestService_Latest_ReturnsSnapshot(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/snapshots/latest",
		Body:   `{"ok":true,"type":"snapshot.latest","snapshot":{"id":"11111111-1111-4111-8111-111111111111","sandboxId":"bx_example","status":"completed","createdAt":"2026-06-24T06:24:00Z"}}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Latest(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got == nil || got.ID.String() != snapshotID {
		t.Fatalf("latest = %#v", got)
	}
}

func TestService_Latest_NullSnapshot(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/snapshots/latest",
		Body:   `{"ok":true,"type":"snapshot.latest","snapshot":null}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	got, err := svc.Latest(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got != nil {
		t.Fatalf("latest = %#v, want nil", got)
	}
}

func TestService_Latest_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/sandboxes/bx_example/snapshots/latest",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no sandbox"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Latest(context.Background(), sandboxID)
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
}
