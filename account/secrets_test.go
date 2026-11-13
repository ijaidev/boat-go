package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

func TestService_Secrets_ReturnsSetup(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/secrets",
		Body: `{
			"ok": true,
			"type": "secrets.info",
			"environmentId": "env_example",
			"envContents": "EXAMPLE_TOKEN=boat_test_value\n",
			"secretFiles": [{"path": ".config/example.json", "contents": "{\"type\":\"example\"}"}]
		}`,
	})

	got, err := svc.Secrets(context.Background())
	if err != nil {
		t.Fatalf("Secrets: %v", err)
	}
	if got.EnvironmentID != "env_example" {
		t.Fatalf("environmentId = %q, want env_example", got.EnvironmentID)
	}
	if got.EnvContents != "EXAMPLE_TOKEN=boat_test_value\n" {
		t.Fatal("env contents mismatch")
	}
	if len(got.SecretFiles) != 1 {
		t.Fatalf("secretFiles = %d, want 1", len(got.SecretFiles))
	}
	if got.SecretFiles[0].Path != ".config/example.json" {
		t.Fatal("secret file path mismatch")
	}
	if got.SecretFiles[0].Contents != `{"type":"example"}` {
		t.Fatal("secret file contents mismatch")
	}
}

func TestService_UpdateSecrets_SendsFullReplacement(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/secrets",
		Body: `{
			"ok": true,
			"type": "secrets.updated",
			"success": true,
			"environmentId": "env_example",
			"envContents": "EXAMPLE_TOKEN=boat_test_value\n",
			"secretFiles": [],
			"pushed": {"updated": 2, "failed": 0}
		}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			body = readBody(t, r)
		},
	})

	got, err := svc.UpdateSecrets(context.Background(), account.UpdateSecretsRequest{
		EnvContents: "EXAMPLE_TOKEN=boat_test_value\n",
		SecretFiles: []model.SecretFile{{
			Path:     ".config/example.json",
			Contents: `{"type":"example"}`,
		}},
	})
	if err != nil {
		t.Fatalf("UpdateSecrets: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("body keys = %d, want envContents and secretFiles", len(body))
	}
	if body["envContents"] != "EXAMPLE_TOKEN=boat_test_value\n" {
		t.Fatal("env contents mismatch")
	}
	files, ok := body["secretFiles"].([]any)
	if !ok || len(files) != 1 {
		t.Fatalf("secretFiles = %v", body["secretFiles"])
	}
	file, ok := files[0].(map[string]any)
	if !ok || file["path"] != ".config/example.json" {
		t.Fatalf("secret file = %v", files[0])
	}
	if got.Pushed == nil {
		t.Fatal("pushed = nil, want counts")
	}
}

func TestService_UpdateSecrets_SendsEmptyArrays(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/secrets",
		Body: `{"ok":true,"type":"secrets.updated","success":true,
			"environmentId":"env_example","envContents":"","secretFiles":[]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			body = readBody(t, r)
		},
	})

	if _, err := svc.UpdateSecrets(context.Background(), account.UpdateSecretsRequest{}); err != nil {
		t.Fatalf("UpdateSecrets: %v", err)
	}
	if body["envContents"] != "" {
		t.Fatalf("envContents = %v, want empty", body["envContents"])
	}
	files, ok := body["secretFiles"].([]any)
	if !ok || len(files) != 0 {
		t.Fatalf("secretFiles = %v, want an empty array", body["secretFiles"])
	}
}
