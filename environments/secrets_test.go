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

func TestService_SetSecretFile_SendsPathAndContents(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/environments/" + envID + "/secret-files",
		Body:   `{"success":true,"versionId":"` + versionID + `","versionNumber":2}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "org", "team_example")
			requireBody(t, r, map[string]any{
				"path":     ".config/service-account.json",
				"contents": "test-secret",
			})
		},
	})

	got, err := svc.SetSecretFile(context.Background(), envID, environments.SecretFileRequest{
		Path:     ".config/service-account.json",
		Contents: "test-secret",
		Org:      "team_example",
	})
	if err != nil {
		t.Fatalf("SetSecretFile: %v", err)
	}
	if got.VersionNumber == nil || *got.VersionNumber != 2 {
		t.Fatalf("version number = %v", got.VersionNumber)
	}
}

func TestService_DeleteSecretFile_SendsPathQuery(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/environments/" + envID + "/secret-files",
		Body:   `{"success":true,"versionId":"` + versionID + `","versionNumber":3}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			requireQuery(t, r, "path", ".config/service-account.json")
			requireQuery(t, r, "org", "team_example")
			requireEmptyBody(t, r)
		},
	})

	got, err := svc.DeleteSecretFile(context.Background(), envID, ".config/service-account.json",
		environments.WithOrg("team_example"))
	if err != nil {
		t.Fatalf("DeleteSecretFile: %v", err)
	}
	if got.VersionNumber == nil || *got.VersionNumber != 3 {
		t.Fatalf("version number = %v", got.VersionNumber)
	}
}

func TestService_SetSecretFile_DoesNotLeakContentsInErrors(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPut,
		Path:   "/environments/" + envID + "/secret-files",
		Status: http.StatusBadRequest,
		Body:   `{"ok":false,"type":"sandbox.error","status":400,"code":"invalid_path","message":"rejected","requestId":"req_1"}`,
	})

	_, err := svc.SetSecretFile(context.Background(), envID, environments.SecretFileRequest{
		Path:     ".env",
		Contents: "test-secret",
	})
	if err == nil {
		t.Fatal("SetSecretFile = nil, want error")
	}
	if strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("error leaks secret contents: %v", err)
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.RequestID != "req_1" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func TestService_SecretFiles_RequireIDs(t *testing.T) {
	t.Parallel()

	svc := environments.New(&testutil.Doer{})
	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "set missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.SetSecretFile(ctx, "", environments.SecretFileRequest{Path: ".env"})
				return err
			},
		},
		{
			name: "set missing path",
			call: func(ctx context.Context) error {
				_, err := svc.SetSecretFile(ctx, envID, environments.SecretFileRequest{})
				return err
			},
		},
		{
			name: "delete missing environment",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteSecretFile(ctx, "", ".env")
				return err
			},
		},
		{
			name: "delete missing path",
			call: func(ctx context.Context) error {
				_, err := svc.DeleteSecretFile(ctx, envID, "")
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
