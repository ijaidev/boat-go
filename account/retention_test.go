package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_DataRetention_ReturnsPolicy(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/account/data-retention",
		Body: `{
			"ok": true,
			"type": "data_retention.info",
			"enabled": true,
			"enabledAt": "2026-11-01T00:00:00Z",
			"snapshotsOff": false,
			"monitoringOff": true,
			"monitoringOffAt": "2026-11-02T00:00:00Z"
		}`,
	})

	got, err := svc.DataRetention(context.Background())
	if err != nil {
		t.Fatalf("DataRetention: %v", err)
	}
	if got.Type != model.DataRetentionInfo {
		t.Fatalf("type = %q, want data_retention.info", got.Type)
	}
	if !got.Enabled || got.EnabledAt == nil {
		t.Fatal("enabled policy missing")
	}
	if got.SnapshotsOff == nil || *got.SnapshotsOff {
		t.Fatal("snapshotsOff = nil or true, want explicit false")
	}
	if got.MonitoringOff == nil || !*got.MonitoringOff || got.MonitoringOffAt == nil {
		t.Fatal("monitoringOff missing")
	}
}

func TestService_UpdateDataRetention_SendsConfirmation(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/account/data-retention",
		Body: `{
			"ok": true,
			"type": "data_retention.updated",
			"enabled": true,
			"enabledAt": "2026-11-13T00:00:00Z",
			"queuedSandboxes": 2,
			"acceptedDeletionOperationsIrreversible": true
		}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body = readBody(t, r)
		},
	})

	got, err := svc.UpdateDataRetention(context.Background(), account.UpdateDataRetentionRequest{
		Enabled:      core.Ptr(true),
		Confirmation: account.DataRetentionConfirmation,
	})
	if err != nil {
		t.Fatalf("UpdateDataRetention: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("body = %v, want only enabled and confirmation", body)
	}
	if body["enabled"] != true {
		t.Fatalf("enabled = %v, want true", body["enabled"])
	}
	if body["confirmation"] != account.DataRetentionConfirmation {
		t.Fatalf("confirmation = %v, want the exact phrase", body["confirmation"])
	}
	if got.Type != model.DataRetentionUpdated {
		t.Fatalf("type = %q, want data_retention.updated", got.Type)
	}
	if got.QueuedSandboxes == nil || *got.QueuedSandboxes != 2 {
		t.Fatal("queuedSandboxes mismatch")
	}
	if got.AcceptedDeletionOperationsIrreversible == nil || !*got.AcceptedDeletionOperationsIrreversible {
		t.Fatal("acceptedDeletionOperationsIrreversible missing")
	}
}

func TestService_UpdateDataRetention_SendsExplicitFalse(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/account/data-retention",
		Body:   `{"ok":true,"type":"data_retention.updated","enabled":false,"enabledAt":null}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body = readBody(t, r)
		},
	})

	if _, err := svc.UpdateDataRetention(context.Background(), account.UpdateDataRetentionRequest{
		SnapshotsOff: core.Ptr(false),
	}); err != nil {
		t.Fatalf("UpdateDataRetention: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("body = %v, want only snapshotsOff", body)
	}
	if value, ok := body["snapshotsOff"]; !ok || value != false {
		t.Fatalf("snapshotsOff = %v, want explicit false", body["snapshotsOff"])
	}
}

func TestService_UpdateDataRetention_MapsAPIKeyForbidden(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPatch,
		Path:   "/account/data-retention",
		Status: http.StatusForbidden,
		Body: `{
			"ok": false,
			"type": "account.error",
			"status": 403,
			"code": "session_required",
			"message": "browser sign-in required"
		}`,
	})

	_, err := svc.UpdateDataRetention(context.Background(), account.UpdateDataRetentionRequest{
		MonitoringOff: core.Ptr(true),
	})
	if !core.IsCode(err, "session_required") {
		t.Fatalf("error = %v, want session_required", err)
	}
}
