package environments_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/environments"
	"github.com/ijaidev/boat-go/internal/testutil"
)

func TestService_SetVar_SendsValueAndVersion(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/environments/" + envID + "/vars/API_TOKEN",
		Body:   `{"success":true,"versionId":"` + versionID + `","versionNumber":3}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireBody(t, r, map[string]any{"value": "test-secret"})
		},
	})

	got, err := svc.SetVar(context.Background(), envID, "API_TOKEN", "test-secret")
	if err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if got.VersionNumber == nil || *got.VersionNumber != 3 {
		t.Fatalf("version number = %v", got.VersionNumber)
	}
	if got.VersionID == nil || *got.VersionID != versionID {
		t.Fatalf("version id = %v", got.VersionID)
	}
}

func TestService_DeleteVar_SendsOrg(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/environments/" + envID + "/vars/API_TOKEN",
		Body:   `{"success":true,"versionId":"` + versionID + `","versionNumber":4}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireEmptyBody(t, r)
		},
	})

	got, err := svc.DeleteVar(context.Background(), envID, "API_TOKEN", environments.WithOrg("team_example"))
	if err != nil {
		t.Fatalf("DeleteVar: %v", err)
	}
	if got.VersionNumber == nil || *got.VersionNumber != 4 {
		t.Fatalf("version number = %v", got.VersionNumber)
	}
}

func TestService_SetVar_DoesNotLeakValueInErrors(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/environments/" + envID + "/vars/API_TOKEN",
		Status: http.StatusBadRequest,
		Body:   `{"ok":false,"type":"sandbox.error","status":400,"code":"invalid_key","message":"rejected","requestId":"req_1"}`,
	})

	_, err := svc.SetVar(context.Background(), envID, "API_TOKEN", "test-secret")
	if err == nil {
		t.Fatal("SetVar = nil, want error")
	}
	if strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("error leaks variable value: %v", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.RequestID != "req_1" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func TestService_Vars_RequireIDs(t *testing.T) {
	t.Parallel()

	svc := environments.New(&testutil.Doer{})
	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "set missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.SetVar(ctx, "", "KEY", "value")
				return err
			},
		},
		{
			name: "set missing key",
			call: func(ctx context.Context) error {
				_, err := svc.SetVar(ctx, envID, "", "value")
				return err
			},
		},
		{
			name: "delete missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteVar(ctx, "", "KEY")
				return err
			},
		},
		{
			name: "delete missing key",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteVar(ctx, envID, "")
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
