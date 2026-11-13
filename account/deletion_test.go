package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_DeletionOperation_ReturnsCurrentState(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/deletion-operations/bdop_example",
		Body: `{
			"ok": true,
			"type": "deletion.operation",
			"operation": {
				"id": "bdop_example",
				"kind": "snapshot",
				"targetId": "snap_example",
				"reason": "explicit",
				"status": "completed",
				"stage": "completed",
				"attemptCount": 1,
				"requestedAt": "2026-11-13T00:00:00Z",
				"completedAt": "2026-11-13T00:01:00Z"
			}
		}`,
	})

	got, err := svc.DeletionOperation(context.Background(), "bdop_example")
	if err != nil {
		t.Fatalf("DeletionOperation: %v", err)
	}
	if got.ID != "bdop_example" {
		t.Fatalf("id = %q, want bdop_example", got.ID)
	}
	if got.Kind != model.DeletionOperationKindSnapshot {
		t.Fatalf("kind = %q, want snapshot", got.Kind)
	}
	if got.Status != model.DeletionOperationStatusCompleted {
		t.Fatalf("status = %q, want completed", got.Status)
	}
	if got.Stage == nil || *got.Stage != model.DeletionOperationStageCompleted {
		t.Fatal("stage = nil or not completed")
	}
	if got.CompletedAt == nil || got.AttemptCount != 1 {
		t.Fatal("completion details missing")
	}
}

func TestService_DeletionOperation_RequiresOperationID(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	if _, err := account.New(d).DeletionOperation(context.Background(), ""); err == nil {
		t.Fatal("DeletionOperation = nil, want operation id error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("Do calls = %d, want 0", len(d.Calls))
	}
}
