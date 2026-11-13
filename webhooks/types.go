package webhooks

import (
	"fmt"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// CreateRequest is the body of POST /webhooks.
//
// URL and Events are required. Name has three states: nil omits it and lets the server choose,
// core.Some sets a name, and core.Null sends an explicit null (no name).
type CreateRequest struct {
	Name   *core.Optional[string]   `json:"name,omitempty"`
	URL    string                   `json:"url"`
	Events []model.WebhookEventType `json:"events"`
}

// UpdateRequest is the body of PATCH /webhooks/{webhookId}.
//
// Only the fields you set are sent; the API requires at least one. Name has three states: nil
// leaves it unchanged, core.Some replaces it, and core.Null removes it with an explicit null. An
// empty Events slice is omitted, not sent as an empty list.
type UpdateRequest struct {
	Name   *core.Optional[string]   `json:"name,omitempty"`
	URL    *string                  `json:"url,omitempty"`
	Events []model.WebhookEventType `json:"events,omitempty"`
}

// SecretResult is returned by Create and Rotate.
//
// Secret is the one-time signing secret. It is never returned by List, Get or Update, and it is
// never logged by the SDK. Its String and GoString methods redact the value so accidental
// formatting cannot leak it; pass Secret to Verify as-is.
type SecretResult struct {
	// Webhook is the endpoint metadata without the secret.
	Webhook model.Webhook
	// Secret is the one-time signing secret. Never log it.
	Secret string
}

// String implements fmt.Stringer and redacts the signing secret.
func (r SecretResult) String() string {
	return fmt.Sprintf("webhooks.SecretResult{Webhook: %q, Secret: [REDACTED]}", r.Webhook.ID)
}

// GoString implements fmt.GoStringer and redacts the signing secret.
func (r SecretResult) GoString() string {
	return r.String()
}
