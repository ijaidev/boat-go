package webhooks

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Create registers an account-wide endpoint for sandbox lifecycle events.
//
// The endpoint must use public HTTPS on port 443. An account can register at most 10 unique URLs.
// The response carries the one-time signing secret; store it now, because List, Get and Update
// never return it. By default Create sends no Idempotency-Key, so the transport does not retry it: a
// failed registration is surfaced once and may be reconciled with List.
func (s *Service) Create(ctx context.Context, req CreateRequest, opts ...core.RequestOption) (*SecretResult, error) {
	var out model.WebhookSecretResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   "/webhooks",
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &SecretResult{Webhook: out.Webhook, Secret: out.Secret}, nil
}

// Update changes the name, endpoint URL or subscribed events without rotating the signing secret.
//
// At least one field must be set; the API rejects an empty body with 400. An update that sets no
// name, URL or events sends "{}" and surfaces that error unchanged.
func (s *Service) Update(ctx context.Context, webhookID string, req UpdateRequest, opts ...core.RequestOption) (*model.Webhook, error) {
	if err := requireID(webhookID); err != nil {
		return nil, err
	}
	var out model.WebhookResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPatch,
		Path:   webhookPath(webhookID),
		JSON:   req,
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Webhook, nil
}
