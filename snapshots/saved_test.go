package snapshots_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/snapshots"
)

func TestService_ListSaved_SendsFiltersAndReadsItems(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/saved-snapshots",
		Body: `{"ok":true,"type":"saved_snapshot.list","snapshots":[
			{"name":"base","sourceSandboxId":"bx_example","status":"ready","createdAt":"2026-11-01T00:00:00Z",
			 "snapshotId":"11111111-1111-4111-8111-111111111111","sizeBytes":1024}],
			"allowance":{"included":10,"used":1}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			q := r.URL.Query()
			if got := q.Get("name"); got != "base" {
				t.Fatalf("name = %q", got)
			}
			if got := q.Get("namePrefix"); got != "ba" {
				t.Fatalf("namePrefix = %q", got)
			}
			if got := q.Get("org"); got != "personal" {
				t.Fatalf("org = %q", got)
			}
		},
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	page, err := svc.ListSaved(context.Background(), snapshots.SavedListRequest{Name: "base", NamePrefix: "ba", Org: "personal"})
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "base" {
		t.Fatalf("page = %#v", page)
	}
	if page.HasMore || page.NextCursor != "" {
		t.Fatalf("page = %#v, want a single page", page)
	}
}

func TestService_ListSaved_MapsUnauthorized(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/saved-snapshots",
		Status: http.StatusUnauthorized,
		Body:   `{"ok":false,"type":"error","status":401,"code":"unauthorized","message":"bad key"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.ListSaved(context.Background(), snapshots.SavedListRequest{})
	if !core.IsCode(err, "unauthorized") {
		t.Fatalf("error = %v, want unauthorized", err)
	}
}

func TestService_Save_SendsBodyAndReturnsSavingSnapshot(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/saved-snapshots",
		Status: http.StatusAccepted,
		Body: `{"ok":true,"type":"saved_snapshot.saving","status":"saving","snapshot":
			{"name":"base","sourceSandboxId":"bx_example","status":"saving","createdAt":"2026-11-13T00:00:00Z"}}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		},
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	saved, err := svc.Save(context.Background(), sandboxID, "base")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if body["sandboxId"] != sandboxID || body["name"] != "base" {
		t.Fatalf("body = %#v", body)
	}
	if saved.Status != model.SavedSnapshotStatusSaving {
		t.Fatalf("status = %q", saved.Status)
	}
}

func TestService_Save_MapsErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		code   string
	}{
		{status: http.StatusBadRequest, code: "invalid_name"},
		{status: http.StatusUnauthorized, code: "unauthorized"},
		{status: http.StatusPaymentRequired, code: "credit_required"},
		{status: http.StatusNotFound, code: "not_found"},
		{status: http.StatusConflict, code: "save_in_progress"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodPost,
				Path:   "/saved-snapshots",
				Status: tt.status,
				Body:   fmt.Sprintf(`{"ok":false,"type":"error","status":%d,"code":%q,"message":"failed"}`, tt.status, tt.code),
			})
			svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			if _, err := svc.Save(context.Background(), sandboxID, "base"); !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestService_GetSaved_ReturnsSnapshot(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/saved-snapshots/base",
		Body: `{"ok":true,"type":"saved_snapshot.info","snapshot":
			{"name":"base","sourceSandboxId":"bx_example","status":"ready","createdAt":"2026-11-13T00:00:00Z",
			 "snapshotId":"11111111-1111-4111-8111-111111111111"}}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	saved, err := svc.GetSaved(context.Background(), "base")
	if err != nil {
		t.Fatalf("GetSaved: %v", err)
	}
	if saved.Status != model.SavedSnapshotStatusReady {
		t.Fatalf("status = %q", saved.Status)
	}
	if saved.SnapshotID == nil || *saved.SnapshotID != snapshotID {
		t.Fatalf("snapshot id = %v", saved.SnapshotID)
	}
}

func TestService_GetSaved_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/saved-snapshots/missing",
		Status: http.StatusNotFound,
		Body:   `{"ok":false,"type":"error","status":404,"code":"not_found","message":"no saved snapshot"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.GetSaved(context.Background(), "missing")
	if !core.IsCode(err, "not_found") {
		t.Fatalf("error = %v, want not_found", err)
	}
}

func TestService_Save_PollsUntilReady(t *testing.T) {
	t.Parallel()

	gets := 0
	d := &testutil.Doer{DoFunc: func(_ context.Context, req *core.Request, out any, _ ...core.RequestOption) error {
		switch req.Method {
		case http.MethodPost:
			*out.(*model.SavedSnapshotSavingResponse) = model.SavedSnapshotSavingResponse{
				Snapshot: model.SavedSnapshot{Name: "base", Status: model.SavedSnapshotStatusSaving},
			}
		case http.MethodGet:
			gets++
			status := model.SavedSnapshotStatusSaving
			if gets == 2 {
				status = model.SavedSnapshotStatusReady
			}
			*out.(*model.SavedSnapshotInfoResponse) = model.SavedSnapshotInfoResponse{
				Snapshot: model.SavedSnapshot{Name: "base", Status: status},
			}
		default:
			t.Fatalf("unexpected method %q", req.Method)
		}
		return nil
	}}
	svc := snapshots.New(d)

	saved, err := svc.Save(context.Background(), sandboxID, "base")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.Status != model.SavedSnapshotStatusSaving {
		t.Fatalf("initial status = %q", saved.Status)
	}

	for range 3 {
		saved, err = svc.GetSaved(context.Background(), "base")
		if err != nil {
			t.Fatalf("GetSaved: %v", err)
		}
		if saved.Status == model.SavedSnapshotStatusReady {
			return
		}
	}
	t.Fatal("saved snapshot never reached ready")
}

func TestService_DeleteSaved_RemovesByName(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/saved-snapshots/base",
		Body:   `{"ok":true,"type":"saved_snapshot.deleted","name":"base","status":"deleted"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	if err := svc.DeleteSaved(context.Background(), "base"); err != nil {
		t.Fatalf("DeleteSaved: %v", err)
	}
}

func TestService_DeleteSaved_MapsSaveInProgress(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/saved-snapshots/base",
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"error","status":409,"code":"save_in_progress","message":"still saving"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	err := svc.DeleteSaved(context.Background(), "base")
	if !core.IsCode(err, "save_in_progress") {
		t.Fatalf("error = %v, want save_in_progress", err)
	}
}

func TestService_SavedSnapshot_RequiresNames(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.Save(context.Background(), "", "base"); err == nil {
		t.Fatal("Save without sandbox id = nil, want error")
	}
	if _, err := svc.Save(context.Background(), sandboxID, ""); err == nil {
		t.Fatal("Save without name = nil, want error")
	}
	if _, err := svc.GetSaved(context.Background(), ""); err == nil {
		t.Fatal("GetSaved without name = nil, want error")
	}
	if err := svc.DeleteSaved(context.Background(), ""); err == nil {
		t.Fatal("DeleteSaved without name = nil, want error")
	}
}
