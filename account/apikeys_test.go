package account_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/account"
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/internal/testutil"
	"github.com/ijaidev/boat-go/model"
)

const exampleAPIKey = `{
	"id": "sak_example",
	"name": "CI worker",
	"credentialLane": "scoped-v1",
	"keyPrefix": "boat_live",
	"keyLastFour": "9abc",
	"sandboxId": null,
	"createdAt": "2026-05-31T12:00:00Z",
	"lastUsedAt": null,
	"usage": {"requests": 7, "windowDays": 30},
	"resources": {"total": 2, "sandboxes": 1, "agents": 1},
	"expiresAt": "2026-11-21T12:00:00Z",
	"expired": false,
	"expiringSoon": false,
	"scope": {
		"actions": ["sandbox.read"],
		"sandboxes": "*",
		"environments": "*",
		"grandfathered": false
	}
}`

const exampleAPIKeyList = `{"ok":true,"type":"api_key.list","apiKeys":[` + exampleAPIKey + `],"catalog":{
	"actions": ["sandbox.read", "*"],
	"presets": {"read-only": ["sandbox.read"]},
	"defaultTtl": "90d",
	"maxTtl": "365d",
	"scopedCreationEnabled": true
}}`

const exampleCreatedSecret = "boat_test_scoped_secret"

func TestService_APIKeys_ReturnsMetadataAndCatalog(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/api-keys",
		Body:   exampleAPIKeyList,
	})

	got, err := svc.APIKeys(context.Background())
	if err != nil {
		t.Fatalf("APIKeys: %v", err)
	}
	if len(got.APIKeys) != 1 {
		t.Fatalf("apiKeys = %d, want 1", len(got.APIKeys))
	}
	key := got.APIKeys[0]
	if key.ID != "sak_example" || key.CredentialLane != model.APIKeyCredentialLaneScopedV1 {
		t.Fatal("api key metadata mismatch")
	}
	if key.SandboxID != nil {
		t.Fatal("sandboxId = non-nil, want null")
	}
	if key.Usage.Requests != 7 || key.Resources.Total != 2 {
		t.Fatal("usage or resources mismatch")
	}
	if key.Scope == nil || key.Scope.Actions == nil || len(*key.Scope.Actions) != 1 {
		t.Fatal("scope actions missing")
	}
	if !got.Catalog.ScopedCreationEnabled {
		t.Fatal("scopedCreationEnabled = false, want true")
	}
	if got.Catalog.DefaultTTL != "90d" || got.Catalog.MaxTTL != "365d" {
		t.Fatal("catalog TTL mismatch")
	}
}

func TestService_CreateScopedAPIKey_SendsScopeAndReturnsSecret(t *testing.T) {
	t.Parallel()

	var body map[string]any
	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/api-keys/scoped",
		Status: http.StatusCreated,
		Body: `{"ok":true,"type":"api_key.created","apiKey":` + exampleAPIKey +
			`,"secret":"` + exampleCreatedSecret + `","apiKeys":[` + exampleAPIKey + `]}`,
		Assert: func(t *testing.T, r *http.Request) {
			t.Helper()
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			body = readBody(t, r)
		},
	})

	got, err := svc.CreateScopedAPIKey(context.Background(), account.CreateScopedAPIKeyRequest{
		Name:           "CI worker",
		TTL:            "24h",
		Preset:         "ci",
		SandboxIDs:     []string{"bx_example"},
		EnvironmentIDs: []string{"env_example"},
	})
	if err != nil {
		t.Fatalf("CreateScopedAPIKey: %v", err)
	}
	if body["name"] != "CI worker" || body["ttl"] != "24h" || body["preset"] != "ci" {
		t.Fatalf("body = %v", body)
	}
	if _, ok := body["actions"]; ok {
		t.Fatal("actions = present, want omitted")
	}
	sandboxes, ok := body["sandboxIds"].([]any)
	if !ok || len(sandboxes) != 1 || sandboxes[0] != "bx_example" {
		t.Fatalf("sandboxIds = %v", body["sandboxIds"])
	}
	environments, ok := body["environmentIds"].([]any)
	if !ok || len(environments) != 1 || environments[0] != "env_example" {
		t.Fatalf("environmentIds = %v", body["environmentIds"])
	}
	if got.Secret != exampleCreatedSecret {
		t.Fatal("secret mismatch")
	}
	if got.APIKey.ID != "sak_example" {
		t.Fatal("created key id mismatch")
	}
}

func TestService_CreateScopedAPIKey_RequiresScope(t *testing.T) {
	t.Parallel()

	d := &testutil.Doer{}
	svc := account.New(d)
	if _, err := svc.CreateScopedAPIKey(context.Background(), account.CreateScopedAPIKeyRequest{Name: "empty"}); err == nil {
		t.Fatal("CreateScopedAPIKey = nil, want scope error")
	}
	if len(d.Calls) != 0 {
		t.Fatalf("Do calls = %d, want 0", len(d.Calls))
	}
}

func TestService_CreateScopedAPIKey_MapsDisabled(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/api-keys/scoped",
		Status: http.StatusServiceUnavailable,
		Body: `{
			"error": "scoped_api_key_creation_disabled",
			"message": "scoped credential creation is not active",
			"scopedCreationEnabled": false
		}`,
	})

	_, err := svc.CreateScopedAPIKey(context.Background(), account.CreateScopedAPIKeyRequest{Preset: "read-only"})
	if err == nil {
		t.Fatal("CreateScopedAPIKey = nil, want an error")
	}
	var apiErr *core.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusServiceUnavailable {
		t.Fatalf("error = %v, want HTTP 503", err)
	}
}

func TestService_RotateAPIKey_ReturnsReplacementSecret(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodPost,
		Path:   "/api-keys/sak_example/rotate",
		Body: `{"ok":true,"type":"api_key.rotated","apiKey":` + exampleAPIKey +
			`,"secret":"` + exampleCreatedSecret + `","apiKeys":[` + exampleAPIKey + `]}`,
	})

	got, err := svc.RotateAPIKey(context.Background(), "sak_example")
	if err != nil {
		t.Fatalf("RotateAPIKey: %v", err)
	}
	if got.Secret != exampleCreatedSecret {
		t.Fatal("secret mismatch")
	}
	if got.APIKey.ID != "sak_example" {
		t.Fatal("rotated key id mismatch")
	}
}

func TestService_RevokeAPIKey_ReturnsRemainingKeys(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodDelete,
		Path:   "/api-keys/sak_example",
		Body:   `{"ok":true,"type":"api_key.revoked","apiKeys":[]}`,
	})

	got, err := svc.RevokeAPIKey(context.Background(), "sak_example")
	if err != nil {
		t.Fatalf("RevokeAPIKey: %v", err)
	}
	if len(got.APIKeys) != 0 {
		t.Fatalf("apiKeys = %d, want 0", len(got.APIKeys))
	}
}

func TestService_APIKeyUsage_ReturnsCreatedResources(t *testing.T) {
	t.Parallel()

	svc := newService(t, testutil.Route{
		Method: http.MethodGet,
		Path:   "/api-keys/sak_example/usage",
		Body: `{"ok":true,"type":"api_key.usage","id":"sak_example","name":"CI worker",
			"credentialLane":"scoped-v1","keyPrefix":"boat_live","keyLastFour":"9abc",
			"sandboxId":null,"createdAt":"2026-05-31T12:00:00Z","lastUsedAt":null,
			"usage":{"requests":7,"windowDays":30},"resources":{"total":1,"sandboxes":1,"agents":0},
			"createdResources":[{"kind":"sandbox","id":"bx_example","name":"CI build","state":"ready","createdAt":"2026-08-24T14:00:00Z"}]}`,
	})

	got, err := svc.APIKeyUsage(context.Background(), "sak_example")
	if err != nil {
		t.Fatalf("APIKeyUsage: %v", err)
	}
	if got.ID != "sak_example" || got.Usage.Requests != 7 {
		t.Fatal("usage mismatch")
	}
	if len(got.CreatedResources) != 1 {
		t.Fatalf("createdResources = %d, want 1", len(got.CreatedResources))
	}
	resource := got.CreatedResources[0]
	if resource.Kind != model.APIKeyCreatedResourceKindSandbox || resource.ID != "bx_example" {
		t.Fatal("created resource mismatch")
	}
}

func TestService_RequiresAPIKeyID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *account.Service) error
	}{
		{
			name: "rotate",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.RotateAPIKey(ctx, "")
				return err
			},
		},
		{
			name: "revoke",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.RevokeAPIKey(ctx, "")
				return err
			},
		},
		{
			name: "usage",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.APIKeyUsage(ctx, "")
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{}
			if err := tt.call(context.Background(), account.New(d)); err == nil {
				t.Fatal("call = nil, want api key id error")
			}
			if len(d.Calls) != 0 {
				t.Fatalf("Do calls = %d, want 0", len(d.Calls))
			}
		})
	}
}

func TestService_MutatingMethodsDisableRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, *account.Service) error
	}{
		{
			name: "create scoped key",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.CreateScopedAPIKey(ctx, account.CreateScopedAPIKeyRequest{Preset: "read-only"})
				return err
			},
		},
		{
			name: "rotate key",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.RotateAPIKey(ctx, "sak_example")
				return err
			},
		},
		{
			name: "revoke key",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.RevokeAPIKey(ctx, "sak_example")
				return err
			},
		},
		{
			name: "update data retention",
			call: func(ctx context.Context, svc *account.Service) error {
				_, err := svc.UpdateDataRetention(ctx, account.UpdateDataRetentionRequest{
					Enabled: core.Ptr(false),
				})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &testutil.Doer{DoFunc: func(context.Context, *core.Request, any, ...core.RequestOption) error {
				return nil
			}}
			if err := tt.call(context.Background(), account.New(d)); err != nil {
				t.Fatalf("call: %v", err)
			}
			if len(d.Calls) != 1 {
				t.Fatalf("Do calls = %d, want 1", len(d.Calls))
			}
			if !d.Calls[0].NoRetry {
				t.Fatal("NoRetry = false, want true")
			}
		})
	}
}
