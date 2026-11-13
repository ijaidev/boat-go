package webhooks_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/ijaidev/boat-go/core"
)

const (
	webhookID  = "wh_0123456789abcdef01234567"
	testSecret = "whsec_test_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	webhookOne = `{
		"id": "wh_0123456789abcdef01234567",
		"name": "CI hooks",
		"url": "https://example.com/hooks/ci",
		"events": ["sandbox.ready", "sandbox.error"],
		"createdAt": "2026-11-13T10:00:00Z",
		"updatedAt": "2026-11-13T10:00:00Z"
	}`
	webhookTwo = `{
		"id": "wh_89abcdef0123456789abcdef",
		"name": null,
		"url": "https://example.com/hooks/audit",
		"events": ["sandbox.archived"],
		"createdAt": "2026-11-12T09:00:00Z",
		"updatedAt": "2026-11-12T09:00:00Z"
	}`
)

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return body
}

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()

	var apiErr *core.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *core.APIError", err)
	}
	if apiErr.Status != status || apiErr.Code != code {
		t.Fatalf("APIError = %#v, want status %d code %q", apiErr, status, code)
	}
}
