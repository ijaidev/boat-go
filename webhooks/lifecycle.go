package webhooks

import (
	"context"
	"net/http"

	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// Delete removes an endpoint and its queued deliveries, and returns the deleted webhook id.
//
// An attempt already in flight when the request lands can still arrive, so a handler must tolerate
// a delivery for an endpoint it has just deleted.
func (s *Service) Delete(ctx context.Context, webhookID string, opts ...core.RequestOption) (string, error) {
	if err := requireID(webhookID); err != nil {
		return "", err
	}
	var out model.WebhookDeleteResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodDelete,
		Path:   webhookPath(webhookID),
	}, &out, opts...); err != nil {
		return "", err
	}
	return out.ID, nil
}

// Rotate replaces the endpoint's signing secret and returns the new one-time secret.
//
// Store the new secret before acknowledging the call. Attempts already in flight are signed with
// the old secret, so accept both secrets briefly after a rotation, then remove the old one. By
// default Rotate sends no Idempotency-Key, so the transport does not retry it.
func (s *Service) Rotate(ctx context.Context, webhookID string, opts ...core.RequestOption) (*SecretResult, error) {
	if err := requireID(webhookID); err != nil {
		return nil, err
	}
	var out model.WebhookSecretResponse
	if err := s.do(ctx, &core.Request{
		Method: http.MethodPost,
		Path:   webhookPath(webhookID, "rotate"),
	}, &out, opts...); err != nil {
		return nil, err
	}
	return &SecretResult{Webhook: out.Webhook, Secret: out.Secret}, nil
}
