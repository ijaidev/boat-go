package snapshots_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/snapshots"
)

const deletionOperation = `{"id":"del_1","kind":"snapshot","reason":"manual","status":"pending","targetId":"` + snapshotID + `","attemptCount":0,"requestedAt":"2026-11-13T00:00:00Z","completedAt":null}`

func TestService_Delete_SendsSnapshotConfirmationHeader(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/snapshots/" + snapshotID,
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"snapshot.deleting","operation":` + deletionOperation + `}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("X-Ascii-Confirm-Delete"); got != snapshotID {
				t.Fatalf("confirm header = %q, want %q", got, snapshotID)
			}
		},
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	op, err := svc.Delete(context.Background(), snapshotID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if op.ID != "del_1" || op.Status != model.DeletionOperationStatusPending {
		t.Fatalf("operation = %#v", op)
	}
}

func TestService_Delete_MapsConflict(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/snapshots/" + snapshotID,
		Status: http.StatusConflict,
		Body:   `{"ok":false,"type":"error","status":409,"code":"confirmation_required","message":"missing confirmation header"}`,
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	_, err := svc.Delete(context.Background(), snapshotID)
	if !core.IsCode(err, "confirmation_required") {
		t.Fatalf("error = %v, want confirmation_required", err)
	}
}

func TestService_Delete_MapsErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		code   string
	}{
		{status: http.StatusForbidden, code: "forbidden"},
		{status: http.StatusNotFound, code: "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			srv := testutil.NewServer(t, testutil.Route{
				Method: http.MethodDelete,
				Path:   "/snapshots/" + snapshotID,
				Status: tt.status,
				Body:   fmt.Sprintf(`{"ok":false,"type":"error","status":%d,"code":%q,"message":"failed"}`, tt.status, tt.code),
			})
			svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

			if _, err := svc.Delete(context.Background(), snapshotID); !core.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestService_Delete_RequiresSnapshotID(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.Delete(context.Background(), ""); err == nil {
		t.Fatal("Delete = nil, want id error")
	}
}

func TestService_DeleteForSandbox_SendsSandboxConfirmationHeader(t *testing.T) {
	t.Parallel()

	srv := testutil.NewServer(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/sandboxes/bx_example/snapshots",
		Status: http.StatusAccepted,
		Body:   `{"ok":true,"type":"snapshot.deleting","operations":[` + deletionOperation + `,` + deletionOperation + `]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("X-Ascii-Confirm-Delete"); got != sandboxID {
				t.Fatalf("confirm header = %q, want %q", got, sandboxID)
			}
		},
	})
	svc := snapshots.New(testutil.Transport(t, srv.URL, "boat_test_key"))

	ops, err := svc.DeleteForSandbox(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("DeleteForSandbox: %v", err)
	}
	if len(ops) != 2 || ops[0].ID != "del_1" {
		t.Fatalf("operations = %#v", ops)
	}
}

func TestService_DeleteForSandbox_RequiresSandboxID(t *testing.T) {
	t.Parallel()

	svc := snapshots.New(&testutil.Doer{})
	if _, err := svc.DeleteForSandbox(context.Background(), ""); err == nil {
		t.Fatal("DeleteForSandbox = nil, want id error")
	}
}
